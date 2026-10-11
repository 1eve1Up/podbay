package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func writeContract(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, DefaultFilename)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_memoHitMissAndParseError(t *testing.T) {
	resetContractLoadMemoForTest()
	dir := t.TempDir()
	p := writeContract(t, dir, "project: demo\nservices:\n  web:\n    image: nginx\n")

	c1, f1, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c1.Project != "demo" {
		t.Fatalf("project=%q", c1.Project)
	}
	c2, f2, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c1 != c2 {
		t.Fatal("same path+bytes should reuse the parsed Contract pointer")
	}
	if f1 != f2 {
		t.Fatalf("file %q vs %q", f1, f2)
	}
	if got := contractLoadParseCountForTest(); got != 1 {
		t.Fatalf("parse count after hit = %d", got)
	}

	if err := os.WriteFile(p, []byte("project: other\nservices:\n  web:\n    image: nginx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c3, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c3 == c1 {
		t.Fatal("byte change must not return the stale Contract")
	}
	if c3.Project != "other" {
		t.Fatalf("after edit project=%q", c3.Project)
	}
	if got := contractLoadParseCountForTest(); got != 2 {
		t.Fatalf("parse count after miss = %d", got)
	}

	if err := os.WriteFile(p, []byte("project: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(dir); err == nil {
		t.Fatal("expected yaml parse error")
	}
}

func TestLoad_sameSizeDigestMiss(t *testing.T) {
	resetContractLoadMemoForTest()
	dir := t.TempDir()
	p := writeContract(t, dir, "project: aaa\nservices:\n  web:\n    image: nginx\n")
	c1, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c1.Project != "aaa" {
		t.Fatalf("project=%q", c1.Project)
	}
	if err := os.WriteFile(p, []byte("project: bbb\nservices:\n  web:\n    image: nginx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c2, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c2 == c1 {
		t.Fatal("same-size byte change must not return the stale Contract")
	}
	if c2.Project != "bbb" {
		t.Fatalf("after same-size edit project=%q", c2.Project)
	}
	if got := contractLoadParseCountForTest(); got != 2 {
		t.Fatalf("parse count after same-size digest miss = %d", got)
	}
}

func TestLoad_differentPathsDoNotShareCache(t *testing.T) {
	resetContractLoadMemoForTest()
	body := "project: twin\nservices:\n  web:\n    image: nginx\n"
	a := writeContract(t, t.TempDir(), body)
	b := writeContract(t, t.TempDir(), body)
	ca, _, err := Load(a)
	if err != nil {
		t.Fatal(err)
	}
	cb, _, err := Load(b)
	if err != nil {
		t.Fatal(err)
	}
	if ca == cb {
		t.Fatal("same digest at different paths must not share one Contract")
	}
	if got := contractLoadParseCountForTest(); got != 2 {
		t.Fatalf("parse count = %d want 2", got)
	}
}

func TestLoad_doesNotWriteDiskCache(t *testing.T) {
	resetContractLoadMemoForTest()
	dir := t.TempDir()
	writeContract(t, dir, "project: x\nservices: {}\n")
	if _, _, err := Load(dir); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != DefaultFilename {
			t.Fatalf("unexpected file: %s", e.Name())
		}
	}
}
