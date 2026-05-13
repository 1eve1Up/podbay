package composeimport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/1eve1Up/podbay/internal/composefile"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestMarshalContractRoundTripSpecLoad(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
services:
  web:
    image: nginx:alpine
    depends_on:
      - api
  api:
    image: api:latest
`
	f, err := composefile.Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ToContract(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := MarshalContract(c)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, spec.DefaultFilename)
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := spec.Load(p)
	if err != nil {
		t.Fatalf("spec.Load: %v\n--- yaml ---\n%s", err, string(raw))
	}
	if len(loaded.Services) != 2 {
		t.Fatalf("services: %d", len(loaded.Services))
	}
	if loaded.Services["web"].DependsOn[0].Service != "api" {
		t.Fatalf("depends_on: %#v", loaded.Services["web"].DependsOn)
	}
	if got := loaded.Services["api"].RedeployPeers; len(got) != 1 || got[0] != "web" {
		t.Fatalf("api.dependents inverse: want [web] got %#v", got)
	}
}

func TestMarshalContractNil(t *testing.T) {
	t.Parallel()
	_, err := MarshalContract(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMarshalContractRoundTripAnsibleVaultPaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	secPath := filepath.Join(dir, "vaulted.txt")
	if err := os.WriteFile(secPath, []byte("$ANSIBLE_VAULT;1.1;AES256\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	yamlDoc := `
secrets:
  s1:
    file: vaulted.txt
services:
  a:
    image: alpine:latest
    secrets:
      - s1
`
	f, err := composefile.Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ToContract(f, dir)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := MarshalContract(c)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, spec.DefaultFilename)
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := spec.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	paths := loaded.Services["a"].AnsibleVaultPaths
	if len(paths) != 1 || paths[0] != secPath {
		t.Fatalf("round-trip ansible_vault_paths: %#v", paths)
	}
}
