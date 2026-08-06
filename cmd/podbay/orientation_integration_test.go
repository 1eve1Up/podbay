package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/explain"
	"github.com/1eve1Up/podbay/internal/orientation"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestOrientationFlow_initOnboardVocabulary(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, spec.DefaultFilename)
	fileFlag := target

	init := initCmd(&fileFlag, target)
	var initBuf bytes.Buffer
	init.SetOut(&initBuf)
	if err := init.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(initBuf.String(), "podbay onboard") {
		t.Fatalf("init next steps: %q", initBuf.String())
	}

	onboard := onboardCmd(&fileFlag, target)
	var onboardBuf bytes.Buffer
	onboard.SetOut(&onboardBuf)
	onboard.SetArgs([]string{"--json"})
	if err := onboard.Execute(); err != nil {
		t.Fatalf("onboard: %v\n%s", err, onboardBuf.String())
	}
	var doc orientation.Document
	if err := json.Unmarshal(onboardBuf.Bytes(), &doc); err != nil {
		t.Fatalf("json: %v\n%s", err, onboardBuf.String())
	}
	if doc.Kind != orientation.Kind || doc.Note != orientation.BoundaryNote {
		t.Fatalf("doc: %+v", doc)
	}
	if len(doc.ActiveServices) == 0 || len(doc.Graph) == 0 || len(doc.NextActions) < 3 {
		t.Fatalf("incomplete orientation: %+v", doc)
	}

	c, loaded, err := spec.Load(target)
	if err != nil {
		t.Fatal(err)
	}
	// Shared vocabulary: explain-path orientation uses the same Build + AttachRuntime helpers.
	live, err := orientation.Build(c, loaded, orientation.BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	orientation.AttachRuntime(live, true, explain.RuntimeRowsFromStatus([]explain.ServiceStatus{
		{Name: "web", Missing: true},
	}))
	if live.Kind != doc.Kind || live.Project != doc.Project {
		t.Fatalf("vocabulary mismatch onboard=%+v live=%+v", doc, live)
	}
	if live.Runtime == nil || !live.Runtime.Available {
		t.Fatalf("explain-path orientation should attach runtime from synthetic status: %+v", live.Runtime)
	}
	if len(live.ActiveServices) != len(doc.ActiveServices) {
		t.Fatalf("active_services mismatch: onboard=%v live=%v", doc.ActiveServices, live.ActiveServices)
	}
}

func TestOrientationDemoScript_offline(t *testing.T) {
	modRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "podbay")
	build := exec.Command("go", "build", "-o", bin, "./cmd/podbay")
	build.Dir = modRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	script := filepath.Join(modRoot, "examples", "ci-orientation-demo.sh")
	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(), "PODBAY_BIN="+bin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("demo: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "ci-orientation-demo: ok") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func findModuleRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
