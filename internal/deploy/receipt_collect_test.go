package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/receipt"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestBuildReceiptServiceRecords_mockInspect(t *testing.T) {
	prev := inspectContainersForReceipt
	t.Cleanup(func() { inspectContainersForReceipt = prev })
	inspectContainersForReceipt = func(names []string) (map[string]runtimestate.ReceiptInspect, error) {
		out := make(map[string]runtimestate.ReceiptInspect, len(names))
		for _, n := range names {
			e := []receipt.EnvVar{}
			m := []receipt.MountSpec{}
			out[n] = runtimestate.ReceiptInspect{
				ID:     "id:" + n,
				Image:  "registry/img:podman",
				Env:    &e,
				Mounts: &m,
			}
		}
		return out, nil
	}

	r := runner.New("demo")
	active := map[string]spec.Service{
		"web": {Image: "my:local"},
		"api": {},
	}
	order := []string{"api", "web"}
	recs, err := buildReceiptServiceRecords(r, order, active, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records", len(recs))
	}
	if recs[0].Service != "api" || recs[0].Image != "registry/img:podman" {
		t.Fatalf("api record %+v", recs[0])
	}
	if recs[1].Service != "web" || recs[1].Image != "my:local" {
		t.Fatalf("web record %+v", recs[1])
	}
	if recs[1].ContainerID == "" {
		t.Fatal("expected container id")
	}
}

func TestWriteDeployReceipt_directoryMode(t *testing.T) {
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
		},
		active: map[string]spec.Service{"web": {Image: "x:1"}},
		r:      runner.New("demo"),
		logf:   func(string, ...any) {},
	}
	if err := writeDeployReceipt(ctx, []string{"web"}); err != nil {
		t.Fatal(err)
	}
	if written == "" || filepath.Dir(written) != store {
		// Dir may be abs-normalized
		absStore, _ := filepath.Abs(store)
		if filepath.Dir(written) != absStore {
			t.Fatalf("written=%q store=%q", written, store)
		}
	}
	raw, err := os.ReadFile(written)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := receipt.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if rec.DeployID == "" || rec.Status != receipt.StatusOK || !strings.HasPrefix(rec.ContractDigest, "sha256:") {
		t.Fatalf("evidence: %+v", rec)
	}
	if !strings.Contains(filepath.Base(written), rec.DeployID) {
		t.Fatalf("filename %q should contain deploy_id %q", written, rec.DeployID)
	}
}

func TestWriteDeployReceipt_evidenceFields(t *testing.T) {
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
	receiptPath := filepath.Join(dir, "r.json")
	ctx := &deployContext{
		contractFile: contractPath,
		project:      "demo",
		opt: Options{
			Quiet:            true,
			ReceiptPath:      receiptPath,
			DeployServices:   []string{"api"},
			DeployDependents: true,
		},
		active:  map[string]spec.Service{"api": {Image: "x:1"}},
		partial: true,
		r:       runner.New("demo"),
		logf:    func(string, ...any) {},
	}
	if err := writeDeployReceipt(ctx, []string{"api"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := receipt.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if rec.DeployID == "" || !strings.HasPrefix(rec.ContractDigest, "sha256:") {
		t.Fatalf("evidence missing: id=%q digest=%q", rec.DeployID, rec.ContractDigest)
	}
	if rec.Status != receipt.StatusOK {
		t.Fatalf("status=%q", rec.Status)
	}
	if len(rec.DeployServices) != 1 || rec.DeployServices[0] != "api" || !rec.DependentsExpand {
		t.Fatalf("selection %+v expand=%v", rec.DeployServices, rec.DependentsExpand)
	}
}
