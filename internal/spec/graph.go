package spec

import (
	"fmt"
	"sort"
	"strings"
)

// DependsOnContains reports whether svc lists peer in depends_on (any condition).
func DependsOnContains(svc Service, peer string) bool {
	peer = strings.TrimSpace(peer)
	if peer == "" {
		return false
	}
	for _, d := range svc.DependsOn {
		if d.Service == peer {
			return true
		}
	}
	return false
}

// ExpandDependentsTransitive returns the union of seeds and every service z in `base` such that z
// lists some service already in the working set under depends_on (transitive BFS downstream within
// `base` only). Empty or nil seeds returns seeds unchanged.
func ExpandDependentsTransitive(base, seeds map[string]Service) map[string]Service {
	if len(seeds) == 0 {
		return seeds
	}
	if len(base) == 0 {
		return seeds
	}
	out := make(map[string]Service, len(seeds))
	for k, v := range seeds {
		out[k] = v
	}
	for {
		var toAdd []string
		for z, zsvc := range base {
			if _, have := out[z]; have {
				continue
			}
			for name := range out {
				if DependsOnContains(zsvc, name) {
					toAdd = append(toAdd, z)
					break
				}
			}
		}
		if len(toAdd) == 0 {
			break
		}
		sort.Strings(toAdd)
		for _, z := range toAdd {
			if _, have := out[z]; !have {
				out[z] = base[z]
			}
		}
	}
	return out
}

// ServicesForDeployTargets returns base unchanged when roots is empty (after trimming and deduplication).
// Otherwise it returns only the named services that exist in base (partial deploy / validate targets only;
// depends_on does not expand the set).
func ServicesForDeployTargets(base map[string]Service, roots []string) (map[string]Service, error) {
	if len(base) == 0 || len(roots) == 0 {
		return base, nil
	}
	var norm []string
	seenRoot := map[string]struct{}{}
	for _, r := range roots {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if _, dup := seenRoot[r]; dup {
			continue
		}
		seenRoot[r] = struct{}{}
		if _, ok := base[r]; !ok {
			return nil, fmt.Errorf("service %q is not active for this profile set (unknown or excluded by --profile)", r)
		}
		norm = append(norm, r)
	}
	if len(norm) == 0 {
		return base, nil
	}
	out := make(map[string]Service, len(norm))
	for _, name := range norm {
		out[name] = base[name]
	}
	return out, nil
}

// ObservabilityActiveServices is the single implementation of partial service selection for all
// commands (validate, deploy, diff, ps, explain, logs, teardown/down). When roots is empty (or
// only whitespace after ServicesForDeployTargets normalization), returns profileActive unchanged;
// otherwise starts from ServicesForDeployTargets(profileActive, roots) and optionally applies
// ExpandDependentsTransitive.
func ObservabilityActiveServices(profileActive map[string]Service, roots []string, expandDependents bool) (map[string]Service, error) {
	active := profileActive
	if len(roots) == 0 {
		return active, nil
	}
	sub, err := ServicesForDeployTargets(profileActive, roots)
	if err != nil {
		return nil, err
	}
	active = sub
	if expandDependents {
		active = ExpandDependentsTransitive(profileActive, sub)
	}
	return active, nil
}

// ServiceNamesSorted returns sorted keys of m.
func ServiceNamesSorted(m map[string]Service) []string {
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ServiceNames returns sorted service keys (full contract).
func (c *Contract) ServiceNames() []string {
	return ServiceNamesSorted(c.Services)
}

// TopologicalOrder returns services in dependency order for the full contract.
func (c *Contract) TopologicalOrder() ([]string, error) {
	return TopologicalOrder(c.Services)
}

// TopologicalOrder returns a topological sort of the given service map.
func TopologicalOrder(services map[string]Service) ([]string, error) {
	inDegree := map[string]int{}
	adj := map[string][]string{}
	for name := range services {
		inDegree[name] = 0
	}
	for name, svc := range services {
		for _, dep := range svc.DependsOn {
			if _, ok := services[dep.Service]; !ok {
				return nil, fmt.Errorf("service %q depends on unknown service %q", name, dep.Service)
			}
			adj[dep.Service] = append(adj[dep.Service], name)
			inDegree[name]++
		}
	}
	var q []string
	for name := range services {
		if inDegree[name] == 0 {
			q = append(q, name)
		}
	}
	sort.Strings(q)
	var order []string
	for len(q) > 0 {
		u := q[0]
		q = q[1:]
		order = append(order, u)
		next := adj[u]
		sort.Strings(next)
		for _, v := range next {
			inDegree[v]--
			if inDegree[v] == 0 {
				q = append(q, v)
			}
		}
		sort.Strings(q)
	}
	if len(order) != len(services) {
		return nil, fmt.Errorf("dependency cycle detected")
	}
	return order, nil
}

// TopologicalOrderSubset is like TopologicalOrder but only includes services in sub.
// depends_on edges to services outside sub are ignored (treated as already satisfied for ordering).
// Cycles involving only services in sub are still detected.
func TopologicalOrderSubset(sub map[string]Service) ([]string, error) {
	inDegree := map[string]int{}
	adj := map[string][]string{}
	for name := range sub {
		inDegree[name] = 0
	}
	for name, svc := range sub {
		for _, dep := range svc.DependsOn {
			if _, ok := sub[dep.Service]; !ok {
				continue
			}
			adj[dep.Service] = append(adj[dep.Service], name)
			inDegree[name]++
		}
	}
	var q []string
	for name := range sub {
		if inDegree[name] == 0 {
			q = append(q, name)
		}
	}
	sort.Strings(q)
	var order []string
	for len(q) > 0 {
		u := q[0]
		q = q[1:]
		order = append(order, u)
		next := adj[u]
		sort.Strings(next)
		for _, v := range next {
			inDegree[v]--
			if inDegree[v] == 0 {
				q = append(q, v)
			}
		}
		sort.Strings(q)
	}
	if len(order) != len(sub) {
		return nil, fmt.Errorf("dependency cycle detected")
	}
	return order, nil
}
