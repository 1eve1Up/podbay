package composefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverDockerfile_matchOrder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := DiscoverDockerfile(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "Dockerfile")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDiscoverDockerfile_firstExistingInOrder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := DiscoverDockerfile(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "dockerfile")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDiscoverDockerfile_missing(t *testing.T) {
	dir := t.TempDir()
	_, err := DiscoverDockerfile(dir, "")
	if err == nil {
		t.Fatal("expected not found")
	}
	if CodeOrEmpty(err) != CodeDockerfileDiscoveryNotFound {
		t.Fatalf("code=%q err=%v", CodeOrEmpty(err), err)
	}
}

func TestDiscoverDockerfile_explicitWins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(dir, "Dockerfile.prod")
	if err := os.WriteFile(custom, []byte("FROM alpine:3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := DiscoverDockerfile(dir, custom)
	if err != nil {
		t.Fatal(err)
	}
	absCustom, err := filepath.Abs(custom)
	if err != nil {
		t.Fatal(err)
	}
	if got != absCustom {
		t.Fatalf("got %q want %q", got, absCustom)
	}
}

func TestDiscoverDockerfile_explicitMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := DiscoverDockerfile(dir, filepath.Join(dir, "nope.Dockerfile"))
	if err == nil {
		t.Fatal("expected missing explicit path error")
	}
	if CodeOrEmpty(err) != CodeDockerfileDiscoveryNotFound {
		t.Fatalf("code=%q err=%v", CodeOrEmpty(err), err)
	}
}

func TestDiscoverDockerfile_skipsDirectoriesNamedLikeDockerfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "Dockerfile"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := DiscoverDockerfile(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "dockerfile")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDiscoverDockerfile_doesNotPickCompose(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := DiscoverDockerfile(dir, "")
	if err == nil {
		t.Fatal("expected not found when only Compose is present")
	}
	if CodeOrEmpty(err) != CodeDockerfileDiscoveryNotFound {
		t.Fatalf("code=%q err=%v", CodeOrEmpty(err), err)
	}
}
