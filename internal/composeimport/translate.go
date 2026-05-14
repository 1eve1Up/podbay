// Package composeimport maps a parsed Docker Compose file into a Podbay spec contract.
package composeimport

import (
	"fmt"
	"sort"
	"strings"

	"github.com/1eve1Up/podbay/internal/composefile"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
)

// ToContract converts a compose File into a podbay Contract (version 1).
// composeDir is the absolute directory containing the Compose file (used to resolve configs:/secrets: file: paths).
// Unsupported network topologies and features produce errors.
func ToContract(f *composefile.File, composeDir string) (*spec.Contract, error) {
	if f == nil {
		return nil, fmt.Errorf("composeimport: nil compose file")
	}
	composeDir = strings.TrimSpace(composeDir)
	if composeDir == "" {
		return nil, fmt.Errorf("composeimport: compose directory is required")
	}

	out := &spec.Contract{
		Version:  "1",
		Services: make(map[string]spec.Service),
		Volumes:  make(map[string]spec.Volume),
		Networks: make(map[string]spec.Network),
	}

	namedVol := map[string]struct{}{}

	for name, cs := range f.Services {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		svc, err := translateService(composeDir, f, name, cs)
		if err != nil {
			return nil, err
		}
		for _, v := range svc.Volumes {
			src, dest, _ := runner.SplitVolumeMount(v)
			if dest == "" {
				continue
			}
			if isBindMountSource(src) {
				continue
			}
			if src != "" {
				namedVol[src] = struct{}{}
			}
		}
		out.Services[name] = svc
	}

	for n := range namedVol {
		if _, ok := f.Volumes[n]; !ok {
			// Compose creates anonymous named volumes when only used in services;
			// declare them in the contract like top-level volumes: { n: {} }.
			out.Volumes[n] = spec.Volume{}
			continue
		}
		v := f.Volumes[n]
		out.Volumes[n] = spec.Volume{Driver: strings.TrimSpace(v.Driver)}
	}

	if err := applyComposeNetworks(f, out); err != nil {
		return nil, err
	}

	syncRedeployPeersFromDependsOn(out.Services)

	if err := validateServiceNames(out.Services); err != nil {
		return nil, err
	}

	return out, nil
}

func applyComposeNetworks(f *composefile.File, out *spec.Contract) error {
	for name, ns := range f.Networks {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out.Networks[name] = spec.Network{
			Driver:   strings.TrimSpace(ns.Driver),
			External: ns.External,
			Name:     strings.TrimSpace(ns.Name),
		}
	}

	for svcName, cs := range f.Services {
		svcName = strings.TrimSpace(svcName)
		if svcName == "" {
			continue
		}
		nets, err := inferAttachedNetworks(cs, f.Networks)
		if err != nil {
			return fmt.Errorf("composeimport: service %q: %w", svcName, err)
		}
		svc := out.Services[svcName]
		svc.Networks = append([]string(nil), nets...)
		out.Services[svcName] = svc
		for _, netName := range nets {
			netName = strings.TrimSpace(netName)
			if netName == "" {
				continue
			}
			if _, ok := f.Networks[netName]; !ok {
				return fmt.Errorf("composeimport: service %q references undefined network %q", svcName, netName)
			}
		}
	}
	return nil
}

func inferAttachedNetworks(cs composefile.ServiceSpec, top map[string]composefile.NetworkSpec) ([]string, error) {
	if len(cs.Networks) > 0 {
		return []string(cs.Networks), nil
	}
	if len(top) == 0 {
		return nil, nil
	}
	if _, ok := top["default"]; ok {
		return []string{"default"}, nil
	}
	if len(top) == 1 {
		for name := range top {
			return []string{name}, nil
		}
	}
	return nil, fmt.Errorf("multiple top-level networks without default: — declare networks: on each service")
}

// syncRedeployPeersFromDependsOn fills each parent's RedeployPeers from inverse depends_on edges
// so contracts satisfy bidirectional dependents validation.
func syncRedeployPeersFromDependsOn(services map[string]spec.Service) {
	inverse := map[string]map[string]struct{}{}
	for childName, child := range services {
		for _, d := range child.DependsOn {
			p := strings.TrimSpace(d.Service)
			if p == "" {
				continue
			}
			if inverse[p] == nil {
				inverse[p] = map[string]struct{}{}
			}
			inverse[p][childName] = struct{}{}
		}
	}
	for parent, children := range inverse {
		svc, ok := services[parent]
		if !ok {
			continue
		}
		m := map[string]struct{}{}
		for _, x := range svc.RedeployPeers {
			x = strings.TrimSpace(x)
			if x != "" {
				m[x] = struct{}{}
			}
		}
		for c := range children {
			m[c] = struct{}{}
		}
		var peers []string
		for x := range m {
			peers = append(peers, x)
		}
		sort.Strings(peers)
		svc.RedeployPeers = peers
		services[parent] = svc
	}
}

func sortedStringKeys(m map[string]struct{}) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}

func validateServiceNames(services map[string]spec.Service) error {
	for name, svc := range services {
		for _, d := range svc.DependsOn {
			if _, ok := services[d.Service]; !ok {
				return fmt.Errorf("composeimport: service %q depends_on unknown service %q", name, d.Service)
			}
		}
		for _, child := range svc.RedeployPeers {
			if _, ok := services[child]; !ok {
				return fmt.Errorf("composeimport: service %q dependents unknown service %q", name, child)
			}
			if !spec.DependsOnContains(services[child], name) {
				return fmt.Errorf("composeimport: service %q dependents lists %q but %q must depends_on %q", name, child, child, name)
			}
		}
	}
	return nil
}

