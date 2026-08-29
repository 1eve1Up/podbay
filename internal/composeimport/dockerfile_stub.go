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
// Ports and health are omitted unless applied via StubFromDockerfileScan.
func StubFromDockerfile(project, dockerfileRel string) *spec.Contract {
	return StubFromDockerfileScan(project, dockerfileRel, DockerfileScan{})
}

// StubFromDockerfileScan is StubFromDockerfile plus declared EXPOSE → expose and
// HEALTHCHECK → health.exec. It does not invent host:container ports.
func StubFromDockerfileScan(project, dockerfileRel string, scan DockerfileScan) *spec.Contract {
	proj := strings.TrimSpace(project)
	if proj == "" {
		proj = "app"
	}
	df := strings.TrimSpace(dockerfileRel)
	if df == "" {
		df = "Dockerfile"
	}
	app := spec.Service{
		Build: &spec.Build{
			Context:    ".",
			Dockerfile: df,
		},
		Image: fmt.Sprintf("localhost/%s:local", cProjectSlug(proj)),
	}
	if len(scan.Expose) > 0 {
		app.Expose = append([]string(nil), scan.Expose...)
	}
	if len(scan.Health) > 0 {
		app.Health = &spec.HealthCheck{
			Exec: &spec.ExecHealth{Command: append([]string(nil), scan.Health...)},
		}
	}
	// No command_exists podman requirement — matches Compose import honesty and keeps
	// offline validate → hand-tighten viable without inventing host tooling checks.
	return &spec.Contract{
		Version: "1",
		Project: proj,
		Services: map[string]spec.Service{
			"app": app,
		},
		Volumes:  map[string]spec.Volume{},
		Networks: map[string]spec.Network{},
	}
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
