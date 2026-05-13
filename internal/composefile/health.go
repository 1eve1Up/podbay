package composefile

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// HealthcheckSpec mirrors Compose healthcheck (not podbay health:).
type HealthcheckSpec struct {
	Test        CommandTest `yaml:"test"`
	Interval    string      `yaml:"interval,omitempty"`
	Timeout     string      `yaml:"timeout,omitempty"`
	Retries     int         `yaml:"retries,omitempty"`
	StartPeriod string      `yaml:"start_period,omitempty"`
}

// CommandTest is healthcheck.test: string or list (CMD / CMD-SHELL).
type CommandTest []string

// UnmarshalYAML decodes test as scalar or sequence.
func (c *CommandTest) UnmarshalYAML(n *yaml.Node) error {
	*c = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind == yaml.ScalarNode {
		s := strings.TrimSpace(n.Value)
		if s != "" {
			*c = []string{"CMD-SHELL", s}
		}
		return nil
	}
	if n.Kind == yaml.SequenceNode {
		for _, item := range n.Content {
			var s string
			if err := item.Decode(&s); err != nil {
				return err
			}
			*c = append(*c, s)
		}
		return nil
	}
	return fmt.Errorf("composefile: healthcheck.test: expected string or sequence")
}
