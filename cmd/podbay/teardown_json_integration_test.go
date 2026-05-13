package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/teardown"
)

const teardownProject = "demo"

func captureTeardownJSON(t *testing.T, res teardown.TeardownResult, runErr error) (string, int) {
	t.Helper()
	cmd := &cobra.Command{}
	var buf strings.Builder
	cmd.SetOut(&buf)
	emitTeardownJSON(cmd, "/app/podbay.yaml", teardownProject, nil, nil, false, res, runErr)
	return buf.String(), teardown.ExitCode(runErr)
}

func captureTeardownLoadJSON(t *testing.T, msg string) string {
	t.Helper()
	cmd := &cobra.Command{}
	var buf strings.Builder
	cmd.SetOut(&buf)
	emitTeardownLoadJSON(cmd, "", "", nil, msg)
	return buf.String()
}

func parseTeardownEnvelope(t *testing.T, payload string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &m); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, payload)
	}
	return m
}

func TestTeardownJSONIntegration_successNoContainers(t *testing.T) {
	res := teardown.TeardownResult{Project: teardownProject, Network: "podbay_demo"}
	out, exit := captureTeardownJSON(t, res, nil)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	m := parseTeardownEnvelope(t, out)
	if m["kind"] != clijson.KindTeardown || m["status"] != clijson.StatusOK {
		t.Fatalf("envelope: %+v", m)
	}
}

func TestTeardownJSONIntegration_fatalVolume(t *testing.T) {
	res := teardown.TeardownResult{Project: teardownProject}
	err := teardown.NewFatalError(teardown.CodeVolumeError, errors.New("volume busy"))
	out, exit := captureTeardownJSON(t, res, err)
	if exit != 1 {
		t.Fatalf("exit = %d", exit)
	}
	m := parseTeardownEnvelope(t, out)
	if m["status"] != clijson.StatusFailed {
		t.Fatalf("status = %v", m["status"])
	}
	issues, _ := m["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("issues = %v", issues)
	}
	first, _ := issues[0].(map[string]any)
	if first["code"] != teardown.CodeVolumeError {
		t.Fatalf("issue code = %v", first["code"])
	}
}

func TestTeardownJSONIntegration_networkWarnStillOK(t *testing.T) {
	res := teardown.TeardownResult{
		Project:        teardownProject,
		Network:        "podbay_demo",
		NetworkWarning: "network rm failed",
	}
	out, exit := captureTeardownJSON(t, res, nil)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	m := parseTeardownEnvelope(t, out)
	if m["status"] != clijson.StatusOK {
		t.Fatalf("status = %v", m["status"])
	}
	issues, _ := m["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("issues = %v", issues)
	}
}

func TestTeardownLoadJSONIntegration(t *testing.T) {
	out := captureTeardownLoadJSON(t, "no such file")
	m := parseTeardownEnvelope(t, out)
	if m["kind"] != clijson.KindTeardown || m["status"] != clijson.StatusFailed {
		t.Fatalf("%+v", m)
	}
	issues, _ := m["issues"].([]any)
	im, _ := issues[0].(map[string]any)
	if im["code"] != "teardown_load_error" {
		t.Fatalf("code = %v", im["code"])
	}
}
