package composefile

import (
	"reflect"
	"testing"
)

func TestParseDependsOnList(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
services:
  web:
    image: nginx:alpine
    depends_on:
      - api
      - db
  api:
    image: api:latest
  db:
    image: postgres:15
`
	f, err := Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	web := f.Services["web"]
	if len(web.DependsOn) != 2 {
		t.Fatalf("depends_on len: got %d want 2", len(web.DependsOn))
	}
	if web.DependsOn[0].Service != "api" || web.DependsOn[0].Condition != "service_started" {
		t.Errorf("first dep: %+v", web.DependsOn[0])
	}
	if web.DependsOn[1].Service != "db" {
		t.Errorf("second dep: %+v", web.DependsOn[1])
	}
}

func TestParseDependsOnMap(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
services:
  web:
    image: nginx:alpine
    depends_on:
      api:
        condition: service_healthy
      db:
        condition: service_started
  api:
    image: api:latest
    healthcheck:
      test: ["CMD", "true"]
  db:
    image: postgres:15
`
	f, err := Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	web := f.Services["web"]
	want := []DependsOnEntry{
		{Service: "api", Condition: "service_healthy"},
		{Service: "db", Condition: "service_started"},
	}
	if !reflect.DeepEqual(want, []DependsOnEntry(web.DependsOn)) {
		t.Fatalf("depends_on: got %#v want %#v", web.DependsOn, want)
	}
}

func TestParseBuildString(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
services:
  app:
    build: .
    image: app:local
`
	f, err := Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	b := f.Services["app"].Build
	if b == nil || b.Context != "." {
		t.Fatalf("build: %+v", b)
	}
}

func TestParseBuildObject(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
services:
  app:
    build:
      context: ./backend
      dockerfile: Dockerfile.dev
    image: app:local
`
	f, err := Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	b := f.Services["app"].Build
	if b == nil || b.Context != "./backend" || b.Dockerfile != "Dockerfile.dev" {
		t.Fatalf("build: %+v", b)
	}
}

func TestParseEnvironmentForms(t *testing.T) {
	t.Parallel()
	const mapForm = `
services:
  a:
    image: x
    environment:
      FOO: bar
      BAZ: "2"
`
	f, err := Parse([]byte(mapForm))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"FOO": "bar", "BAZ": "2"}
	if !reflect.DeepEqual(want, map[string]string(f.Services["a"].Environment)) {
		t.Fatalf("env map: got %#v", f.Services["a"].Environment)
	}

	const listForm = `
services:
  b:
    image: x
    environment:
      - FOO=bar
      - BAZ=2
`
	f2, err := Parse([]byte(listForm))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, map[string]string(f2.Services["b"].Environment)) {
		t.Fatalf("env list: got %#v", f2.Services["b"].Environment)
	}
}

func TestParseHealthcheckAndPorts(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
services:
  api:
    image: api
    ports:
      - "8080:80"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://127.0.0.1/"]
      interval: 5s
      timeout: 3s
      retries: 3
      start_period: 10s
`
	f, err := Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	s := f.Services["api"]
	if len(s.Ports) != 1 || s.Ports[0] != "8080:80" {
		t.Fatalf("ports: %#v", s.Ports)
	}
	if s.Healthcheck == nil || len(s.Healthcheck.Test) != 4 {
		t.Fatalf("healthcheck: %#v", s.Healthcheck)
	}
	if s.Healthcheck.Interval != "5s" || s.Healthcheck.StartPeriod != "10s" {
		t.Fatalf("healthcheck timing: %+v", s.Healthcheck)
	}
}

func TestParsePortsLongFormNormalized(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
services:
  api:
    image: api
    ports:
      - target: 80
        published: 8080
`
	f, err := Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	got := f.Services["api"].Ports
	if len(got) != 1 || got[0] != "8080:80" {
		t.Fatalf("ports: %#v", got)
	}
}

func TestParsePortsLongFormHostIPUDP(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
services:
  api:
    image: api
    ports:
      - target: 53
        published: 5353
        host_ip: 127.0.0.1
        protocol: udp
`
	f, err := Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	got := f.Services["api"].Ports
	want := "127.0.0.1:5353:53/udp"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("ports: %#v want %q", got, want)
	}
}

func TestParseEmptyServices(t *testing.T) {
	t.Parallel()
	f, err := Parse([]byte(`version: "3"`))
	if err != nil {
		t.Fatal(err)
	}
	if f.Services == nil || len(f.Services) != 0 {
		t.Fatalf("services: %#v", f.Services)
	}
}

func TestParseTopLevelConfigsSecrets(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
configs:
  app_conf:
    file: ./config/app.json
secrets:
  db_pw:
    file: ./secrets/db.txt
services:
  web:
    image: nginx:alpine
`
	f, err := Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	if f.Configs["app_conf"].File != "./config/app.json" {
		t.Fatalf("configs app_conf: %#v", f.Configs["app_conf"])
	}
	if f.Secrets["db_pw"].File != "./secrets/db.txt" {
		t.Fatalf("secrets db_pw: %#v", f.Secrets["db_pw"])
	}
	if f.Configs["app_conf"].External {
		t.Fatal("expected external false")
	}
}

func TestParseServiceConfigsSecretsRefs(t *testing.T) {
	t.Parallel()
	const yamlDoc = `
configs:
  httpd_conf:
    file: ./httpd.conf
secrets:
  db_password:
    file: ./db_password.txt
services:
  web:
    image: nginx:alpine
    configs:
      - source: httpd_conf
        target: /etc/httpd/conf/httpd.conf
    secrets:
      - db_password
      - source: db_password
        target: /run/secrets/custom_db
`
	f, err := Parse([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	w := f.Services["web"]
	if len(w.Configs) != 1 || w.Configs[0].Source != "httpd_conf" || w.Configs[0].Target != "/etc/httpd/conf/httpd.conf" {
		t.Fatalf("configs: %#v", w.Configs)
	}
	if len(w.Secrets) != 2 {
		t.Fatalf("secrets len: %#v", w.Secrets)
	}
	if w.Secrets[0].Source != "db_password" || w.Secrets[0].Target != "" {
		t.Fatalf("secret short: %#v", w.Secrets[0])
	}
	if w.Secrets[1].Target != "/run/secrets/custom_db" {
		t.Fatalf("secret long: %#v", w.Secrets[1])
	}
}
