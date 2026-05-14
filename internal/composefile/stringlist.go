package composefile

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// StringOrList is command: string or list of strings.
//
// Compose semantics: a scalar form is shell-form and is wrapped as ["sh","-c",value]
// so the container sees a single shell command rather than a single binary literally
// named "echo hi". The sequence form is preserved as-is (exec form).
type StringOrList []string

// UnmarshalYAML decodes a scalar in shell-form (wrapped as sh -c) or a sequence as exec-form argv.
func (s *StringOrList) UnmarshalYAML(n *yaml.Node) error {
	*s = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind == yaml.ScalarNode {
		v := strings.TrimSpace(n.Value)
		if v != "" {
			*s = []string{"sh", "-c", v}
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
