package orientation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/spec"
)

func TestBuild_idleContract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	yaml := `
version: "1"
project: orientdemo
services:
  web:
    image: docker.io/library/nginx:alpine
    depends_on:
      api:
        condition: service_healthy
  api:
    image: docker.io/library/nginx:alpine
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	c, loaded, err := spec.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := Build(c, loaded, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if doc.FormatVersion != FormatVersion || doc.Kind != Kind {
		t.Fatalf("identity kind/version: %+v", doc)
	}
	if doc.Project != "orientdemo" || doc.ContractPath != loaded {
		t.Fatalf("identity fields: project=%q path=%q", doc.Project, doc.ContractPath)
	}
	if doc.Note != BoundaryNote {
		t.Fatalf("note: %q", doc.Note)
	}
	if len(doc.ActiveServices) != 2 {
		t.Fatalf("active_services: %v", doc.ActiveServices)
	}
	if doc.Runtime != nil {
		t.Fatalf("offline build must omit runtime, got %+v", doc.Runtime)
	}
	if len(doc.Graph) != 2 {
		t.Fatalf("graph len=%d", len(doc.Graph))
	}
	var web *GraphService
	for i := range doc.Graph {
		if doc.Graph[i].Name == "web" {
			web = &doc.Graph[i]
			break
		}
	}
	if web == nil || len(web.DependsOn) != 1 || web.DependsOn[0].Service != "api" {
		t.Fatalf("web graph skim: %+v", web)
	}
	if len(doc.NextActions) < 3 {
		t.Fatalf("next_actions too short: %v", doc.NextActions)
	}
	joined := strings.Join(doc.NextActions, "\n")
	for _, gate := range []string{"validate", "deploy", "diff", "explain"} {
		if !strings.Contains(joined, "podbay "+gate) {
			t.Fatalf("next_actions missing %s: %v", gate, doc.NextActions)
		}
	}
	for _, bad := range []string{"auto-fix", "remediat", "root-cause", "rollback"} {
		if strings.Contains(strings.ToLower(joined), bad) {
			t.Fatalf("next_actions must not invent remediation (%s): %v", bad, doc.NextActions)
		}
	}
}

func TestBuild_requirementsSkim(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	yaml := `
version: "1"
project: reqs
services:
  web:
    image: docker.io/library/nginx:alpine
    ports:
      - "8080:80"
    health:
      http:
        url: http://127.0.0.1:8080/
  app:
    build:
      context: .
      dockerfile: Dockerfile
    image: localhost/reqs:local
    expose:
      - "8080"
    health:
      exec:
        command: [wget, -q, -O-, http://127.0.0.1/]
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	c, loaded, err := spec.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Build(c, loaded, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]GraphService{}
	for _, g := range doc.Graph {
		byName[g.Name] = g
	}
	web := byName["web"]
	if web.Source != SourceImage || web.Health != HealthHTTP {
		t.Fatalf("web skim: %+v", web)
	}
	if len(web.Ports) != 1 || web.Ports[0] != "8080:80" {
		t.Fatalf("web ports: %v", web.Ports)
	}
	app := byName["app"]
	if app.Source != SourceBuild || app.Health != HealthExec {
		t.Fatalf("app skim: %+v", app)
	}
	if len(app.Expose) != 1 || app.Expose[0] != "8080" {
		t.Fatalf("app expose: %v", app.Expose)
	}
	if len(app.Ports) != 0 {
		t.Fatalf("app must not invent ports: %v", app.Ports)
	}
}

func TestBuild_handTightenHintOnThinContract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	yaml := `
version: "1"
project: thin
services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    image: localhost/thin:local
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	c, loaded, err := spec.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Build(c, loaded, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(doc.NextActions, "\n")
	if !strings.Contains(joined, "podbay validate") {
		t.Fatalf("still need validate: %v", doc.NextActions)
	}
	if !strings.Contains(joined, HandTightenHint) {
		t.Fatalf("missing hand-tighten: %v", doc.NextActions)
	}
}

func TestBuild_noHandTightenWhenPortsAndHealthPresent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	yaml := `
version: "1"
project: ready
services:
  web:
    image: docker.io/library/nginx:alpine
    ports:
      - "8080:80"
    health:
      http:
        url: http://127.0.0.1:8080/
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	c, loaded, err := spec.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Build(c, loaded, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(doc.NextActions, "\n"), HandTightenHint) {
		t.Fatalf("complete contract should not hint hand-tighten: %v", doc.NextActions)
	}
}

func TestBuild_nilContract(t *testing.T) {
	_, err := Build(nil, "/x/podbay.yaml", BuildOptions{})
	if err == nil {
		t.Fatal("expected error for nil contract")
	}
}

func TestBuild_emptyPath(t *testing.T) {
	_, err := Build(&spec.Contract{Project: "x", Services: map[string]spec.Service{"a": {}}}, "", BuildOptions{})
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestBuild_noServices(t *testing.T) {
	_, err := Build(&spec.Contract{Project: "x"}, "/tmp/podbay.yaml", BuildOptions{})
	if err == nil {
		t.Fatal("expected error for empty services")
	}
}

func TestBuild_partialRoots(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	yaml := `
version: "1"
project: partial
services:
  web:
    image: nginx
    depends_on:
      - api
    dependents: []
  api:
    image: nginx
    dependents: [web]
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	c, loaded, err := spec.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Build(c, loaded, BuildOptions{DeployRoots: []string{"api"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.DeployServices) != 1 || doc.DeployServices[0] != "api" {
		t.Fatalf("deploy_services: %v", doc.DeployServices)
	}
	if len(doc.ActiveServices) != 1 || doc.ActiveServices[0] != "api" {
		t.Fatalf("active without dependents expand: %v", doc.ActiveServices)
	}
	joined := strings.Join(doc.NextActions, " ")
	if !strings.Contains(joined, "api") {
		t.Fatalf("next_actions should include root: %v", doc.NextActions)
	}
}
