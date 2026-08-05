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

func TestFilterEntries_status(t *testing.T) {
	entries := []ListEntry{
		{DeployID: "ok1", Status: StatusOK},
		{DeployID: "fail1", Status: StatusFailed},
		{DeployID: "legacy", Status: ""},
	}
	okOnly := FilterEntries(entries, StatusOK)
	if len(okOnly) != 2 || okOnly[0].DeployID != "ok1" || okOnly[1].DeployID != "legacy" {
		t.Fatalf("ok filter: %+v", okOnly)
	}
	failOnly := FilterEntries(entries, StatusFailed)
	if len(failOnly) != 1 || failOnly[0].DeployID != "fail1" {
		t.Fatalf("failed filter: %+v", failOnly)
	}
	all := FilterEntries(entries, "")
	if len(all) != 3 {
		t.Fatalf("unfiltered: %+v", all)
	}
}

func TestListDir_statusFilterMixed(t *testing.T) {
	dir := t.TempDir()
	write := func(name, status, id string, at time.Time) {
		r := &Receipt{
			FormatVersion: CurrentFormatVersion,
			GeneratedAt:   at,
			ContractPath:  "/c.yaml",
			Project:       "demo",
			Services:      []ServiceRecord{{Service: "web", ContainerName: "c"}},
			DeployID:      id,
			Status:        status,
		}
		if status == StatusFailed {
			r.Failure = &FailureSummary{Code: "deploy_health_timeout", Service: "web", Class: "timeout"}
		}
		raw, err := Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("ok.json", StatusOK, "ok-id", time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC))
	write("fail.json", StatusFailed, "fail-id", time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC))
	entries, _, err := ListDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	failed := FilterEntries(entries, StatusFailed)
	if len(failed) != 1 || failed[0].DeployID != "fail-id" {
		t.Fatalf("%+v", failed)
	}
}
