package deploy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/1eve1Up/podbay/internal/spec"
)

func TestApplyAnsibleVaultMountsNoopWithoutPaths(t *testing.T) {
	t.Parallel()
	svc := spec.Service{
		Image:   "alpine",
		Volumes: []string{"/a:/b:ro"},
	}
	out, cleanup, err := applyAnsibleVaultMounts("/tmp", svc)
	if err != nil {
		t.Fatal(err)
	}
	cleanup()
	if len(out.AnsibleVaultPaths) != 0 {
		t.Fatalf("paths: %#v", out.AnsibleVaultPaths)
	}
	if len(out.Volumes) != 1 || out.Volumes[0] != "/a:/b:ro" {
		t.Fatalf("volumes: %#v", out.Volumes)
	}
}

func TestApplyAnsibleVaultMountsRequiresAnsibleVault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	secret := filepath.Join(dir, "sec.txt")
	// Looks like vault but will fail decrypt without password
	if err := os.WriteFile(secret, []byte("$ANSIBLE_VAULT;1.1;AES256\nxxx\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := spec.Service{
		Image:             "alpine",
		Volumes:           []string{secret + ":/run/secrets/x:ro"},
		AnsibleVaultPaths: []string{secret},
	}
	_, _, err := applyAnsibleVaultMounts(dir, svc)
	if err == nil {
		t.Fatal("expected error decrypting without valid vault password / ciphertext")
	}
}
