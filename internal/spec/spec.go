package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultFilename = "podbay.yaml"

// Condition* are normalized depends_on conditions (Compose: service_started / service_healthy).
const (
	ConditionStarted = "started"
	ConditionHealthy = "healthy"
)

// Contract is the root podbay.yaml document.
type Contract struct {
	Version      string             `yaml:"version"`
	Project      string             `yaml:"project"`
	HostEnvFiles []string           `yaml:"host_env_files"` // optional; default loads .env.example + .env when present
	Services     map[string]Service `yaml:"services"`
	Volumes      map[string]Volume  `yaml:"volumes"`
	Networks     map[string]Network `yaml:"networks"`
	// Network (singular) sets options for the implicit podbay_<project> bridge Podbay creates (not Compose networks:).
	Network *ProjectBridge `yaml:"network"`
	// Podman adjusts Podbay-specific defaults so Compose-shaped YAML works without extra Podman-only suffixes.
	Podman       *PodmanSection `yaml:"podman"`
	Requirements []Requirement  `yaml:"requirements"`
}

// PodmanSection is optional; all fields are off/zero by default (sensible Podman parity).
type PodmanSection struct {
	// DisableAutoVolumeU, when true, stops Podbay from appending :U to named volume mounts that omit options.
	// Leave false (omit podman:) so declared named volumes get Podman :U automatically (closer to Docker Desktop data-dir behavior).
	DisableAutoVolumeU bool `yaml:"disable_auto_volume_u"`
	// NetworkDNS is passed as repeated podman network create --dns when the project bridge is first created.
	// When empty, macOS/Windows still get a Podbay default (see DisableDefaultBridgeDNS).
	NetworkDNS []string `yaml:"network_dns"`
	// DisableDefaultBridgeDNS skips Podbay’s Podman-Machine default bridge DNS when network_dns is empty (advanced).
	DisableDefaultBridgeDNS bool `yaml:"disable_default_bridge_dns"`
}

// ProjectBridge is optional tuning for the single Podman bridge network created per project.
type ProjectBridge struct {
	// MTU is passed as podman network create --opt mtu=<n> when the network is first created (VPN / slirp issues).
	MTU int `yaml:"mtu"`
}

type Volume struct {
	Driver string `yaml:"driver"`
}

type Network struct {
	Driver string `yaml:"driver,omitempty"`
	// External, when true, means the Podman network already exists; deploy only joins it.
	External bool `yaml:"external,omitempty"`
	// Name is the Podman network name for external networks. When empty, the logical
	// networks: key is used (Compose default for external: true).
	Name string `yaml:"name,omitempty"`
}

// ExtraHostList matches Compose extra_hosts as either a list of "hostname:ip" strings or a map hostname → ip.
type ExtraHostList []string

// UnmarshalYAML accepts ["host.docker.internal:host-gateway"] or { host.docker.internal: host-gateway }.
func (e *ExtraHostList) UnmarshalYAML(n *yaml.Node) error {
	*e = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind == yaml.SequenceNode {
		for _, item := range n.Content {
			var s string
			if err := item.Decode(&s); err != nil {
				return err
			}
			*e = append(*e, s)
		}
		return nil
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i < len(n.Content); i += 2 {
			var k, v string
			if err := n.Content[i].Decode(&k); err != nil {
				return err
			}
			if err := n.Content[i+1].Decode(&v); err != nil {
				return err
			}
			k, v = strings.TrimSpace(k), strings.TrimSpace(v)
			if k == "" || v == "" {
				continue
			}
			*e = append(*e, k+":"+v)
		}
		return nil
	}
	return fmt.Errorf("extra_hosts: expected sequence or mapping")
}

// Build describes an image build (Compose-style); executed before run when set.
type Build struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}

