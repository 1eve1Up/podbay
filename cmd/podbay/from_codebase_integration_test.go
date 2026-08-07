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
	"github.com/1eve1Up/podbay/internal/orientation"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestFromCodebaseFlow_initOnboardValidate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(`services:
  web:
    image: docker.io/library/nginx:alpine
`), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, spec.DefaultFilename)
	fileFlag := target

	init := initCmd(&fileFlag, target)
	init.SetArgs([]string{"--from-codebase", dir, "--json"})
	var initBuf bytes.Buffer
	init.SetOut(&initBuf)
	if err := init.Execute(); err != nil {
		t.Fatalf("init: %v\n%s", err, initBuf.String())
	}
	var initDoc clijson.Document
	if err := json.Unmarshal(initBuf.Bytes(), &initDoc); err != nil {
		t.Fatalf("init json: %v\n%s", err, initBuf.String())
	}
	if initDoc.Kind != clijson.KindInit || initDoc.Status != clijson.StatusOK {
		t.Fatalf("%+v", initDoc)
	}

	onboard := onboardCmd(&fileFlag, target)
	var onboardBuf bytes.Buffer
	onboard.SetOut(&onboardBuf)
	onboard.SetArgs([]string{"--json"})
	if err := onboard.Execute(); err != nil {
		t.Fatalf("onboard: %v\n%s", err, onboardBuf.String())
	}
	var orient orientation.Document
	if err := json.Unmarshal(onboardBuf.Bytes(), &orient); err != nil {
		t.Fatalf("onboard json: %v", err)
	}
	if orient.Kind != orientation.Kind || len(orient.ActiveServices) == 0 {
		t.Fatalf("%+v", orient)
	}

	val := validateCmd(&fileFlag, target)
	var valBuf bytes.Buffer
	val.SetOut(&valBuf)
	val.SetArgs([]string{"--json"})
	if err := val.Execute(); err != nil {
		t.Fatalf("validate: %v\n%s", err, valBuf.String())
	}
	if !strings.Contains(valBuf.String(), `"status": "ok"`) && !strings.Contains(valBuf.String(), `"status":"ok"`) {
		t.Fatalf("validate not ok: %s", valBuf.String())
	}
}

func TestFromCodebaseDemoScript_offline(t *testing.T) {
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
	script := filepath.Join(modRoot, "examples", "ci-from-codebase-demo.sh")
	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(), "PODBAY_BIN="+bin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("demo: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "ci-from-codebase-demo: ok") {
		t.Fatalf("unexpected output: %s", out)
	}
}
