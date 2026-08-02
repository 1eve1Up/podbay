package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveReceiptWritePath_fileMode(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "r.json")
	got, err := resolveReceiptWritePath(file, "abc-id", time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(file)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveReceiptWritePath_directoryExisting(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 8, 2, 15, 4, 5, 0, time.UTC)
	got, err := resolveReceiptWritePath(dir, "deploy-uuid", at)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Base(got)
	if base != "20260802T150405Z-deploy-uuid.json" {
		t.Fatalf("basename %q", base)
	}
	if filepath.Dir(got) != dir && filepath.Dir(got) != filepath.Clean(dir) {
		absDir, _ := filepath.Abs(dir)
		if filepath.Dir(got) != absDir {
			t.Fatalf("dir %q want under %q", got, dir)
		}
	}
}

func TestResolveReceiptWritePath_trailingSlash(t *testing.T) {
	dir := t.TempDir()
	arg := dir + string(os.PathSeparator)
	got, err := resolveReceiptWritePath(arg, "id1", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, "20260102T030405Z-id1.json") {
		t.Fatalf("got %q", got)
	}
}
