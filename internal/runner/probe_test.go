package runner

import (
	"context"
	"fmt"
	"os"
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
	if err := EnsurePodman(); err != nil {
		t.Skip("podman not available:", err)
	}
	// Missing containers fail immediately (exit 125); a running container is required
	// so sleep can outlive the probe deadline and exercise the timeout path.
	name := fmt.Sprintf("podbay-probe-timeout-%d", os.Getpid())
	_ = exec.Command("podman", "rm", "-f", name).Run()
	out, err := exec.Command(
		"podman", "run", "-d", "--name", name,
		"docker.io/library/alpine:latest", "sleep", "300",
	).CombinedOutput()
	if err != nil {
		t.Skipf("could not start probe container: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	t.Cleanup(func() {
		_ = exec.Command("podman", "rm", "-f", name).Run()
	})

	start := time.Now()
	_, err = ExecOnceWithTimeout(name, []string{"sleep", "30"}, 150*time.Millisecond)
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

func TestPodmanExecCombinedContext_failsQuicklyMissingContainer(t *testing.T) {
	if err := EnsurePodman(); err != nil {
		t.Skip("podman not available:", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	_, err := podmanExecCombinedContext(ctx, "unused-container-missing", []string{"true"})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("command took too long: %v", elapsed)
	}
}
