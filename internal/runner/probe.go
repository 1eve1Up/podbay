package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const defaultHTTPProbeTimeout = 5 * time.Second

// HTTPProbeOnce performs a single HTTP GET and returns the status code.
// timeout bounds curl --max-time and the process via CommandContext.
func HTTPProbeOnce(rawURL string, insecure bool, timeout time.Duration) (int, error) {
	if timeout <= 0 {
		timeout = defaultHTTPProbeTimeout
	}
	maxSec := int(timeout.Round(time.Second).Seconds())
	if maxSec < 1 {
		maxSec = 1
	}
	args := []string{"-s", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", strconv.Itoa(maxSec)}
	if insecure {
		args = append(args, "-k")
	}
	args = append(args, rawURL)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "curl", args...)
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("probe timed out after %s", timeout)
		}
		return 0, err
	}
	code, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, err
	}
	return code, nil
}

const defaultExecProbeTimeout = 5 * time.Second

// ExecOnceWithTimeout runs a single non-interactive exec with a wall-clock deadline.
func ExecOnceWithTimeout(container string, argv []string, timeout time.Duration) ([]byte, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("exec health: empty command")
	}
	if timeout <= 0 {
		timeout = defaultExecProbeTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := podmanExecCombinedContext(ctx, container, argv)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return out, fmt.Errorf("probe timed out after %s", timeout)
		}
		return out, err
	}
	return out, nil
}
