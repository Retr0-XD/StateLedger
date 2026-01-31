package collectors

import (
	"encoding/json"
	"errors"
	"strings"
)

type CodePayload struct {
	Repo      string   `json:"repo"`
	Commit    string   `json:"commit"`
	Artifacts []string `json:"artifacts,omitempty"`
	Lockfiles []string `json:"lockfiles,omitempty"`
}

type ConfigPayload struct {
	Source   string `json:"source"`
	Version  string `json:"version"`
	Hash     string `json:"hash"`
	Snapshot string `json:"snapshot"`
}

type EnvironmentPayload struct {
	OS         string   `json:"os"`
	Kernel     string   `json:"kernel"`
	Container  string   `json:"container"`
	Runtime    string   `json:"runtime"`
	Arch       string   `json:"arch"`
	Flags      []string `json:"flags,omitempty"`
	TimeSource string   `json:"time_source"`
}

type MutationPayload struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Source      string `json:"source"`
	Hash        string `json:"hash"`
	ExternalRef string `json:"external_ref,omitempty"`
}

func (p CodePayload) Validate() error {
	if strings.TrimSpace(p.Repo) == "" {
		return errors.New("repo is required")
	}
	if strings.TrimSpace(p.Commit) == "" {
		return errors.New("commit is required")
	}
	return nil
}

func (p ConfigPayload) Validate() error {
	if strings.TrimSpace(p.Source) == "" {
		return errors.New("source is required")
	}
	if strings.TrimSpace(p.Hash) == "" {
		return errors.New("hash is required")
	}
	if strings.TrimSpace(p.Snapshot) == "" {
		return errors.New("snapshot is required")
	}
	return nil
}

func (p EnvironmentPayload) Validate() error {
	if strings.TrimSpace(p.OS) == "" {
		return errors.New("os is required")
	}
	if strings.TrimSpace(p.Runtime) == "" {
		return errors.New("runtime is required")
	}
	if strings.TrimSpace(p.Arch) == "" {
		return errors.New("arch is required")
	}
	if strings.TrimSpace(p.TimeSource) == "" {
		return errors.New("time_source is required")
	}
	return nil
}

func (p MutationPayload) Validate() error {
	if strings.TrimSpace(p.Type) == "" {
		return errors.New("type is required")
	}
	if strings.TrimSpace(p.ID) == "" {
		return errors.New("id is required")
	}
	if strings.TrimSpace(p.Source) == "" {
		return errors.New("source is required")
	}
	return nil
}

func MarshalPayload[T any](payload T) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ParseJSON[T any](raw string, out *T) error {
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}
