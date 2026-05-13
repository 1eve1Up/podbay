package main

// Integration coverage for two-arg receipt diff --json: the same emitReceiptPairErrorJSON,
// emitReceiptPairResultJSON, and receiptPairJSONExitCode helpers that runReceiptPairDiff uses
// (avoids os.Exit in tests while locking envelope shape and exit-code parity).

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/receipt"
)

func captureReceiptPairErrorJSON(t *testing.T, code, msg string) string {
	t.Helper()
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	emitReceiptPairErrorJSON(cmd, code, msg)
	return buf.String()
}

func captureReceiptPairResultJSON(t *testing.T, res receipt.ReceiptDiffResult) (payload string, exitCode int) {
	t.Helper()
	return captureReceiptPairResultJSONRaw(t, res, false)
}

func captureReceiptPairResultJSONRaw(t *testing.T, res receipt.ReceiptDiffResult, showRawEnv bool) (payload string, exitCode int) {
	t.Helper()
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := emitReceiptPairResultJSON(cmd, res, showRawEnv); err != nil {
		t.Fatalf("emitReceiptPairResultJSON: %v", err)
	}
	return buf.String(), receiptPairJSONExitCode(res)
}

func assertReceiptPairDiffEnvelopeBase(t *testing.T, m map[string]any, wantStatus string) {
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
	if _, ok := m["contract_path"]; ok {
		t.Errorf("receipt pair success doc should omit top-level contract_path")
	}
	if _, ok := m["project"]; ok {
		t.Errorf("receipt pair success doc should omit top-level project")
	}
}

func receiptFixture(project, contract string, profiles []string, svcs []receipt.ServiceRecord) *receipt.Receipt {
	return &receipt.Receipt{
		FormatVersion: receipt.CurrentFormatVersion,
		GeneratedAt:   time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		ContractPath:  contract,
		Project:       project,
		Profiles:      profiles,
		Services:      svcs,
	}
}

func TestReceiptPairJSONIntegration_loadErrorEnvelope(t *testing.T) {
	out := captureReceiptPairErrorJSON(t, clijson.CodeReceiptDiffLoadError, "read /no/such: no such file")

	m := parseEnvelope(t, out)
	if v, _ := m["format_version"].(float64); int(v) != clijson.FormatVersion {
		t.Errorf("format_version = %v", v)
	}
	if v, _ := m["kind"].(string); v != clijson.KindDiff {
		t.Errorf("kind = %v", v)
	}
	if v, _ := m["status"].(string); v != clijson.StatusFailed {
		t.Errorf("status = %v", v)
	}
	if _, ok := m["drift"]; ok {
		t.Errorf("drift should be omitted, got %v", m["drift"])
	}
	if _, ok := m["receipt_pair"]; ok {
		t.Errorf("receipt_pair should be omitted on load error")
	}
	issues, _ := m["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("issues = %v", m["issues"])
	}
	is, _ := issues[0].(map[string]any)
	if is["code"] != clijson.CodeReceiptDiffLoadError {
		t.Errorf("issue.code = %v", is["code"])
	}
}

func TestReceiptPairJSONIntegration_decodeErrorEnvelope(t *testing.T) {
	out := captureReceiptPairErrorJSON(t, clijson.CodeReceiptDiffDecodeError, "/tmp/r.json: receipt: decode: invalid character")

	m := parseEnvelope(t, out)
	issues, _ := m["issues"].([]any)
	is, _ := issues[0].(map[string]any)
	if is["code"] != clijson.CodeReceiptDiffDecodeError {
		t.Errorf("issue.code = %v", is["code"])
	}
}

func TestReceiptPairJSONIntegration_usageErrorEnvelope(t *testing.T) {
	out := captureReceiptPairErrorJSON(t, clijson.CodeReceiptDiffUsageError, "diff: --profile ...")

	m := parseEnvelope(t, out)
	issues, _ := m["issues"].([]any)
	is, _ := issues[0].(map[string]any)
	if is["code"] != clijson.CodeReceiptDiffUsageError {
		t.Errorf("issue.code = %v", is["code"])
	}
}

func TestReceiptPairJSONIntegration_envRedactedVsRawPolicy(t *testing.T) {
	e1 := []receipt.EnvVar{{Name: "FOO", Value: "secret-a"}}
	e2 := []receipt.EnvVar{{Name: "FOO", Value: "secret-b"}}
	a := receiptFixture("demo", "/app/p.yaml", nil, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1", Env: &e1},
	})
	b := receiptFixture("demo", "/app/p.yaml", nil, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1", Env: &e2},
	})
	res := receipt.CompareReceipts(a, b)
	outRedact, _ := captureReceiptPairResultJSONRaw(t, res, false)
	m := parseEnvelope(t, outRedact)
	if m["env_value_display_policy"] != "redacted" {
		t.Fatalf("policy = %v", m["env_value_display_policy"])
	}
	pair, _ := m["receipt_pair"].(map[string]any)
	svcs, _ := pair["services"].([]any)
	row, _ := svcs[0].(map[string]any)
	first, _ := row["first"].(map[string]any)
	envs, _ := first["env"].([]any)
	ev, _ := envs[0].(map[string]any)
	if ev["value"] != "(redacted)" {
		t.Fatalf("expected redacted env, got %v", ev["value"])
	}

	outRaw, _ := captureReceiptPairResultJSONRaw(t, res, true)
	m2 := parseEnvelope(t, outRaw)
	if m2["env_value_display_policy"] != "raw" {
		t.Fatalf("policy = %v", m2["env_value_display_policy"])
	}
	pair2, _ := m2["receipt_pair"].(map[string]any)
	svcs2, _ := pair2["services"].([]any)
	row2, _ := svcs2[0].(map[string]any)
	first2, _ := row2["first"].(map[string]any)
	envs2, _ := first2["env"].([]any)
	ev2, _ := envs2[0].(map[string]any)
	if ev2["value"] != "secret-a" {
		t.Fatalf("raw first env = %v", ev2["value"])
	}
}

