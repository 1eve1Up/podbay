package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/diff"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
)

// These tests integration-cover the podbay diff --json pipeline that the
// CLI handler runs in --json mode (success path):
//
//	diff.Compute(stub InspectFunc) -> clijson.FromDiff -> clijson.MarshalIndent -> stdout
//
// They use the same internal helpers (emitDiffJSON / emitDiffErrorJSON /
// diffJSONExitCode) that diffCmd uses, with a bytes.Buffer in place of
// stdout, so they do not require a live Podman binary while still locking
// the JSON envelope contract and exit-code semantics for each case.

const project = "demo"

// captureDiffJSON runs the same composition diffCmd uses for the --json
// success path and returns the marshalled stdout payload plus the implied
// process exit code.
func captureDiffJSON(t *testing.T, services []string, inspect diff.InspectFunc, extras []string, extrasErr error) (string, int) {
	t.Helper()

	r := runner.New(project)
	res, err := diff.Compute(r, services, inspect, extras, extrasErr)
	if err != nil {
		t.Fatalf("diff.Compute returned error: %v", err)
	}

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	emitDiffJSON(cmd, "/app/podbay.yaml", project, []string{"dev"}, nil, false, res)
	return buf.String(), diffJSONExitCode(res)
}

// captureDiffErrorJSON exercises the failure-path envelope helpers for
// load/runtime failures. Exit code 1 is implicit in diffCmd via os.Exit.
func captureDiffErrorJSON(t *testing.T, contractPath, code, msg string) string {
	t.Helper()
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	emitDiffErrorJSON(cmd, contractPath, project, []string{"dev"}, code, msg)
	return buf.String()
}

func parseEnvelope(t *testing.T, payload string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &m); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, payload)
	}
	return m
}

func assertEnvelopeBase(t *testing.T, m map[string]any, wantStatus string) {
	t.Helper()
	if v, _ := m["format_version"].(float64); int(v) != clijson.FormatVersion {
		t.Errorf("format_version = %v, want %d", v, clijson.FormatVersion)
	}
	if v, _ := m["kind"].(string); v != clijson.KindDiff {
		t.Errorf("kind = %v, want %s", v, clijson.KindDiff)
	}
	if v, _ := m["status"].(string); v != wantStatus {
		t.Errorf("status = %v, want %s", v, wantStatus)
	}
	if v, _ := m["contract_path"].(string); v != "/app/podbay.yaml" {
		t.Errorf("contract_path = %v, want /app/podbay.yaml", v)
	}
	if v, _ := m["project"].(string); v != project {
		t.Errorf("project = %v, want %s", v, project)
	}
}

func TestDiffJSONIntegration_noDrift_singleRunningService(t *testing.T) {
	running := &runtimestate.ContainerState{State: "running"}
	inspect := func(name string) (*runtimestate.ContainerState, error) {
		if name == "podbay_demo_api" {
			return running, nil
		}
		return nil, nil
	}

	out, exit := captureDiffJSON(t, []string{"api"}, inspect, nil, nil)

	if exit != 0 {
		t.Errorf("exit = %d, want 0", exit)
	}
	m := parseEnvelope(t, out)
	assertEnvelopeBase(t, m, clijson.StatusOK)
	if v, ok := m["drift"].(bool); !ok || v {
		t.Errorf("drift = %v, want explicit false", m["drift"])
	}
	svcs, _ := m["services_status"].([]any)
	if len(svcs) != 1 {
		t.Fatalf("services_status len = %d, want 1", len(svcs))
	}
	first, _ := svcs[0].(map[string]any)
	if first["name"] != "api" || first["container_name"] != "podbay_demo_api" || first["status"] != "ok" || first["state"] != "running" {
		t.Errorf("first service mismatch: %+v", first)
	}
	if _, ok := m["extras"]; ok {
		t.Errorf("no extras expected, got: %v", m["extras"])
	}
	if _, ok := m["issues"]; ok {
		t.Errorf("no issues expected for ok path, got: %v", m["issues"])
	}
}

func TestDiffJSONIntegration_missing_drives_failed_status_and_exit1(t *testing.T) {
	inspect := func(string) (*runtimestate.ContainerState, error) { return nil, nil }
	out, exit := captureDiffJSON(t, []string{"api"}, inspect, nil, nil)

	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}
	m := parseEnvelope(t, out)
	assertEnvelopeBase(t, m, clijson.StatusFailed)
	if v, ok := m["drift"].(bool); !ok || !v {
		t.Errorf("drift = %v, want true", m["drift"])
	}
	svcs, _ := m["services_status"].([]any)
	first, _ := svcs[0].(map[string]any)
	if first["status"] != "missing" {
		t.Errorf("status = %v, want missing", first["status"])
	}
	if _, has := first["state"]; has {
		t.Errorf("missing service should omit state, got: %+v", first)
	}
}

