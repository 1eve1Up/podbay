package clijson

import (
	"encoding/json"
	"testing"

	"github.com/1eve1Up/podbay/internal/diff"
)

func TestFromDiff_noDrift_singleOKService(t *testing.T) {
	res := diff.DriftResult{
		Project: "demo",
		Services: []diff.ServiceDrift{
			{Name: "api", ContainerName: "podbay_demo_api", Status: diff.StatusOK, State: "running"},
		},
	}
	d := FromDiff("/app/podbay.yaml", "demo", []string{"dev"}, res)

	if d.FormatVersion != FormatVersion {
		t.Errorf("format_version=%d, want %d", d.FormatVersion, FormatVersion)
	}
	if d.Kind != KindDiff {
		t.Errorf("kind=%q, want %q", d.Kind, KindDiff)
	}
	if d.Status != StatusOK {
		t.Errorf("status=%q, want %q", d.Status, StatusOK)
	}
	if d.Drift == nil || *d.Drift != false {
		t.Errorf("drift=%v, want pointer to false", d.Drift)
	}
	if len(d.ServicesStatus) != 1 || d.ServicesStatus[0].Name != "api" || d.ServicesStatus[0].Status != "ok" || d.ServicesStatus[0].State != "running" {
		t.Errorf("services_status mismatch: %+v", d.ServicesStatus)
	}
	if len(d.Extras) != 0 {
		t.Errorf("extras=%v, want empty", d.Extras)
	}
	if len(d.Issues) != 0 {
		t.Errorf("no inspect_error => no issues, got %+v", d.Issues)
	}
}

func TestFromDiff_missing_singleService(t *testing.T) {
	res := diff.DriftResult{
		Project: "demo",
		Services: []diff.ServiceDrift{
			{Name: "api", ContainerName: "podbay_demo_api", Status: diff.StatusMissing},
		},
		Drift: true,
	}
	d := FromDiff("/app/podbay.yaml", "demo", nil, res)

	if d.Status != StatusFailed {
		t.Errorf("status=%q, want %q", d.Status, StatusFailed)
	}
	if d.Drift == nil || *d.Drift != true {
		t.Errorf("drift=%v, want pointer to true", d.Drift)
	}
	if d.ServicesStatus[0].Status != "missing" {
		t.Errorf("status=%q, want missing", d.ServicesStatus[0].Status)
	}
	if d.ServicesStatus[0].State != "" || d.ServicesStatus[0].ExitCode != 0 || d.ServicesStatus[0].Error != "" {
		t.Errorf("missing service should have empty optional fields, got %+v", d.ServicesStatus[0])
	}
	if len(d.Issues) != 0 {
		t.Errorf("missing != inspect_error; expected no issues, got %+v", d.Issues)
	}
}

func TestFromDiff_wrongState_singleService(t *testing.T) {
	res := diff.DriftResult{
		Project: "demo",
		Services: []diff.ServiceDrift{
			{Name: "web", ContainerName: "podbay_demo_web", Status: diff.StatusWrongState, State: "exited", ExitCode: 137, Error: "OOM"},
		},
		Drift: true,
	}
	d := FromDiff("/app/podbay.yaml", "demo", nil, res)

	if d.Status != StatusFailed {
		t.Errorf("status=%q, want failed", d.Status)
	}
	got := d.ServicesStatus[0]
	if got.Status != "wrong_state" || got.State != "exited" || got.ExitCode != 137 || got.Error != "OOM" {
		t.Errorf("wrong_state mapping mismatch: %+v", got)
	}
	if len(d.Issues) != 0 {
		t.Errorf("wrong_state != inspect_error; expected no issues, got %+v", d.Issues)
	}
}

func TestFromDiff_inspectError_singleServiceCarriesIssue(t *testing.T) {
	res := diff.DriftResult{
		Project: "demo",
		Services: []diff.ServiceDrift{
			{Name: "metrics", ContainerName: "podbay_demo_metrics", Status: diff.StatusInspectError, Error: "podman gone"},
		},
		Drift: true,
	}
	d := FromDiff("/app/podbay.yaml", "demo", nil, res)

	if d.Status != StatusFailed {
		t.Errorf("status=%q, want failed", d.Status)
	}
	if d.ServicesStatus[0].Status != "inspect_error" || d.ServicesStatus[0].Error != "podman gone" {
		t.Errorf("service entry mismatch: %+v", d.ServicesStatus[0])
	}
	if len(d.Issues) != 1 {
		t.Fatalf("expected one inspect_error issue, got %+v", d.Issues)
	}
	issue := d.Issues[0]
	if issue.Code != "diff_inspect_error" {
		t.Errorf("issue.code=%q, want diff_inspect_error", issue.Code)
	}
	if issue.Service != "metrics" {
		t.Errorf("issue.service=%q, want metrics", issue.Service)
	}
	if issue.Message == "" {
		t.Error("issue.message empty")
	}
}

