package composeimport

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/1eve1Up/podbay/internal/spec"
)

// StubFromDockerfile builds a first-pass single-service contract for a Dockerfile-only repo.
// project is a logical project name (directory basename is typical). dockerfileRel is the
// dockerfile path relative to build context (usually "Dockerfile").
// The stub includes build + image tag (required by validate/deploy).
// Ports, health, and host tooling requirements are omitted — validate / hand-tighten remains required.
func StubFromDockerfile(project, dockerfileRel string) *spec.Contract {
	proj := strings.TrimSpace(project)
	if proj == "" {
		proj = "app"
	}
	df := strings.TrimSpace(dockerfileRel)
	if df == "" {
		df = "Dockerfile"
	}
	// No command_exists podman requirement — matches Compose import honesty and keeps
	// offline validate → hand-tighten viable without inventing host tooling checks.
	c := &spec.Contract{
		Version: "1",
		Project: proj,
		Services: map[string]spec.Service{
			"app": {
				Build: &spec.Build{
					Context:    ".",
					Dockerfile: df,
				},
				Image: fmt.Sprintf("localhost/%s:local", cProjectSlug(proj)),
			},
		},
		Volumes:  map[string]spec.Volume{},
		Networks: map[string]spec.Network{},
	}
	return c
}

// DockerfileRelForStub returns a dockerfile path suitable for build.dockerfile
// relative to absDir when possible.
func DockerfileRelForStub(absDir, absDockerfile string) string {
	rel, err := filepath.Rel(absDir, absDockerfile)
	if err != nil || rel == "" || strings.HasPrefix(rel, "..") {
		return filepath.Base(absDockerfile)
	}
	return filepath.ToSlash(rel)
}

func cProjectSlug(project string) string {
	c := &spec.Contract{Project: project}
	return c.ProjectName("app")
}
