package composefile

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML accepts build: . or build: { context: ., dockerfile: Dockerfile }.
func (b *BuildSpec) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind == yaml.ScalarNode {
		*b = BuildSpec{Context: n.Value}
		return nil
	}
	if n.Kind == yaml.MappingNode {
		type raw struct {
			Context    string `yaml:"context"`
			Dockerfile string `yaml:"dockerfile"`
		}
		var r raw
		if err := n.Decode(&r); err != nil {
			return err
		}
		if r.Context == "" {
			return fmt.Errorf("composefile: build: missing context")
		}
		*b = BuildSpec{Context: r.Context, Dockerfile: r.Dockerfile}
		return nil
	}
	return fmt.Errorf("composefile: build: expected string or mapping")
}
