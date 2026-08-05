package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/1eve1Up/podbay/internal/receipt"
)

func TestReceiptListStatusFilter_CLI(t *testing.T) {
	dir := t.TempDir()
	write := func(name, status, id string, at time.Time) {
		r := &receipt.Receipt{
			FormatVersion: receipt.CurrentFormatVersion,
			GeneratedAt:   at,
			ContractPath:  "/c.yaml",
			Project:       "demo",
			Services:      []receipt.ServiceRecord{{Service: "web", ContainerName: "c"}},
			DeployID:      id,
			Status:        status,
		}
		if status == receipt.StatusFailed {
			r.Failure = &receipt.FailureSummary{Code: "deploy_health_timeout", Service: "web", Class: "timeout"}
		}
		raw, err := receipt.Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("ok.json", receipt.StatusOK, "ok-id", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	write("fail.json", receipt.StatusFailed, "fail-id", time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC))

	cmd := receiptCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"list", dir, "--status", "failed", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, buf.String())
	}
	if m["kind"] != "receipt_list" {
		t.Fatalf("kind=%v", m["kind"])
	}
	recs, _ := m["receipts"].([]any)
	if len(recs) != 1 {
		t.Fatalf("receipts=%v", m["receipts"])
	}
	first, _ := recs[0].(map[string]any)
	if first["status"] != "failed" || first["deploy_id"] != "fail-id" {
		t.Fatalf("%+v", first)
	}
}
