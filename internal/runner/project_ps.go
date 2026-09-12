package runner

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ListProjectContainers returns container names that carry label podbay.project=<Runner.Project>.
// A Podman ps race on a stale/missing container (exit 125 with "no such container") is treated
// as an empty list so already-down teardown can proceed to network cleanup.
func (r *Runner) ListProjectContainers() ([]string, error) {
	out, err := r.ProjectPs("{{.Names}}")
	if err != nil {
		return nil, err
	}
	return parsePsNameLines(out), nil
}

// ProjectPs runs `podman ps -a --filter label=podbay.project=<project> --format <format>`.
// Stderr is captured for diagnostics. Stale-container races are empty success; other
// failures (including a stopped Podman machine) still return an error that includes stderr.
func (r *Runner) ProjectPs(format string) ([]byte, error) {
	out, stderr, err := r.projectPsOnce(format)
	if err == nil {
		return out, nil
	}
	if isBenignPodmanListError(out, stderr, err) {
		return nil, nil
	}
	if isPodmanExit(err, 125) {
		out2, stderr2, err2 := r.projectPsOnce(format)
		if err2 == nil {
			return out2, nil
		}
		if isBenignPodmanListError(out2, stderr2, err2) {
			return nil, nil
		}
		return nil, wrapPodmanPsError(err2, stderr2)
	}
	return nil, wrapPodmanPsError(err, stderr)
}

func (r *Runner) projectPsOnce(format string) (stdout, stderr []byte, err error) {
	cmd := r.podman("ps", "-a", "--filter", "label=podbay.project="+r.Project, "--format", format)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	out, err := cmd.Output()
	return out, errBuf.Bytes(), err
}

func wrapPodmanPsError(err error, stderr []byte) error {
	trim := bytes.TrimSpace(stderr)
	if len(trim) > 0 {
		return fmt.Errorf("podman ps: %w: %s", err, trim)
	}
	return fmt.Errorf("podman ps: %w", err)
}

func isPodmanExit(err error, code int) bool {
	var ee *exec.ExitError
	return errors.As(err, &ee) && ee.ExitCode() == code
}

// IsPodmanUnavailable reports whether err is a stopped machine / missing socket,
// not a listing or template failure. Used by teardown so `down` stays idempotent
// when the engine is off; observability commands still fail closed.
func IsPodmanUnavailable(err error) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(err.Error())
	return strings.Contains(low, "cannot connect to podman") ||
		strings.Contains(low, "unable to connect to podman socket")
}

func isBenignPodmanListError(stdout, stderr []byte, err error) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(string(stdout) + " " + string(stderr) + " " + err.Error())
	return strings.Contains(low, "no such container") ||
		strings.Contains(low, "no container with name") ||
		strings.Contains(low, "does not exist in database")
}

func parsePsNameLines(out []byte) []string {
	var names []string
	for _, line := range strings.Split(string(bytes.TrimSpace(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		for _, name := range strings.Split(line, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}
