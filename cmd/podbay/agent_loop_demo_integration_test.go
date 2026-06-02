package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAgentLoopDemo_happy(t *testing.T) {
	runAgentLoopDemo(t, "happy")
}

func TestAgentLoopDemo_fail(t *testing.T) {
	runAgentLoopDemo(t, "fail")
}

func runAgentLoopDemo(t *testing.T, mode string) {
	t.Helper()
	if exec.Command("podman", "version").Run() != nil {
		t.Skip("podman not on PATH")
	}
	root, err := filepath.Abs(modRoot())
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "examples", "ci-partial-agent-loop-demo.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "podbay")
	build := exec.Command("go", "build", "-o", bin, "./cmd/podbay")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build podbay: %v\n%s", err, out)
	}
	cmd := exec.Command("bash", script, mode)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PODBAY_BIN="+bin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mode=%s: %v\n%s", mode, err, out)
	}
}
