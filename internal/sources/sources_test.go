package sources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Retr0-XD/StateLedger/internal/collectors"
)

func TestCaptureEnvironment(t *testing.T) {
	payload, err := CaptureEnvironment()
	if err != nil {
		t.Fatalf("CaptureEnvironment() error = %v", err)
	}

	if payload.OS != runtime.GOOS {
		t.Errorf("OS = %v, want %v", payload.OS, runtime.GOOS)
	}
	if payload.Arch != runtime.GOARCH {
		t.Errorf("Arch = %v, want %v", payload.Arch, runtime.GOARCH)
	}
	if payload.Runtime != runtime.Version() {
		t.Errorf("Runtime = %v, want %v", payload.Runtime, runtime.Version())
	}
	if payload.TimeSource != "system" {
		t.Errorf("TimeSource = %v, want system", payload.TimeSource)
	}

	// Validate the payload
	if err := payload.Validate(); err != nil {
		t.Errorf("Generated payload should be valid: %v", err)
	}
}

func TestCaptureConfig(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "app.yaml")
		content := "key: value\nport: 8080"

		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		payload, err := CaptureConfig(configPath)
		if err != nil {
			t.Fatalf("CaptureConfig() error = %v", err)
		}

		if payload.Source != configPath {
			t.Errorf("Source = %v, want %v", payload.Source, configPath)
		}
		if payload.Snapshot != content {
			t.Errorf("Snapshot = %v, want %v", payload.Snapshot, content)
		}
		if !strings.HasPrefix(payload.Hash, "sha256:") {
			t.Errorf("Hash should start with sha256:, got %v", payload.Hash)
		}
		if payload.Version != "1" {
			t.Errorf("Version = %v, want 1", payload.Version)
		}

		// Validate the payload
		if err := payload.Validate(); err != nil {
			t.Errorf("Generated payload should be valid: %v", err)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := CaptureConfig("/nonexistent/file.yaml")
		if err == nil {
			t.Error("CaptureConfig() should have failed on nonexistent file")
		}
	})

	t.Run("empty source", func(t *testing.T) {
		_, err := CaptureConfig("")
		if err == nil {
			t.Error("CaptureConfig() should have failed on empty source")
		}
		if !strings.Contains(err.Error(), "required") {
			t.Errorf("Expected 'required' error, got: %v", err)
		}
	})
}

func TestComputeConfigHash(t *testing.T) {
	// Hash should be deterministic
	content := "test content"
	hash1 := computeConfigHash(content)
	hash2 := computeConfigHash(content)

	if hash1 != hash2 {
		t.Errorf("Hash should be deterministic: %v != %v", hash1, hash2)
	}

	if !strings.HasPrefix(hash1, "sha256:") {
		t.Errorf("Hash should start with sha256:, got %v", hash1)
	}

	// Different content should produce different hash
	hash3 := computeConfigHash("different content")
	if hash1 == hash3 {
		t.Error("Different content should produce different hash")
	}
}

func TestCaptureFromManifest(t *testing.T) {
	t.Run("environment collector", func(t *testing.T) {
		result, err := CaptureFromManifest("environment", "", nil)
		if err != nil {
			t.Fatalf("CaptureFromManifest() error = %v", err)
		}

		if result.Kind != "environment" {
			t.Errorf("Kind = %v, want environment", result.Kind)
		}
		if result.Error != "" {
			t.Errorf("Error should be empty, got: %v", result.Error)
		}
		if result.Payload == "" {
			t.Error("Payload should not be empty")
		}

		// Verify payload is valid JSON
		var payload collectors.EnvironmentPayload
		if err := json.Unmarshal([]byte(result.Payload), &payload); err != nil {
			t.Errorf("Payload should be valid JSON: %v", err)
		}
	})

	t.Run("config collector", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte("key: value"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		result, err := CaptureFromManifest("config", configPath, nil)
		if err != nil {
			t.Fatalf("CaptureFromManifest() error = %v", err)
		}

		if result.Kind != "config" {
			t.Errorf("Kind = %v, want config", result.Kind)
		}
		if result.Source != configPath {
			t.Errorf("Source = %v, want %v", result.Source, configPath)
		}
		if result.Error != "" {
			t.Errorf("Error should be empty, got: %v", result.Error)
		}
		if result.Payload == "" {
			t.Error("Payload should not be empty")
		}

		// Verify payload is valid JSON
		var payload collectors.ConfigPayload
		if err := json.Unmarshal([]byte(result.Payload), &payload); err != nil {
			t.Errorf("Payload should be valid JSON: %v", err)
		}
	})

	t.Run("config collector with error", func(t *testing.T) {
		result, err := CaptureFromManifest("config", "/nonexistent/file", nil)
		if err != nil {
			t.Fatalf("CaptureFromManifest() should not return error, got: %v", err)
		}

		if result.Kind != "config" {
			t.Errorf("Kind = %v, want config", result.Kind)
		}
		if result.Error == "" {
			t.Error("Error should be set when capture fails")
		}
		if result.Payload != "" {
			t.Errorf("Payload should be empty on error, got: %v", result.Payload)
		}
	})

	t.Run("unsupported kind", func(t *testing.T) {
		_, err := CaptureFromManifest("unsupported", "source", nil)
		if err == nil {
			t.Error("CaptureFromManifest() should have failed on unsupported kind")
		}
		if !strings.Contains(err.Error(), "unsupported") {
			t.Errorf("Expected 'unsupported' error, got: %v", err)
		}
	})
}

// TestCaptureGit is intentionally minimal since it requires a real git repo.
// More comprehensive tests would need integration test setup with git.
func TestCaptureGit(t *testing.T) {
	t.Run("not a git repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, err := CaptureGit(tmpDir)
		if err == nil {
			t.Error("CaptureGit() should have failed on non-git directory")
		}
		if !strings.Contains(err.Error(), "not a git repository") {
			t.Errorf("Expected 'not a git repository' error, got: %v", err)
		}
	})
}
