package clijson

import (
	"encoding/json"
	"testing"

	"github.com/1eve1Up/podbay/internal/validate"
)

// TestKindDiff_constant locks the public string for the diff document kind.
func TestKindDiff_constant(t *testing.T) {
	if KindDiff != "diff" {
		t.Fatalf("KindDiff = %q, want %q", KindDiff, "diff")
	}
}

// TestDocument_diffFields_omitForOtherKinds asserts the new diff payload
// fields are omitted in validate/deploy/receipt JSON. This is the additive
// existing envelope bytes do not change.
func TestDocument_diffFields_omitForOtherKinds(t *testing.T) {
	cases := []struct {
		name string
		doc  *Document
	}{
		{
			name: "validate ok",
			doc:  FromValidate("/app/podbay.yaml", "p", []string{"dev"}, nil, []validate.Result{{OK: true, Level: validate.LevelOK, Message: "ok"}}, false),
		},
		{
			name: "deploy success",
			doc:  DeployOutcome("/app/podbay.yaml", "p", nil, nil, "/tmp/r.json", nil, false),
		},
		{
			name: "deploy failure",
			doc:  DeployOutcome("/app/podbay.yaml", "p", nil, nil, "", errFake("boom"), false),
		},
		{
			name: "receipt read ok",
			doc:  ReceiptReadSuccess("/tmp/r.json", []byte(`{"contract":{}}`)),
		},
		{
			name: "receipt read failure",
			doc:  ReceiptReadFailure("/tmp/r.json", errFake("nope")),
		},
	}
	for _, c := range cases {
		raw, err := MarshalIndent(c.doc)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		for _, k := range []string{"drift", "services_status", "extras", "receipt_pair", "dependents_expand"} {
			if _, ok := m[k]; ok {
				t.Errorf("%s: unexpected key %q present in non-diff document:\n%s", c.name, k, string(raw))
			}
		}
	}
}

// TestDocument_diffFields_serializeForDiff verifies the new fields land
// under the documented JSON keys when set on a KindDiff document.
func TestDocument_diffFields_serializeForDiff(t *testing.T) {
	driftTrue := true
	d := &Document{
		FormatVersion: FormatVersion,
		Kind:          KindDiff,
		Status:        StatusFailed,
		ContractPath:  "/app/podbay.yaml",
		Project:       "demo",
		Profiles:      []string{"dev"},
		Drift:         &driftTrue,
		ServicesStatus: []DiffServiceStatus{
			{Name: "api", ContainerName: "podbay_demo_api", Status: "missing"},
			{Name: "web", ContainerName: "podbay_demo_web", Status: "wrong_state", State: "exited", ExitCode: 1, Error: "boom"},
			{Name: "metrics", ContainerName: "podbay_demo_metrics", Status: "inspect_error", Error: "podman gone"},
			{Name: "worker", ContainerName: "podbay_demo_worker", Status: "ok", State: "running"},
		},
		Extras: []string{"podbay_demo_debug"},
	}
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if got, _ := m["kind"].(string); got != "diff" {
		t.Errorf("kind=%v, want diff", m["kind"])
	}
	if got, _ := m["drift"].(bool); !got {
		t.Errorf("drift=%v, want true", m["drift"])
	}
	if _, ok := m["services_status"]; !ok {
		t.Errorf("services_status missing in:\n%s", string(raw))
	}
	if _, ok := m["extras"]; !ok {
		t.Errorf("extras missing in:\n%s", string(raw))
	}

	svcs, _ := m["services_status"].([]any)
	if len(svcs) != 4 {
		t.Fatalf("services_status len=%d, want 4", len(svcs))
	}
	first, _ := svcs[0].(map[string]any)
	if first["name"] != "api" || first["container_name"] != "podbay_demo_api" || first["status"] != "missing" {
		t.Errorf("first service entry mismatch: %+v", first)
	}
	if _, has := first["state"]; has {
		t.Errorf("missing service should omit state, got: %+v", first)
	}
	if _, has := first["exit_code"]; has {
		t.Errorf("missing service should omit exit_code, got: %+v", first)
	}
	if _, has := first["error"]; has {
		t.Errorf("missing service should omit error, got: %+v", first)
	}

	wrong, _ := svcs[1].(map[string]any)
	if wrong["state"] != "exited" {
		t.Errorf("wrong_state state=%v, want exited", wrong["state"])
	}
	if wrong["exit_code"].(float64) != 1 {
		t.Errorf("wrong_state exit_code=%v, want 1", wrong["exit_code"])
	}
	if wrong["error"] != "boom" {
		t.Errorf("wrong_state error=%v, want boom", wrong["error"])
	}

	ok, _ := svcs[3].(map[string]any)
	if _, has := ok["error"]; has {
		t.Errorf("ok service should omit error, got: %+v", ok)
	}
	if _, has := ok["exit_code"]; has {
		t.Errorf("ok service should omit exit_code, got: %+v", ok)
	}
}

// TestDocument_diffFalse_serializesExplicitFalse asserts that a Document
// with Drift pointing to false emits the key (not omitted), so consumers
// can distinguish "no drift" from "this is not a diff document".
func TestDocument_diffFalse_serializesExplicitFalse(t *testing.T) {
	driftFalse := false
	d := &Document{
		FormatVersion:  FormatVersion,
		Kind:           KindDiff,
		Status:         StatusOK,
		Project:        "demo",
		Drift:          &driftFalse,
		ServicesStatus: []DiffServiceStatus{{Name: "api", ContainerName: "podbay_demo_api", Status: "ok", State: "running"}},
	}
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	v, ok := m["drift"]
	if !ok {
		t.Fatalf("drift key missing on KindDiff document; got:\n%s", string(raw))
	}
	if b, _ := v.(bool); b {
		t.Fatalf("drift = %v, want false", v)
	}
}

// TestDocument_existingShapeStable_validateOk asserts that an existing
// validate ok envelope still has its documented top-level keys after the
// schema additions (no rename, no removal).
func TestDocument_existingShapeStable_validateOk(t *testing.T) {
	d := FromValidate("/app/podbay.yaml", "demo", []string{"dev"}, nil, []validate.Result{
		{OK: true, Level: validate.LevelOK, Message: "ok"},
	}, false)
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"format_version", "kind", "status", "contract_path", "project", "profiles", "issues"} {
		if _, ok := m[k]; !ok {
			t.Errorf("expected key %q present, got:\n%s", k, string(raw))
		}
	}
	if v, _ := m["format_version"].(float64); v != float64(FormatVersion) {
		t.Errorf("format_version = %v, want %d", v, FormatVersion)
	}
}
