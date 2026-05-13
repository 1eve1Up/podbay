package clijson

import (
	"encoding/json"
	"testing"

	"github.com/1eve1Up/podbay/internal/validate"
)

func TestDiffError_loadError(t *testing.T) {
	d := DiffError("/app/podbay.yaml", "", nil, "diff_load_error", "open podbay.yaml: no such file")
	if d.FormatVersion != FormatVersion {
		t.Errorf("format_version=%d, want %d", d.FormatVersion, FormatVersion)
	}
	if d.Kind != KindDiff {
		t.Errorf("kind=%q, want %q", d.Kind, KindDiff)
	}
	if d.Status != StatusFailed {
		t.Errorf("status=%q, want %q", d.Status, StatusFailed)
	}
	if d.Drift != nil {
		t.Errorf("drift=%v, want nil for load/runtime errors (drift unknown)", d.Drift)
	}
	if len(d.Issues) != 1 {
		t.Fatalf("issues len=%d, want 1", len(d.Issues))
	}
	is := d.Issues[0]
	if is.Code != "diff_load_error" || is.Level != validate.LevelFail || is.Message == "" {
		t.Errorf("issue mismatch: %+v", is)
	}

	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["drift"]; ok {
		t.Errorf("drift key should be omitted on DiffError (unknown), got:\n%s", string(raw))
	}
	if _, ok := m["services_status"]; ok {
		t.Errorf("services_status should be omitted, got:\n%s", string(raw))
	}
	if _, ok := m["extras"]; ok {
		t.Errorf("extras should be omitted, got:\n%s", string(raw))
	}
}

func TestDiffError_runtimeError_carriesProjectAndProfiles(t *testing.T) {
	d := DiffError("/app/podbay.yaml", "demo", []string{"dev"}, "diff_runtime_error", "podman not available")
	if d.Status != StatusFailed {
		t.Errorf("status=%q, want failed", d.Status)
	}
	if d.Project != "demo" {
		t.Errorf("project=%q, want demo", d.Project)
	}
	if len(d.Profiles) != 1 || d.Profiles[0] != "dev" {
		t.Errorf("profiles=%v, want [dev]", d.Profiles)
	}
	if d.Issues[0].Code != "diff_runtime_error" {
		t.Errorf("code=%q, want diff_runtime_error", d.Issues[0].Code)
	}
}

func TestDiffError_pathNormalization(t *testing.T) {
	d := DiffError("/app/./podbay.yaml", "demo", nil, "diff_load_error", "boom")
	if d.ContractPath != "/app/podbay.yaml" {
		t.Errorf("contract_path=%q, want cleaned /app/podbay.yaml", d.ContractPath)
	}
	dEmpty := DiffError("", "", nil, "diff_load_error", "boom")
	if dEmpty.ContractPath != "" {
		t.Errorf("empty contract_path should remain empty, got %q", dEmpty.ContractPath)
	}
}
