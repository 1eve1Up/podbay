package explain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/orientation"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestExplainJSONV1RequiredKeys(t *testing.T) {
	doc := explainJSONV1{
		FormatVersion:  ExplainJSONFormatVersion,
		Project:        "demo",
		ContractPath:   "/app/podbay.yaml",
		ActiveServices: []string{"web"},
		Services:       []serviceJSON{{Name: "web", Container: "podbay_demo_web", Running: true}},
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"format_version", "kind", "status", "project", "contract_path", "active_services", "issues", "services"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %q", k)
		}
	}
	if int(m["format_version"].(float64)) != ExplainJSONFormatVersion {
		t.Fatalf("format_version got %v", m["format_version"])
	}
}

func TestServiceStatusToJSON_healthHTTP(t *testing.T) {
	j := serviceStatusToJSON(ServiceStatus{
		Name: "api", Container: "podbay_x_api", Running: true, State: "running",
		HTTPURL: "http://127.0.0.1:1/", HTTPStatus: 200,
	})
	if j.Health == nil || j.Health.HTTP == nil || j.Health.HTTP.URL == "" {
		t.Fatalf("expected http health, got %+v", j.Health)
	}
}

func TestBuildLiveOrientation_preservesServicesAndVocabulary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	yaml := `
version: "1"
project: explainorient
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
	statuses := []ServiceStatus{{Name: "web", Missing: true, Container: "podbay_explainorient_web"}}
	doc, err := buildLiveOrientation(c, loaded, nil, nil, false, statuses)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Kind != orientation.Kind || doc.Project != "explainorient" {
		t.Fatalf("orientation identity: %+v", doc)
	}
	if doc.Runtime == nil || !doc.Runtime.Available || len(doc.Runtime.Services) != 1 {
		t.Fatalf("runtime: %+v", doc.Runtime)
	}
	out := explainJSONV1{
		FormatVersion:  ExplainJSONFormatVersion,
		Kind:           "explain",
		Status:         "ok",
		Project:        "explainorient",
		ContractPath:   loaded,
		ActiveServices: []string{"web"},
		Issues:         []explainIssue{},
		Services:       []serviceJSON{{Name: "web", Container: "podbay_explainorient_web", Missing: true}},
		Orientation:    doc,
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"services", "orientation", "active_services", "kind"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %q", k)
		}
	}
	orient, ok := m["orientation"].(map[string]any)
	if !ok || orient["kind"] != orientation.Kind {
		t.Fatalf("orientation block: %v", m["orientation"])
	}
}

func TestWriteOrientationPreamble(t *testing.T) {
	var b strings.Builder
	writeOrientationPreamble(&b, &orientation.Document{
		NextActions: []string{"podbay logs --json", "podbay explain --json"},
	})
	s := b.String()
	if !strings.Contains(s, "Orientation:") || !strings.Contains(s, "next: podbay logs --json") {
		t.Fatalf("preamble: %q", s)
	}
}

func TestBuildFocusDepsJSON_dependentsSorted(t *testing.T) {
	active := map[string]spec.Service{
		"web": {DependsOn: []spec.Dependency{{Service: "api", Condition: spec.ConditionHealthy}}},
		"api": {},
		"z":   {DependsOn: []spec.Dependency{{Service: "web", Condition: spec.ConditionStarted}}},
		"a":   {DependsOn: []spec.Dependency{{Service: "web", Condition: spec.ConditionStarted}}},
	}
	f := buildFocusDepsJSON(active, "web")
	if f == nil {
		t.Fatal("nil focus deps")
	}
	if len(f.Dependents) != 2 || f.Dependents[0].Service != "a" || f.Dependents[1].Service != "z" {
		t.Fatalf("dependents not sorted: %+v", f.Dependents)
	}
}
