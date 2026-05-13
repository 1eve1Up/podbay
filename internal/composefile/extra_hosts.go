package composefile

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExtraHostCompose is extra_hosts as list or map (Compose).
type ExtraHostCompose []string

// UnmarshalYAML accepts ["host:ip"] or { host: ip }.
func (e *ExtraHostCompose) UnmarshalYAML(n *yaml.Node) error {
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
			*e = append(*e, strings.TrimSpace(s))
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
	return fmt.Errorf("composefile: extra_hosts: expected sequence or mapping")
}
