package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/receipt"
)

func TestReceiptLastOK_CLI(t *testing.T) {
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
	cmd.SetArgs([]string{"last-ok", dir, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, buf.String())
	}
	if m["kind"] != "receipt_last_ok" || m["status"] != "ok" {
		t.Fatalf("%+v", m)
	}
	if m["receipt_path"] == nil || m["receipt_path"] == "" {
		t.Fatalf("missing receipt_path: %+v", m)
	}
	path, _ := m["receipt_path"].(string)
	if !strings.HasSuffix(path, "ok.json") {
		t.Fatalf("path=%q", path)
	}
}

func TestReceiptLastOK_humanNoPriorOK(t *testing.T) {
	dir := t.TempDir()
	r := &receipt.Receipt{
		FormatVersion: receipt.CurrentFormatVersion,
		GeneratedAt:   time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		ContractPath:  "/c.yaml",
		Project:       "demo",
		Services:      []receipt.ServiceRecord{{Service: "web", ContainerName: "c"}},
		DeployID:      "fail-id",
		Status:        receipt.StatusFailed,
		Failure:       &receipt.FailureSummary{Code: "deploy_health_timeout", Service: "web", Class: "timeout"},
	}
	raw, err := receipt.Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fail.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := receiptCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"last-ok", dir})
	err = cmd.Execute()
	if !errors.Is(err, receipt.ErrNoLastOK) {
		t.Fatalf("err=%v want ErrNoLastOK\n%s", err, buf.String())
	}
}

func TestReceiptLastOK_JSONHelpers(t *testing.T) {
	entry := receipt.ListEntry{
		Path: "/tmp/ok.json", DeployID: "d1", GeneratedAt: "2026-08-01T00:00:00Z",
		Project: "demo", Status: receipt.StatusOK, ServiceCount: 1,
	}
	ok := clijson.ReceiptLastOKSuccess("/tmp/store", entry)
	if ok.Kind != clijson.KindReceiptLastOK || ok.Status != clijson.StatusOK || ok.ReceiptPath != entry.Path {
		t.Fatalf("%+v", ok)
	}
	fail := clijson.ReceiptLastOKFailure("/tmp/store", "receipt_no_last_ok", receipt.ErrNoLastOK)
	if fail.Status != clijson.StatusFailed || len(fail.Issues) != 1 || fail.Issues[0].Code != "receipt_no_last_ok" {
		t.Fatalf("%+v", fail)
	}
	if fail.ReceiptPath != "" {
		t.Fatalf("invented path %q", fail.ReceiptPath)
	}
}