func TestFromDiff_extrasOnly_runningServicePlusExtra(t *testing.T) {
	res := diff.DriftResult{
		Project: "demo",
		Services: []diff.ServiceDrift{
			{Name: "api", ContainerName: "podbay_demo_api", Status: diff.StatusOK, State: "running"},
		},
		Extras: []string{"podbay_demo_debug"},
		Drift:  true,
	}
	d := FromDiff("/app/podbay.yaml", "demo", nil, res)

	if d.Status != StatusFailed {
		t.Errorf("status=%q, want failed (extras present)", d.Status)
	}
	if d.Drift == nil || !*d.Drift {
		t.Errorf("drift=%v, want pointer to true", d.Drift)
	}
	if len(d.Extras) != 1 || d.Extras[0] != "podbay_demo_debug" {
		t.Errorf("extras=%v, want [podbay_demo_debug]", d.Extras)
	}
	if d.ServicesStatus[0].Status != "ok" {
		t.Errorf("running service should remain ok, got %+v", d.ServicesStatus[0])
	}
	if len(d.Issues) != 0 {
		t.Errorf("extras alone should not add issues, got %+v", d.Issues)
	}
}

func TestFromDiff_mixedCase_inspectErrorIssuesEnumerated(t *testing.T) {
	res := diff.DriftResult{
		Project: "demo",
		Services: []diff.ServiceDrift{
			{Name: "api", ContainerName: "podbay_demo_api", Status: diff.StatusOK, State: "running"},
			{Name: "worker", ContainerName: "podbay_demo_worker", Status: diff.StatusWrongState, State: "exited", ExitCode: 1, Error: "boom"},
			{Name: "db", ContainerName: "podbay_demo_db", Status: diff.StatusMissing},
			{Name: "metrics", ContainerName: "podbay_demo_metrics", Status: diff.StatusInspectError, Error: "podman gone"},
			{Name: "logger", ContainerName: "podbay_demo_logger", Status: diff.StatusInspectError, Error: "timeout"},
		},
		Extras: []string{"podbay_demo_debug"},
		Drift:  true,
	}
	d := FromDiff("/app/podbay.yaml", "demo", []string{"dev"}, res)

	if d.Status != StatusFailed {
		t.Errorf("status=%q, want failed", d.Status)
	}
	if d.Drift == nil || !*d.Drift {
		t.Errorf("drift=%v, want pointer to true", d.Drift)
	}
	if len(d.ServicesStatus) != 5 {
		t.Fatalf("services_status len=%d, want 5", len(d.ServicesStatus))
	}
	wantOrder := []string{"api", "worker", "db", "metrics", "logger"}
	for i, want := range wantOrder {
		if d.ServicesStatus[i].Name != want {
			t.Errorf("services_status[%d]=%q, want %q (order must match input)", i, d.ServicesStatus[i].Name, want)
		}
	}
	if len(d.Issues) != 2 {
		t.Fatalf("expected 2 inspect_error issues, got %+v", d.Issues)
	}
	if d.Issues[0].Service != "metrics" || d.Issues[1].Service != "logger" {
		t.Errorf("issue order = %q,%q ; want metrics,logger", d.Issues[0].Service, d.Issues[1].Service)
	}
	for _, is := range d.Issues {
		if is.Code != "diff_inspect_error" {
			t.Errorf("issue.code=%q, want diff_inspect_error", is.Code)
		}
	}
}

func TestFromDiff_doesNotMutateInputSlices(t *testing.T) {
	services := []diff.ServiceDrift{
		{Name: "api", ContainerName: "podbay_demo_api", Status: diff.StatusOK, State: "running"},
	}
	extras := []string{"podbay_demo_debug"}
	res := diff.DriftResult{Project: "demo", Services: services, Extras: extras, Drift: true}

	d := FromDiff("/app/podbay.yaml", "demo", nil, res)

	if len(d.ServicesStatus) > 0 && len(services) > 0 {
		if &d.ServicesStatus[0] == (*DiffServiceStatus)(nil) {
			t.Fatal("nil services_status entry")
		}
	}
	if len(d.Extras) > 0 && &d.Extras[0] == &extras[0] {
		t.Fatal("FromDiff aliased input Extras slice; want defensive copy")
	}
	if d.Drift == &res.Drift {
		t.Fatal("FromDiff aliased &res.Drift; want defensive copy of bool")
	}
}

