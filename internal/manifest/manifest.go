package manifest

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

type Manifest struct {
	Version    string      `json:"version"`
	Name       string      `json:"name"`
	Collectors []Collector `json:"collectors"`
}

type Collector struct {
	Kind   string            `json:"kind"`
	Source string            `json:"source"`
	Params map[string]string `json:"params,omitempty"`
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}

	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}

	return m, nil
}

func (m Manifest) Validate() error {
	if strings.TrimSpace(m.Version) == "" {
		return errors.New("version is required")
	}
	if strings.TrimSpace(m.Name) == "" {
		return errors.New("name is required")
	}
	if len(m.Collectors) == 0 {
		return errors.New("at least one collector is required")
	}

	for i, c := range m.Collectors {
		if err := c.Validate(); err != nil {
			return errors.New("collector " + string(rune(i)) + ": " + err.Error())
		}
	}

	return nil
}

func (c Collector) Validate() error {
	validKinds := map[string]bool{"code": true, "config": true, "environment": true, "mutation": true}
	if !validKinds[c.Kind] {
		return errors.New("invalid kind: " + c.Kind)
	}
	if strings.TrimSpace(c.Source) == "" {
		return errors.New("source is required")
	}
	return nil
}

func NewManifest(name string) Manifest {
	return Manifest{
		Version:    "1.0",
		Name:       name,
		Collectors: []Collector{},
	}
}

func (m *Manifest) AddCollector(kind, source string, params map[string]string) {
	m.Collectors = append(m.Collectors, Collector{
		Kind:   kind,
		Source: source,
		Params: params,
	})
}

func (m Manifest) ToJSON() (string, error) {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
