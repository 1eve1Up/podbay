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
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected discovery error")
	}
	if composefile.CodeOrEmpty(err) != composefile.CodeCodebaseDiscoveryNotFound {
		t.Fatalf("code=%q err=%v", composefile.CodeOrEmpty(err), err)
	}
}

func TestInit_fromCodebase_dockerfileOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine:3.20\n"), 0o644); err != nil {
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
	s := string(data)
	if strings.Contains(s, "nginx:alpine") {
		t.Fatalf("wrote greenfield template: %s", s)
	}
	if !strings.Contains(s, "app:") || !strings.Contains(s, "dockerfile: Dockerfile") {
		t.Fatalf("missing Dockerfile stub: %s", s)
	}
	if !strings.Contains(s, "image: localhost/") {
		t.Fatalf("missing image tag: %s", s)
	}
	out := buf.String()
	if !strings.Contains(out, "Wrote "+target) || !strings.Contains(out, "Dockerfile") {
		t.Fatalf("missing wrote/source: %q", out)
	}
	if !strings.Contains(out, "podbay onboard") || !strings.Contains(out, "podbay validate") {
		t.Fatalf("missing next steps: %q", out)
	}
	if !strings.Contains(out, clijson.InitHandTightenHint) {
		t.Fatalf("missing hand-tighten hint: %q", out)
	}
}

func TestInit_fromCodebase_dockerfileCopiesExposeAndHealth(t *testing.T) {
	dir := t.TempDir()
	body := "FROM alpine:3.20\nEXPOSE 8080\nHEALTHCHECK CMD wget -q -O- http://127.0.0.1/\n"
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(body), 0o644); err != nil {
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
	s := string(data)
	if !strings.Contains(s, "expose:") || !strings.Contains(s, "8080") {
		t.Fatalf("missing expose: %s", s)
	}
	if !strings.Contains(s, "health:") || !strings.Contains(s, "wget") {
		t.Fatalf("missing health.exec: %s", s)
	}
	if strings.Contains(s, "8080:8080") || strings.Contains(s, "ports:") {
		t.Fatalf("invented published ports: %s", s)
	}
}

func TestInit_fromCodebase_composePreferredOverDockerfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine:3.20\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services:\n  api:\n    image: docker.io/library/nginx:alpine\n"), 0o644); err != nil {
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
	s := string(data)
	if !strings.Contains(s, "api:") {
		t.Fatalf("expected Compose import, got: %s", s)
	}
	if strings.Contains(s, "localhost/") {
		t.Fatalf("took Dockerfile stub despite Compose: %s", s)
	}
	if !strings.Contains(buf.String(), "compose.yaml") {
		t.Fatalf("expected compose source in output: %q", buf.String())
	}
}

func TestInit_fromCodebase_dockerfileFlagForcesStub(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine:3.20\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services:\n  api:\n    image: docker.io/library/nginx:alpine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, spec.DefaultFilename)
	fileFlag := target
	cmd := initCmd(&fileFlag, filepath.Join(dir, "ignored.yaml"))
	cmd.SetArgs([]string{"--from-codebase", dir, "--dockerfile", filepath.Join(dir, "Dockerfile")})
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
	if strings.Contains(string(data), "api:") {
		t.Fatalf("--dockerfile should skip Compose: %s", data)
	}
	if !strings.Contains(string(data), "dockerfile: Dockerfile") {
		t.Fatalf("missing stub: %s", data)
	}
}

