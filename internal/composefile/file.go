// Package composefile parses Docker Compose YAML for the import ingest phase only.
// Runtime commands use spec, not composefile; see docs/architecture.md.
package composefile

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// File is the top-level compose document (subset used for Podbay import).
type File struct {
	Version  string                 `yaml:"version"`
	Include  []IncludeEntry         `yaml:"include,omitempty"`
	Services map[string]ServiceSpec `yaml:"services"`
	Volumes  map[string]VolumeSpec  `yaml:"volumes"`
	Networks map[string]NetworkSpec `yaml:"networks"`
	Configs  map[string]ConfigSpec  `yaml:"configs,omitempty"`
	Secrets  map[string]SecretSpec  `yaml:"secrets,omitempty"`
}

// VolumeSpec is a named volume declaration (Compose top-level volumes:).
type VolumeSpec struct {
	Driver string `yaml:"driver,omitempty"`
}

// ServiceSpec is one compose service (subset).
type ServiceSpec struct {
	Extends     *ExtendsRef         `yaml:"extends,omitempty"`
	Image       string              `yaml:"image,omitempty"`
	Build       *BuildSpec          `yaml:"build,omitempty"`
	Ports       PortStrings         `yaml:"ports,omitempty"`
	Expose      []string            `yaml:"expose,omitempty"`
	Volumes     []string            `yaml:"volumes,omitempty"`
	Environment StringOrMap         `yaml:"environment,omitempty"`
	EnvFile     EnvFileList         `yaml:"env_file,omitempty"`
	DependsOn   DependsOnList       `yaml:"depends_on,omitempty"`
	Healthcheck *HealthcheckSpec    `yaml:"healthcheck,omitempty"`
	Profiles    []string            `yaml:"profiles,omitempty"`
	Command     StringOrList        `yaml:"command,omitempty"`
	Restart     string              `yaml:"restart,omitempty"`
	Labels      map[string]string   `yaml:"labels,omitempty"`
	ExtraHosts  ExtraHostCompose    `yaml:"extra_hosts,omitempty"`
	User        string              `yaml:"user,omitempty"`
	DNS         []string            `yaml:"dns,omitempty"`
	Networks    ServiceNetworkNames `yaml:"networks,omitempty"`
	Configs     ServiceConfigs      `yaml:"configs,omitempty"`
	Secrets     ServiceSecrets      `yaml:"secrets,omitempty"`
}

// BuildSpec is build: context or { context, dockerfile }.
type BuildSpec struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile,omitempty"`
}

// Parse decodes compose YAML into a File. Omitted maps are initialized empty.
func Parse(data []byte) (*File, error) {
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, ParseError(err)
	}
	if f.Services == nil {
		f.Services = map[string]ServiceSpec{}
	}
	if f.Volumes == nil {
		f.Volumes = map[string]VolumeSpec{}
	}
	if f.Networks == nil {
		f.Networks = map[string]NetworkSpec{}
	}
	if f.Configs == nil {
		f.Configs = map[string]ConfigSpec{}
	}
	if f.Secrets == nil {
		f.Secrets = map[string]SecretSpec{}
	}
	return &f, nil
}

// Load reads a compose file from disk, parses it, resolves Compose `include:` merges,
// then resolves `extends:` chains.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, ReadError(path, err)
	}
	f, err := Parse(data)
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	if err := ResolveIncludeMergeOnly(abs, f, nil); err != nil {
		return nil, fmt.Errorf("composefile: %w", err)
	}
	if err := ResolveExtends(abs, f); err != nil {
		return nil, fmt.Errorf("composefile: %w", err)
	}
	return f, nil
}
