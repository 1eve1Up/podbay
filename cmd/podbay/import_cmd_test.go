package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestImportCompose_jsonFlagRegistered(t *testing.T) {
	cmd := importCmd()
	var compose *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "compose" {
			compose = c
			break
		}
	}
	if compose == nil {
		t.Fatal("compose subcommand missing")
	}
	if compose.Flags().Lookup("json") == nil {
		t.Fatalf("expected --json on import compose; flags:\n%s", compose.Flags().FlagUsages())
	}
}

func TestImportCompose_helpMentionsJSON(t *testing.T) {
	cmd := importCmd()
	var compose *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "compose" {
			compose = c
			break
		}
	}
	var buf bytes.Buffer
	compose.SetOut(&buf)
	compose.SetErr(&buf)
	compose.SetArgs([]string{"--help"})
	if err := compose.Help(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"--json", "import_compose", "format_version"} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q\n%s", want, out)
		}
	}
}

func TestImportCompose_jsonMissingFile_goRun(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "absent-compose.yml")
	modRoot := filepath.Clean(filepath.Join("..", ".."))
	exe := exec.Command("go", "run", "./cmd/podbay", "import", "compose", "--json", missing)
	exe.Dir = modRoot
	var stdout bytes.Buffer
	exe.Stdout = &stdout
	exe.Stderr = io.Discard
	err := exe.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("want exit 1, err=%v stdout=%q", err, stdout.String())
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, stdout.String())
	}
	if m["kind"] != "import_compose" {
		t.Fatalf("kind=%v", m["kind"])
	}
	issues, _ := m["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("issues=%v", issues)
	}
	is0, _ := issues[0].(map[string]any)
	if is0["code"] != "import_compose_file_not_found" {
		t.Fatalf("code=%v", is0["code"])
	}
}

func modRoot() string {
	return filepath.Clean(filepath.Join("..", ".."))
}

func runImportComposeJSONExpectCode(t *testing.T, composePath string, wantCode string) {
	t.Helper()
	exe := exec.Command("go", "run", "./cmd/podbay", "import", "compose", "--json", composePath)
	exe.Dir = modRoot()
	var stdout bytes.Buffer
	exe.Stdout = &stdout
	exe.Stderr = io.Discard
	err := exe.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("want exit 1, err=%v stdout=%q", err, stdout.String())
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, stdout.String())
	}
	issues, _ := m["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("issues=%v", issues)
	}
	is0, _ := issues[0].(map[string]any)
	if is0["code"] != wantCode {
		t.Fatalf("code=%v want %q", is0["code"], wantCode)
	}
}

func TestImportCompose_jsonIncludeCycle_goRun(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.yml")
	b := filepath.Join(dir, "b.yml")
	if err := os.WriteFile(a, []byte("include: [\"./b.yml\"]\nservices: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("include: [\"./a.yml\"]\nservices: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runImportComposeJSONExpectCode(t, a, "import_include_cycle")
}

func TestImportCompose_jsonIncludePathEscape_goRun(t *testing.T) {
	dir := t.TempDir()
	primary := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(primary, []byte("include: [\"../../../outside.yml\"]\nservices: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runImportComposeJSONExpectCode(t, primary, "import_include_path_escape")
}

func TestImportCompose_jsonIncludeDepth_goRun(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 16; i++ {
		p := filepath.Join(dir, fmt.Sprintf("f%d.yml", i))
		body := fmt.Sprintf("include: [\"./f%d.yml\"]\nservices: {}\n", i+1)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	last := filepath.Join(dir, "f16.yml")
	if err := os.WriteFile(last, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runImportComposeJSONExpectCode(t, filepath.Join(dir, "f0.yml"), "import_include_depth")
}

func TestImportCompose_jsonInvalidYAML_goRun(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.yml")
	if err := os.WriteFile(bad, []byte("services: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runImportComposeJSONExpectCode(t, bad, "import_compose_parse")
}

func TestImportCompose_jsonIncludeUnsupportedURL_goRun(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "docker-compose.yml")
	body := "include:\n  - \"http://example.com/bad.yml\"\nservices: {}\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	runImportComposeJSONExpectCode(t, p, "import_include_unsupported")
}

func TestImportCompose_jsonSuccess_goRun(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "docker-compose.yml")
	body := "services:\n  web:\n    image: nginx:alpine\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := exec.Command("go", "run", "./cmd/podbay", "import", "compose", "--json", p)
	exe.Dir = modRoot()
	var stdout bytes.Buffer
	exe.Stdout = &stdout
	exe.Stderr = io.Discard
	err := exe.Run()
	if err != nil {
		t.Fatalf("want exit 0, err=%v stdout=%q", err, stdout.String())
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, stdout.String())
	}
	if m["kind"] != "import_compose" || m["status"] != "ok" {
		t.Fatalf("kind/status=%v/%v", m["kind"], m["status"])
	}
	if int(m["format_version"].(float64)) != 1 {
		t.Fatalf("format_version=%v", m["format_version"])
	}
	cy, _ := m["contract_yaml"].(string)
	if cy == "" || !strings.Contains(cy, "web") {
		t.Fatalf("contract_yaml missing web: %q", cy)
	}
	if m["contract_path"] == "" {
		t.Fatalf("contract_path empty")
	}
	if m["issues"] != nil {
		t.Fatalf("unexpected issues: %v", m["issues"])
	}
	sc, ok := m["service_count"].(float64)
	if !ok || int(sc) != 1 {
		t.Fatalf("service_count=%v", m["service_count"])
	}
}

func TestImportCompose_jsonSuccess_withOutput_goRun(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "docker-compose.yml")
	out := filepath.Join(dir, "out.yaml")
	body := "services:\n  web:\n    image: nginx:alpine\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := exec.Command("go", "run", "./cmd/podbay", "import", "compose", "--json", p, "-o", out)
	exe.Dir = modRoot()
	var stdout bytes.Buffer
	exe.Stdout = &stdout
	exe.Stderr = io.Discard
	if err := exe.Run(); err != nil {
		t.Fatalf("want exit 0, err=%v stdout=%q", err, stdout.String())
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, stdout.String())
	}
	if m["status"] != "ok" {
		t.Fatalf("status=%v", m["status"])
	}
	op, _ := m["output_path"].(string)
	if op == "" {
		t.Fatalf("output_path empty: %v", m)
	}
	disk, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	cy, _ := m["contract_yaml"].(string)
	if string(disk) != cy {
		t.Fatalf("disk yaml differs from contract_yaml field")
	}
}
