package composeimport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/composefile"
	"github.com/1eve1Up/podbay/internal/spec"
	"github.com/1eve1Up/podbay/internal/validate"
)

// Sample compose used for import → validate integration (no published ports → no port/health warnings as fails).
const integrationCompose = `
services:
  web:
    image: docker.io/library/nginx:alpine
    depends_on:
      - api
  api:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
`

func TestImportComposePipelineValidate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(integrationCompose), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := composefile.Load(composePath)
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
	contractPath := filepath.Join(dir, spec.DefaultFilename)
	if err := os.WriteFile(contractPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := spec.Load(contractPath)
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}
	out := validate.NewRunOutcome(loaded, contractPath, nil, nil, false)
	if out.HasFailure() {
		for _, r := range out.Results {
			if r.Level == validate.LevelFail {
				t.Logf("validate: %s", r.Message)
			}
		}
		t.Fatal("validate reported failures")
	}
}

func TestImportComposeIncludePipelineValidate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	frag := filepath.Join(dir, "fragment.yml")
	root := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(frag, []byte(`
services:
  api:
    image: docker.io/library/alpine:latest
    command: ["sleep", "300"]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte(`
include:
  - ./fragment.yml
services:
  web:
    image: docker.io/library/nginx:alpine
`), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := composefile.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.Services["api"]; !ok {
		t.Fatal("expected api from included fragment")
	}
	c, err := ToContract(f, dir)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := MarshalContract(c)
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(dir, spec.DefaultFilename)
	if err := os.WriteFile(contractPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := spec.Load(contractPath)
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}
	out := validate.NewRunOutcome(loaded, contractPath, nil, nil, false)
	if out.HasFailure() {
		for _, r := range out.Results {
			if r.Level == validate.LevelFail {
				t.Logf("validate: %s", r.Message)
			}
		}
		t.Fatal("validate reported failures")
	}
}

const integrationComposeNetworksLongPorts = `
networks:
  app_net:
services:
  web:
    image: docker.io/library/nginx:alpine
    networks:
      - app_net
    ports:
      - target: 80
        published: 8080
  api:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    networks:
      - app_net
`

func TestImportComposeNetworksAndLongPortsValidate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(integrationComposeNetworksLongPorts), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := composefile.Load(composePath)
	if err != nil {
		t.Fatal(err)
	}
	c, err := ToContract(f, dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Networks["app_net"]; !ok {
		t.Fatalf("contract networks: %#v", c.Networks)
	}
	wPorts := c.Services["web"].Ports
	if len(wPorts) != 1 || wPorts[0] != "8080:80" {
		t.Fatalf("web ports: %#v", wPorts)
	}
	raw, err := MarshalContract(c)
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(dir, spec.DefaultFilename)
	if err := os.WriteFile(contractPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := spec.Load(contractPath)
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}
	out := validate.NewRunOutcome(loaded, contractPath, nil, nil, false)
	if out.HasFailure() {
		t.Fatal("validate reported failures")
	}
}

func TestImportComposeValidateJSONEnvelope(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(integrationCompose), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := composefile.Load(composePath)
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
	contractPath := filepath.Join(dir, spec.DefaultFilename)
	if err := os.WriteFile(contractPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := spec.Load(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	out := validate.NewRunOutcome(loaded, contractPath, nil, nil, false)
	proj := loaded.ProjectName(filepath.Base(dir))
	doc := clijson.FromValidate(contractPath, proj, nil, nil, out.Results, false)
	jsonBytes, err := clijson.MarshalIndent(doc)
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(jsonBytes, &top); err != nil {
		t.Fatal(err)
	}
	if _, ok := top["format_version"]; !ok {
		t.Fatalf("missing format_version: %s", string(jsonBytes))
	}
}

func TestImportComposeConfigsSecretsValidate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "httpd.conf"), []byte("# conf\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "db_password.txt"), []byte("s3cr3t\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	const doc = `
configs:
  httpd_conf:
    file: ./httpd.conf
secrets:
  db_password:
    file: ./db_password.txt
services:
  web:
    image: docker.io/library/nginx:alpine
    configs:
      - source: httpd_conf
        target: /etc/httpd/conf/httpd.conf
    secrets:
      - db_password
  api:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
`
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := composefile.Load(composePath)
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
	contractPath := filepath.Join(dir, spec.DefaultFilename)
	if err := os.WriteFile(contractPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := spec.Load(contractPath)
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}
	out := validate.NewRunOutcome(loaded, contractPath, nil, nil, false)
	if out.HasFailure() {
		t.Fatal("validate reported failures")
	}
	w := loaded.Services["web"]
	if len(w.Volumes) < 2 {
		t.Fatalf("expected config+secret mounts on web, got %#v", w.Volumes)
	}
}

func TestImportComposeRejectsAbsoluteConfigFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(`
configs:
  bad:
    file: /etc/passwd
services:
  web:
    image: docker.io/library/nginx:alpine
    configs:
      - source: bad
`), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := composefile.Load(composePath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ToContract(f, dir)
	if err == nil {
		t.Fatal("expected error for absolute config file path")
	}
	if !strings.Contains(err.Error(), "absolute host paths are not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportComposeRejectsParentEscapeSecretFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(`
secrets:
  bad:
    file: ../../../etc/shadow
services:
  web:
    image: docker.io/library/nginx:alpine
    secrets:
      - bad
`), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := composefile.Load(composePath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ToContract(f, dir)
	if err == nil {
		t.Fatal("expected error for secret path escape")
	}
	if !strings.Contains(err.Error(), "escapes compose directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportComposeExtendsValidate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	commonPath := filepath.Join(dir, "common.yml")
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(commonPath, []byte(`
services:
  svc_base:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    environment:
      FROM_COMMON: "1"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(composePath, []byte(`
services:
  web:
    extends:
      file: common.yml
      service: svc_base
    image: docker.io/library/nginx:alpine
`), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := composefile.Load(composePath)
	if err != nil {
		t.Fatal(err)
	}
	c, err := ToContract(f, dir)
	if err != nil {
		t.Fatal(err)
	}
	w := c.Services["web"]
	if w.Image != "docker.io/library/nginx:alpine" {
		t.Fatalf("image: %q", w.Image)
	}
	if w.Environment["FROM_COMMON"] != "1" {
		t.Fatalf("env: %#v", w.Environment)
	}
	raw, err := MarshalContract(c)
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(dir, spec.DefaultFilename)
	if err := os.WriteFile(contractPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := spec.Load(contractPath)
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}
	out := validate.NewRunOutcome(loaded, contractPath, nil, nil, false)
	if out.HasFailure() {
		t.Fatal("validate reported failures")
	}
}

func TestImportComposeExternalValidate(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
networks:
  edge:
    external: true
    name: some_preexisting_net
services:
  web:
    image: docker.io/library/nginx:alpine
    networks:
      - edge
`
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(yamlDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := composefile.Load(composePath)
	if err != nil {
		t.Fatal(err)
	}
	c, err := ToContract(f, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Networks["edge"].External || c.Networks["edge"].Name != "some_preexisting_net" {
		t.Fatalf("network edge: %#v", c.Networks["edge"])
	}
	raw, err := MarshalContract(c)
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(dir, spec.DefaultFilename)
	if err := os.WriteFile(contractPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := spec.Load(contractPath)
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}
	out := validate.NewRunOutcome(loaded, contractPath, nil, nil, false)
	if out.HasFailure() {
		for _, r := range out.Results {
			if r.Level == validate.LevelFail {
				t.Logf("validate: %s", r.Message)
			}
		}
		t.Fatal("validate reported failures")
	}
}
