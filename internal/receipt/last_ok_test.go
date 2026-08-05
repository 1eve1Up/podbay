package receipt

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLastOK_mixedDirectoryNewestOK(t *testing.T) {
	dir := t.TempDir()
	writeReceipt := func(name, status, id string, at time.Time) {
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
	writeReceipt("older-ok.json", StatusOK, "older-ok", time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC))
	writeReceipt("legacy.json", "", "legacy-ok", time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC))
	writeReceipt("fail.json", StatusFailed, "fail-id", time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC))
	writeReceipt("newer-ok.json", StatusOK, "newer-ok", time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC))

	got, err := LastOK(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.DeployID != "newer-ok" {
		t.Fatalf("LastOK DeployID=%q want newer-ok (failed must not win; newer ok beats legacy)", got.DeployID)
	}
	if got.Status != StatusOK {
		t.Fatalf("Status=%q", got.Status)
	}
	if got.Path == "" {
		t.Fatal("Path empty")
	}
}

func TestLastOK_emptyStore(t *testing.T) {
	dir := t.TempDir()
	got, err := LastOK(dir)
	if !errors.Is(err, ErrNoLastOK) {
		t.Fatalf("err=%v want ErrNoLastOK", err)
	}
	if got != nil {
		t.Fatalf("got=%+v want nil", got)
	}
}

func TestLastOK_failedOnly(t *testing.T) {
	dir := t.TempDir()
	r := &Receipt{
		FormatVersion: CurrentFormatVersion,
		GeneratedAt:   time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC),
		ContractPath:  "/c.yaml",
		Project:       "demo",
		Services:      []ServiceRecord{{Service: "web", ContainerName: "c"}},
		DeployID:      "fail-id",
		Status:        StatusFailed,
		Failure:       &FailureSummary{Code: "deploy_health_timeout", Service: "web", Class: "timeout"},
	}
	raw, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fail.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LastOK(dir)
	if !errors.Is(err, ErrNoLastOK) {
		t.Fatalf("err=%v want ErrNoLastOK", err)
	}
	if got != nil {
		t.Fatalf("got=%+v want nil", got)
	}
}

func TestLastOKFromEntries_preservesNewestFirst(t *testing.T) {
	entries := []ListEntry{
		{DeployID: "fail", Status: StatusFailed, Path: "/f.json"},
		{DeployID: "ok-new", Status: StatusOK, Path: "/ok-new.json"},
		{DeployID: "ok-old", Status: StatusOK, Path: "/ok-old.json"},
	}
	got, err := LastOKFromEntries(entries)
	if err != nil {
		t.Fatal(err)
	}
	if got.DeployID != "ok-new" {
		t.Fatalf("got=%+v", got)
	}
}
