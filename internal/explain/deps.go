package explain

import (
	"fmt"
	"sort"
	"strings"

	"github.com/1eve1Up/podbay/internal/spec"
)

// DependencySummary returns a short depends_on / dependents / deploy-order view for one
// service within the active (profile-filtered) graph.
func DependencySummary(active map[string]spec.Service, focus string) string {
	svc, ok := active[focus]
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Focus: %s\n", focus))

	if len(svc.DependsOn) == 0 {
		b.WriteString("Depends on: (none)\n")
	} else {
		b.WriteString("Depends on:\n")
		for _, d := range svc.DependsOn {
			cond := conditionLabel(d.Condition)
			if _, ok := active[d.Service]; !ok {
				b.WriteString(fmt.Sprintf("  %s (%s) [inactive in this profile]\n", d.Service, cond))
				continue
			}
			b.WriteString(fmt.Sprintf("  %s (%s)\n", d.Service, cond))
		}
	}

	var dependents []string
	for name, s := range active {
		if name == focus {
			continue
		}
		for _, d := range s.DependsOn {
			if d.Service == focus {
				dependents = append(dependents, fmt.Sprintf("%s (%s)", name, conditionLabel(d.Condition)))
				break
			}
		}
	}
	sort.Strings(dependents)
	if len(dependents) == 0 {
		b.WriteString("Dependents: (none)\n")
	} else {
		b.WriteString("Dependents:\n")
		for _, line := range dependents {
			b.WriteString("  " + line + "\n")
		}
	}

	if order, err := spec.TopologicalOrder(active); err == nil && len(order) > 0 {
		b.WriteString("Deploy order (active): " + strings.Join(order, ", ") + "\n")
	}

	return b.String()
}

func conditionLabel(c string) string {
	switch spec.NormalizeCondition(c) {
	case spec.ConditionHealthy:
		return "healthy"
	case spec.ConditionStarted:
		return "started"
	default:
		return spec.NormalizeCondition(c)
	}
}
