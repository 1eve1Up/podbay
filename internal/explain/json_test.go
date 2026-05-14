package explain

import (
	"encoding/json"
	"testing"

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
			t.Errorf("missing top-level key %q", k)
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
