package composefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover_matchOrder(t *testing.T) {
	dir := t.TempDir()
	// Lower-priority file present first; higher-priority should still win when both exist.
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "compose.yaml")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDiscover_firstExistingInOrder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "docker-compose.yaml")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDiscover_missing(t *testing.T) {
	dir := t.TempDir()
	_, err := Discover(dir, "")
	if err == nil {
		t.Fatal("expected not found")
	}
	if CodeOrEmpty(err) != CodeComposeDiscoveryNotFound {
		t.Fatalf("code=%q err=%v", CodeOrEmpty(err), err)
	}
}

func TestDiscover_explicitWins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(dir, "my-compose.yml")
	if err := os.WriteFile(custom, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(dir, custom)
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

func TestDiscover_explicitMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := Discover(dir, filepath.Join(dir, "nope.yml"))
	if err == nil {
		t.Fatal("expected missing explicit path error")
	}
	if CodeOrEmpty(err) != CodeImportComposeFileNotFound {
		t.Fatalf("code=%q err=%v", CodeOrEmpty(err), err)
	}
}

func TestDiscover_skipsDirectoriesNamedLikeCompose(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "compose.yaml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "compose.yml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "compose.yml")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
