package composeimport

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStubFromDockerfile_emitsBuildAndImage(t *testing.T) {
	c := StubFromDockerfile("myapp", "Dockerfile")
	if c == nil || c.Project != "myapp" {
		t.Fatalf("project: %+v", c)
	}
	svc, ok := c.Services["app"]
	if !ok || svc.Build == nil {
		t.Fatalf("services: %+v", c.Services)
	}
	if svc.Build.Context != "." || svc.Build.Dockerfile != "Dockerfile" {
		t.Fatalf("build: %+v", svc.Build)
	}
	if svc.Image != "localhost/myapp:local" {
		t.Fatalf("image=%q", svc.Image)
	}
	if len(svc.Ports) != 0 || svc.Health != nil {
		t.Fatalf("stub must not invent ports/health: %+v", svc)
	}
	raw, err := MarshalContract(c)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "dockerfile: Dockerfile") || !strings.Contains(s, "image: localhost/myapp:local") {
		t.Fatalf("yaml: %s", s)
	}
}

func TestDockerfileRelForStub(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	if got := DockerfileRelForStub(dir, df); got != "Dockerfile" {
		t.Fatalf("got %q", got)
	}
}
