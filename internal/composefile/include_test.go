package composefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_includeMergesServices(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	common := filepath.Join(dir, "common.yml")
	root := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(common, []byte(`
services:
  api:
    image: docker.io/library/alpine:latest
    command: ["sleep", "300"]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte(`
include:
  - ./common.yml
services:
  web:
    image: docker.io/library/nginx:alpine
`), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.Services["web"]; !ok {
		t.Fatal("missing web from root")
	}
	if _, ok := f.Services["api"]; !ok {
		t.Fatal("missing api from included file")
	}
}

func TestLoad_includeLaterIncludeOverwritesService(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a := filepath.Join(dir, "a.yml")
	b := filepath.Join(dir, "b.yml")
	root := filepath.Join(dir, "root.yml")
	if err := os.WriteFile(a, []byte(`
services:
  x:
    image: docker.io/library/alpine:latest
    command: ["sleep", "1"]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(`
services:
  x:
    image: docker.io/library/nginx:alpine
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte(`
include:
  - ./a.yml
  - ./b.yml
`), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	sx, ok := f.Services["x"]
	if !ok {
		t.Fatal("missing x")
	}
	if want := "docker.io/library/nginx:alpine"; sx.Image != want {
		t.Fatalf("later include should win: got image %q want %q", sx.Image, want)
	}
}

func TestLoad_includeRootOverwritesIncluded(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	inc := filepath.Join(dir, "inc.yml")
	root := filepath.Join(dir, "root.yml")
	if err := os.WriteFile(inc, []byte(`
services:
  web:
    image: docker.io/library/alpine:latest
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte(`
include:
  - ./inc.yml
services:
  web:
    image: docker.io/library/nginx:alpine
`), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if f.Services["web"].Image != "docker.io/library/nginx:alpine" {
		t.Fatalf("root should win: %q", f.Services["web"].Image)
	}
}

func TestLoad_includeCycle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a := filepath.Join(dir, "a.yml")
	b := filepath.Join(dir, "b.yml")
	if err := os.WriteFile(a, []byte(`
include:
  - ./b.yml
services:
  x:
    image: docker.io/library/alpine:latest
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(`
include:
  - ./a.yml
services:
  y:
    image: docker.io/library/alpine:latest
`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(a)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestLoad_includeURLError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root := filepath.Join(dir, "root.yml")
	if err := os.WriteFile(root, []byte(`
include:
  - https://example.com/compose.yml
`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	if err == nil {
		t.Fatal("expected URL error")
	}
}

func TestLoad_includePathEscape(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root := filepath.Join(dir, "root.yml")
	if err := os.WriteFile(root, []byte(`
include:
  - ../outside.yml
`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	if err == nil {
		t.Fatal("expected escape error")
	}
}

func TestLoad_includeMapPathForm(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	common := filepath.Join(dir, "common.yml")
	root := filepath.Join(dir, "root.yml")
	if err := os.WriteFile(common, []byte(`
services:
  api:
    image: docker.io/library/alpine:latest
    command: ["sleep", "1"]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte(`
include:
  - path: ./common.yml
services: {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.Services["api"]; !ok {
		t.Fatal("expected api from map-path include")
	}
}
