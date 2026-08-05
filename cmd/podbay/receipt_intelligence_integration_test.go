package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/1eve1Up/podbay/internal/receipt"
)

func TestReceiptIntelligenceFlow_integration(t *testing.T) {
	dir := t.TempDir()
	write := func(name, status, id, digest, image string, at time.Time) string {
		r := &receipt.Receipt{
			FormatVersion:  receipt.CurrentFormatVersion,
			GeneratedAt:    at,
			ContractPath:   "/c.yaml",
			Project:        "demo",
			ContractDigest: digest,
			DeployID:       id,
			Status:         status,
			Services:       []receipt.ServiceRecord{{Service: "web", ContainerName: "c", Image: image}},
		}
		if status == receipt.StatusFailed {
			r.Failure = &receipt.FailureSummary{Code: "deploy_health_timeout", Service: "web", Class: "timeout"}
		}
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
	okPath := write("ok.json", receipt.StatusOK, "ok-id",
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "web:1",
		time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	failPath := write("fail.json", receipt.StatusFailed, "fail-id",
		"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "web:2",
		time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC))

	// list → last-ok → handoff via CLI (no os.Exit paths)
	listCmd := receiptCmd()
	var listBuf bytes.Buffer
	listCmd.SetOut(&listBuf)
	listCmd.SetArgs([]string{"list", dir, "--status", "failed", "--json"})
	if err := listCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	lastCmd := receiptCmd()
	var lastBuf bytes.Buffer
	lastCmd.SetOut(&lastBuf)
	lastCmd.SetArgs([]string{"last-ok", dir, "--json"})
	if err := lastCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var last map[string]any
	if err := json.Unmarshal(lastBuf.Bytes(), &last); err != nil {
		t.Fatal(err)
	}
	if last["kind"] != "receipt_last_ok" {
		t.Fatalf("%+v", last)
	}

	entry, err := receipt.LastOK(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(entry.Path) != "ok.json" {
		t.Fatalf("path=%s", entry.Path)
	}

	a, err := receipt.Decode(mustRead(t, okPath))
	if err != nil {
		t.Fatal(err)
	}
	b, err := receipt.Decode(mustRead(t, failPath))
	if err != nil {
		t.Fatal(err)
	}
	viaPaths := receipt.CompareReceipts(a, b)
	viaLast, err := receipt.Decode(mustRead(t, entry.Path))
	if err != nil {
		t.Fatal(err)
	}
	viaResolve := receipt.CompareReceipts(viaLast, b)
	if viaPaths.Drift != viaResolve.Drift || !viaPaths.Drift {
		t.Fatalf("vs-last-ok drift mismatch: paths=%v resolve=%v", viaPaths.Drift, viaResolve.Drift)
	}

	handCmd := receiptCmd()
	var handBuf bytes.Buffer
	handCmd.SetOut(&handBuf)
	handCmd.SetArgs([]string{"handoff", failPath, "--store", dir, "--json"})
	if err := handCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var hand map[string]any
	if err := json.Unmarshal(handBuf.Bytes(), &hand); err != nil {
		t.Fatal(err)
	}
	h, _ := hand["handoff"].(map[string]any)
	if h == nil || h["deploy_id"] != "fail-id" {
		t.Fatalf("%+v", hand)
	}
	if h["drift"] != true {
		t.Fatalf("drift=%v", h["drift"])
	}
}

func TestReceiptIntelligenceDemoScript(t *testing.T) {
	root, err := filepath.Abs(modRoot())
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "examples", "ci-receipt-intelligence-demo.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "podbay")
	build := exec.Command("go", "build", "-o", bin, "./cmd/podbay")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build podbay: %v\n%s", err, out)
	}
	cmd := exec.Command("bash", script)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PODBAY_BIN="+bin, "PODBAY_DEMO_RECEIPT_DIR="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("demo failed: %v\n%s", err, out)
	}
}
