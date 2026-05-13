package composefile

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// PortStrings normalizes ports: to []string (short syntax and Compose long-form mappings).
type PortStrings []string

// UnmarshalYAML accepts scalar strings or long-form mapping entries (published/target/...).
func (p *PortStrings) UnmarshalYAML(n *yaml.Node) error {
	*p = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind != yaml.SequenceNode {
		return fmt.Errorf("composefile: ports: expected sequence")
	}
	for _, item := range n.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			s := strings.TrimSpace(item.Value)
			if s != "" {
				*p = append(*p, s)
			}
		case yaml.MappingNode:
			s, err := portMappingFromLongYAML(item)
			if err != nil {
				return err
			}
			if s != "" {
				*p = append(*p, s)
			}
		default:
			return fmt.Errorf("composefile: ports: unsupported entry kind")
		}
	}
	return nil
}

func portMappingFromLongYAML(item *yaml.Node) (string, error) {
	var raw map[string]interface{}
	if err := item.Decode(&raw); err != nil {
		return "", fmt.Errorf("composefile: ports: long-form: %w", err)
	}
	mode, _ := raw["mode"].(string)
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "host" {
		return "", fmt.Errorf("composefile: ports: mode=host is not supported in import (use podman/host networking explicitly outside import)")
	}
	tgt, err := ifacePort(raw["target"])
	if err != nil {
		return "", fmt.Errorf("composefile: ports: target: %w", err)
	}
	pubVal, ok := raw["published"]
	if !ok || pubVal == nil {
		return "", fmt.Errorf("composefile: ports: long-form mapping requires published and target (omit published only when using short \"host:container\" syntax)")
	}
	pub, err := ifacePort(pubVal)
	if err != nil {
		return "", fmt.Errorf("composefile: ports: published: %w", err)
	}
	proto := ""
	if s, ok := raw["protocol"].(string); ok {
		proto = strings.TrimSpace(strings.ToLower(s))
	}
	hostIP := ""
	if s, ok := raw["host_ip"].(string); ok {
		hostIP = strings.TrimSpace(s)
	}

	core := fmt.Sprintf("%d:%d", pub, tgt)
	if proto != "" && proto != "tcp" {
		core = core + "/" + proto
	}
	if hostIP != "" {
		return hostIP + ":" + core, nil
	}
	return core, nil
}

func ifacePort(v interface{}) (int, error) {
	if v == nil {
		return 0, fmt.Errorf("missing port")
	}
	switch x := v.(type) {
	case int:
		return x, nil
	case int64:
		return int(x), nil
	case float64:
		return int(x), nil
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, fmt.Errorf("empty port")
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("parse port %q: %w", s, err)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("invalid port type %T", v)
	}
}
