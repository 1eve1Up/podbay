package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractPathAndDeployServices_withFileFlag(t *testing.T) {
	p, svc, err := contractPathAndDeployServices("/proj/podbay.yaml", []string{"web", "api"}, "/def")
	if err != nil {
		t.Fatal(err)
	}
	if p != "/proj/podbay.yaml" || len(svc) != 2 || svc[0] != "web" || svc[1] != "api" {
		t.Fatalf("got path=%q svc=%v", p, svc)
	}
	p2, svc2, err := contractPathAndDeployServices("/proj/podbay.yaml", nil, "/def")
	if err != nil || len(svc2) != 0 {
		t.Fatalf("got svc2=%v err=%v", svc2, err)
	}
	if p2 != "/proj/podbay.yaml" {
		t.Fatalf("path %q", p2)
	}
}

func TestContractPathAndDeployServices_positional(t *testing.T) {
	def := filepath.Join(t.TempDir(), "podbay.yaml")
	p, svc, err := contractPathAndDeployServices("", []string{"/x/podbay.yaml"}, def)
	if err != nil || p != "/x/podbay.yaml" || svc != nil {
		t.Fatalf("got %q %v %v", p, svc, err)
	}
	p, svc, err = contractPathAndDeployServices("", []string{"/x/podbay.yaml", "web"}, def)
	if err != nil || p != "/x/podbay.yaml" || len(svc) != 1 || svc[0] != "web" {
		t.Fatalf("got %q %v", p, svc)
	}
	p, svc, err = contractPathAndDeployServices("", nil, def)
	if err != nil || p != def || len(svc) != 0 {
		t.Fatalf("default got %q %v", p, svc)
	}
}

func TestContractPathAndDeployServices_singleTokenServiceUsesDefaultContract(t *testing.T) {
	dir := t.TempDir()
	def := filepath.Join(dir, "podbay.yaml")
	yaml := `version: "1"
services:
  web:
    image: docker.io/library/nginx:alpine
`
	if err := os.WriteFile(def, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	p, svc, err := contractPathAndDeployServices("", []string{"web"}, def)
	if err != nil {
		t.Fatal(err)
	}
	if p != def || len(svc) != 1 || svc[0] != "web" {
		t.Fatalf("got path=%q svc=%v", p, svc)
	}
}

func TestContractPathAndDeployServices_singleTokenUnknownServiceStillPath(t *testing.T) {
	dir := t.TempDir()
	def := filepath.Join(dir, "podbay.yaml")
	yaml := `version: "1"
services:
  web:
    image: docker.io/library/nginx:alpine
`
	if err := os.WriteFile(def, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	p, svc, err := contractPathAndDeployServices("", []string{"missing"}, def)
	if err != nil {
		t.Fatal(err)
	}
	if p != "missing" || svc != nil {
		t.Fatalf("expected path-only fallback got path=%q svc=%v", p, svc)
	}
}

func TestValidateDependentsFlagRegistered(t *testing.T) {
	var file string
	cmd := validateCmd(&file, "/def")
	if cmd.Flags().Lookup("dependents") == nil {
		t.Fatal("expected --dependents on validate")
	}
}

func TestDeployDependentsFlagRegistered(t *testing.T) {
	var file string
	cmd := deployCmd(&file, "/def")
	if cmd.Flags().Lookup("dependents") == nil {
		t.Fatal("expected --dependents on deploy")
	}
}

func TestValidateHelpMentionsDependents(t *testing.T) {
	var file string
	cmd := validateCmd(&file, "/def")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Help(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"--dependents", "transitive"} {
		if !strings.Contains(out, want) {
			t.Errorf("validate help missing %q\n%s", want, out)
		}
	}
}

func TestDeployHelpMentionsDependents(t *testing.T) {
	var file string
	cmd := deployCmd(&file, "/def")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Help(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"--dependents", "transitive"} {
		if !strings.Contains(out, want) {
			t.Errorf("deploy help missing %q\n%s", want, out)
		}
	}
}
