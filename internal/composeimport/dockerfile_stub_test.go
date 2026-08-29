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

func TestStubFromDockerfileScan_exposeAndHealth(t *testing.T) {
	c := StubFromDockerfileScan("myapp", "Dockerfile", DockerfileScan{
		Expose: []string{"80", "443"},
		Health: []string{"wget", "-q", "-O-", "http://127.0.0.1/"},
	})
	svc := c.Services["app"]
	if len(svc.Ports) != 0 {
		t.Fatalf("must not invent published ports: %+v", svc.Ports)
	}
	if !stringsEqual(svc.Expose, []string{"80", "443"}) {
		t.Fatalf("expose=%v", svc.Expose)
	}
	if svc.Health == nil || svc.Health.Exec == nil {
		t.Fatalf("health: %+v", svc.Health)
	}
	if !stringsEqual(svc.Health.Exec.Command, []string{"wget", "-q", "-O-", "http://127.0.0.1/"}) {
		t.Fatalf("health.exec=%v", svc.Health.Exec.Command)
	}
	raw, err := MarshalContract(c)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "expose:") || !strings.Contains(s, "- \"80\"") || !strings.Contains(s, "health:") {
		t.Fatalf("yaml: %s", s)
	}
	if strings.Contains(s, "ports:") {
		t.Fatalf("yaml must not invent ports: %s", s)
	}
}

func TestStubFromDockerfileScan_emptyStaysBuildOnly(t *testing.T) {
	c := StubFromDockerfileScan("myapp", "Dockerfile", DockerfileScan{})
	svc := c.Services["app"]
	if len(svc.Expose) != 0 || len(svc.Ports) != 0 || svc.Health != nil {
		t.Fatalf("empty scan must stay build-only: %+v", svc)
	}
}

func stringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDockerfileRelForStub(t *testing.T) {
	dir := t.TempDir()
	df := filepath.Join(dir, "Dockerfile")
	if got := DockerfileRelForStub(dir, df); got != "Dockerfile" {
		t.Fatalf("got %q", got)
	}
}
