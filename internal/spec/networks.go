package spec

import (
	"fmt"
	"sort"
	"strings"
)

// EffectiveServiceNetworks returns logical network keys from the contract for this service.
// When the contract defines no top-level networks:, returns (nil, nil) to mean the implicit
// project bridge (podbay_<project>) only — callers must not treat nil as an error.
func EffectiveServiceNetworks(c *Contract, svcName string, svc Service) ([]string, error) {
	if len(c.Networks) == 0 {
		if len(svc.Networks) > 0 {
			return nil, fmt.Errorf("service %q: networks: is set but the contract has no top-level networks: block (declare networks or remove service networks)", svcName)
		}
		return nil, nil
	}

	keys := NetworkKeysSorted(c.Networks)
	if len(keys) == 1 {
		sole := keys[0]
		if len(svc.Networks) == 0 {
			return []string{sole}, nil
		}
		for _, n := range svc.Networks {
			n = strings.TrimSpace(n)
			if n != sole {
				return nil, fmt.Errorf("service %q: network %q is not defined (this contract has a single network %q)", svcName, n, sole)
			}
		}
		return dedupeNetworkNames(svc.Networks), nil
	}

	if len(svc.Networks) == 0 {
		return nil, fmt.Errorf("service %q: must declare networks: when multiple top-level networks are defined (%v)", svcName, keys)
	}
	var out []string
	for _, n := range svc.Networks {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := c.Networks[n]; !ok {
			return nil, fmt.Errorf("service %q: network %q is not declared under top-level networks:", svcName, n)
		}
		out = append(out, n)
	}
	out = dedupeNetworkNames(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("service %q: networks: must list at least one defined network", svcName)
	}
	return out, nil
}

// NetworkKeysSorted returns sorted logical network names from the contract.
func NetworkKeysSorted(nets map[string]Network) []string {
	s := make([]string, 0, len(nets))
	for k := range nets {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}

// ValidateNetworkDrivers returns an error if any network uses an unsupported driver.
func ValidateNetworkDrivers(nets map[string]Network) error {
	for name, n := range nets {
		d := strings.ToLower(strings.TrimSpace(n.Driver))
		if n.External {
			if d != "" && d != "bridge" {
				return fmt.Errorf("networks %q: external networks cannot set driver %q (omit driver; Podbay does not create external networks)", name, strings.TrimSpace(n.Driver))
			}
			continue
		}
		if d == "" || d == "bridge" {
			continue
		}
		return fmt.Errorf("networks %q: driver %q is not supported (only bridge is supported in this release)", name, strings.TrimSpace(n.Driver))
	}
	return nil
}

func dedupeNetworkNames(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
