package composeimport

import (
	"sort"
	"strings"

	"github.com/1eve1Up/podbay/internal/spec"
)

// emitRoot is a YAML-friendly contract shape for import output (spec.Load-compatible).
type emitRoot struct {
	Version      string                 `yaml:"version"`
	Project      string                 `yaml:"project,omitempty"`
	HostEnvFiles []string               `yaml:"host_env_files,omitempty"`
	Services     map[string]emitService `yaml:"services"`
	Volumes      map[string]emitVolume  `yaml:"volumes,omitempty"`
	Networks     map[string]emitNetwork `yaml:"networks,omitempty"`
	Requirements []spec.Requirement     `yaml:"requirements,omitempty"`
}

type emitVolume struct {
	Driver string `yaml:"driver,omitempty"`
}

type emitNetwork struct {
	Driver   string `yaml:"driver,omitempty"`
	External bool   `yaml:"external,omitempty"`
	Name     string `yaml:"name,omitempty"`
}

type emitService struct {
	Build        *spec.Build         `yaml:"build,omitempty"`
	Image        string              `yaml:"image,omitempty"`
	DependsOn    interface{}         `yaml:"depends_on,omitempty"` // []string or map[string]map[string]string
	Dependents   []string            `yaml:"dependents,omitempty"` // parent lists dependents (child depends_on parent)
	Ports        []string            `yaml:"ports,omitempty"`
	Expose       []string            `yaml:"expose,omitempty"`
	Volumes      []string            `yaml:"volumes,omitempty"`
	Environment  map[string]string   `yaml:"environment,omitempty"`
	EnvFile      spec.EnvFileEntries `yaml:"env_file,omitempty"`
	Command      []string            `yaml:"command,omitempty"`
	Health       *spec.HealthCheck   `yaml:"health,omitempty"`
	Requirements []spec.Requirement  `yaml:"requirements,omitempty"`
	Labels       map[string]string   `yaml:"labels,omitempty"`
	Restart      string              `yaml:"restart,omitempty"`
	ExtraHosts   spec.ExtraHostList  `yaml:"extra_hosts,omitempty"`
	Profiles     []string            `yaml:"profiles,omitempty"`
	User         string              `yaml:"user,omitempty"`
	DNS          []string            `yaml:"dns,omitempty"`
	// AnsibleVaultPaths round-trips spec.Service.AnsibleVaultPaths (optional).
	AnsibleVaultPaths []string `yaml:"ansible_vault_paths,omitempty"`
	// Networks lists logical keys from top-level networks: (Compose services[].networks).
	Networks []string `yaml:"networks,omitempty"`
}

func contractToEmitRoot(c *spec.Contract) *emitRoot {
	if c == nil {
		return nil
	}
	ver := c.Version
	if ver == "" {
		ver = "1"
	}
	out := &emitRoot{
		Version:      ver,
		Project:      c.Project,
		HostEnvFiles: append([]string(nil), c.HostEnvFiles...),
		Services:     make(map[string]emitService),
		Requirements: append([]spec.Requirement(nil), c.Requirements...),
	}
	if len(c.Volumes) > 0 {
		out.Volumes = make(map[string]emitVolume)
		for k, v := range c.Volumes {
			out.Volumes[k] = emitVolume{Driver: v.Driver}
		}
	}
	if len(c.Networks) > 0 {
		out.Networks = make(map[string]emitNetwork)
		for k, v := range c.Networks {
			out.Networks[k] = emitNetwork{
				Driver:   v.Driver,
				External: v.External,
				Name:     v.Name,
			}
		}
	}
	for name, svc := range c.Services {
		out.Services[name] = emitServiceFromSpec(c, name, svc)
	}
	return out
}

func dependentsInContract(c *spec.Contract, parent string) []string {
	if c == nil {
		return nil
	}
	parent = strings.TrimSpace(parent)
	var out []string
	for childName, ch := range c.Services {
		if spec.DependsOnContains(ch, parent) {
			out = append(out, childName)
		}
	}
	sort.Strings(out)
	return out
}

func mergedDependentsYAML(c *spec.Contract, name string, svc spec.Service) []string {
	m := map[string]struct{}{}
	for _, x := range svc.RedeployPeers {
		x = strings.TrimSpace(x)
		if x != "" {
			m[x] = struct{}{}
		}
	}
	for _, x := range dependentsInContract(c, name) {
		m[x] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for x := range m {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

func emitServiceFromSpec(c *spec.Contract, name string, svc spec.Service) emitService {
	es := emitService{
		Build:        svc.Build,
		Image:        svc.Image,
		DependsOn:    dependsOnForEmit(svc.DependsOn),
		Ports:        append([]string(nil), svc.Ports...),
		Expose:       append([]string(nil), svc.Expose...),
		Volumes:      append([]string(nil), svc.Volumes...),
		Command:      append([]string(nil), svc.Command...),
		Health:       svc.Health,
		Requirements: append([]spec.Requirement(nil), svc.Requirements...),
		Restart:      svc.Restart,
		User:         svc.User,
		Profiles:     append([]string(nil), svc.Profiles...),
		DNS:          append([]string(nil), svc.DNS...),
	}
	if len(svc.Environment) > 0 {
		es.Environment = make(map[string]string)
		for k, v := range svc.Environment {
			es.Environment[k] = v
		}
	}
	if len(svc.EnvFile) > 0 {
		es.EnvFile = append(spec.EnvFileEntries(nil), svc.EnvFile...)
	}
	if len(svc.Labels) > 0 {
		es.Labels = make(map[string]string)
		for k, v := range svc.Labels {
			es.Labels[k] = v
		}
	}
	if len(svc.ExtraHosts) > 0 {
		es.ExtraHosts = append(spec.ExtraHostList(nil), svc.ExtraHosts...)
	}
	if len(svc.AnsibleVaultPaths) > 0 {
		es.AnsibleVaultPaths = append([]string(nil), svc.AnsibleVaultPaths...)
	}
	if len(svc.Networks) > 0 {
		es.Networks = append([]string(nil), svc.Networks...)
	}
	if deps := mergedDependentsYAML(c, name, svc); len(deps) > 0 {
		es.Dependents = deps
	}
	return es
}

func dependsOnForEmit(deps spec.Dependencies) interface{} {
	if len(deps) == 0 {
		return nil
	}
	allStarted := true
	for _, d := range deps {
		if d.Condition != spec.ConditionStarted {
			allStarted = false
			break
		}
	}
	if allStarted {
		out := make([]string, len(deps))
		for i, d := range deps {
			out[i] = d.Service
		}
		return out
	}
	m := make(map[string]map[string]string)
	for _, d := range deps {
		cond := "service_started"
		if d.Condition == spec.ConditionHealthy {
			cond = "service_healthy"
		}
		m[d.Service] = map[string]string{"condition": cond}
	}
	return m
}
