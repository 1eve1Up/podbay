package composefile

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ServiceNetworkNames is Compose services[].networks as a list of network names or mapping keys only.
type ServiceNetworkNames []string

// UnmarshalYAML accepts a sequence of strings or a mapping (keys are network names; values ignored for import v2).
func (s *ServiceNetworkNames) UnmarshalYAML(n *yaml.Node) error {
	*s = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind == yaml.SequenceNode {
		for _, item := range n.Content {
			var name string
			if err := item.Decode(&name); err != nil {
				return fmt.Errorf("composefile: service networks: %w", err)
			}
			name = strings.TrimSpace(name)
			if name != "" {
				*s = append(*s, name)
			}
		}
		return nil
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i < len(n.Content); i += 2 {
			var key string
			if err := n.Content[i].Decode(&key); err != nil {
				return fmt.Errorf("composefile: service networks: %w", err)
			}
			key = strings.TrimSpace(key)
			if key != "" {
				*s = append(*s, key)
			}
		}
		return nil
	}
	return fmt.Errorf("composefile: service networks: expected sequence or mapping")
}
