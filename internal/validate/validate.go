package validate

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/spec"
)

// Level values for Result.Level (validate text output and --json issues).
const (
	LevelOK   = "ok"
	LevelWarn = "warn"
	LevelFail = "fail"
)

// Result is one validation finding.
type Result struct {
	OK      bool
	Level   string // ok | warn | fail
	Message string
	// Code, when non-empty, becomes Issue.code in validate --json (instead of generic validation_fail / validation_warn).
	Code string
	// Service optionally scopes the finding to a service name in JSON output.
	Service string
}

// Run executes validation checks. profiles selects Compose-style profiles (empty = default set only).
// deployRoots, when non-empty, selects a partial deploy subset: each name must be in the profile-active
// set. The effective map is explicit targets only, or includes transitive dependents when expandDependents is true.
func Run(c *spec.Contract, contractPath string, profiles []string, deployRoots []string, expandDependents bool) []Result {
	var out []Result
	root := filepath.Dir(contractPath)

	hostSubst, err := expand.LoadHostSubst(root, c.HostEnvFiles)
	if err != nil {
		out = append(out, Result{OK: false, Level: "fail", Message: fmt.Sprintf("host env: %v", err)})
		return out
	}

	profileActive := c.ServicesForProfiles(profiles)
	active := profileActive
	if len(deployRoots) > 0 {
		sub, err := spec.ServicesForDeployTargets(profileActive, deployRoots)
		if err != nil {
			out = append(out, Result{
				OK: false, Level: "fail", Message: err.Error(),
				Code: "deploy_service_selection",
			})
			return out
		}
		active = sub
		if expandDependents {
			active = spec.ExpandDependentsTransitive(profileActive, sub)
		}
	}
	if len(c.Services) > 0 && len(active) == 0 {
		out = append(out, Result{OK: false, Level: "fail", Message: "No services selected for this profile set"})
	}

	if len(active) == 0 && len(c.Services) == 0 {
		out = append(out, Result{OK: false, Level: "fail", Message: "No services defined"})
	}

	for parentName, parentSvc := range c.Services {
		for _, child := range parentSvc.RedeployPeers {
			if _, ok := c.Services[child]; !ok {
				out = append(out, Result{
					OK: false, Level: "fail",
					Message: fmt.Sprintf("Service %q: dependents lists %q, which is not a defined service", parentName, child),
					Code:    "dependents_unknown_service",
					Service: parentName,
				})
				continue
			}
			childSvc := c.Services[child]
			if !spec.DependsOnContains(childSvc, parentName) {
				out = append(out, Result{
					OK: false, Level: "fail",
					Message: fmt.Sprintf("Service %q: dependents lists %q, but %q must list %q under depends_on (child depends on parent)", parentName, child, child, parentName),
					Code:    "dependents_invalid_dependent",
					Service: parentName,
				})
			}
		}
	}

	for childName, child := range c.Services {
		for _, d := range child.DependsOn {
			parent := strings.TrimSpace(d.Service)
			if parent == "" {
				continue
			}
			parentSvc, ok := c.Services[parent]
			if !ok {
				continue
			}
			if !redeployPeersContains(parentSvc, childName) {
				out = append(out, Result{
					OK: false, Level: "fail",
					Message: fmt.Sprintf("Service %q: depends_on %q but %q must list %q under dependents (mirror depends_on on the dependency service)", childName, parent, parent, childName),
					Code:    "dependents_missing_inverse",
					Service: parent,
				})
			}
		}
	}

	if c.Network != nil && c.Network.MTU > 0 {
		out = append(out, Result{
			OK: true, Level: "warn",
			Message: "network.mtu applies only when Podbay first creates the bridge; an existing podbay_<project> network keeps its MTU until you remove it (podman network rm podbay_<project>) and redeploy",
		})
	}

	partial := len(deployRoots) > 0
	var order []string
	var topoErr error
	if partial {
		order, topoErr = spec.TopologicalOrderSubset(active)
	} else {
		order, topoErr = spec.TopologicalOrder(active)
	}
	if topoErr != nil {
		out = append(out, Result{OK: false, Level: "fail", Message: topoErr.Error(), Code: "dependency_invalid"})
	} else if len(order) > 0 {
		msg := "Dependency graph is acyclic (active services)"
		if partial {
			if expandDependents {
				msg = "Dependency graph is acyclic (partial selection: explicit targets and dependents-expanded services)"
			} else {
				msg = "Dependency graph is acyclic (partial selection: explicit targets only)"
			}
		}
		out = append(out, Result{OK: true, Level: "ok", Message: msg})
	}

	if err := spec.ValidateNetworkDrivers(c.Networks); err != nil {
		out = append(out, Result{
			OK: false, Level: "fail", Message: err.Error(),
			Code: "network_driver_unsupported",
		})
	}

	for name, svc := range active {
		if _, err := spec.EffectiveServiceNetworks(c, name, svc); err != nil {
			out = append(out, Result{
				OK: false, Level: "fail", Message: err.Error(),
				Code: "network_invalid", Service: name,
			})
		}
	}

	for name, svc := range active {
		for _, ef := range svc.EnvFile {
			p := filepath.Join(root, ef.Path)
			if _, err := os.Stat(p); err != nil {
				if ef.Required {
					out = append(out, Result{
						OK: false, Level: "fail",
						Message: fmt.Sprintf("Service %q: required env_file missing: %s", name, ef.Path),
					})
				}
			}
		}
	}

	for name, svc := range active {
		svc = expandService(svc, hostSubst)
		if svc.Build != nil && strings.TrimSpace(svc.Image) == "" {
			out = append(out, Result{
				OK: false, Level: "fail",
				Message: fmt.Sprintf("Service %q: build requires image tag", name),
			})
		}
		for _, d := range svc.DependsOn {
			if _, ok := active[d.Service]; !ok {
				if partial {
					if _, def := c.Services[d.Service]; !def {
						out = append(out, Result{
							OK: false, Level: "fail",
							Message: fmt.Sprintf("Service %q: depends on unknown service %q", name, d.Service),
						})
					}
				} else {
					out = append(out, Result{
						OK: false, Level: "fail",
						Message: fmt.Sprintf("Service %q: depends on %q which is not active (profile?)", name, d.Service),
					})
				}
			}
			if d.Condition == spec.ConditionHealthy {
				dep := c.Services[d.Service]
				if !dep.Health.HasProbe() {
					out = append(out, Result{
						OK: false, Level: "fail",
						Message: fmt.Sprintf("Service %q: depends on %q as healthy but that service has no health.http / health.exec", name, d.Service),
					})
				}
			}
		}
		for _, p := range svc.Ports {
			if !strings.Contains(p, ":") {
				out = append(out, Result{
					OK: false, Level: "fail",
					Message: fmt.Sprintf("Service %q: port %q must be host:container (e.g. 8080:80)", name, p),
				})
			}
		}
		for _, req := range svc.Requirements {
			out = append(out, evalRequirement(req, name, root)...)
		}
		if svc.Health != nil {
			if svc.Health.HTTP != nil && svc.Health.HTTP.URL == "" && svc.Health.Exec == nil {
				out = append(out, Result{OK: false, Level: "fail", Message: fmt.Sprintf("Service %q: health.http.url is empty", name)})
			}
			if svc.Health.Exec != nil && len(svc.Health.Exec.Command) == 0 && svc.Health.HTTP == nil {
				out = append(out, Result{OK: false, Level: "fail", Message: fmt.Sprintf("Service %q: health.exec.command is empty", name)})
			}
		}
	}

	for _, req := range c.Requirements {
		out = append(out, evalRequirement(req, "(contract)", root)...)
	}

	for name, svc := range active {
		svc = expandService(svc, hostSubst)
		if len(svc.Ports) > 0 && !svc.Health.HasProbe() {
			out = append(out, Result{
				OK: false, Level: "warn",
				Message: fmt.Sprintf("Service %q exposes host ports but has no health check", name),
			})
		}
	}

	return out
}

