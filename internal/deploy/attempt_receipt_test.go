package deploy

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/receipt"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestFailureSummaryFromHealthGate(t *testing.T) {
	h := &HealthGateFailure{
		Service:      "api",
		ProbeKind:    ProbeHTTP,
		FailureClass: HealthFailureTimeout,
		Message:      "health check failed: timeout",
	}
	sum := failureSummaryFromHealthGate(h)
	if sum == nil || sum.Code != "deploy_health_timeout" || sum.Class != "timeout" || sum.Service != "api" {
		t.Fatalf("%+v", sum)
	}
	h.ExternalDep = true
	h.RequestedBy = "web"
	sum = failureSummaryFromHealthGate(h)
	if sum.Code != "deploy_external_dep_unhealthy" || !sum.ExternalDep || sum.RequestedBy != "web" {
		t.Fatalf("%+v", sum)
	}
}

func TestMaybeWriteAttemptReceipt_healthGate(t *testing.T) {
	prev := inspectContainersForReceipt
	t.Cleanup(func() { inspectContainersForReceipt = prev })
	inspectContainersForReceipt = func(names []string) (map[string]runtimestate.ReceiptInspect, error) {
		out := make(map[string]runtimestate.ReceiptInspect, len(names))
		for _, n := range names {
			e := []receipt.EnvVar{}
			m := []receipt.MountSpec{}
			out[n] = runtimestate.ReceiptInspect{ID: "cid", Image: "img:1", Env: &e, Mounts: &m}
		}
		return out, nil
	}

	dir := t.TempDir()
	contractPath := filepath.Join(dir, "podbay.yaml")
	if err := os.WriteFile(contractPath, []byte("project: demo\nservices: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(dir, "receipts")
	if err := os.MkdirAll(store, 0o755); err != nil {
		t.Fatal(err)
	}
	var written string
	ctx := &deployContext{
		contractFile: contractPath,
		project:      "demo",
		opt: Options{
			Quiet:              true,
			ReceiptPath:        store + string(os.PathSeparator),
			WrittenReceiptPath: &written,
			DeployServices:     []string{"api"},
		},
		active:  map[string]spec.Service{"api": {Image: "x:1"}},
		partial: true,
		r:       runner.New("demo"),
		logf:    func(string, ...any) {},
	}
	hg := &HealthGateFailure{
		Service:      "api",
		ProbeKind:    ProbeHTTP,
		FailureClass: HealthFailureTimeout,
		Message:      "health check failed: timeout",
	}
	if err := maybeWriteAttemptReceipt(ctx, []string{"api"}, hg); err != nil {
		t.Fatal(err)
	}
	if written == "" {
		t.Fatal("expected WrittenReceiptPath")
	}
	raw, err := os.ReadFile(written)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := receipt.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Status != receipt.StatusFailed || rec.DeployID == "" || !strings.HasPrefix(rec.ContractDigest, "sha256:") {
		t.Fatalf("evidence %+v", rec)
	}
	if rec.Failure == nil || rec.Failure.Code != "deploy_health_timeout" {
		t.Fatalf("failure %+v", rec.Failure)
	}
	if len(rec.DeployServices) != 1 || rec.DeployServices[0] != "api" {
		t.Fatalf("selection %+v", rec.DeployServices)
	}
}

func TestMaybeWriteAttemptReceipt_skipsNonHealth(t *testing.T) {
	dir := t.TempDir()
	receiptPath := filepath.Join(dir, "r.json")
	ctx := &deployContext{
		opt:  Options{ReceiptPath: receiptPath, Quiet: true},
		logf: func(string, ...any) {},
	}
	if err := maybeWriteAttemptReceipt(ctx, nil, errors.New("build failed")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(receiptPath); !os.IsNotExist(err) {
		t.Fatal("must not write receipt for non-health error")
	}
}

func TestMaybeWriteAttemptReceipt_skipsEmptyReceiptPath(t *testing.T) {
	ctx := &deployContext{
		opt:  Options{Quiet: true},
		logf: func(string, ...any) {},
	}
	hg := &HealthGateFailure{Service: "api", FailureClass: HealthFailureProbeError, Message: "boom"}
	if err := maybeWriteAttemptReceipt(ctx, []string{"api"}, hg); err != nil {
		t.Fatal(err)
	}
}

func TestBuildReceiptServiceRecordsBestEffort_missingInspect(t *testing.T) {
	prev := inspectContainersForReceipt
	t.Cleanup(func() { inspectContainersForReceipt = prev })
	inspectContainersForReceipt = func(names []string) (map[string]runtimestate.ReceiptInspect, error) {
		return map[string]runtimestate.ReceiptInspect{}, nil
	}
	r := runner.New("demo")
	active := map[string]spec.Service{"web": {Image: "x:1"}}
	recs := buildReceiptServiceRecordsBestEffort(r, []string{"web"}, active, nil)
	if len(recs) != 1 || recs[0].Service != "web" || recs[0].ContainerID != "" {
		t.Fatalf("%+v", recs)
	}
}
