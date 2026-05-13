package composefile

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveExtendsBareString(t *testing.T) {
	t.Parallel()
	const doc = `
services:
  base:
    image: docker.io/library/alpine:latest
  web:
    extends: base
    image: docker.io/library/nginx:alpine
`
	f, err := Parse([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if err := ResolveExtends(filepath.Join(t.TempDir(), "c.yml"), f); err != nil {
		t.Fatal(err)
	}
	w := f.Services["web"]
	if w.Image != "docker.io/library/nginx:alpine" {
		t.Fatalf("image: %q", w.Image)
	}
}

func TestResolveExtendsSameFile(t *testing.T) {
	t.Parallel()
	const doc = `
services:
  base:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    environment:
      FOO: from_base
  web:
    extends:
      service: base
    image: docker.io/library/nginx:alpine
    environment:
      BAR: from_web
`
	f, err := Parse([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if err := ResolveExtends("/tmp/compose.yml", f); err != nil {
		t.Fatal(err)
	}
	w := f.Services["web"]
	if w.Image != "docker.io/library/nginx:alpine" {
		t.Fatalf("image: %q", w.Image)
	}
	if len(w.Command) != 2 || w.Command[0] != "sleep" {
		t.Fatalf("command from base: %#v", w.Command)
	}
	env := map[string]string(w.Environment)
	if env["FOO"] != "from_base" || env["BAR"] != "from_web" {
		t.Fatalf("env: %#v", env)
	}
	if w.Extends != nil {
		t.Fatal("extends should be cleared")
	}
}

func TestResolveExtendsCycle(t *testing.T) {
	t.Parallel()
	const doc = `
services:
  a:
    extends:
      service: b
    image: x
  b:
    extends:
      service: a
    image: y
`
	f, err := Parse([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	err = ResolveExtends("/tmp/cycle.yml", f)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestResolveExtendsCrossFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	commonPath := filepath.Join(dir, "common.yml")
	rootPath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(commonPath, []byte(`
services:
  app:
    image: docker.io/library/alpine:latest
    command: ["sleep", "infinity"]
    environment:
      SHARED: "1"
networks:
  net_common:
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootPath, []byte(`
services:
  web:
    extends:
      file: common.yml
      service: app
    image: docker.io/library/nginx:alpine
`), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.Networks["net_common"]; !ok {
		t.Fatalf("expected net_common merged: %#v", f.Networks)
	}
	w := f.Services["web"]
	if w.Image != "docker.io/library/nginx:alpine" {
		t.Fatalf("image: %q", w.Image)
	}
	if len(w.Command) < 1 {
		t.Fatalf("command: %#v", w.Command)
	}
	if w.Environment["SHARED"] != "1" {
		t.Fatalf("env: %#v", w.Environment)
	}
}

func TestParseIgnoresXExtensionKeys(t *testing.T) {
	t.Parallel()
	const doc = `
x-logging: &default
  driver: json-file
services:
  web:
    image: nginx:alpine
    x-metadata: ignored
`
	f, err := Parse([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if f.Services["web"].Image != "nginx:alpine" {
		t.Fatalf("service: %+v", f.Services["web"])
	}
}

func TestMergeDependsOrder(t *testing.T) {
	t.Parallel()
	base := DependsOnList{{Service: "db", Condition: "service_started"}}
	over := DependsOnList{{Service: "api", Condition: "service_healthy"}, {Service: "db", Condition: "service_healthy"}}
	got := mergeDependsOn(base, over)
	want := DependsOnList{
		{Service: "db", Condition: "service_healthy"},
		{Service: "api", Condition: "service_healthy"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
