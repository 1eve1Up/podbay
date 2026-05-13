package composefile

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// StringOrMap is environment: KEY: val or list of KEY=val.
type StringOrMap map[string]string

// UnmarshalYAML decodes a mapping or a sequence of KEY=VAL strings.
func (m *StringOrMap) UnmarshalYAML(n *yaml.Node) error {
	*m = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind == yaml.MappingNode {
		out := make(map[string]string)
		for i := 0; i < len(n.Content); i += 2 {
			var k, v string
			if err := n.Content[i].Decode(&k); err != nil {
				return err
			}
			if err := n.Content[i+1].Decode(&v); err != nil {
				return err
			}
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			out[k] = v
		}
		*m = out
		return nil
	}
	if n.Kind == yaml.SequenceNode {
		out := make(map[string]string)
		for _, item := range n.Content {
			var s string
			if err := item.Decode(&s); err != nil {
				return err
			}
			k, v, ok := strings.Cut(s, "=")
			if !ok {
				return fmt.Errorf("composefile: environment list entry %q: expected KEY=VAL", s)
			}
			k = strings.TrimSpace(k)
			if k == "" {
				return fmt.Errorf("composefile: environment list entry %q: empty key", s)
			}
			out[k] = strings.TrimSpace(v)
		}
		*m = out
		return nil
	}
	return fmt.Errorf("composefile: environment: expected mapping or sequence")
}

// EnvFileEntry matches Compose env_file item.
type EnvFileEntry struct {
	Path     string `yaml:"path"`
	Required bool   `yaml:"required"`
}

// EnvFileList is env_file as scalar, sequence of scalars, or sequence of objects.
type EnvFileList []EnvFileEntry

// UnmarshalYAML for env_file.
func (e *EnvFileList) UnmarshalYAML(n *yaml.Node) error {
	*e = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind == yaml.ScalarNode {
		*e = append(*e, EnvFileEntry{Path: strings.TrimSpace(n.Value), Required: false})
		return nil
	}
	if n.Kind != yaml.SequenceNode {
		return fmt.Errorf("composefile: env_file: expected string or sequence")
	}
	for _, item := range n.Content {
		if item.Kind == yaml.ScalarNode {
			*e = append(*e, EnvFileEntry{Path: strings.TrimSpace(item.Value), Required: false})
			continue
		}
		var row EnvFileEntry
		if err := item.Decode(&row); err != nil {
			return err
		}
		row.Path = strings.TrimSpace(row.Path)
		if row.Path == "" {
			return fmt.Errorf("composefile: env_file: empty path")
		}
		*e = append(*e, row)
	}
	return nil
}
