package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestStore(t *testing.T) {
	t.Run("store new artifact", func(t *testing.T) {
		tmpDir := t.TempDir()
		storeRoot := filepath.Join(tmpDir, "artifacts")
		if err := os.MkdirAll(storeRoot, 0755); err != nil {
			t.Fatalf("Failed to create store root: %v", err)
		}

		// Create test file
		testFile := filepath.Join(tmpDir, "test.txt")
		content := []byte("test content")
		if err := os.WriteFile(testFile, content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Store the artifact
		artifact, err := Store(storeRoot, testFile)
		if err != nil {
			t.Fatalf("Store() error = %v", err)
		}

		// Verify checksum
		expectedHash := sha256.Sum256(content)
		expectedChecksum := hex.EncodeToString(expectedHash[:])
		if artifact.Checksum != expectedChecksum {
			t.Errorf("Checksum = %v, want %v", artifact.Checksum, expectedChecksum)
		}

		// Verify size
		if artifact.Size != int64(len(content)) {
			t.Errorf("Size = %v, want %v", artifact.Size, len(content))
		}

		// Verify file was written to correct location
		expectedPath := filepath.Join(storeRoot, expectedChecksum)
		if artifact.Path != expectedPath {
			t.Errorf("Path = %v, want %v", artifact.Path, expectedPath)
		}

		// Verify content was written correctly
		stored, err := os.ReadFile(artifact.Path)
		if err != nil {
			t.Fatalf("Failed to read stored artifact: %v", err)
		}
		if string(stored) != string(content) {
			t.Errorf("Stored content = %v, want %v", string(stored), string(content))
		}
	})

	t.Run("nonexistent source file", func(t *testing.T) {
		tmpDir := t.TempDir()
		storeRoot := filepath.Join(tmpDir, "artifacts")
		if err := os.MkdirAll(storeRoot, 0755); err != nil {
			t.Fatalf("Failed to create store root: %v", err)
		}

		_, err := Store(storeRoot, "/nonexistent/file")
		if err == nil {
			t.Error("Store() should have failed on nonexistent file")
		}
	})

	t.Run("deduplicate identical files", func(t *testing.T) {
		tmpDir := t.TempDir()
		storeRoot := filepath.Join(tmpDir, "artifacts")
		if err := os.MkdirAll(storeRoot, 0755); err != nil {
			t.Fatalf("Failed to create store root: %v", err)
		}

		content := []byte("identical content")

		// Create two files with same content
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")
		if err := os.WriteFile(file1, content, 0644); err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}
		if err := os.WriteFile(file2, content, 0644); err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}

		// Store both
		artifact1, err := Store(storeRoot, file1)
		if err != nil {
			t.Fatalf("Store(file1) error = %v", err)
		}

		artifact2, err := Store(storeRoot, file2)
		if err != nil {
			t.Fatalf("Store(file2) error = %v", err)
		}

		// Both should have same checksum and path (deduplication)
		if artifact1.Checksum != artifact2.Checksum {
			t.Errorf("Checksums should match: %v != %v", artifact1.Checksum, artifact2.Checksum)
		}
		if artifact1.Path != artifact2.Path {
			t.Errorf("Paths should match: %v != %v", artifact1.Path, artifact2.Path)
		}
	})
}

func TestRetrieve(t *testing.T) {
	t.Run("retrieve existing artifact", func(t *testing.T) {
		tmpDir := t.TempDir()
		storeRoot := filepath.Join(tmpDir, "artifacts")
		if err := os.MkdirAll(storeRoot, 0755); err != nil {
			t.Fatalf("Failed to create store root: %v", err)
		}

		// Store an artifact first
		testFile := filepath.Join(tmpDir, "test.txt")
		content := []byte("test content")
		if err := os.WriteFile(testFile, content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		artifact, err := Store(storeRoot, testFile)
		if err != nil {
			t.Fatalf("Store() error = %v", err)
		}

		// Retrieve it
		path, err := Retrieve(storeRoot, artifact.Checksum)
		if err != nil {
			t.Fatalf("Retrieve() error = %v", err)
		}

		if path != artifact.Path {
			t.Errorf("Retrieved path = %v, want %v", path, artifact.Path)
		}

		// Verify content
		retrieved, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read retrieved artifact: %v", err)
		}
		if string(retrieved) != string(content) {
			t.Errorf("Retrieved content = %v, want %v", string(retrieved), string(content))
		}
	})

	t.Run("nonexistent checksum", func(t *testing.T) {
		tmpDir := t.TempDir()
		storeRoot := filepath.Join(tmpDir, "artifacts")
		if err := os.MkdirAll(storeRoot, 0755); err != nil {
			t.Fatalf("Failed to create store root: %v", err)
		}

		_, err := Retrieve(storeRoot, "nonexistentchecksum")
		if err == nil {
			t.Error("Retrieve() should have failed on nonexistent checksum")
		}
	})
}

func TestExists(t *testing.T) {
	t.Run("artifact exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		storeRoot := filepath.Join(tmpDir, "artifacts")
		if err := os.MkdirAll(storeRoot, 0755); err != nil {
			t.Fatalf("Failed to create store root: %v", err)
		}

		// Store an artifact
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		artifact, err := Store(storeRoot, testFile)
		if err != nil {
			t.Fatalf("Store() error = %v", err)
		}

		// Check existence
		if !Exists(storeRoot, artifact.Checksum) {
			t.Error("Exists() should return true for stored artifact")
		}
	})

	t.Run("artifact does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		storeRoot := filepath.Join(tmpDir, "artifacts")
		if err := os.MkdirAll(storeRoot, 0755); err != nil {
			t.Fatalf("Failed to create store root: %v", err)
		}

		if Exists(storeRoot, "nonexistentchecksum") {
			t.Error("Exists() should return false for nonexistent artifact")
		}
	})
}