// Service is one runnable unit.
type Service struct {
	Build     *Build       `yaml:"build"`
	Image     string       `yaml:"image"`
	DependsOn Dependencies `yaml:"depends_on"`
	// RedeployPeers (YAML: dependents) lists dependents: each named service X must list this service under
	// depends_on (Y is a dependency of X). With bidirectional validate, every depends_on edge is mirrored here.
	RedeployPeers []string          `yaml:"dependents,omitempty"`
	Ports         []string          `yaml:"ports"`
	Expose        []string          `yaml:"expose"` // informational; same network reaches ports without publish
	Volumes       []string          `yaml:"volumes"`
	Environment   map[string]string `yaml:"environment"`
	EnvFile       EnvFileEntries    `yaml:"env_file"`
	Command       []string          `yaml:"command"`
	Health        *HealthCheck      `yaml:"health"`
	Requirements  []Requirement     `yaml:"requirements"`
	Labels        map[string]string `yaml:"labels"`
	Restart       string            `yaml:"restart"`
	ExtraHosts    ExtraHostList     `yaml:"extra_hosts"`
	Profiles      []string          `yaml:"profiles"` // Compose profiles: omitted = always on; set = only with --profile
	// User is passed to podman run --user (uid:gid or name). Use with :U on named volumes when the image
	// defaults to root but the app runs as a fixed UID (e.g. 1000) so Podman chowns the volume correctly.
	User string `yaml:"user"`
	// DNS is passed as repeated podman run --dns <addr> (resolver / VPN parity with Compose dns:).
	DNS []string `yaml:"dns"`
	// AnsibleVaultPaths lists host-side bind sources (absolute paths) that must be decrypted with
	// ansible-vault before mounting. Optional; empty means all volume sources are plaintext.
	AnsibleVaultPaths []string `yaml:"ansible_vault_paths,omitempty"`
	// Networks lists logical keys from the contract's top-level networks: block. When the contract
	// defines exactly one network, this may be omitted (that network is implied). When multiple
	// networks exist, each active service must list the networks it joins.
	Networks []string `yaml:"networks,omitempty"`
}

// Dependency is an entry in depends_on.
type Dependency struct {
	Service   string `yaml:"service"`
	Condition string `yaml:"condition"` // started | healthy (Compose service_* accepted)
}

// Dependencies supports Compose list or map form.
type Dependencies []Dependency

// UnmarshalYAML accepts ["api"] or { api: { condition: service_healthy } }.
func (d *Dependencies) UnmarshalYAML(n *yaml.Node) error {
	*d = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind == yaml.SequenceNode {
		for _, c := range n.Content {
			var s string
			if err := c.Decode(&s); err != nil {
				return err
			}
			*d = append(*d, Dependency{Service: s, Condition: ConditionStarted})
		}
		return nil
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i < len(n.Content); i += 2 {
			var key string
			if err := n.Content[i].Decode(&key); err != nil {
				return err
			}
			var v struct {
				Condition string `yaml:"condition"`
			}
			if err := n.Content[i+1].Decode(&v); err != nil {
				return err
			}
			*d = append(*d, Dependency{Service: key, Condition: NormalizeCondition(v.Condition)})
		}
		return nil
	}
	return fmt.Errorf("depends_on: expected sequence or mapping")
}

// NormalizeCondition maps Compose condition strings to started|healthy.
func NormalizeCondition(c string) string {
	switch strings.ToLower(strings.TrimSpace(c)) {
	case "service_healthy", "healthy":
		return ConditionHealthy
	case "service_started", "started", "":
		return ConditionStarted
	default:
		return strings.ToLower(strings.TrimSpace(c))
	}
}

// EnvFileEntry is one env_file item (path + required).
type EnvFileEntry struct {
	Path     string `yaml:"path"`
	Required bool   `yaml:"required"`
}

// EnvFileEntries unmarshals a list of paths or objects.
type EnvFileEntries []EnvFileEntry