func translateService(composeDir string, f *composefile.File, name string, cs composefile.ServiceSpec) (spec.Service, error) {
	img := strings.TrimSpace(cs.Image)
	hasBuild := cs.Build != nil && strings.TrimSpace(cs.Build.Context) != ""
	if img == "" && !hasBuild {
		return spec.Service{}, fmt.Errorf("composeimport: service %q: need image or build", name)
	}
	if hasBuild && img == "" {
		return spec.Service{}, fmt.Errorf("composeimport: service %q: build requires an image tag in Compose (podbay deploy needs image for podman run)", name)
	}

	var b *spec.Build
	if cs.Build != nil {
		b = &spec.Build{
			Context:    strings.TrimSpace(cs.Build.Context),
			Dockerfile: strings.TrimSpace(cs.Build.Dockerfile),
		}
	}

	deps := make(spec.Dependencies, 0, len(cs.DependsOn))
	for _, d := range cs.DependsOn {
		svc := strings.TrimSpace(d.Service)
		if svc == "" {
			continue
		}
		cond := spec.NormalizeCondition(d.Condition)
		if cond != spec.ConditionStarted && cond != spec.ConditionHealthy {
			return spec.Service{}, fmt.Errorf("composeimport: service %q: depends_on %q: unsupported condition %q (use service_started or service_healthy)", name, svc, strings.TrimSpace(d.Condition))
		}
		deps = append(deps, spec.Dependency{Service: svc, Condition: cond})
	}

	env := map[string]string{}
	for k, v := range cs.Environment {
		env[k] = v
	}

	envFile := make(spec.EnvFileEntries, 0, len(cs.EnvFile))
	for _, e := range cs.EnvFile {
		envFile = append(envFile, spec.EnvFileEntry{Path: e.Path, Required: e.Required})
	}

	var health *spec.HealthCheck
	if cs.Healthcheck != nil && len(cs.Healthcheck.Test) > 0 {
		argv := normalizeHealthcheckTest(cs.Healthcheck.Test)
		if len(argv) > 0 {
			health = &spec.HealthCheck{
				Exec: &spec.ExecHealth{
					Command: argv,
				},
				Interval:    cs.Healthcheck.Interval,
				Timeout:     cs.Healthcheck.Timeout,
				Retries:     cs.Healthcheck.Retries,
				StartPeriod: cs.Healthcheck.StartPeriod,
			}
		}
	}

	labels := map[string]string{}
	for k, v := range cs.Labels {
		labels[k] = v
	}

	extra := spec.ExtraHostList(nil)
	for _, h := range cs.ExtraHosts {
		if strings.TrimSpace(h) != "" {
			extra = append(extra, strings.TrimSpace(h))
		}
	}

	ports := append([]string(nil), cs.Ports...)
	expose := append([]string(nil), cs.Expose...)
	vols := append([]string(nil), cs.Volumes...)
	cmd := append([]string(nil), cs.Command...)

	svc := spec.Service{
		Build:       b,
		Image:       img,
		DependsOn:   deps,
		Ports:       ports,
		Expose:      expose,
		Volumes:     vols,
		Environment: env,
		EnvFile:     envFile,
		Command:     cmd,
		Health:      health,
		Labels:      labels,
		Restart:     strings.TrimSpace(cs.Restart),
		ExtraHosts:  extra,
		Profiles:    append([]string(nil), cs.Profiles...),
		User:        strings.TrimSpace(cs.User),
		DNS:         append([]string(nil), cs.DNS...),
	}
	if err := appendConfigSecretVolumes(composeDir, f, name, cs, &svc); err != nil {
		return spec.Service{}, err
	}
	return svc, nil
}

// normalizeHealthcheckTest strips Compose's leading directive token from a healthcheck.test
// array so the resulting argv can be handed to `podman exec` directly.
//
//   - CMD <argv...>           → <argv...>
//   - CMD-SHELL <string...>   → ["sh", "-c", strings.Join(rest, " ")]
//   - NONE                    → nil (caller emits no Health probe)
//   - anything else           → returned as-is (already a bare argv)
func normalizeHealthcheckTest(test []string) []string {
	if len(test) == 0 {
		return nil
	}
	switch strings.ToUpper(strings.TrimSpace(test[0])) {
	case "NONE":
		return nil
	case "CMD":
		return append([]string(nil), test[1:]...)
	case "CMD-SHELL":
		rest := strings.TrimSpace(strings.Join(test[1:], " "))
		if rest == "" {
			return nil
		}
		return []string{"sh", "-c", rest}
	default:
		return append([]string(nil), test...)
	}
}

func isBindMountSource(src string) bool {
	src = strings.TrimSpace(src)
	if src == "" {
		return false
	}
	if strings.HasPrefix(src, "./") || strings.HasPrefix(src, "../") {
		return true
	}
	if strings.HasPrefix(src, "~") {
		return true
	}
	if strings.HasPrefix(src, "/") {
		return true
	}
	if len(src) >= 2 && (src[1] == ':' || src[1] == '|') {
		// Windows drive or Docker Desktop pipe paths
		return true
	}
	return false
}
