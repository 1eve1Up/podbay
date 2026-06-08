package expand

import (
	"testing"

	"github.com/1eve1Up/podbay/internal/spec"
)

func TestExpandService_substitution(t *testing.T) {
	host := map[string]string{"PORT": "8080", "HOST": "127.0.0.1", "USER_ID": "1000"}
	svc := spec.Service{
		Ports:       []string{"${HOST}:${PORT}"},
		Environment: map[string]string{"BIND": "${HOST}"},
		User:        "${USER_ID}",
		Health: &spec.HealthCheck{
			HTTP: &spec.HTTPHealth{URL: "http://${HOST}:${PORT}/health"},
		},
	}

	got := ExpandService(svc, host)

	if got.Ports[0] != "127.0.0.1:8080" {
		t.Fatalf("ports: got %q", got.Ports[0])
	}
	if got.Environment["BIND"] != "127.0.0.1" {
		t.Fatalf("environment: got %q", got.Environment["BIND"])
	}
	if got.User != "1000" {
		t.Fatalf("user: got %q", got.User)
	}
	if got.Health.HTTP.URL != "http://127.0.0.1:8080/health" {
		t.Fatalf("health url: got %q", got.Health.HTTP.URL)
	}
}

func TestExpandService_AnsibleVaultPaths(t *testing.T) {
	host := map[string]string{"SECRET": "/run/secrets/app"}
	svc := spec.Service{
		AnsibleVaultPaths: []string{"${SECRET}/vault.yml"},
		Volumes:           []string{"${SECRET}/data:/data"},
		DNS:               []string{"${SECRET}"},
		ExtraHosts:        spec.ExtraHostList{"cache:${SECRET}"},
	}

	got := ExpandService(svc, host)

	if len(got.AnsibleVaultPaths) != 1 || got.AnsibleVaultPaths[0] != "/run/secrets/app/vault.yml" {
		t.Fatalf("ansible vault paths: %#v", got.AnsibleVaultPaths)
	}
	if got.Volumes[0] != "/run/secrets/app/data:/data" {
		t.Fatalf("volumes: got %q", got.Volumes[0])
	}
	if got.DNS[0] != "/run/secrets/app" {
		t.Fatalf("dns: got %q", got.DNS[0])
	}
	if string(got.ExtraHosts[0]) != "cache:/run/secrets/app" {
		t.Fatalf("extra_hosts: got %q", got.ExtraHosts[0])
	}
}

func TestExpandService_nilHealth(t *testing.T) {
	svc := spec.Service{Ports: []string{"80:80"}}
	got := ExpandService(svc, map[string]string{})
	if got.Health != nil {
		t.Fatal("expected nil health unchanged")
	}
}
