package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectorValidation(t *testing.T) {
	tests := []struct {
		name    string
		col     Collector
		wantErr bool
	}{
		{
			name:    "valid code collector",
			col:     Collector{Kind: "code", Source: "/path/to/repo"},
			wantErr: false,
		},
		{
			name:    "valid config collector",
			col:     Collector{Kind: "config", Source: "app.yaml"},
			wantErr: false,
		},
		{
			name:    "valid environment collector without source",
			col:     Collector{Kind: "environment"},
			wantErr: false,
		},
		{
			name:    "valid mutation collector",
			col:     Collector{Kind: "mutation", Source: "kafka://topic"},
			wantErr: false,
		},
		{
			name:    "invalid kind",
			col:     Collector{Kind: "invalid", Source: "source"},
			wantErr: true,
		},
		{
			name:    "code collector missing source",
			col:     Collector{Kind: "code"},
			wantErr: true,
		},
		{
			name:    "config collector missing source",
			col:     Collector{Kind: "config"},
			wantErr: true,
		},
		{
			name:    "mutation collector missing source",
			col:     Collector{Kind: "mutation"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.col.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManifestValidation(t *testing.T) {
	tests := []struct {
		name    string
		m       Manifest
		wantErr bool
	}{
		{
			name: "valid manifest",
			m: Manifest{
				Version: "1.0",
				Name:    "test-manifest",
				Collectors: []Collector{
					{Kind: "code", Source: "/repo"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			m: Manifest{
				Name: "test-manifest",
				Collectors: []Collector{
					{Kind: "code", Source: "/repo"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			m: Manifest{
				Version: "1.0",
				Collectors: []Collector{
					{Kind: "code", Source: "/repo"},
				},
			},
			wantErr: true,
		},
		{
			name: "no collectors",
			m: Manifest{
				Version:    "1.0",
				Name:       "test-manifest",
				Collectors: []Collector{},
			},
			wantErr: true,
		},
		{
			name: "invalid collector",
			m: Manifest{
				Version: "1.0",
				Name:    "test-manifest",
				Collectors: []Collector{
					{Kind: "invalid", Source: "/repo"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.m.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewManifest(t *testing.T) {
	m := NewManifest("test-manifest")
	if m.Version != "1.0" {
		t.Errorf("Version = %v, want 1.0", m.Version)
	}
	if m.Name != "test-manifest" {
		t.Errorf("Name = %v, want test-manifest", m.Name)
	}
	if len(m.Collectors) != 0 {
		t.Errorf("Collectors length = %v, want 0", len(m.Collectors))
	}
}

func TestAddCollector(t *testing.T) {
	m := NewManifest("test")
	m.AddCollector("code", "/repo", map[string]string{"branch": "main"})

	if len(m.Collectors) != 1 {
		t.Fatalf("Expected 1 collector, got %d", len(m.Collectors))
	}

	col := m.Collectors[0]
	if col.Kind != "code" {
		t.Errorf("Kind = %v, want code", col.Kind)
	}
	if col.Source != "/repo" {
		t.Errorf("Source = %v, want /repo", col.Source)
	}
	if col.Params["branch"] != "main" {
		t.Errorf("Params[branch] = %v, want main", col.Params["branch"])
	}
}

func TestManifestToJSON(t *testing.T) {
	m := NewManifest("test")
	m.AddCollector("code", "/repo", nil)

	jsonStr, err := m.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify it's valid JSON and can be unmarshaled
	var decoded Manifest
	if err := json.Unmarshal([]byte(jsonStr), &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if decoded.Name != "test" {
		t.Errorf("Decoded name = %v, want test", decoded.Name)
	}
	if len(decoded.Collectors) != 1 {
		t.Errorf("Decoded collectors length = %v, want 1", len(decoded.Collectors))
	}
}

func TestLoadManifest(t *testing.T) {
	t.Run("valid manifest file", func(t *testing.T) {
		// Create temp file
		tmpDir := t.TempDir()
		manifestPath := filepath.Join(tmpDir, "manifest.json")

		content := `{
  "version": "1.0",
  "name": "test-manifest",
  "collectors": [
    {
      "kind": "code",
      "source": "/repo"
    },
    {
      "kind": "environment"
    }
  ]
}`
		if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		m, err := LoadManifest(manifestPath)
		if err != nil {
			t.Fatalf("LoadManifest() error = %v", err)
		}

		if m.Name != "test-manifest" {
			t.Errorf("Name = %v, want test-manifest", m.Name)
		}
		if len(m.Collectors) != 2 {
			t.Errorf("Collectors length = %v, want 2", len(m.Collectors))
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := filepath.Join(tmpDir, "invalid.json")

		if err := os.WriteFile(manifestPath, []byte("{invalid json"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err := LoadManifest(manifestPath)
		if err == nil {
			t.Error("LoadManifest() should have failed on invalid JSON")
		}
	})

	t.Run("invalid manifest", func(t *testing.T) {
		tmpDir := t.TempDir()
		manifestPath := filepath.Join(tmpDir, "invalid-manifest.json")

		content := `{
  "version": "1.0",
  "name": "",
  "collectors": []
}`
		if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err := LoadManifest(manifestPath)
		if err == nil {
			t.Error("LoadManifest() should have failed on invalid manifest")
		}
		if !strings.Contains(err.Error(), "name is required") && !strings.Contains(err.Error(), "at least one collector") {
			t.Errorf("Expected validation error, got: %v", err)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := LoadManifest("/nonexistent/path")
		if err == nil {
			t.Error("LoadManifest() should have failed on nonexistent file")
		}
	})
}
