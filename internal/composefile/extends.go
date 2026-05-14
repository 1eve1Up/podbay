package composefile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxExtendsDepth = 16

// ExtendsRef is Compose `extends:` (same file or file+service).
type ExtendsRef struct {
	File    string `yaml:"file,omitempty"`
	Service string `yaml:"service,omitempty"`
}

// UnmarshalYAML accepts a bare service name (same file) or { file?, service }.
func (e *ExtendsRef) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		e.File = ""
		e.Service = strings.TrimSpace(n.Value)
		if e.Service == "" {
			return fmt.Errorf("extends: empty service name")
		}
		return nil
	}
	if n.Kind == yaml.MappingNode {
		type raw struct {
			File    string `yaml:"file"`
			Service string `yaml:"service"`
		}
		var r raw
		if err := n.Decode(&r); err != nil {
			return err
		}
		e.File = strings.TrimSpace(r.File)
		e.Service = strings.TrimSpace(r.Service)
		if e.Service == "" {
			return fmt.Errorf("extends: service name is required")
		}
		return nil
	}
	return fmt.Errorf("extends: expected string or mapping")
}

// mergeTopLevelInto copies networks, volumes, configs, and secrets from src into dst
// when the key is missing in dst (primary compose file wins on conflict).
func mergeTopLevelInto(dst, src *File) {
	if dst == nil || src == nil {
		return
	}
	for k, v := range src.Networks {
		if _, ok := dst.Networks[k]; !ok {
			dst.Networks[k] = v
		}
	}
	for k, v := range src.Volumes {
		if _, ok := dst.Volumes[k]; !ok {
			dst.Volumes[k] = v
		}
	}
	for k, v := range src.Configs {
		if _, ok := dst.Configs[k]; !ok {
			dst.Configs[k] = v
		}
	}
	for k, v := range src.Secrets {
		if _, ok := dst.Secrets[k]; !ok {
			dst.Secrets[k] = v
		}
	}
}

