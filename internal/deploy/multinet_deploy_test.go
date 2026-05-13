package deploy

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
	"github.com/1eve1Up/podbay/internal/teardown"
)

func TestDeploy_multiNetworkAttachment(t *testing.T) {
	if err := runner.EnsurePodman(); err != nil {
		t.Skip("podman not available:", err)
	}
	dir := t.TempDir()
	proj := "mnettest"
	yamlPath := filepath.Join(dir, "podbay.yaml")
	doc := `version: "1"
project: ` + proj + `
networks:
  front:
  back:
services:
  web:
    image: docker.io/library/alpine:latest
    command: ["sleep", "300"]
    networks:
      - front
  api:
    image: docker.io/library/alpine:latest
    command: ["sleep", "300"]
    networks:
      - back
`
	if err := os.WriteFile(yamlPath, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	c, file, err := spec.Load(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	pname := c.ProjectName(proj)
	r := runner.New(pname)
	defer func() {
		_ = r.RemoveService("web")
		_ = r.RemoveService("api")
		_ = r.RemoveNamedNetwork(r.NetworkPodmanName("front"))
		_ = r.RemoveNamedNetwork(r.NetworkPodmanName("back"))
	}()

	if err := Deploy(c, file, pname, Options{Quiet: true}); err != nil {
		t.Fatal(err)
	}

	webC := r.ContainerName("web")
	out, err := exec.Command("podman", "inspect", webC, "--format", "{{json .NetworkSettings.Networks}}").Output()
	if err != nil {
		t.Fatalf("inspect web: %v", err)
	}
	var nets map[string]any
	if err := json.Unmarshal(out, &nets); err != nil {
		t.Fatalf("json: %v", err)
	}
	want := r.NetworkPodmanName("front")
	if _, ok := nets[want]; !ok {
		t.Fatalf("web missing network %q, have keys %v", want, keysOf(nets))
	}
	if len(nets) != 1 {
		t.Fatalf("web expected 1 network, got %d: %v", len(nets), keysOf(nets))
	}

	apiC := r.ContainerName("api")
	out2, err := exec.Command("podman", "inspect", apiC, "--format", "{{json .NetworkSettings.Networks}}").Output()
	if err != nil {
		t.Fatalf("inspect api: %v", err)
	}
	if err := json.Unmarshal(out2, &nets); err != nil {
		t.Fatalf("json: %v", err)
	}
	wantB := r.NetworkPodmanName("back")
	if _, ok := nets[wantB]; !ok {
		t.Fatalf("api missing network %q, have keys %v", wantB, keysOf(nets))
	}
}

func keysOf(m map[string]any) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}

func TestDeploy_multiNetworkDNSNameOnBridge(t *testing.T) {
	if err := runner.EnsurePodman(); err != nil {
		t.Skip("podman not available:", err)
	}
	dir := t.TempDir()
	proj := "mnetdns"
	yamlPath := filepath.Join(dir, "podbay.yaml")
	doc := `version: "1"
project: ` + proj + `
networks:
  appnet:
services:
  one:
    image: docker.io/library/alpine:latest
    command: ["sleep", "300"]
  two:
    image: docker.io/library/alpine:latest
    command: ["sleep", "300"]
    depends_on:
      - one
`
	if err := os.WriteFile(yamlPath, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	c, file, err := spec.Load(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	pname := c.ProjectName(proj)
	r := runner.New(pname)
	defer func() {
		_ = r.RemoveService("one")
		_ = r.RemoveService("two")
		_ = r.RemoveNamedNetwork(r.NetworkPodmanName("appnet"))
	}()
	if err := Deploy(c, file, pname, Options{Quiet: true}); err != nil {
		t.Fatal(err)
	}
	two := r.ContainerName("two")
	// two should resolve service name "one" on shared user-defined network
	out, err := exec.Command("podman", "exec", two, "getent", "hosts", "one").CombinedOutput()
	if err != nil {
		t.Logf("getent output: %s", strings.TrimSpace(string(out)))
		t.Fatalf("exec getent: %v", err)
	}
	if !strings.Contains(string(out), "one") {
		t.Fatalf("expected hosts line for one, got %q", string(out))
	}
}

func TestDeploy_externalNetworkJoin(t *testing.T) {
	if err := runner.EnsurePodman(); err != nil {
		t.Skip("podman not available:", err)
	}
	extNet := "podbay_s13_extnet_test"
	_ = exec.Command("podman", "network", "rm", extNet).Run()
	createOut, err := exec.Command("podman", "network", "create", extNet).CombinedOutput()
	if err != nil {
		t.Fatalf("podman network create: %v: %s", err, strings.TrimSpace(string(createOut)))
	}
	defer func() { _ = exec.Command("podman", "network", "rm", extNet).Run() }()

	dir := t.TempDir()
	proj := "extntest"
	yamlPath := filepath.Join(dir, "podbay.yaml")
	doc := `version: "1"
project: ` + proj + `
networks:
  edge:
    external: true
    name: ` + extNet + `
services:
  web:
    image: docker.io/library/alpine:latest
    command: ["sleep", "300"]
    networks:
      - edge
`
	if err := os.WriteFile(yamlPath, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	c, file, err := spec.Load(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	pname := c.ProjectName(proj)
	r := runner.New(pname)
	defer func() {
		_ = r.RemoveService("web")
	}()

	if err := Deploy(c, file, pname, Options{Quiet: true}); err != nil {
		t.Fatal(err)
	}

	webC := r.ContainerName("web")
	inspectOut, err := exec.Command("podman", "inspect", webC, "--format", "{{json .NetworkSettings.Networks}}").Output()
	if err != nil {
		t.Fatalf("inspect web: %v", err)
	}
	var nets map[string]any
	if err := json.Unmarshal(inspectOut, &nets); err != nil {
		t.Fatalf("json: %v", err)
	}
	if _, ok := nets[extNet]; !ok {
		t.Fatalf("web missing external network %q, have keys %v", extNet, keysOf(nets))
	}

	_, tearErr := teardown.Execute(c, pname, teardown.Options{Quiet: true})
	if tearErr != nil {
		t.Fatalf("teardown: %v", tearErr)
	}
	if exec.Command("podman", "network", "exists", extNet).Run() != nil {
		t.Fatal("external network should still exist after teardown")
	}
}
