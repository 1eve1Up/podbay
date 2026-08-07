package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/composefile"
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

func TestInit_greenfieldUnchangedBesideFromCodebase(t *testing.T) {
	dir := t.TempDir()
	// from-codebase in a sibling dir must not change bare-init defaults.
	composeDir := filepath.Join(dir, "compose-proj")
	if err := os.Mkdir(composeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(composeDir, "compose.yaml"), []byte("services:\n  api:\n    image: docker.io/library/nginx:alpine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gfTarget := filepath.Join(dir, "greenfield", spec.DefaultFilename)
	if err := os.Mkdir(filepath.Dir(gfTarget), 0o755); err != nil {
		t.Fatal(err)
	}
	fileFlag := gfTarget
	cmd := initCmd(&fileFlag, filepath.Join(dir, "ignored.yaml"))
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("greenfield: %v\n%s", err, buf.String())
	}
	data, err := os.ReadFile(gfTarget)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "project: myapp") || !strings.Contains(string(data), "nginx:alpine") {
		t.Fatalf("greenfield template drifted: %s", data)
	}
	if strings.Contains(string(data), "api:") {
		t.Fatalf("greenfield picked up from-codebase service names: %s", data)
	}

	cbTarget := filepath.Join(composeDir, spec.DefaultFilename)
	fileFlag2 := cbTarget
	cmd2 := initCmd(&fileFlag2, filepath.Join(composeDir, "ignored.yaml"))
	cmd2.SetArgs([]string{"--from-codebase", composeDir})
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("from-codebase: %v\n%s", err, buf2.String())
	}
	// Re-read greenfield — must be unchanged after from-codebase ran elsewhere.
	data2, err := os.ReadFile(gfTarget)
	if err != nil {
		t.Fatal(err)
	}
	if string(data2) != string(data) {
		t.Fatal("greenfield contract mutated after from-codebase")
	}
	cbData, err := os.ReadFile(cbTarget)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cbData), "project: myapp") {
		t.Fatalf("from-codebase wrote greenfield template: %s", cbData)
	}
	if !strings.Contains(string(cbData), "api:") {
		t.Fatalf("from-codebase missing imported service: %s", cbData)
	}
}

func TestInit_fromCodebase_writesContract(t *testing.T) {
	dir := t.TempDir()
	composeDst := filepath.Join(dir, "docker-compose.yml")
	composeYAML := "services:\n  web:\n    image: docker.io/library/nginx:alpine\n    ports:\n      - \"8080:80\"\n"
	if err := os.WriteFile(composeDst, []byte(composeYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, spec.DefaultFilename)
	fileFlag := target
	cmd := initCmd(&fileFlag, filepath.Join(dir, "ignored.yaml"))
	cmd.SetArgs([]string{"--from-codebase", dir})
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
	if strings.Contains(string(data), "project: myapp") {
		t.Fatalf("expected imported contract, not greenfield template: %s", data)
	}
	if !strings.Contains(string(data), "web:") {
		t.Fatalf("missing imported service: %s", data)
	}
	out := buf.String()
	if !strings.Contains(out, "Wrote "+target) {
		t.Fatalf("missing wrote line: %q", out)
	}
	if !strings.Contains(out, "docker-compose.yml") {
		t.Fatalf("missing compose source in output: %q", out)
	}
	if !strings.Contains(out, "podbay onboard") || !strings.Contains(out, "podbay validate") {
		t.Fatalf("missing next steps: %q", out)
	}
}

func TestInit_fromCodebase_refusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services:\n  web:\n    image: nginx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, spec.DefaultFilename)
	if err := os.WriteFile(target, []byte("version: \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fileFlag := target
	cmd := initCmd(&fileFlag, target)
	cmd.SetArgs([]string{"--from-codebase", dir})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected overwrite error")
	}
}

func TestInit_fromCodebase_noCompose(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, spec.DefaultFilename)
	fileFlag := target
	cmd := initCmd(&fileFlag, filepath.Join(dir, "ignored.yaml"))
	cmd.SetArgs([]string{"--from-codebase", dir})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected discovery error")
	}
}

func TestInit_fromCodebase_jsonSuccess(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services:\n  web:\n    image: docker.io/library/nginx:alpine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, spec.DefaultFilename)
	fileFlag := target
	cmd := initCmd(&fileFlag, filepath.Join(dir, "ignored.yaml"))
	cmd.SetArgs([]string{"--from-codebase", dir, "--json"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	var doc clijson.Document
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("json: %v\n%s", err, buf.String())
	}
	if doc.Kind != clijson.KindInit || doc.Status != clijson.StatusOK {
		t.Fatalf("%+v", doc)
	}
	if doc.ContractPath != target {
		t.Fatalf("contract_path=%q", doc.ContractPath)
	}
	if !strings.HasSuffix(doc.ComposeSource, "compose.yaml") {
		t.Fatalf("compose_source=%q", doc.ComposeSource)
	}
	if doc.ImportServiceCount != 1 {
		t.Fatalf("service_count=%d", doc.ImportServiceCount)
	}
	if len(doc.NextActions) < 2 {
		t.Fatalf("next_actions=%v", doc.NextActions)
	}
	joined := strings.Join(doc.NextActions, "\n")
	if !strings.Contains(joined, "podbay onboard") || !strings.Contains(joined, "podbay validate") {
		t.Fatalf("next_actions missing orient gates: %v", doc.NextActions)
	}
	for _, a := range doc.NextActions {
		if strings.Contains(strings.ToLower(a), "fix") || strings.Contains(strings.ToLower(a), "remediat") {
			t.Fatalf("remediation verb in next_actions: %q", a)
		}
	}
}

func TestInit_fromCodebase_jsonNoCompose(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, spec.DefaultFilename)
	out := runInitJSONExpectFail(t, dir, target, []string{"--from-codebase", dir, "--json"})
	var doc clijson.Document
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if doc.Status != clijson.StatusFailed || len(doc.Issues) == 0 {
		t.Fatalf("%+v", doc)
	}
	if doc.Issues[0].Code != composefile.CodeComposeDiscoveryNotFound {
		t.Fatalf("code=%q", doc.Issues[0].Code)
	}
}

func TestInit_fromCodebase_jsonOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services:\n  web:\n    image: nginx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, spec.DefaultFilename)
	if err := os.WriteFile(target, []byte("version: \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runInitJSONExpectFail(t, dir, target, []string{"--from-codebase", dir, "--json"})
	var doc clijson.Document
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if doc.Issues[0].Code != clijson.CodeInitTargetExists {
		t.Fatalf("code=%q doc=%+v", doc.Issues[0].Code, doc)
	}
}

// runInitJSONExpectFail runs init in a subprocess because JSON failure paths call os.Exit(1).
func runInitJSONExpectFail(t *testing.T, dir, target string, args []string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "podbay-test")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = "."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	cmdArgs := append([]string{"init", "-f", target}, args...)
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit\n%s", out)
	}
	return string(out)
}
