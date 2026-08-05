package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/1eve1Up/podbay/internal/receipt"
)

func TestReceiptHandoff_CLIJSON(t *testing.T) {
	dir := t.TempDir()
	write := func(name, status, id, digest string, at time.Time) string {
		r := &receipt.Receipt{
			FormatVersion:  receipt.CurrentFormatVersion,
			GeneratedAt:    at,
			ContractPath:   "/c.yaml",
			Project:        "demo",
			ContractDigest: digest,
			DeployID:       id,
			Status:         status,
			Services:       []receipt.ServiceRecord{{Service: "web", ContainerName: "c", Image: "web:1"}},
		}
		if status == receipt.StatusFailed {
			r.Failure = &receipt.FailureSummary{Code: "deploy_health_timeout", Service: "web", Class: "timeout"}
			r.Services[0].Image = "web:2"
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
	write("ok.json", receipt.StatusOK, "ok-id", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	failPath := write("fail.json", receipt.StatusFailed, "fail-id", "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC))

	cmd := receiptCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"handoff", failPath, "--store", dir, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, buf.String())
	}
	if m["kind"] != "receipt_handoff" || m["status"] != "ok" {
		t.Fatalf("%+v", m)
	}
	h, _ := m["handoff"].(map[string]any)
	if h == nil {
		t.Fatalf("missing handoff: %+v", m)
	}
	if h["deploy_id"] != "fail-id" {
		t.Fatalf("%+v", h)
	}
	failObj, _ := h["failure"].(map[string]any)
	if failObj == nil || failObj["code"] != "deploy_health_timeout" {
		t.Fatalf("failure=%v", h["failure"])
	}
	actions, _ := h["next_actions"].([]any)
	if len(actions) < 3 {
		t.Fatalf("next_actions=%v", actions)
	}
	if !strings.Contains(actions[0].(string), "logs") {
		t.Fatalf("actions=%v", actions)
	}
}