func TestFromDiff_pathNormalization(t *testing.T) {
	res := diff.DriftResult{Project: "demo"}
	d := FromDiff("/app/./podbay.yaml", "demo", nil, res)
	if d.ContractPath != "/app/podbay.yaml" {
		t.Errorf("contract_path=%q, want cleaned /app/podbay.yaml", d.ContractPath)
	}
	dEmpty := FromDiff("", "demo", nil, res)
	if dEmpty.ContractPath != "" {
		t.Errorf("empty contract_path should remain empty, got %q", dEmpty.ContractPath)
	}
}

func TestFromDiff_goldenJSON_mixedCase(t *testing.T) {
	res := diff.DriftResult{
		Project: "demo",
		Services: []diff.ServiceDrift{
			{Name: "api", ContainerName: "podbay_demo_api", Status: diff.StatusOK, State: "running"},
			{Name: "worker", ContainerName: "podbay_demo_worker", Status: diff.StatusWrongState, State: "exited", ExitCode: 1, Error: "boom"},
			{Name: "db", ContainerName: "podbay_demo_db", Status: diff.StatusMissing},
			{Name: "metrics", ContainerName: "podbay_demo_metrics", Status: diff.StatusInspectError, Error: "podman gone"},
		},
		Extras: []string{"podbay_demo_debug"},
		Drift:  true,
	}
	d := FromDiff("/app/podbay.yaml", "demo", []string{"dev"}, res)
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "format_version": 1,
  "kind": "diff",
  "status": "failed",
  "contract_path": "/app/podbay.yaml",
  "project": "demo",
  "profiles": [
    "dev"
  ],
  "issues": [
    {
      "level": "fail",
      "code": "diff_inspect_error",
      "message": "podman gone",
      "service": "metrics"
    }
  ],
  "drift": true,
  "services_status": [
    {
      "name": "api",
      "container_name": "podbay_demo_api",
      "status": "ok",
      "state": "running"
    },
    {
      "name": "worker",
      "container_name": "podbay_demo_worker",
      "status": "wrong_state",
      "state": "exited",
      "exit_code": 1,
      "error": "boom"
    },
    {
      "name": "db",
      "container_name": "podbay_demo_db",
      "status": "missing"
    },
    {
      "name": "metrics",
      "container_name": "podbay_demo_metrics",
      "status": "inspect_error",
      "error": "podman gone"
    }
  ],
  "extras": [
    "podbay_demo_debug"
  ]
}`
	if string(raw) != want {
		t.Fatalf("golden JSON mismatch.\n got:\n%s\nwant:\n%s", string(raw), want)
	}
}

func TestFromDiff_goldenJSON_noDriftKeysPresent(t *testing.T) {
	res := diff.DriftResult{
		Project: "demo",
		Services: []diff.ServiceDrift{
			{Name: "api", ContainerName: "podbay_demo_api", Status: diff.StatusOK, State: "running"},
		},
	}
	d := FromDiff("/app/podbay.yaml", "demo", nil, res)
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if v, ok := m["drift"]; !ok || v != false {
		t.Fatalf("drift = %v (present=%v); want explicit false on KindDiff document\n%s", v, ok, string(raw))
	}
	if _, ok := m["issues"]; ok {
		t.Errorf("no inspect_error => issues should be omitted, got:\n%s", string(raw))
	}
	if _, ok := m["extras"]; ok {
		t.Errorf("no extras => key should be omitted, got:\n%s", string(raw))
	}
}

func TestFromDiffWithPartial_setsDeployFields(t *testing.T) {
	res := diff.DriftResult{Project: "demo", Drift: false}
	d := FromDiffWithPartial("/app/p.yaml", "demo", nil, []string{"web"}, true, res)
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	ds, _ := m["deploy_services"].([]any)
	if len(ds) != 1 || ds[0] != "web" {
		t.Fatalf("deploy_services=%v", m["deploy_services"])
	}
	if m["dependents_expand"] != true {
		t.Fatalf("dependents_expand=%v", m["dependents_expand"])
	}
}
