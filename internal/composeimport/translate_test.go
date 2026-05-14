package composeimport

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/composefile"
)

func TestToContractDependsOnAndVolumes(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
services:
  web:
    image: nginx:alpine
    depends_on:
      - api
    volumes:
      - data:/usr/share/nginx/html:ro
  api:
    image: api:latest
volumes:
  data:
`
	f, err := composefile.Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ToContract(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	web := c.Services["web"]
	if len(web.DependsOn) != 1 || web.DependsOn[0].Service != "api" {
		t.Fatalf("deps: %#v", web.DependsOn)
	}
	if len(web.Volumes) != 1 {
		t.Fatalf("volumes: %#v", web.Volumes)
	}
	if _, ok := c.Volumes["data"]; !ok {
		t.Fatalf("named volume data missing: %#v", c.Volumes)
	}
	api := c.Services["api"]
	if want := []string{"web"}; !reflect.DeepEqual(api.RedeployPeers, want) {
		t.Fatalf("api.dependents: want %v got %v", want, api.RedeployPeers)
	}
}

func TestToContractAllowsDefaultNetworksBlock(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
networks:
  default:
services:
  a:
    image: x
`
	f, err := composefile.Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ToContract(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Networks["default"]; !ok {
		t.Fatalf("expected default network in contract: %#v", c.Networks)
	}
}

func TestToContractSplitNetworks(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
networks:
  front:
  back:
services:
  web:
    image: nginx:alpine
    networks:
      - front
  api:
    image: alpine:latest
    command: ["true"]
    networks:
      - back
`
	f, err := composefile.Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ToContract(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if w, ok := c.Services["web"]; !ok || len(w.Networks) != 1 || w.Networks[0] != "front" {
		t.Fatalf("web networks: %#v", w.Networks)
	}
	if a, ok := c.Services["api"]; !ok || len(a.Networks) != 1 || a.Networks[0] != "back" {
		t.Fatalf("api networks: %#v", a.Networks)
	}
	if len(c.Networks) != 2 {
		t.Fatalf("contract networks: %#v", c.Networks)
	}
}

func TestToContractExternalNetwork(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
networks:
  edge:
    external: true
services:
  a:
    image: x
    networks:
      - edge
`
	f, err := composefile.Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ToContract(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	en, ok := c.Networks["edge"]
	if !ok || !en.External {
		t.Fatalf("edge network: %#v", c.Networks)
	}
	if len(c.Services["a"].Networks) != 1 || c.Services["a"].Networks[0] != "edge" {
		t.Fatalf("service networks: %#v", c.Services["a"].Networks)
	}
}

func TestToContractExternalNetworkExplicitName(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
networks:
  edge:
    external:
      name: real_net
services:
  a:
    image: x
    networks:
      - edge
`
	f, err := composefile.Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ToContract(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	en := c.Networks["edge"]
	if !en.External || en.Name != "real_net" {
		t.Fatalf("edge: %#v", en)
	}
}

func TestToContractBuildRequiresImage(t *testing.T) {
	t.Parallel()
	f := &composefile.File{
		Services: map[string]composefile.ServiceSpec{
			"a": {
				Build: &composefile.BuildSpec{Context: "."},
			},
		},
	}
	_, err := ToContract(f, t.TempDir())
	if err == nil {
		t.Fatal("expected error for build without image")
	}
}

func TestToContractHealthExec(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		in       []string
		wantCmd  []string
		wantSkip bool
	}{
		{name: "CMD stripped", in: []string{"CMD", "true"}, wantCmd: []string{"true"}},
		{name: "CMD-SHELL wraps", in: []string{"CMD-SHELL", "curl -f http://x/"},
			wantCmd: []string{"sh", "-c", "curl -f http://x/"}},
		{name: "NONE disables", in: []string{"NONE"}, wantSkip: true},
		{name: "bare argv preserved", in: []string{"true"}, wantCmd: []string{"true"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &composefile.File{
				Services: map[string]composefile.ServiceSpec{
					"a": {
						Image: "x",
						Healthcheck: &composefile.HealthcheckSpec{
							Test:     tc.in,
							Interval: "5s",
						},
					},
				},
			}
			c, err := ToContract(f, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			h := c.Services["a"].Health
			if tc.wantSkip {
				if h != nil {
					t.Fatalf("expected nil Health for %v, got %#v", tc.in, h)
				}
				return
			}
			if h == nil || h.Exec == nil || !reflect.DeepEqual(h.Exec.Command, tc.wantCmd) {
				t.Fatalf("input %v: got %#v want %v", tc.in, h, tc.wantCmd)
			}
		})
	}
}

func TestToContractPassesExpose(t *testing.T) {
	t.Parallel()
	f := &composefile.File{
		Services: map[string]composefile.ServiceSpec{
			"api": {
				Image:  "x",
				Expose: []string{"8000", "9090"},
			},
		},
	}
	c, err := ToContract(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Services["api"].Expose; !reflect.DeepEqual(got, []string{"8000", "9090"}) {
		t.Fatalf("expose: got %#v", got)
	}
}

func TestToContractUnknownDependency(t *testing.T) {
	t.Parallel()
	f := &composefile.File{
		Services: map[string]composefile.ServiceSpec{
			"web": {
				Image:     "x",
				DependsOn: []composefile.DependsOnEntry{{Service: "missing", Condition: "service_started"}},
			},
		},
	}
	_, err := ToContract(f, t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestToContractImplicitNamedVolume(t *testing.T) {
	t.Parallel()
	// No top-level volumes: — still declare named vol in contract
	const yamlDoc = `
services:
  a:
    image: x
    volumes:
      - mydata:/data
`
	f, err := composefile.Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ToContract(f, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Volumes["mydata"]; !ok {
		t.Fatalf("volumes: %#v", c.Volumes)
	}
}

func TestToContractConfigSecretFileMounts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "app.json")
	secPath := filepath.Join(dir, "db.txt")
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secPath, []byte(`secret`), 0o600); err != nil {
		t.Fatal(err)
	}
	relCfg := filepath.Base(cfgPath)
	relSec := filepath.Base(secPath)
	yamlDoc := `
configs:
  app_conf:
    file: ` + relCfg + `
secrets:
  db_pw:
    file: ` + relSec + `
services:
  web:
    image: nginx:alpine
    configs:
      - source: app_conf
        target: /etc/app/config.json
    secrets:
      - db_pw
`
	f, err := composefile.Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ToContract(f, dir)
	if err != nil {
		t.Fatal(err)
	}
	w := c.Services["web"]
	var foundCfg, foundSec bool
	for _, v := range w.Volumes {
		if strings.HasPrefix(v, cfgPath) && strings.Contains(v, ":/etc/app/config.json:ro") {
			foundCfg = true
		}
		if strings.HasPrefix(v, secPath) && strings.Contains(v, ":/run/secrets/db_pw:ro") {
			foundSec = true
		}
	}
	if !foundCfg || !foundSec {
		t.Fatalf("volumes: %#v", w.Volumes)
	}
	if len(w.AnsibleVaultPaths) != 0 {
		t.Fatalf("ansible paths: %#v", w.AnsibleVaultPaths)
	}
}

func TestToContractAnsibleVaultPathsTagged(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	secPath := filepath.Join(dir, "vaulted.txt")
	payload := ansibleVaultPrefix + ";1.1;AES256\n0000\n"
	if err := os.WriteFile(secPath, []byte(payload), 0o600); err != nil {
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
	paths := c.Services["a"].AnsibleVaultPaths
	if len(paths) != 1 || paths[0] != secPath {
		t.Fatalf("ansible_vault_paths: %#v", paths)
	}
}
