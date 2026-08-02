package receipt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDeployID_uniqueNonEmptyUUID(t *testing.T) {
	a := NewDeployID()
	b := NewDeployID()
	if a == "" || b == "" {
		t.Fatal("empty deploy_id")
	}
	if a == b {
		t.Fatalf("expected distinct ids, got %q twice", a)
	}
	// UUID string shape: 8-4-4-4-12 hex with dashes
	if len(a) != 36 || strings.Count(a, "-") != 4 {
		t.Fatalf("unexpected uuid shape: %q", a)
	}
}

func TestContractDigestFile_knownFixture(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	content := []byte("project: demo\nservices:\n  web:\n    image: nginx:alpine\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ContractDigestFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "sha256:") {
		t.Fatalf("prefix: %q", got)
	}
	hexPart := strings.TrimPrefix(got, "sha256:")
	if len(hexPart) != 64 {
		t.Fatalf("hex length %d: %q", len(hexPart), got)
	}
	again, err := ContractDigestFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Fatalf("digest not stable: %q vs %q", got, again)
	}
}

func TestContractDigestFile_missing(t *testing.T) {
	_, err := ContractDigestFile(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
}
