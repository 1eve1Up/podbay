package spec

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML accepts ["host.docker.internal:host-gateway"] or { host.docker.internal: host-gateway }.
func (e *ExtraHostList) UnmarshalYAML(n *yaml.Node) error {
	*e = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind == yaml.SequenceNode {
		for _, item := range n.Content {
			var s string
			if err := item.Decode(&s); err != nil {
				return err
			}
			*e = append(*e, s)
		}
		return nil
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i < len(n.Content); i += 2 {
			var k, v string
			if err := n.Content[i].Decode(&k); err != nil {
				return err
			}
			if err := n.Content[i+1].Decode(&v); err != nil {
				return err
			}
			k, v = strings.TrimSpace(k), strings.TrimSpace(v)
			if k == "" || v == "" {
				continue
			}
			*e = append(*e, k+":"+v)
		}
		return nil
	}
	return fmt.Errorf("extra_hosts: expected sequence or mapping")
}

// UnmarshalYAML accepts ["api"] or { api: { condition: service_healthy } }.
func (d *Dependencies) UnmarshalYAML(n *yaml.Node) error {
	*d = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind == yaml.SequenceNode {
		for _, c := range n.Content {
			var s string
			if err := c.Decode(&s); err != nil {
				return err
			}
			*d = append(*d, Dependency{Service: s, Condition: ConditionStarted})
		}
		return nil
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i < len(n.Content); i += 2 {
			var key string
			if err := n.Content[i].Decode(&key); err != nil {
				return err
			}
			var v struct {
				Condition string `yaml:"condition"`
			}
			if err := n.Content[i+1].Decode(&v); err != nil {
				return err
			}
			*d = append(*d, Dependency{Service: key, Condition: NormalizeCondition(v.Condition)})
		}
		return nil
	}
	return fmt.Errorf("depends_on: expected sequence or mapping")
}

// NormalizeCondition maps Compose condition strings to started|healthy.
func NormalizeCondition(c string) string {
	switch strings.ToLower(strings.TrimSpace(c)) {
	case "service_healthy", "healthy":
		return ConditionHealthy
	case "service_started", "started", "":
		return ConditionStarted
	default:
		return strings.ToLower(strings.TrimSpace(c))
	}
}

// UnmarshalYAML accepts ".env" or { path: .env, required: true }.
func (e *EnvFileEntries) UnmarshalYAML(n *yaml.Node) error {
	*e = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind != yaml.SequenceNode {
		return fmt.Errorf("env_file: expected sequence")
	}
	for _, item := range n.Content {
		if item.Kind == yaml.ScalarNode {
			*e = append(*e, EnvFileEntry{Path: item.Value, Required: false})
			continue
		}
		var row EnvFileEntry
		if err := item.Decode(&row); err != nil {
			return err
		}
		*e = append(*e, row)
	}
	return nil
}