func TestReceiptPairJSONIntegration_envIncomparable_warnExit0(t *testing.T) {
	env := []receipt.EnvVar{{Name: "K", Value: "secret"}}
	a := receiptFixture("demo", "/app/p.yaml", nil, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1", Env: &env},
	})
	b := receiptFixture("demo", "/app/p.yaml", nil, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1"},
	})
	res := receipt.CompareReceipts(a, b)
	out, code := captureReceiptPairResultJSON(t, res)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	m := parseEnvelope(t, out)
	if m["env_value_display_policy"] != "redacted" {
		t.Fatalf("policy = %v", m["env_value_display_policy"])
	}
	issues, _ := m["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("issues = %v", issues)
	}
	is, _ := issues[0].(map[string]any)
	if is["level"] != "warn" || is["code"] != receipt.CodeEnvIncomparable {
		t.Fatalf("issue = %v", is)
	}
}

func TestReceiptPairJSONIntegration_noDriftEnvelope(t *testing.T) {
	a := receiptFixture("demo", "/app/p.yaml", []string{"a"}, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1", Image: "i1"},
	})
	b := receiptFixture("demo", "/app/p.yaml", []string{"a"}, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1", Image: "i1"},
	})
	res := receipt.CompareReceipts(a, b)
	out, code := captureReceiptPairResultJSON(t, res)
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	m := parseEnvelope(t, out)
	assertReceiptPairDiffEnvelopeBase(t, m, clijson.StatusOK)
	if drift, ok := m["drift"].(bool); !ok || drift {
		t.Fatalf("drift = %v", m["drift"])
	}
	pair, ok := m["receipt_pair"].(map[string]any)
	if !ok {
		t.Fatalf("missing receipt_pair: %s", out)
	}
	if pair["project_match"] != true {
		t.Fatalf("project_match = %v", pair["project_match"])
	}
	if _, has := m["services_status"]; has {
		t.Errorf("contract field services_status must be absent")
	}
}

func TestReceiptPairJSONIntegration_driftEnvelope(t *testing.T) {
	a := receiptFixture("p1", "/a.yaml", nil, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1", Image: "i1"},
	})
	b := receiptFixture("p2", "/b.yaml", nil, []receipt.ServiceRecord{
		{Service: "web", ContainerName: "n2", Image: "i2"},
	})
	res := receipt.CompareReceipts(a, b)
	out, code := captureReceiptPairResultJSON(t, res)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	m := parseEnvelope(t, out)
	assertReceiptPairDiffEnvelopeBase(t, m, clijson.StatusFailed)
	if drift, ok := m["drift"].(bool); !ok || !drift {
		t.Fatalf("drift = %v", m["drift"])
	}
	issues, _ := m["issues"].([]any)
	if len(issues) < 3 {
		t.Fatalf("expected global + service issues, got %d: %v", len(issues), issues)
	}
	pair, ok := m["receipt_pair"].(map[string]any)
	if !ok {
		t.Fatal("missing receipt_pair")
	}
	if pair["project_match"] != false {
		t.Fatalf("project_match = %v", pair["project_match"])
	}
	svcs, _ := pair["services"].([]any)
	if len(svcs) != 2 {
		t.Fatalf("receipt_pair.services len = %d", len(svcs))
	}
}

func TestReceiptPairJSONIntegration_emitTrailingNewline(t *testing.T) {
	a := receiptFixture("p", "/x.yaml", nil, nil)
	b := receiptFixture("p", "/x.yaml", nil, nil)
	res := receipt.CompareReceipts(a, b)

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := emitReceiptPairResultJSON(cmd, res, false); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.HasSuffix(got, "}\n") {
		t.Errorf("output should end with '}\\n', got tail: %q", got[max(0, len(got)-8):])
	}
	if strings.HasSuffix(got, "}\n\n") {
		t.Errorf("no blank trailing line, got tail: %q", got[max(0, len(got)-12):])
	}
}

func TestReceiptPairJSONExitCode_table(t *testing.T) {
	cases := []struct {
		name string
		res  receipt.ReceiptDiffResult
		want int
	}{
		{"no drift", receipt.ReceiptDiffResult{}, 0},
		{"drift", receipt.ReceiptDiffResult{Drift: true}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := receiptPairJSONExitCode(c.res); got != c.want {
				t.Fatalf("receiptPairJSONExitCode = %d, want %d", got, c.want)
			}
		})
	}
}