// UnmarshalYAML accepts ".env" or { path: .env, required: true }.
func (e *EnvFileEntries) UnmarshalYAML(n *yaml.Node) error {
	*e = nil
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil
	}
	if n.Kind != yaml.SequenceNode {
		return fmt.Errorf("env_file: expected sequence")
	}
	for _, item := range n.Content {
		if item.Kind == yaml.ScalarNode {
			*e = append(*e, EnvFileEntry{Path: item.Value, Required: false})
			continue
		}
		var row EnvFileEntry
		if err := item.Decode(&row); err != nil {
			return err
		}
		*e = append(*e, row)
	}
	return nil
}

type HealthCheck struct {
	HTTP        *HTTPHealth `yaml:"http"`
	Exec        *ExecHealth `yaml:"exec"`
	Interval    string      `yaml:"interval"`
	Timeout     string      `yaml:"timeout"`
	Retries     int         `yaml:"retries"`
	StartPeriod string      `yaml:"start_period"`
}

type HTTPHealth struct {
	URL      string `yaml:"url"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
	Insecure bool   `yaml:"insecure_tls"` // curl -k
}

type ExecHealth struct {
	Command []string `yaml:"command"`
}

// HasProbe reports whether any health probe is defined.
func (h *HealthCheck) HasProbe() bool {
	return h != nil && ((h.HTTP != nil && h.HTTP.URL != "") || (h.Exec != nil && len(h.Exec.Command) > 0))
}

// Requirement is a simple check type (contract-level or per-service).
type Requirement struct {
	Type    string `yaml:"type"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
	Command string `yaml:"command"`
}

// Load reads and parses podbay.yaml from path (file or directory).
func Load(path string) (*Contract, string, error) {
	var file string
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", err
	}
	if info.IsDir() {
		file = filepath.Join(path, DefaultFilename)
	} else {
		file = path
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, file, err
	}
	var c Contract
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, file, fmt.Errorf("parse yaml: %w", err)
	}
	if c.Version == "" {
		c.Version = "1"
	}
	if c.Services == nil {
		c.Services = map[string]Service{}
	}
	for name, svc := range c.Services {
		for i := range svc.DependsOn {
			svc.DependsOn[i].Condition = NormalizeCondition(svc.DependsOn[i].Condition)
			if svc.DependsOn[i].Condition != ConditionStarted && svc.DependsOn[i].Condition != ConditionHealthy {
				return nil, file, fmt.Errorf("service %q: depends_on %q: unknown condition %q (use started or healthy)", name, svc.DependsOn[i].Service, svc.DependsOn[i].Condition)
			}
		}
		svc.RedeployPeers = normalizeRedeployPeerNames(svc.RedeployPeers)
		c.Services[name] = svc
	}
	return &c, file, nil
}

func normalizeRedeployPeerNames(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

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

// ProjectName returns an identifier used for labels and container names.
func (c *Contract) ProjectName(defaultName string) string {
	if c.Project != "" {
		return sanitizeName(c.Project)
	}
	return sanitizeName(defaultName)
}

func sanitizeName(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out = append(out, r)
		case r == '-' || r == '_':
			out = append(out, '_')
		default:
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "podbay"
	}
	return string(out)
}

// IsIncluded implements Compose profiles: services without profiles always run;
// services with profiles run only when a matching --profile is passed.
func (s *Service) IsIncluded(profiles []string) bool {
	if len(s.Profiles) == 0 {
		return true
	}
	if len(profiles) == 0 {
		return false
	}
	want := map[string]struct{}{}
	for _, p := range profiles {
		want[p] = struct{}{}
	}
	for _, p := range s.Profiles {
		if _, ok := want[p]; ok {
			return true
		}
	}
	return false
}

// ServicesForProfiles returns the subset of services that should run.
func (c *Contract) ServicesForProfiles(profiles []string) map[string]Service {
	out := make(map[string]Service)
	for n, svc := range c.Services {
		if svc.IsIncluded(profiles) {
			out[n] = svc
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