func mergeStringOrMap(base, over StringOrMap) StringOrMap {
	if len(base) == 0 && len(over) == 0 {
		return nil
	}
	out := make(StringOrMap)
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

func mergeStringMaps(base, over map[string]string) map[string]string {
	if len(base) == 0 && len(over) == 0 {
		return nil
	}
	out := make(map[string]string)
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

func mergeDependsOn(base, over DependsOnList) DependsOnList {
	if len(base) == 0 && len(over) == 0 {
		return nil
	}
	byName := make(map[string]DependsOnEntry)
	var order []string
	for _, e := range base {
		s := strings.TrimSpace(e.Service)
		if s == "" {
			continue
		}
		if _, ok := byName[s]; !ok {
			order = append(order, s)
		}
		byName[s] = e
	}
	for _, e := range over {
		s := strings.TrimSpace(e.Service)
		if s == "" {
			continue
		}
		if _, ok := byName[s]; !ok {
			order = append(order, s)
		}
		byName[s] = e
	}
	out := make(DependsOnList, 0, len(order))
	for _, s := range order {
		out = append(out, byName[s])
	}
	return out
}

func unionSortedStrings(a, b []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range a {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	for _, s := range b {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// mergeServiceSpecs merges parent (base) into child (over). Compose-style: child fields
// override when set; lists for ports/volumes/env_file/extra_hosts append parent then child.
func mergeServiceSpecs(base, over ServiceSpec) ServiceSpec {
	out := base

	if strings.TrimSpace(over.Image) != "" {
		out.Image = over.Image
	}
	if over.Build != nil {
		out.Build = over.Build
	}
	out.Ports = append(append([]string{}, base.Ports...), over.Ports...)
	out.Volumes = append(append([]string{}, base.Volumes...), over.Volumes...)
	out.Environment = mergeStringOrMap(base.Environment, over.Environment)
	if len(base.EnvFile) > 0 || len(over.EnvFile) > 0 {
		out.EnvFile = append(append([]EnvFileEntry{}, base.EnvFile...), over.EnvFile...)
	}
	out.DependsOn = mergeDependsOn(base.DependsOn, over.DependsOn)

	if over.Healthcheck != nil {
		out.Healthcheck = over.Healthcheck
	}
	out.Profiles = unionSortedStrings(base.Profiles, over.Profiles)
	if len(over.Command) > 0 {
		out.Command = over.Command
	}
	if strings.TrimSpace(over.Restart) != "" {
		out.Restart = over.Restart
	}
	out.Labels = mergeStringMaps(base.Labels, over.Labels)
	if len(base.ExtraHosts) > 0 || len(over.ExtraHosts) > 0 {
		out.ExtraHosts = append(append(ExtraHostCompose{}, base.ExtraHosts...), over.ExtraHosts...)
	}
	if strings.TrimSpace(over.User) != "" {
		out.User = over.User
	}
	if len(base.DNS) > 0 || len(over.DNS) > 0 {
		out.DNS = append(append([]string{}, base.DNS...), over.DNS...)
	}
	if len(over.Networks) > 0 {
		out.Networks = over.Networks
	}
	if len(base.Configs) > 0 || len(over.Configs) > 0 {
		out.Configs = append(append(ServiceConfigs{}, base.Configs...), over.Configs...)
	}
	if len(base.Secrets) > 0 || len(over.Secrets) > 0 {
		out.Secrets = append(append(ServiceSecrets{}, base.Secrets...), over.Secrets...)
	}
	return out
}

type extendsCacheKey struct {
	file string
	svc  string
}

// ResolveExtends expands services[].extends in-place on the primary file f.
// primaryAbs must be an absolute path to the compose file that produced f.
func ResolveExtends(primaryAbs string, f *File) error {
	if f == nil {
		return nil
	}
	primaryAbs = filepath.Clean(primaryAbs)
	primaryDir := filepath.Dir(primaryAbs)
	fileMemo := make(map[string]*File) // abs -> parsed (unresolved body)
	fileMemo[primaryAbs] = f

	loadUnresolved := func(abs string) (*File, error) {
		abs = filepath.Clean(abs)
		if ff, ok := fileMemo[abs]; ok {
			return ff, nil
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		parsed, err := Parse(data)
		if err != nil {
			return nil, err
		}
		fileMemo[abs] = parsed
		return parsed, nil
	}

	toplevelMerged := make(map[string]bool)
	resolved := make(map[extendsCacheKey]ServiceSpec)

	var resolveInFile func(fileAbs string, fileF *File, svcName string, chain []string) (ServiceSpec, error)
	resolveInFile = func(fileAbs string, fileF *File, svcName string, chain []string) (ServiceSpec, error) {
		fileAbs = filepath.Clean(fileAbs)
		ck := extendsCacheKey{file: fileAbs, svc: svcName}
		if r, ok := resolved[ck]; ok {
			return r, nil
		}
		loopKey := fileAbs + "#" + svcName
		for _, c := range chain {
			if c == loopKey {
				return ServiceSpec{}, fmt.Errorf("cyclic extends involving service %q", svcName)
			}
		}
		if len(chain) >= maxExtendsDepth {
			return ServiceSpec{}, fmt.Errorf("extends: chain deeper than %d", maxExtendsDepth)
		}

		s, ok := fileF.Services[svcName]
		if !ok {
			return ServiceSpec{}, fmt.Errorf("unknown service %q", svcName)
		}
		if s.Extends == nil {
			resolved[ck] = s
			return s, nil
		}
		ref := s.Extends
		if strings.TrimSpace(ref.Service) == "" {
			return ServiceSpec{}, fmt.Errorf("extends: service name is required")
		}
		if strings.Contains(ref.File, "://") {
			return ServiceSpec{}, fmt.Errorf("extends: URL files are not supported (%q)", ref.File)
		}
		if filepath.IsAbs(ref.File) {
			return ServiceSpec{}, fmt.Errorf("extends: absolute file paths are not supported (%q)", ref.File)
		}

		nextChain := append(append([]string(nil), chain...), loopKey)

		var base ServiceSpec
		var err error
		if ref.File == "" {
			base, err = resolveInFile(fileAbs, fileF, ref.Service, nextChain)
		} else {
			subAbs := filepath.Clean(filepath.Join(filepath.Dir(fileAbs), ref.File))
			if err := assertIncludePathUnderComposeRoot(primaryDir, subAbs); err != nil {
				return ServiceSpec{}, fmt.Errorf("extends: %q: %w", ref.File, err)
			}
			subF, err2 := loadUnresolved(subAbs)
			if err2 != nil {
				return ServiceSpec{}, fmt.Errorf("extends: read %q: %w", ref.File, err2)
			}
			if !toplevelMerged[subAbs] {
				mergeTopLevelInto(f, subF)
				toplevelMerged[subAbs] = true
			}
			base, err = resolveInFile(subAbs, subF, ref.Service, nextChain)
		}
		if err != nil {
			return ServiceSpec{}, err
		}
		merged := mergeServiceSpecs(base, s)
		merged.Extends = nil
		resolved[ck] = merged
		return merged, nil
	}

	names := make([]string, 0, len(f.Services))
	for k := range f.Services {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		rs, err := resolveInFile(primaryAbs, f, name, nil)
		if err != nil {
			return fmt.Errorf("service %q: %w", name, err)
		}
		f.Services[name] = rs
	}
	return nil
}
