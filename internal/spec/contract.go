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

// EnvFileEntry is one env_file item (path + required).
type EnvFileEntry struct {
	Path     string `yaml:"path"`
	Required bool   `yaml:"required"`
}

// EnvFileEntries unmarshals a list of paths or objects.
type EnvFileEntries []EnvFileEntry

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
