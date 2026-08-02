package receipt

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListDir_newestFirstSkipsJunk(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, at time.Time, deployID string) {
		r := &Receipt{
			FormatVersion:  CurrentFormatVersion,
			GeneratedAt:    at,
			ContractPath:   "/c.yaml",
			Project:        "demo",
			Services:       []ServiceRecord{{Service: "web", ContainerName: "c"}},
			DeployID:       deployID,
			ContractDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Status:         StatusOK,
		}
		raw, err := Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("old.json", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "old-id")
	write("new.json", time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC), "new-id")
	if err := os.WriteFile(filepath.Join(dir, "junk.json"), []byte(`{"not":"a receipt"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, skipped, err := ListDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries=%d skipped=%v", len(entries), skipped)
	}
	if entries[0].DeployID != "new-id" || entries[1].DeployID != "old-id" {
		t.Fatalf("order: %+v", entries)
	}
	if len(skipped) != 1 {
		t.Fatalf("skipped=%v", skipped)
	}
}
