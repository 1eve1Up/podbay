package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/spec"
)

func TestInit_printsOrientationNextSteps(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, spec.DefaultFilename)
	fileFlag := target
	cmd := initCmd(&fileFlag, filepath.Join(dir, "ignored.yaml"))
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "project: myapp") {
		t.Fatalf("template not written: %s", data)
	}
	out := buf.String()
	if !strings.Contains(out, "Wrote "+target) {
		t.Fatalf("missing wrote line: %q", out)
	}
	if !strings.Contains(out, "podbay onboard") || !strings.Contains(out, "podbay validate") {
		t.Fatalf("missing next steps: %q", out)
	}
}

func TestInit_refusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, spec.DefaultFilename)
	if err := os.WriteFile(target, []byte("version: \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fileFlag := target
	cmd := initCmd(&fileFlag, target)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected overwrite error")
	}
}
