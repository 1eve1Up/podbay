package orientation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/spec"
)

func loadFixtureContract(t *testing.T) (*spec.Contract, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	yaml := `
version: "1"
project: livefix
services:
  web:
    image: docker.io/library/nginx:alpine
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	c, loaded, err := spec.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return c, loaded
}

func TestAttachRuntime_unhealthyAdjustsNextActions(t *testing.T) {
	c, path := loadFixtureContract(t)
	doc, err := Build(c, path, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	idle := append([]string(nil), doc.NextActions...)
	healthy := false
	AttachRuntime(doc, true, []RuntimeService{
		{Name: "web", Running: true, State: "running", Healthy: &healthy},
	})
	if doc.Runtime == nil || !doc.Runtime.Available || len(doc.Runtime.Services) != 1 {
		t.Fatalf("runtime: %+v", doc.Runtime)
	}
	joined := strings.Join(doc.NextActions, "\n")
	if !strings.Contains(joined, "podbay logs") || !strings.Contains(joined, "podbay explain") || !strings.Contains(joined, "podbay down") {
		t.Fatalf("unhealthy next_actions: %v", doc.NextActions)
	}
	if strings.Join(idle, "\n") == joined {
		t.Fatal("next_actions should change from idle when unhealthy")
	}
	if strings.Contains(joined, HandTightenHint) {
		t.Fatalf("unhealthy playbook should not add hand-tighten: %v", doc.NextActions)
	}
}

func TestAttachRuntime_unavailableKeepsIdleHints(t *testing.T) {
	c, path := loadFixtureContract(t)
	doc, err := Build(c, path, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	AttachRuntime(doc, false, nil)
	if doc.Runtime == nil || doc.Runtime.Available {
		t.Fatalf("expected unavailable runtime, got %+v", doc.Runtime)
	}
	joined := strings.Join(doc.NextActions, "\n")
	if !strings.Contains(joined, "podbay validate") || !strings.Contains(joined, "podbay deploy") {
		t.Fatalf("idle next_actions expected: %v", doc.NextActions)
	}
	if !strings.Contains(joined, HandTightenHint) {
		t.Fatalf("unavailable attach should keep hand-tighten on a thin contract: %v", doc.NextActions)
	}
}

func TestAttachRuntime_allMissingKeepsHandTighten(t *testing.T) {
	c, path := loadFixtureContract(t)
	doc, err := Build(c, path, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	AttachRuntime(doc, true, []RuntimeService{
		{Name: "web", Missing: true},
	})
	joined := strings.Join(doc.NextActions, "\n")
	if !strings.Contains(joined, "podbay validate") || !strings.Contains(joined, "podbay deploy") {
		t.Fatalf("all-missing next_actions: %v", doc.NextActions)
	}
	if !strings.Contains(joined, HandTightenHint) {
		t.Fatalf("arrive with Podman but nothing deployed must keep hand-tighten: %v", doc.NextActions)
	}
}

func TestAttachRuntime_healthyDiffPath(t *testing.T) {
	c, path := loadFixtureContract(t)
	doc, err := Build(c, path, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ok := true
	AttachRuntime(doc, true, []RuntimeService{
		{Name: "web", Running: true, State: "running", Healthy: &ok},
	})
	joined := strings.Join(doc.NextActions, "\n")
	if !strings.Contains(joined, "podbay diff") {
		t.Fatalf("healthy path should suggest diff: %v", doc.NextActions)
	}
	if strings.Contains(joined, "podbay down") {
		t.Fatalf("healthy path should not lead with down: %v", doc.NextActions)
	}
	if strings.Contains(joined, HandTightenHint) {
		t.Fatalf("healthy playbook should not add hand-tighten: %v", doc.NextActions)
	}
}
