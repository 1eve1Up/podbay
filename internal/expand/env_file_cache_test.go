package expand

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHostSubst_envFileMemoHitMissAndProcessEnv(t *testing.T) {
	resetHostEnvFileMemoForTest()
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("FROM_FILE=a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("FROM_PROC", "first")
	m1, err := LoadHostSubst(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m1["FROM_FILE"] != "a" {
		t.Fatalf("FROM_FILE=%q", m1["FROM_FILE"])
	}
	if m1["FROM_PROC"] != "first" {
		t.Fatalf("FROM_PROC=%q", m1["FROM_PROC"])
	}
	if got := hostEnvFileParseCountForTest(); got != 1 {
		t.Fatalf("parse count after first load = %d", got)
	}

	t.Setenv("FROM_PROC", "second")
	m2, err := LoadHostSubst(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m2["FROM_FILE"] != "a" {
		t.Fatalf("cached file map lost FROM_FILE: %q", m2["FROM_FILE"])
	}
	if m2["FROM_PROC"] != "second" {
		t.Fatalf("process env after first call not visible: %q", m2["FROM_PROC"])
	}
	if got := hostEnvFileParseCountForTest(); got != 1 {
		t.Fatalf("unchanged file should be a memo hit, parse count = %d", got)
	}

	if err := os.WriteFile(envPath, []byte("FROM_FILE=b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m3, err := LoadHostSubst(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m3["FROM_FILE"] != "b" {
		t.Fatalf("stale memo after byte change: FROM_FILE=%q", m3["FROM_FILE"])
	}
	if got := hostEnvFileParseCountForTest(); got != 2 {
		t.Fatalf("byte change should re-parse, parse count = %d", got)
	}
}

func TestLoadHostSubst_sameSizeDigestMiss(t *testing.T) {
	resetHostEnvFileMemoForTest()
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("FROM_FILE=aa\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m1, err := LoadHostSubst(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m1["FROM_FILE"] != "aa" {
		t.Fatalf("FROM_FILE=%q", m1["FROM_FILE"])
	}
	if err := os.WriteFile(envPath, []byte("FROM_FILE=bb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m2, err := LoadHostSubst(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m2["FROM_FILE"] != "bb" {
		t.Fatalf("stale memo after same-size digest miss: FROM_FILE=%q", m2["FROM_FILE"])
	}
	if got := hostEnvFileParseCountForTest(); got != 2 {
		t.Fatalf("same-size byte change should re-parse, parse count = %d", got)
	}
}

func TestLoadHostSubst_fileOverridesProcessEnv(t *testing.T) {
	resetHostEnvFileMemoForTest()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SHARED=from-file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHARED", "from-proc")
	m, err := LoadHostSubst(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m["SHARED"] != "from-file" {
		t.Fatalf("file should still override process env, got %q", m["SHARED"])
	}
}

func TestLoadHostSubst_noDiskCache(t *testing.T) {
	resetHostEnvFileMemoForTest()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("K=v\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadHostSubst(dir, nil); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != ".env" && e.Name() != ".env.example" {
			t.Fatalf("unexpected file written next to contract: %s", e.Name())
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".podbay")); err == nil {
		t.Fatal("must not write .podbay cache")
	}
}