func TestDiffJSONIntegration_wrongState_carriesRuntimeDetail(t *testing.T) {
	exited := &runtimestate.ContainerState{State: "exited", ExitCode: 137, Error: "OOM"}
	inspect := func(string) (*runtimestate.ContainerState, error) { return exited, nil }

	out, exit := captureDiffJSON(t, []string{"web"}, inspect, nil, nil)

	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}
	m := parseEnvelope(t, out)
	assertEnvelopeBase(t, m, clijson.StatusFailed)
	svcs, _ := m["services_status"].([]any)
	first, _ := svcs[0].(map[string]any)
	if first["status"] != "wrong_state" {
		t.Errorf("status = %v, want wrong_state", first["status"])
	}
	if first["state"] != "exited" || first["exit_code"].(float64) != 137 || first["error"] != "OOM" {
		t.Errorf("runtime detail mismatch: %+v", first)
	}
}

func TestDiffJSONIntegration_inspectError_alsoSurfacesAsIssue(t *testing.T) {
	inspect := func(string) (*runtimestate.ContainerState, error) {
		return nil, errors.New("podman gone")
	}

	out, exit := captureDiffJSON(t, []string{"metrics"}, inspect, nil, nil)

	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}
	m := parseEnvelope(t, out)
	assertEnvelopeBase(t, m, clijson.StatusFailed)

	svcs, _ := m["services_status"].([]any)
	first, _ := svcs[0].(map[string]any)
	if first["status"] != "inspect_error" || first["error"] != "podman gone" {
		t.Errorf("service entry mismatch: %+v", first)
	}

	issues, _ := m["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("expected one inspect_error issue, got: %v", m["issues"])
	}
	is, _ := issues[0].(map[string]any)
	if is["code"] != "diff_inspect_error" {
		t.Errorf("issue.code = %v, want diff_inspect_error", is["code"])
	}
	if is["service"] != "metrics" {
		t.Errorf("issue.service = %v, want metrics", is["service"])
	}
}

func TestDiffJSONIntegration_extras_runningService_plusExtra(t *testing.T) {
	running := &runtimestate.ContainerState{State: "running"}
	inspect := func(string) (*runtimestate.ContainerState, error) { return running, nil }

	out, exit := captureDiffJSON(t, []string{"api"}, inspect, []string{"podbay_demo_debug"}, nil)

	if exit != 1 {
		t.Errorf("exit = %d, want 1 (extras present)", exit)
	}
	m := parseEnvelope(t, out)
	assertEnvelopeBase(t, m, clijson.StatusFailed)
	if v, ok := m["drift"].(bool); !ok || !v {
		t.Errorf("drift = %v, want true", m["drift"])
	}

	extras, _ := m["extras"].([]any)
	if len(extras) != 1 || extras[0].(string) != "podbay_demo_debug" {
		t.Errorf("extras = %v, want [podbay_demo_debug]", extras)
	}
	if _, ok := m["issues"]; ok {
		t.Errorf("extras alone should not surface as issues, got: %v", m["issues"])
	}
}

func TestDiffJSONIntegration_mixedCase_endToEnd(t *testing.T) {
	running := &runtimestate.ContainerState{State: "running"}
	exited := &runtimestate.ContainerState{State: "exited", ExitCode: 1, Error: "boom"}
	inspect := func(name string) (*runtimestate.ContainerState, error) {
		switch name {
		case "podbay_demo_api":
			return running, nil
		case "podbay_demo_worker":
			return exited, nil
		case "podbay_demo_db":
			return nil, nil
		case "podbay_demo_metrics":
			return nil, errors.New("podman gone")
		}
		return nil, nil
	}

	out, exit := captureDiffJSON(t, []string{"api", "worker", "db", "metrics"}, inspect, []string{"podbay_demo_debug"}, nil)

	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}
	m := parseEnvelope(t, out)
	assertEnvelopeBase(t, m, clijson.StatusFailed)
	if v, ok := m["drift"].(bool); !ok || !v {
		t.Errorf("drift = %v, want true", m["drift"])
	}

	wantStatuses := []string{"ok", "wrong_state", "missing", "inspect_error"}
	svcs, _ := m["services_status"].([]any)
	if len(svcs) != 4 {
		t.Fatalf("services_status len = %d, want 4", len(svcs))
	}
	for i, want := range wantStatuses {
		got, _ := svcs[i].(map[string]any)
		if got["status"] != want {
			t.Errorf("services_status[%d].status = %v, want %s", i, got["status"], want)
		}
	}

	extras, _ := m["extras"].([]any)
	if len(extras) != 1 || extras[0].(string) != "podbay_demo_debug" {
		t.Errorf("extras = %v, want [podbay_demo_debug]", extras)
	}
	issues, _ := m["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("expected exactly one inspect_error issue (metrics), got: %v", m["issues"])
	}
	is, _ := issues[0].(map[string]any)
	if is["code"] != "diff_inspect_error" || is["service"] != "metrics" {
		t.Errorf("issue mismatch: %+v", is)
	}
}

