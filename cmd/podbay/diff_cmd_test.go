package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/receipt"
)

// TestDiffCmd_jsonFlagRegistered ensures the --json flag is wired on the
// diff subcommand. will add full end-to-end integration tests; this
// is the minimum guard that left the flag in place.
func TestDiffCmd_jsonFlagRegistered(t *testing.T) {
	f := ""
	cmd := diffCmd(&f, "podbay.yaml")

	if cmd.Flags().Lookup("json") == nil {
		t.Fatalf("expected --json flag on diff subcommand; flags:\n%s", cmd.Flags().FlagUsages())
	}
	if cmd.Flags().Lookup("profile") == nil {
		t.Fatal("--profile flag must remain on diff subcommand")
	}
	if cmd.Flags().Lookup("receipt-diff-show-env") == nil {
		t.Fatal("expected --receipt-diff-show-env on diff subcommand")
	}
	if cmd.Flags().Lookup("dependents") == nil {
		t.Fatal("expected --dependents flag on diff subcommand")
	}
}

// TestDiffCmd_helpDocumentsJSONAndExitCodes asserts --help mentions the
// new --json flag and keeps the exit-code documentation block intact.
func TestDiffCmd_helpDocumentsJSONAndExitCodes(t *testing.T) {
	f := ""
	cmd := diffCmd(&f, "podbay.yaml")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Help(); err != nil {
		t.Fatalf("help: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"--json", "Exit codes:", "0  No drift", "1  Drift", "receipt", "decode", "receipt-diff-show-env", "redacted", "--dependents"} {
		if !strings.Contains(out, want) {
			t.Errorf("--help missing %q. Output:\n%s", want, out)
		}
	}
}

func TestDiffCmd_receiptPairJSON_envDrift_goRun(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "r1.json")
	p2 := filepath.Join(dir, "r2.json")
	fixed := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	e1 := []receipt.EnvVar{{Name: "PORT", Value: "8080"}}
	e2 := []receipt.EnvVar{{Name: "PORT", Value: "9090"}}
	r1 := &receipt.Receipt{
		FormatVersion: receipt.CurrentFormatVersion,
		GeneratedAt:   fixed,
		ContractPath:  "/app/p.yaml",
		Project:       "demo",
		Services: []receipt.ServiceRecord{{
			Service: "api", ContainerName: "n1", Image: "img:v1",
			Env: &e1,
		}},
	}
	r2 := &receipt.Receipt{
		FormatVersion: receipt.CurrentFormatVersion,
		GeneratedAt:   fixed,
		ContractPath:  "/app/p.yaml",
		Project:       "demo",
		Services: []receipt.ServiceRecord{{
			Service: "api", ContainerName: "n1", Image: "img:v1",
			Env: &e2,
		}},
	}
	raw1, err := receipt.Encode(r1)
	if err != nil {
		t.Fatal(err)
	}
	raw2, err := receipt.Encode(r2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p1, raw1, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p2, raw2, 0o644); err != nil {
		t.Fatal(err)
	}

	modRoot := filepath.Clean(filepath.Join("..", ".."))
	exe := exec.Command("go", "run", "./cmd/podbay", "diff", "--json", p1, p2)
	exe.Dir = modRoot
	var stdout bytes.Buffer
	exe.Stdout = &stdout
	exe.Stderr = io.Discard
	err = exe.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("want exit 1, err=%v stdout=%s", err, stdout.String())
	}
	out := stdout.String()
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if m["drift"] != true {
		t.Fatalf("drift=%v", m["drift"])
	}
	issues, _ := m["issues"].([]any)
	var sawEnv bool
	for _, it := range issues {
		is, _ := it.(map[string]any)
		if is["code"] == receipt.CodeEnvChanged {
			sawEnv = true
			break
		}
	}
	if !sawEnv {
		t.Fatalf("missing env changed issue: %v", issues)
	}
}

func TestDiffCmd_receiptPairJSON_noDrift(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "r1.json")
	p2 := filepath.Join(dir, "r2.json")
	fixed := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	r := &receipt.Receipt{
		FormatVersion: receipt.CurrentFormatVersion,
		GeneratedAt:   fixed,
		ContractPath:  "/app/p.yaml",
		Project:       "demo",
		Profiles:      []string{"dev"},
		Services:      []receipt.ServiceRecord{{Service: "api", ContainerName: "n1", Image: "img:v1"}},
	}
	raw, err := receipt.Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p1, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p2, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	f := ""
	cmd := diffCmd(&f, "podbay.yaml")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--json", p1, p2})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, buf.String())
	}
	if m["kind"] != clijson.KindDiff {
		t.Fatalf("kind=%v", m["kind"])
	}
	if _, ok := m["receipt_pair"]; !ok {
		t.Fatalf("missing receipt_pair: %s", buf.String())
	}
}

func TestDiffCmd_receiptPairProfileRejected(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "r1.json")
	p2 := filepath.Join(dir, "r2.json")
	fixed := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	r := &receipt.Receipt{
		FormatVersion: receipt.CurrentFormatVersion,
		GeneratedAt:   fixed,
		ContractPath:  "/x.yaml",
		Project:       "p",
		Services:      []receipt.ServiceRecord{{Service: "s", ContainerName: "c"}},
	}
	raw, _ := receipt.Encode(r)
	_ = os.WriteFile(p1, raw, 0o644)
	_ = os.WriteFile(p2, raw, 0o644)

	f := ""
	cmd := diffCmd(&f, "podbay.yaml")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	// Human mode: must return error (JSON path would os.Exit before test finishes).
	cmd.SetArgs([]string{"--profile", "dev", p1, p2})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--profile") {
		t.Fatalf("err=%v", err)
	}
}
