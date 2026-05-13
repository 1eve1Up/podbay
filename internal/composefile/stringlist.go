package composefile

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// StringOrList is command: string or list of strings.
type StringOrList []string

// UnmarshalYAML decodes a scalar as one-element list or a sequence of strings.
func (s *StringOrList) UnmarshalYAML(n *yaml.Node) error {
	*s = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind == yaml.ScalarNode {
		v := strings.TrimSpace(n.Value)
		if v != "" {
			*s = []string{v}
		}
		return nil
	}
	if n.Kind == yaml.SequenceNode {
		for _, item := range n.Content {
			var t string
			if err := item.Decode(&t); err != nil {
				return err
			}
			*s = append(*s, t)
		}
		return nil
	}
	return fmt.Errorf("composefile: expected string or sequence")
}
