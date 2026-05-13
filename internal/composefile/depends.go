package composefile

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// DependsOnEntry is one depends_on edge (Compose condition strings preserved).
type DependsOnEntry struct {
	Service   string
	Condition string // service_started | service_healthy | started | healthy | empty
}

// DependsOnList is Compose depends_on as list or map.
type DependsOnList []DependsOnEntry

// UnmarshalYAML accepts ["api"] or { api: { condition: service_healthy } }.
func (d *DependsOnList) UnmarshalYAML(n *yaml.Node) error {
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
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			*d = append(*d, DependsOnEntry{Service: s, Condition: "service_started"})
		}
		return nil
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i < len(n.Content); i += 2 {
			var key string
			if err := n.Content[i].Decode(&key); err != nil {
				return err
			}
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			var v struct {
				Condition string `yaml:"condition"`
			}
			if err := n.Content[i+1].Decode(&v); err != nil {
				return err
			}
			cond := strings.TrimSpace(v.Condition)
			if cond == "" {
				cond = "service_started"
			}
			*d = append(*d, DependsOnEntry{Service: key, Condition: cond})
		}
		return nil
	}
	return fmt.Errorf("composefile: depends_on: expected sequence or mapping")
}