func expandService(svc spec.Service, host map[string]string) spec.Service {
	svc.Ports = expandStrs(svc.Ports, host)
	svc.Volumes = expandStrs(svc.Volumes, host)
	svc.Environment = expandMap(svc.Environment, host)
	svc.User = expand.String(svc.User, host)
	svc.DNS = expandStrs(svc.DNS, host)
	svc.ExtraHosts = spec.ExtraHostList(expandStrs([]string(svc.ExtraHosts), host))
	if svc.Health != nil && svc.Health.HTTP != nil {
		svc.Health.HTTP.URL = expand.String(svc.Health.HTTP.URL, host)
	}
	return svc
}

func expandStrs(in []string, m map[string]string) []string {
	if len(in) == 0 {
		return in
	}
	o := make([]string, len(in))
	for i, s := range in {
		o[i] = expand.String(s, m)
	}
	return o
}

func expandMap(in map[string]string, m map[string]string) map[string]string {
	if len(in) == 0 {
		return in
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = expand.String(v, m)
	}
	return out
}

func evalRequirement(r spec.Requirement, ctx string, root string) []Result {
	switch r.Type {
	case "":
		return []Result{{OK: false, Level: "fail", Message: fmt.Sprintf("%s: requirement missing type", ctx)}}
	case "port_available":
		if r.Port <= 0 || r.Port > 65535 {
			return []Result{{OK: false, Level: "fail", Message: fmt.Sprintf("%s: port_available needs valid port", ctx)}}
		}
		ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(r.Port))
		if err != nil {
			return []Result{{OK: false, Level: "fail", Message: fmt.Sprintf("%s: port %d not available (%v)", ctx, r.Port, err)}}
		}
		_ = ln.Close()
		return []Result{{OK: true, Level: "ok", Message: fmt.Sprintf("%s: port %d available", ctx, r.Port)}}
	case "path_writable":
		if r.Path == "" {
			return []Result{{OK: false, Level: "fail", Message: fmt.Sprintf("%s: path_writable needs path", ctx)}}
		}
		p := r.Path
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		if err := os.MkdirAll(p, 0o755); err != nil {
			return []Result{{OK: false, Level: "fail", Message: fmt.Sprintf("%s: cannot create path %q: %v", ctx, p, err)}}
		}
		f, err := os.CreateTemp(p, ".podbay-write-test-*")
		if err != nil {
			return []Result{{OK: false, Level: "fail", Message: fmt.Sprintf("%s: path not writable %q: %v", ctx, p, err)}}
		}
		_ = f.Close()
		_ = os.Remove(f.Name())
		return []Result{{OK: true, Level: "ok", Message: fmt.Sprintf("%s: path writable %q", ctx, p)}}
	case "command_exists":
		if r.Command == "" {
			return []Result{{OK: false, Level: "fail", Message: fmt.Sprintf("%s: command_exists needs command", ctx)}}
		}
		_, err := exec.LookPath(r.Command)
		if err != nil {
			return []Result{{OK: false, Level: "fail", Message: fmt.Sprintf("%s: command not found: %q", ctx, r.Command)}}
		}
		return []Result{{OK: true, Level: "ok", Message: fmt.Sprintf("%s: command exists: %q", ctx, r.Command)}}
	case "health_defined":
		return []Result{{OK: true, Level: "ok", Message: fmt.Sprintf("%s: health_defined check skipped at validate time (use deploy/explain)", ctx)}}
	default:
		return []Result{{OK: false, Level: "warn", Message: fmt.Sprintf("%s: unknown requirement type %q", ctx, r.Type)}}
	}
}

func redeployPeersContains(parent spec.Service, child string) bool {
	child = strings.TrimSpace(child)
	for _, x := range parent.RedeployPeers {
		if strings.TrimSpace(x) == child {
			return true
		}
	}
	return false
}
