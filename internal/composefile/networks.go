package composefile

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// NetworkSpec is a subset of Compose top-level networks: entries.
type NetworkSpec struct {
	Driver   string `yaml:"driver,omitempty"`
	Name     string `yaml:"name,omitempty"`
	External bool   `yaml:"external,omitempty"`
}

// UnmarshalYAML accepts Compose network definitions including external: true or external: { name: ... }.
func (n *NetworkSpec) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode && value.Tag == "!!null" {
		return nil
	}
	var raw map[string]interface{}
	if err := value.Decode(&raw); err != nil {
		return fmt.Errorf("composefile: networks: %w", err)
	}
	if v, ok := raw["driver"].(string); ok {
		n.Driver = v
	}
	if v, ok := raw["name"].(string); ok {
		n.Name = v
	}
	ext, ok := raw["external"]
	if !ok || ext == nil {
		return nil
	}
	switch t := ext.(type) {
	case bool:
		n.External = t
	case map[string]interface{}:
		n.External = true
		if nm, ok := t["name"].(string); ok && n.Name == "" {
			n.Name = nm
		}
	default:
		n.External = true
	}
	return nil
}
