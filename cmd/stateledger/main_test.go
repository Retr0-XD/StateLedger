package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLIWorkflow tests the full CLI workflow end-to-end
func TestCLIWorkflow(t *testing.T) {
	// Build the binary first
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "stateledger")
	
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, output)
	}

	// Create test directory structure
	testDir := filepath.Join(tmpDir, "test-ledger")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	dbPath := filepath.Join(testDir, "state.db")
	artifactsDir := filepath.Join(testDir, "artifacts")

	t.Run("init", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "init", "-db", dbPath, "-artifacts", artifactsDir)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("init command failed: %v\n%s", err, output)
		}
		
		// Verify database was created
		if _, err := os.Stat(dbPath); err != nil {
			t.Errorf("Database file not created: %v", err)
		}
		
		// Verify artifacts dir was created
		if _, err := os.Stat(artifactsDir); err != nil {
			t.Errorf("Artifacts dir not created: %v", err)
		}
	})

	t.Run("manifest create", func(t *testing.T) {
		manifestPath := filepath.Join(testDir, "manifest.json")
		cmd := exec.Command(binaryPath, "manifest", "create", "-output", manifestPath, "-name", "test-manifest")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("manifest create failed: %v\n%s", err, output)
		}
		
		// Verify manifest file was created
		if _, err := os.Stat(manifestPath); err != nil {
			t.Errorf("Manifest file not created: %v", err)
		}
		
		// Verify it's valid JSON
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatalf("Failed to read manifest: %v", err)
		}
		
		var manifest map[string]any
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Errorf("Manifest is not valid JSON: %v", err)
		}
	})

	t.Run("collect environment", func(t *testing.T) {
		// Capture environment first
		capture := exec.Command(binaryPath, "capture", "-kind", "environment", "-path", "/tmp")
		payload, err := capture.CombinedOutput()
		if err != nil {
			t.Fatalf("capture failed: %v\n%s", err, payload)
		}
		
		// Extract payload JSON from capture result
		var captureResult map[string]any
		if err := json.Unmarshal(payload, &captureResult); err != nil {
			t.Fatalf("Capture output should be JSON: %v", err)
		}
		
		payloadJSON := captureResult["payload"].(string)
		
		// Collect it to DB
		cmd := exec.Command(binaryPath, "collect", "-db", dbPath, "-kind", "environment", "-payload-json", payloadJSON)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("collect command failed: %v\n%s", err, output)
		}
		
		// Output is JSON record
		var result map[string]any
		if err := json.Unmarshal(output, &result); err != nil {
			t.Errorf("Output should be valid JSON: %v\nGot: %s", err, output)
		}
	})

	t.Run("collect config", func(t *testing.T) {
		// Create a test config file
		configPath := filepath.Join(testDir, "test.yaml")
		if err := os.WriteFile(configPath, []byte("key: value\nport: 8080"), 0644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}
		
		// Capture config first
		capture := exec.Command(binaryPath, "capture", "-kind", "config", "-path", configPath)
		payload, err := capture.CombinedOutput()
		if err != nil {
			t.Fatalf("capture failed: %v\n%s", err, payload)
		}
		
		// Extract payload JSON
		var captureResult map[string]any
		if err := json.Unmarshal(payload, &captureResult); err != nil {
			t.Fatalf("Capture output should be JSON: %v", err)
		}
		
		payloadJSON := captureResult["payload"].(string)
		
		// Collect it to DB
		cmd := exec.Command(binaryPath, "collect", "-db", dbPath, "-kind", "config", "-source", configPath, "-payload-json", payloadJSON)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("collect config failed: %v\n%s", err, output)
		}
		
		// Output is JSON record
		var result map[string]any
		if err := json.Unmarshal(output, &result); err != nil {
			t.Errorf("Output should be valid JSON: %v\nGot: %s", err, output)
		}
	})

	t.Run("query records", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "query", "-db", dbPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("query command failed: %v\n%s", err, output)
		}
		
		// Output is newline-separated JSON objects (not a JSON array)
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) < 2 {
			t.Errorf("Expected at least 2 records, got %d lines", len(lines))
		}
		
		// Verify each line is valid JSON
		for i, line := range lines {
			if line == "" {
				continue
			}
			var record map[string]any
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				t.Errorf("Line %d should be valid JSON: %v\nGot: %s", i, err, line)
			}
		}
	})

	t.Run("verify chain", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "verify", "-db", dbPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("verify command failed: %v\n%s", err, output)
		}
		
		// Output is JSON
		var result map[string]any
		if err := json.Unmarshal(output, &result); err != nil {
			t.Errorf("Output should be valid JSON: %v", err)
		}
		
		// Should have "ok": true
		if ok, exists := result["ok"].(bool); !exists || !ok {
			t.Errorf("Expected ok: true in verification result, got: %s", output)
		}
	})

	t.Run("snapshot", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "snapshot", "-db", dbPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("snapshot command failed: %v\n%s", err, output)
		}
		
		// Output is JSON
		var result map[string]any
		if err := json.Unmarshal(output, &result); err != nil {
			t.Errorf("Output should be valid JSON: %v", err)
		}
		
		// Should have determinism_score field
		if _, exists := result["determinism_score"]; !exists {
			t.Errorf("Expected determinism_score field, got: %s", output)
		}
	})

	t.Run("advisory", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "advisory", "-db", dbPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("advisory command failed: %v\n%s", err, output)
		}
		
		// Advisory outputs formatted text with sections
		outStr := string(output)
		if !strings.Contains(outStr, "Advisory") && !strings.Contains(outStr, "Explanation") {
			t.Errorf("Expected advisory output sections, got: %s", outStr)
		}
	})

	t.Run("audit bundle", func(t *testing.T) {
		auditPath := filepath.Join(testDir, "audit.json")
		cmd := exec.Command(binaryPath, "audit", "-db", dbPath, "-out", auditPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("audit command failed: %v\n%s", err, output)
		}
		
		// Verify audit bundle was created
		if _, err := os.Stat(auditPath); err != nil {
			t.Errorf("Audit bundle not created: %v", err)
		}
		
		// Verify it's valid JSON
		data, err := os.ReadFile(auditPath)
		if err != nil {
			t.Fatalf("Failed to read audit bundle: %v", err)
		}
		
		var bundle map[string]any
		if err := json.Unmarshal(data, &bundle); err != nil {
			t.Errorf("Audit bundle is not valid JSON: %v", err)
		}
		
		// Verify bundle has expected fields
		if _, ok := bundle["snapshot"]; !ok {
			t.Error("Audit bundle missing 'snapshot' field")
		}
		if _, ok := bundle["proof"]; !ok {
			t.Error("Audit bundle missing 'proof' field")
		}
	})

	t.Run("artifact store", func(t *testing.T) {
		// Create a test artifact file
		artifactFile := filepath.Join(testDir, "test-artifact.bin")
		content := []byte("test artifact content")
		if err := os.WriteFile(artifactFile, content, 0644); err != nil {
			t.Fatalf("Failed to create test artifact: %v", err)
		}
		
		cmd := exec.Command(binaryPath, "artifact", "put", "-artifacts", artifactsDir, "-file", artifactFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("artifact put failed: %v\n%s", err, output)
		}
		
		// Output is JSON
		var result map[string]any
		if err := json.Unmarshal(output, &result); err != nil {
			t.Errorf("Output should be valid JSON: %v\nGot: %s", err, output)
		}
		
		// Should have checksum field
		if _, exists := result["checksum"]; !exists {
			t.Errorf("Expected checksum field, got: %s", output)
		}
	})
}

// TestCLIErrorHandling tests error cases
func TestCLIErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "stateledger")
	
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, output)
	}

	t.Run("init without required flags", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "init")
		_, err := cmd.CombinedOutput()
		// Init has defaults, so it won't fail
		if err != nil {
			// This is actually OK - some systems might not allow writing to default paths
			t.Logf("Init without flags failed as expected: %v", err)
		}
	})

	t.Run("verify nonexistent database", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "verify", "-db", "/nonexistent/db.sqlite")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("verify should fail on nonexistent database")
		}
		_ = output // Error is expected
	})

	t.Run("collect without kind", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "collect", "-db", "/tmp/test.db")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("collect should fail without -kind flag")
		}
		if !strings.Contains(string(output), "kind") {
			t.Errorf("Error message should mention missing kind, got: %s", output)
		}
	})

	t.Run("manifest run nonexistent file", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "manifest", "run", "-db", "/tmp/test.db", "-manifest", "/nonexistent/manifest.json")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("manifest run should fail on nonexistent file")
		}
		_ = output // Error is expected
	})
}
