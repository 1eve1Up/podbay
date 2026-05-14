package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/clijson"
)

func TestLogsCmd_jsonFlagRegistered(t *testing.T) {
	fileFlag := ""
	defaultFile := filepath.Join(t.TempDir(), "podbay.yaml")
	cmd := logsCmd(&fileFlag, defaultFile)
	if cmd.Flags().Lookup("json") == nil {
		t.Fatalf("expected --json on logs; flags:\n%s", cmd.Flags().FlagUsages())
	}
}

func TestLogsCmd_helpMentionsJSON(t *testing.T) {
	fileFlag := ""
	cmd := logsCmd(&fileFlag, filepath.Join(t.TempDir(), "podbay.yaml"))
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Help(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"--json", "kind logs", "format_version"} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q\n%s", want, out)
		}
	}
}

func runLogsJSON(t *testing.T, extraArgs ...string) (stdout string, exitCode int) {
	t.Helper()
	args := append([]string{"run", "./cmd/podbay", "logs"}, extraArgs...)
	exe := exec.Command("go", args...)
	exe.Dir = modRoot()
	var out bytes.Buffer
	exe.Stdout = &out
	exe.Stderr = io.Discard
	err := exe.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return strings.TrimSpace(out.String()), exitErr.ExitCode()
	}
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return strings.TrimSpace(out.String()), 0
}

func TestLogsJSON_goRun_jsonFollowUsage(t *testing.T) {
	nginx := filepath.Join("examples", "nginx", "podbay.yaml")
	out, code := runLogsJSON(t, "--json", "--follow", nginx, "web")
	if code != 1 {
		t.Fatalf("exit = %d, out=%q", code, out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if m["kind"] != clijson.KindLogs {
		t.Fatalf("kind=%v", m["kind"])
	}
	issues, _ := m["issues"].([]any)
	im, _ := issues[0].(map[string]any)
	if im["code"] != clijson.CodeLogsUsageJSONFollow {
		t.Fatalf("code=%v", im["code"])
	}
}

func TestLogsJSON_goRun_missingContract(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope", "podbay.yaml")
	out, code := runLogsJSON(t, "--json", missing, "web")
	if code != 1 {
		t.Fatalf("exit = %d out=%q", code, out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("json: %v", err)
	}
	if m["kind"] != clijson.KindLogs {
		t.Fatalf("kind=%v", m["kind"])
	}
	issues, _ := m["issues"].([]any)
	im, _ := issues[0].(map[string]any)
	if im["code"] != clijson.CodeLogsLoadError {
		t.Fatalf("code=%v", im["code"])
	}
}

func TestLogsJSON_goRun_serviceNotActive(t *testing.T) {
	nginx := filepath.Join("examples", "nginx", "podbay.yaml")
	out, code := runLogsJSON(t, "--json", nginx, "not-a-service")
	if code != 1 {
		t.Fatalf("exit = %d out=%q", code, out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("json: %v", err)
	}
	issues, _ := m["issues"].([]any)
	im, _ := issues[0].(map[string]any)
	if im["code"] != clijson.CodeLogsServiceNotActive {
		t.Fatalf("code=%v", im["code"])
	}
}

func TestLogsJSON_goRun_successShape(t *testing.T) {
	if exec.Command("podman", "version").Run() != nil {
		t.Skip("podman not on PATH")
	}
	nginx := filepath.Join("examples", "nginx", "podbay.yaml")
	out, code := runLogsJSON(t, "--json", nginx, "web")
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if code != 0 {
		issues, _ := m["issues"].([]any)
		if len(issues) > 0 {
			im, _ := issues[0].(map[string]any)
			if im["code"] == clijson.CodeLogsRuntimeError || im["code"] == clijson.CodeLogsPodmanUnavailable {
				t.Skipf("no container or runtime: %v", im["message"])
			}
		}
		t.Fatalf("exit = %d payload=%+v", code, m)
	}
	if m["kind"] != clijson.KindLogs || m["status"] != clijson.StatusOK {
		t.Fatalf("envelope: %+v", m)
	}
	if m["service"] != "web" {
		t.Fatalf("service=%v", m["service"])
	}
	if _, ok := m["container_name"]; !ok {
		t.Fatalf("missing container_name: %v", m)
	}
	if _, ok := m["log_body"]; !ok {
		t.Fatalf("missing log_body: %v", m)
	}
}
