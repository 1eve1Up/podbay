package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/deploy"
)

func captureDeployJSON(t *testing.T, deployErr error, deployServices []string, dependents bool) string {
	t.Helper()
	return captureDeployJSONWithReceipt(t, deployErr, deployServices, dependents, "")
}

func captureDeployJSONWithReceipt(t *testing.T, deployErr error, deployServices []string, dependents bool, receiptPath string) string {
	t.Helper()
	cmd := &cobra.Command{}
	var buf strings.Builder
	cmd.SetOut(&buf)
	doc := clijson.DeployOutcome("/app/podbay.yaml", "demo", nil, deployServices, receiptPath, deployErr, dependents)
	raw, err := clijson.MarshalIndent(doc)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = buf.Write(raw)
	return buf.String()
}

func parseDeployEnvelope(t *testing.T, payload string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, payload)
	}
	return m
}

func TestDeployJSONIntegration_healthTimeoutIssue(t *testing.T) {
	err := &deploy.HealthGateFailure{
		Service:      "web",
		ProbeKind:    deploy.ProbeHTTP,
		FailureClass: deploy.HealthFailureTimeout,
		Message:      "health check failed: timeout",
	}
	out := captureDeployJSON(t, err, nil, false)
	m := parseDeployEnvelope(t, out)
	if m["kind"] != clijson.KindDeploy || m["status"] != clijson.StatusFailed {
		t.Fatalf("envelope: %+v", m)
	}
	issues, _ := m["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("issues = %v", issues)
	}
	first, _ := issues[0].(map[string]any)
	if first["code"] != clijson.CodeDeployHealthTimeout {
		t.Fatalf("code = %v", first["code"])
	}
	if first["service"] != "web" {
		t.Fatalf("service = %v", first["service"])
	}
}

func TestDeployJSONIntegration_genericDeployError(t *testing.T) {
	out := captureDeployJSON(t, errFakeDeploy("build failed"), nil, false)
	m := parseDeployEnvelope(t, out)
	issues, _ := m["issues"].([]any)
	first, _ := issues[0].(map[string]any)
	if first["code"] != clijson.CodeDeployError {
		t.Fatalf("code = %v", first["code"])
	}
}

type errFakeDeploy string

func (e errFakeDeploy) Error() string { return string(e) }

func TestDeployJSONIntegration_successUnchanged(t *testing.T) {
	out := captureDeployJSON(t, nil, nil, false)
	m := parseDeployEnvelope(t, out)
	if m["status"] != clijson.StatusOK {
		t.Fatalf("status = %v", m["status"])
	}
	if _, ok := m["issues"]; ok {
		t.Fatalf("unexpected issues on success: %v", m["issues"])
	}
}

func TestDeployJSONIntegration_externalDepUnhealthy(t *testing.T) {
	err := &deploy.HealthGateFailure{
		Service:      "db",
		ProbeKind:    deploy.ProbeHTTP,
		FailureClass: deploy.HealthFailureProbeError,
		Message:      "health check failed: HTTP 500",
		ExternalDep:  true,
		RequestedBy:  "api",
	}
	out := captureDeployJSON(t, err, []string{"api"}, true)
	m := parseDeployEnvelope(t, out)
	if m["dependents_expand"] != true {
		t.Fatalf("dependents_expand = %v", m["dependents_expand"])
	}
	ds, _ := m["deploy_services"].([]any)
	if len(ds) != 1 || ds[0] != "api" {
		t.Fatalf("deploy_services = %v", m["deploy_services"])
	}
	issues, _ := m["issues"].([]any)
	first, _ := issues[0].(map[string]any)
	if first["code"] != clijson.CodeDeployExternalDepUnhealthy {
		t.Fatalf("code = %v", first["code"])
	}
}

func TestDeployJSONIntegration_healthProbeFailed(t *testing.T) {
	err := &deploy.HealthGateFailure{
		Service:      "api",
		ProbeKind:    deploy.ProbeExec,
		FailureClass: deploy.HealthFailureProbeError,
		Message:      "exec health failed: exit 1",
	}
	out := captureDeployJSON(t, err, []string{"api"}, false)
	m := parseDeployEnvelope(t, out)
	issues, _ := m["issues"].([]any)
	first, _ := issues[0].(map[string]any)
	if first["code"] != clijson.CodeDeployHealthProbeFailed {
		t.Fatalf("code = %v", first["code"])
	}
}

func TestDeployJSONIntegration_failedWithAttemptReceiptPath(t *testing.T) {
	err := &deploy.HealthGateFailure{
		Service:      "web",
		ProbeKind:    deploy.ProbeHTTP,
		FailureClass: deploy.HealthFailureTimeout,
		Message:      "health check failed: timeout",
	}
	out := captureDeployJSONWithReceipt(t, err, nil, false, "/tmp/attempt.json")
	m := parseDeployEnvelope(t, out)
	if m["status"] != clijson.StatusFailed {
		t.Fatalf("status = %v", m["status"])
	}
	if m["receipt_path"] != "/tmp/attempt.json" {
		t.Fatalf("receipt_path = %v", m["receipt_path"])
	}
	issues, _ := m["issues"].([]any)
	first, _ := issues[0].(map[string]any)
	if first["code"] != clijson.CodeDeployHealthTimeout {
		t.Fatalf("code = %v", first["code"])
	}
}
