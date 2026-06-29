package runner

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestHTTPProbeOnce_timesOutQuickly(t *testing.T) {
	// Unreachable localhost port; should fail within short deadline (not hang).
	start := time.Now()
	_, err := HTTPProbeOnce("http://127.0.0.1:1/", false, 200*time.Millisecond)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error for unreachable probe URL")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("probe took too long: %v", elapsed)
	}
}

func TestExecOnceWithTimeout_rejectsEmptyCommand(t *testing.T) {
	_, err := ExecOnceWithTimeout("unused-container", nil, time.Second)
	if err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestExecOnceWithTimeout_timesOutQuickly(t *testing.T) {
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman not on PATH")
	}
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not on PATH")
	}
	start := time.Now()
	_, err := ExecOnceWithTimeout("unused-container", []string{"sleep", "30"}, 150*time.Millisecond)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("exec probe took too long: %v", elapsed)
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout message, got: %v", err)
	}
}

func TestPodmanExecCombinedContext_timesOutWithoutPodman(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not on PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := podmanExecCombinedContext(ctx, "unused", []string{"sleep", "30"})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("command took too long: %v", elapsed)
	}
}
