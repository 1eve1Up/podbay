package clijson

import (
	"encoding/json"
	"testing"

	"github.com/1eve1Up/podbay/internal/validate"
)

func TestFromValidate_ok(t *testing.T) {
	results := []validate.Result{
		{OK: true, Level: validate.LevelOK, Message: "all good"},
	}
	d := FromValidate("/app/podbay.yaml", "myproj", []string{"dev"}, nil, results, false)
	if d.FormatVersion != FormatVersion {
		t.Fatalf("format_version: got %d", d.FormatVersion)
	}
	if d.Kind != KindValidate || d.Status != StatusOK {
		t.Fatalf("kind/status: got %q %q", d.Kind, d.Status)
	}
	if d.Project != "myproj" || len(d.Profiles) != 1 || d.Profiles[0] != "dev" {
		t.Fatalf("project/profiles: %+v", d)
	}
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["format_version"]; !ok {
		t.Fatalf("missing format_version in %s", string(raw))
	}
}

func TestFromValidate_failed(t *testing.T) {
	results := []validate.Result{
		{OK: true, Level: validate.LevelOK, Message: "graph ok"},
		{OK: false, Level: validate.LevelFail, Message: "port 80 in use"},
	}
	d := FromValidate("podbay.yaml", "p", nil, nil, results, false)
	if d.Status != StatusFailed {
		t.Fatalf("status: %s", d.Status)
	}
	if len(d.Issues) != 2 {
		t.Fatalf("issues len: %d", len(d.Issues))
	}
	found := false
	for _, is := range d.Issues {
		if is.Code == "validation_fail" && is.Message == "port 80 in use" {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues: %+v", d.Issues)
	}
}

func TestDeployFromValidateResults_kind(t *testing.T) {
	d := DeployFromValidateResults("p.yaml", "p", nil, nil, []validate.Result{
		{OK: false, Level: validate.LevelFail, Message: "bad"},
	}, false)
	if d.Kind != KindDeploy || d.Status != StatusFailed {
		t.Fatalf("%+v", d)
	}
}

func TestDeployOutcome_failure(t *testing.T) {
	d := DeployOutcome("/c/podbay.yaml", "x", []string{}, nil, "", errFake("boom"), false)
	if d.Status != StatusFailed || len(d.Issues) != 1 || d.Issues[0].Code != CodeDeployError {
		t.Fatalf("%+v", d)
	}
}

func TestDeployOutcome_success(t *testing.T) {
	d := DeployOutcome("/c/p.yaml", "x", nil, nil, "/tmp/r.json", nil, false)
	if d.Status != StatusOK || d.ReceiptPath != "/tmp/r.json" {
		t.Fatalf("%+v", d)
	}
}

func TestFromValidate_includesDeployServicesWhenSet(t *testing.T) {
	results := []validate.Result{{OK: true, Level: validate.LevelOK, Message: "ok"}}
	d := FromValidate("/app/p.yaml", "p", nil, []string{"web", "api"}, results, false)
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	ds, ok := m["deploy_services"].([]any)
	if !ok || len(ds) != 2 {
		t.Fatalf("deploy_services in JSON: %v", m["deploy_services"])
	}
	if _, ok := m["dependents_expand"]; ok {
		t.Fatalf("unexpected dependents_expand in JSON: %s", string(raw))
	}
}

func TestFromValidate_partialRoots_dependentsExpandInJSON(t *testing.T) {
	results := []validate.Result{{OK: true, Level: validate.LevelOK, Message: "ok"}}
	d := FromValidate("/app/p.yaml", "p", nil, []string{"web"}, results, true)
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if v, ok := m["dependents_expand"].(bool); !ok || !v {
		t.Fatalf("dependents_expand in JSON: %v", m["dependents_expand"])
	}
}

type errFake string

func (e errFake) Error() string { return string(e) }
