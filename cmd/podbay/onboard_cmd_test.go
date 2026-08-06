package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/orientation"
)

func TestOnboard_CLI_offlineJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	yaml := `
version: "1"
project: onboarddemo
services:
  web:
    image: docker.io/library/nginx:alpine
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	fileFlag := path
	cmd := onboardCmd(&fileFlag, path)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, buf.String())
	}
	if m["kind"] != orientation.Kind {
		t.Fatalf("kind: %+v", m)
	}
	if m["project"] != "onboarddemo" {
		t.Fatalf("project: %+v", m)
	}
	if m["note"] != orientation.BoundaryNote {
		t.Fatalf("note: %+v", m)
	}
	actions, _ := m["next_actions"].([]any)
	if len(actions) < 3 {
		t.Fatalf("next_actions: %v", actions)
	}
	joined := ""
	for _, a := range actions {
		joined += a.(string) + "\n"
	}
	if !strings.Contains(joined, "validate") || !strings.Contains(joined, "deploy") {
		t.Fatalf("next_actions: %s", joined)
	}
}

func TestOnboard_CLI_missingContractJSON(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope", "podbay.yaml")
	fileFlag := missing
	cmd := onboardCmd(&fileFlag, missing)
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"--json"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	var m map[string]any
	if jerr := json.Unmarshal(out.Bytes(), &m); jerr != nil {
		t.Fatalf("json: %v\nout=%s\nerr=%s", jerr, out.String(), errBuf.String())
	}
	if m["kind"] != orientation.Kind || m["status"] != "failed" {
		t.Fatalf("%+v", m)
	}
	issues, _ := m["issues"].([]any)
	if len(issues) == 0 {
		t.Fatalf("expected issues: %+v", m)
	}
}
