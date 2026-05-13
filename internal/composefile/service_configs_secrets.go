package composefile

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ServiceBindRef is one Compose service configs: or secrets: entry (source → container path).
type ServiceBindRef struct {
	Source string `yaml:"source"`
	Target string `yaml:"target,omitempty"`
}

// ServiceConfigs is Compose service-level configs: (list of names or {source,target,...}).
type ServiceConfigs []ServiceBindRef

// UnmarshalYAML accepts ["myconf"] or [{source: myconf, target: /etc/app.conf}].
func (c *ServiceConfigs) UnmarshalYAML(n *yaml.Node) error {
	list, err := unmarshalBindRefList(n)
	if err != nil {
		return fmt.Errorf("configs: %w", err)
	}
	*c = ServiceConfigs(list)
	return nil
}

// ServiceSecrets is Compose service-level secrets:.
type ServiceSecrets []ServiceBindRef

// UnmarshalYAML accepts ["db"] or [{source: db, target: /run/secrets/db}] .
func (s *ServiceSecrets) UnmarshalYAML(n *yaml.Node) error {
	list, err := unmarshalBindRefList(n)
	if err != nil {
		return fmt.Errorf("secrets: %w", err)
	}
	*s = ServiceSecrets(list)
	return nil
}

func unmarshalBindRefList(n *yaml.Node) ([]ServiceBindRef, error) {
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil, nil
	}
	if n.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("expected sequence")
	}
	var out []ServiceBindRef
	for _, item := range n.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			src := item.Value
			if src != "" {
				out = append(out, ServiceBindRef{Source: src})
			}
		case yaml.MappingNode:
			var row ServiceBindRef
			if err := item.Decode(&row); err != nil {
				return nil, err
			}
			if row.Source == "" {
				continue
			}
			out = append(out, row)
		default:
			return nil, fmt.Errorf("expected scalar or mapping")
		}
	}
	return out, nil
}