func TestInit_greenfieldUnchangedBesideDockerfileFromCodebase(t *testing.T) {
	dir := t.TempDir()
	dfDir := filepath.Join(dir, "df-proj")
	if err := os.Mkdir(dfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dfDir, "Dockerfile"), []byte("FROM alpine:3.20\n"), 0o644); err != nil {
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
	gfData, err := os.ReadFile(gfTarget)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gfData), "nginx:alpine") {
		t.Fatalf("greenfield drifted: %s", gfData)
	}

	dfTarget := filepath.Join(dfDir, spec.DefaultFilename)
	fileFlag2 := dfTarget
	cmd2 := initCmd(&fileFlag2, filepath.Join(dfDir, "ignored.yaml"))
	cmd2.SetArgs([]string{"--from-codebase", dfDir})
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("dockerfile from-codebase: %v\n%s", err, buf2.String())
	}
	gfData2, err := os.ReadFile(gfTarget)
	if err != nil {
		t.Fatal(err)
	}
	if string(gfData2) != string(gfData) {
		t.Fatal("greenfield contract mutated after Dockerfile from-codebase")
	}
	dfData, err := os.ReadFile(dfTarget)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dfData), "nginx:alpine") {
		t.Fatalf("dockerfile path wrote greenfield template: %s", dfData)
	}
	if !strings.Contains(string(dfData), "dockerfile: Dockerfile") {
		t.Fatalf("missing stub: %s", dfData)
	}
}

func TestInit_fromCodebase_dockerfileRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine:3.20\n"), 0o644); err != nil {
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

func TestInit_fromCodebase_composeAndDockerfileFlagsRejected(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, spec.DefaultFilename)
	fileFlag := target
	cmd := initCmd(&fileFlag, filepath.Join(dir, "ignored.yaml"))
	cmd.SetArgs([]string{"--from-codebase", dir, "--compose", "compose.yaml", "--dockerfile", "Dockerfile"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected mutual exclusion error")
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
	if doc.SourceKind != clijson.InitSourceCompose || doc.DockerfileSource != "" {
		t.Fatalf("source_kind=%q dockerfile_source=%q", doc.SourceKind, doc.DockerfileSource)
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

func TestInit_fromCodebase_dockerfileOrientMatchesComposeDialect(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine:3.20\n"), 0o644); err != nil {
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
	wantGates := clijson.InitOrientNextActions(target)
	if len(doc.NextActions) < len(wantGates) {
		t.Fatalf("next_actions=%v want prefix %v", doc.NextActions, wantGates)
	}
	for i := range wantGates {
		if doc.NextActions[i] != wantGates[i] {
			t.Fatalf("next_actions[%d]=%q want %q", i, doc.NextActions[i], wantGates[i])
		}
	}
	if !strings.Contains(strings.Join(doc.NextActions, "\n"), clijson.InitHandTightenHint) {
		t.Fatalf("missing hand-tighten hint: %v", doc.NextActions)
	}
	if strings.Contains(strings.ToLower(strings.Join(doc.NextActions, " ")), "deploy") {
		t.Fatalf("must not auto-suggest deploy: %v", doc.NextActions)
	}
}

func TestInit_fromCodebase_jsonDockerfileSuccess(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine:3.20\n"), 0o644); err != nil {
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
	if doc.SourceKind != clijson.InitSourceDockerfile {
		t.Fatalf("source_kind=%q", doc.SourceKind)
	}
	if !strings.HasSuffix(doc.DockerfileSource, "Dockerfile") {
		t.Fatalf("dockerfile_source=%q", doc.DockerfileSource)
	}
	if len(doc.Extracted) != 0 {
		t.Fatalf("bare Dockerfile extracted=%v", doc.Extracted)
	}
	if !strings.Contains(strings.Join(doc.Gaps, ","), clijson.InitFieldPublishedPorts) {
		t.Fatalf("gaps=%v", doc.Gaps)
	}
	if doc.ComposeSource != "" {
		t.Fatalf("compose_source should be empty: %q", doc.ComposeSource)
	}
	if doc.ImportServiceCount != 1 {
		t.Fatalf("service_count=%d", doc.ImportServiceCount)
	}
	joined := strings.Join(doc.NextActions, "\n")
	if !strings.Contains(joined, "podbay onboard") || !strings.Contains(joined, "podbay validate") {
		t.Fatalf("next_actions=%v", doc.NextActions)
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
	if doc.Issues[0].Code != composefile.CodeCodebaseDiscoveryNotFound {
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