func TestDiffJSONIntegration_loadErrorEnvelope(t *testing.T) {
	out := captureDiffErrorJSON(t, "/app/missing.yaml", "diff_load_error", "open /app/missing.yaml: no such file")

	m := parseEnvelope(t, out)
	if v, _ := m["format_version"].(float64); int(v) != clijson.FormatVersion {
		t.Errorf("format_version = %v, want %d", v, clijson.FormatVersion)
	}
	if v, _ := m["kind"].(string); v != clijson.KindDiff {
		t.Errorf("kind = %v, want %s", v, clijson.KindDiff)
	}
	if v, _ := m["status"].(string); v != clijson.StatusFailed {
		t.Errorf("status = %v, want %s", v, clijson.StatusFailed)
	}
	if _, ok := m["drift"]; ok {
		t.Errorf("drift key should be omitted on load_error (unknown), got: %v", m["drift"])
	}
	if _, ok := m["services_status"]; ok {
		t.Errorf("services_status should be omitted on load_error, got: %v", m["services_status"])
	}
	issues, _ := m["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("expected one issue, got: %v", m["issues"])
	}
	is, _ := issues[0].(map[string]any)
	if is["code"] != "diff_load_error" {
		t.Errorf("issue.code = %v, want diff_load_error", is["code"])
	}
}

func TestDiffJSONIntegration_runtimeErrorEnvelope(t *testing.T) {
	out := captureDiffErrorJSON(t, "/app/podbay.yaml", "diff_runtime_error", "podman not available")

	m := parseEnvelope(t, out)
	if v, _ := m["status"].(string); v != clijson.StatusFailed {
		t.Errorf("status = %v, want %s", v, clijson.StatusFailed)
	}
	issues, _ := m["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("expected one issue, got: %v", m["issues"])
	}
	is, _ := issues[0].(map[string]any)
	if is["code"] != "diff_runtime_error" {
		t.Errorf("issue.code = %v, want diff_runtime_error", is["code"])
	}
	if is["message"] != "podman not available" {
		t.Errorf("issue.message = %v, want %q", is["message"], "podman not available")
	}
}

// TestDiffJSONIntegration_emitOmitsTrailingNewline asserts the helper writes
// exactly one trailing newline (not the marshal-default trailing newline +
// the Fprintln newline) — important for downstream tools that pipe stdout
// into jq -c or similar.
func TestDiffJSONIntegration_emitOmitsTrailingNewline(t *testing.T) {
	r := runner.New(project)
	res, err := diff.Compute(r, []string{"api"}, func(string) (*runtimestate.ContainerState, error) {
		return &runtimestate.ContainerState{State: "running"}, nil
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	emitDiffJSON(cmd, "/app/podbay.yaml", project, nil, nil, false, res)

	got := buf.String()
	if !strings.HasSuffix(got, "}\n") {
		t.Errorf("output should end with '}\\n', got tail: %q", got[max(0, len(got)-8):])
	}
	if strings.HasSuffix(got, "}\n\n") {
		t.Errorf("output should not have a blank trailing line, got tail: %q", got[max(0, len(got)-12):])
	}
}

// TestDiffJSONExitCode_table locks the success-path exit code contract that
// diffCmd applies after emitDiffJSON.
func TestDiffJSONExitCode_table(t *testing.T) {
	cases := []struct {
		name string
		res  diff.DriftResult
		want int
	}{
		{"no drift", diff.DriftResult{Project: project}, 0},
		{"drift true (services)", diff.DriftResult{Project: project, Drift: true}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := diffJSONExitCode(c.res); got != c.want {
				t.Fatalf("diffJSONExitCode = %d, want %d", got, c.want)
			}
		})
	}
}
