// Package vault provides optional Ansible Vault integration for deploy-time file materialization.
package vault

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DecryptToTemp runs ansible-vault view on hostPath and writes decrypted content to a new temp file.
// cleanup removes the temp file (call after the container is started; best-effort on error paths).
// Uses PATH and the same env as the process (e.g. ANSIBLE_VAULT_PASSWORD_FILE).
func DecryptToTemp(hostPath string) (tmpPath string, cleanup func(), err error) {
	noop := func() {}
	exe, err := exec.LookPath("ansible-vault")
	if err != nil {
		return "", noop, fmt.Errorf("ansible-vault not in PATH: %w", err)
	}
	cmd := exec.Command(exe, "view", hostPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return "", noop, fmt.Errorf("ansible-vault view %q: %w (%s)", hostPath, err, msg)
		}
		return "", noop, fmt.Errorf("ansible-vault view %q: %w", hostPath, err)
	}
	f, err := os.CreateTemp("", "podbay-ansible-vault-*")
	if err != nil {
		return "", noop, err
	}
	tmpPath = f.Name()
	cleanup = func() { _ = os.Remove(tmpPath) }
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		cleanup()
		return "", noop, err
	}
	if _, err := f.Write(out); err != nil {
		_ = f.Close()
		cleanup()
		return "", noop, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", noop, err
	}
	return tmpPath, cleanup, nil
}
