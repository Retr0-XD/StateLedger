package collectors

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCodePayloadValidation(t *testing.T) {
	tests := []struct {
		name    string
		payload CodePayload
		wantErr bool
	}{
		{
			name: "valid payload",
			payload: CodePayload{
				Repo:   "myrepo",
				Commit: "abc123",
			},
			wantErr: false,
		},
		{
			name: "missing repo",
			payload: CodePayload{
				Commit: "abc123",
			},
			wantErr: true,
		},
		{
			name: "missing commit",
			payload: CodePayload{
				Repo: "myrepo",
			},
			wantErr: true,
		},
		{
			name: "whitespace only repo",
			payload: CodePayload{
				Repo:   "  ",
				Commit: "abc123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigPayloadValidation(t *testing.T) {
	tests := []struct {
		name    string
		payload ConfigPayload
		wantErr bool
	}{
		{
			name: "valid payload",
			payload: ConfigPayload{
				Source:   "app.yaml",
				Version:  "1.0",
				Hash:     "abc123",
				Snapshot: "key: value",
			},
			wantErr: false,
		},
		{
			name: "missing source",
			payload: ConfigPayload{
				Hash:     "abc123",
				Snapshot: "key: value",
			},
			wantErr: true,
		},
		{
			name: "missing hash",
			payload: ConfigPayload{
				Source:   "app.yaml",
				Snapshot: "key: value",
			},
			wantErr: true,
		},
		{
			name: "missing snapshot",
			payload: ConfigPayload{
				Source: "app.yaml",
				Hash:   "abc123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvironmentPayloadValidation(t *testing.T) {
	tests := []struct {
		name    string
		payload EnvironmentPayload
		wantErr bool
	}{
		{
			name: "valid payload",
			payload: EnvironmentPayload{
				OS:         "linux",
				Kernel:     "5.15.0",
				Runtime:    "go1.21",
				Arch:       "amd64",
				TimeSource: "ntp",
			},
			wantErr: false,
		},
		{
			name: "missing os",
			payload: EnvironmentPayload{
				Runtime:    "go1.21",
				Arch:       "amd64",
				TimeSource: "ntp",
			},
			wantErr: true,
		},
		{
			name: "missing runtime",
			payload: EnvironmentPayload{
				OS:         "linux",
				Arch:       "amd64",
				TimeSource: "ntp",
			},
			wantErr: true,
		},
		{
			name: "missing arch",
			payload: EnvironmentPayload{
				OS:         "linux",
				Runtime:    "go1.21",
				TimeSource: "ntp",
			},
			wantErr: true,
		},
		{
			name: "missing time_source",
			payload: EnvironmentPayload{
				OS:      "linux",
				Runtime: "go1.21",
				Arch:    "amd64",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMutationPayloadValidation(t *testing.T) {
	tests := []struct {
		name    string
		payload MutationPayload
		wantErr bool
	}{
		{
			name: "valid payload",
			payload: MutationPayload{
				Type:   "kafka",
				ID:     "event-123",
				Source: "topic-orders",
				Hash:   "abc123",
			},
			wantErr: false,
		},
		{
			name: "missing type",
			payload: MutationPayload{
				ID:     "event-123",
				Source: "topic-orders",
			},
			wantErr: true,
		},
		{
			name: "missing id",
			payload: MutationPayload{
				Type:   "kafka",
				Source: "topic-orders",
			},
			wantErr: true,
		},
		{
			name: "missing source",
			payload: MutationPayload{
				Type: "kafka",
				ID:   "event-123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMarshalPayload(t *testing.T) {
	payload := CodePayload{
		Repo:      "test-repo",
		Commit:    "abc123",
		Artifacts: []string{"bin/app"},
	}

	result, err := MarshalPayload(payload)
	if err != nil {
		t.Fatalf("MarshalPayload() error = %v", err)
	}

	// Verify it's valid JSON
	var decoded CodePayload
	if err := json.Unmarshal([]byte(result), &decoded); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if decoded.Repo != payload.Repo {
		t.Errorf("Repo mismatch: got %v, want %v", decoded.Repo, payload.Repo)
	}
	if decoded.Commit != payload.Commit {
		t.Errorf("Commit mismatch: got %v, want %v", decoded.Commit, payload.Commit)
	}
}

func TestParseJSON(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		raw := `{"repo":"test-repo","commit":"abc123","artifacts":["bin/app"]}`
		var payload CodePayload
		err := ParseJSON(raw, &payload)
		if err != nil {
			t.Fatalf("ParseJSON() error = %v", err)
		}
		if payload.Repo != "test-repo" {
			t.Errorf("Repo = %v, want test-repo", payload.Repo)
		}
		if payload.Commit != "abc123" {
			t.Errorf("Commit = %v, want abc123", payload.Commit)
		}
		if len(payload.Artifacts) != 1 || payload.Artifacts[0] != "bin/app" {
			t.Errorf("Artifacts = %v, want [bin/app]", payload.Artifacts)
		}
	})

	t.Run("unknown fields rejected", func(t *testing.T) {
		raw := `{"repo":"test-repo","commit":"abc123","unknown_field":"should fail"}`
		var payload CodePayload
		err := ParseJSON(raw, &payload)
		if err == nil {
			t.Error("ParseJSON() should have rejected unknown field")
		}
		if !strings.Contains(err.Error(), "unknown field") {
			t.Errorf("Error should mention unknown field, got: %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		raw := `{invalid json`
		var payload CodePayload
		err := ParseJSON(raw, &payload)
		if err == nil {
			t.Error("ParseJSON() should have failed on invalid JSON")
		}
	})
}
