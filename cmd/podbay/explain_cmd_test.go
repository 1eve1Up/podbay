package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/runner"
)

func TestExplainCmd_noProbesFlagRegistered(t *testing.T) {
	f := ""
	cmd := explainCmd(&f, "podbay.yaml")
	if cmd.Flags().Lookup("no-probes") == nil {
		t.Fatalf("expected --no-probes on explain; flags:\n%s", cmd.Flags().FlagUsages())
	}
	if cmd.Flags().Lookup("json") == nil {
		t.Fatal("--json must remain on explain")
	}
	if cmd.Flags().Lookup("dependents") == nil {
		t.Fatal("--dependents must remain on explain")
	}
}

func TestExplainCmd_helpDocumentsNoProbes(t *testing.T) {
	f := ""
	cmd := explainCmd(&f, "podbay.yaml")
	help := cmd.Long + "\n" + cmd.Flags().FlagUsages()
	for _, want := range []string{"--no-probes", "--json", "5s"} {
		if !strings.Contains(help, want) {
			t.Errorf("explain help missing %q:\n%s", want, help)
		}
	}
}

func TestExplainCmd_jsonNoProbesSkipsHealth(t *testing.T) {
	if err := runner.EnsurePodman(); err != nil {
		t.Skip("podman not available:", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	yaml := `
version: "1"
project: explainnoprobes
services:
  web:
    image: docker.io/library/nginx:alpine
    health:
      http:
        url: http://127.0.0.1:1/
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	fileFlag := path
	cmd := explainCmd(&fileFlag, path)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--json", "--no-probes"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v\n%s", err, buf.String())
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("json: %v\n%s", err, buf.String())
	}
	if m["kind"] != "explain" || m["probes_skipped"] != true {
		t.Fatalf("envelope: %+v", m)
	}
	if _, ok := m["orientation"]; !ok {
		t.Fatal("orientation must remain on --no-probes JSON")
	}
	svcs, _ := m["services"].([]any)
	if len(svcs) != 1 {
		t.Fatalf("services: %v", m["services"])
	}
	svc := svcs[0].(map[string]any)
	if svc["name"] != "web" {
		t.Fatalf("inspect missing: %+v", svc)
	}
	if _, ok := svc["health"]; ok {
		t.Fatalf("--no-probes JSON must omit health: %+v", svc)
	}
}
