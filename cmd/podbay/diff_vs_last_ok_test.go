package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/receipt"
)

func writeReceiptFile(t *testing.T, dir, name string, r *receipt.Receipt) string {
	t.Helper()
	raw, err := receipt.Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDiffVsLastOK_matchesTwoPathCompare(t *testing.T) {
	dir := t.TempDir()
	okRec := &receipt.Receipt{
		FormatVersion:  receipt.CurrentFormatVersion,
		GeneratedAt:    time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		ContractPath:   "/c.yaml",
		Project:        "demo",
		ContractDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Status:         receipt.StatusOK,
		Services:       []receipt.ServiceRecord{{Service: "web", ContainerName: "c", Image: "web:1"}},
		DeployID:       "ok-id",
	}
	curRec := &receipt.Receipt{
		FormatVersion:  receipt.CurrentFormatVersion,
		GeneratedAt:    time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		ContractPath:   "/c.yaml",
		Project:        "demo",
		ContractDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Status:         receipt.StatusFailed,
		Failure:        &receipt.FailureSummary{Code: "deploy_health_timeout", Service: "web", Class: "timeout"},
		Services:       []receipt.ServiceRecord{{Service: "web", ContainerName: "c", Image: "web:2"}},
		DeployID:       "fail-id",
	}
	okPath := writeReceiptFile(t, dir, "ok.json", okRec)
	curPath := writeReceiptFile(t, dir, "fail.json", curRec)

	entry, err := receipt.LastOK(dir)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Path != okPath && filepath.Base(entry.Path) != "ok.json" {
		t.Fatalf("LastOK path=%q want %q", entry.Path, okPath)
	}

	dataA, err := os.ReadFile(entry.Path)
	if err != nil {
		t.Fatal(err)
	}
	dataB, err := os.ReadFile(curPath)
	if err != nil {
		t.Fatal(err)
	}
	a, err := receipt.Decode(dataA)
	if err != nil {
		t.Fatal(err)
	}
	b, err := receipt.Decode(dataB)
	if err != nil {
		t.Fatal(err)
	}
	viaLastOK := receipt.CompareReceipts(a, b)

	a2, err := receipt.Decode(mustRead(t, okPath))
	if err != nil {
		t.Fatal(err)
	}
	b2, err := receipt.Decode(mustRead(t, curPath))
	if err != nil {
		t.Fatal(err)
	}
	twoPath := receipt.CompareReceipts(a2, b2)

	if viaLastOK.Drift != twoPath.Drift {
		t.Fatalf("drift mismatch: vs-last-ok=%v two-path=%v", viaLastOK.Drift, twoPath.Drift)
	}
	if !viaLastOK.Drift {
		t.Fatal("expected digest/image drift between ok and current")
	}
	out, code := captureReceiptPairResultJSON(t, viaLastOK)
	if code != 1 {
		t.Fatalf("exit=%d want 1", code)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatal(err)
	}
	assertReceiptPairDiffEnvelopeBase(t, m, clijson.StatusFailed)
}

func TestDiffVsLastOK_noPriorOKNoFalseDrift(t *testing.T) {
	dir := t.TempDir()
	writeReceiptFile(t, dir, "fail.json", &receipt.Receipt{
		FormatVersion: receipt.CurrentFormatVersion,
		GeneratedAt:   time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		ContractPath:  "/c.yaml",
		Project:       "demo",
		Status:        receipt.StatusFailed,
		Failure:       &receipt.FailureSummary{Code: "deploy_health_timeout", Service: "web", Class: "timeout"},
		Services:      []receipt.ServiceRecord{{Service: "web", ContainerName: "c"}},
		DeployID:      "fail-id",
	})
	_, err := receipt.LastOK(dir)
	if !errors.Is(err, receipt.ErrNoLastOK) {
		t.Fatalf("err=%v", err)
	}
	out := captureReceiptPairErrorJSON(t, clijson.CodeReceiptNoLastOK, receipt.ErrNoLastOK.Error())
	m := parseEnvelope(t, out)
	if _, ok := m["drift"]; ok {
		t.Fatalf("false drift present: %+v", m)
	}
	if m["status"] != clijson.StatusFailed {
		t.Fatalf("%+v", m)
	}
	issues, _ := m["issues"].([]any)
	if len(issues) < 1 {
		t.Fatalf("%+v", m)
	}
	first, _ := issues[0].(map[string]any)
	if first["code"] != clijson.CodeReceiptNoLastOK {
		t.Fatalf("%+v", first)
	}
}

func TestDiffVsLastOK_humanNoPriorOK(t *testing.T) {
	dir := t.TempDir()
	cur := writeReceiptFile(t, dir, "fail.json", &receipt.Receipt{
		FormatVersion: receipt.CurrentFormatVersion,
		GeneratedAt:   time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		ContractPath:  "/c.yaml",
		Project:       "demo",
		Status:        receipt.StatusFailed,
		Failure:       &receipt.FailureSummary{Code: "deploy_health_timeout", Service: "web", Class: "timeout"},
		Services:      []receipt.ServiceRecord{{Service: "web", ContainerName: "c"}},
		DeployID:      "fail-id",
	})
	file := ""
	cmd := diffCmd(&file, "podbay.yaml")
	cmd.SetArgs([]string{"--vs-last-ok", dir, cur})
	err := cmd.Execute()
	if !errors.Is(err, receipt.ErrNoLastOK) {
		t.Fatalf("err=%v want ErrNoLastOK", err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
