package deploy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestDeploy_noReceiptOnTopologicalError(t *testing.T) {
	if err := runner.EnsurePodman(); err != nil {
		t.Skip("podman not available:", err)
	}
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "podbay.yaml")
	cycle := `version: "1"
project: cycletest
services:
  a:
    image: docker.io/library/alpine:latest
    depends_on:
      - b
  b:
    image: docker.io/library/alpine:latest
    depends_on:
      - a
`
	if err := os.WriteFile(yamlPath, []byte(cycle), 0o644); err != nil {
		t.Fatal(err)
	}
	c, file, err := spec.Load(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(dir, "receipt.json")
	err = Deploy(c, file, c.ProjectName("cycletest"), Options{
		Quiet:       true,
		ReceiptPath: receiptPath,
	})
	if err == nil {
		t.Fatal("expected cycle error from deploy")
	}
	if _, err := os.Stat(receiptPath); !os.IsNotExist(err) {
		t.Fatalf("receipt must not exist, stat err=%v", err)
	}
}
