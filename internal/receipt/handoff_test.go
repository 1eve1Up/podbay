package receipt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildHandoff_failedWithLastOK(t *testing.T) {
	dir := t.TempDir()
	ok := &Receipt{
		FormatVersion:  CurrentFormatVersion,
		GeneratedAt:    time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		ContractPath:   "/c.yaml",
		Project:        "demo",
		ContractDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Status:         StatusOK,
		Services:       []ServiceRecord{{Service: "web", ContainerName: "c", Image: "web:1"}},
		DeployID:       "ok-id",
	}
	fail := &Receipt{
		FormatVersion:  CurrentFormatVersion,
		GeneratedAt:    time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		ContractPath:   "/c.yaml",
		Project:        "demo",
		ContractDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Status:         StatusFailed,
		Failure:        &FailureSummary{Code: "deploy_health_timeout", Service: "web", Class: "timeout"},
		Services:       []ServiceRecord{{Service: "web", ContainerName: "c", Image: "web:2"}},
		DeployID:       "fail-id",
	}
	okRaw, err := Encode(ok)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.json"), okRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	failPath := filepath.Join(dir, "fail.json")
	failRaw, err := Encode(fail)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(failPath, failRaw, 0o644); err != nil {
		t.Fatal(err)
	}

	h, err := BuildHandoff(fail, failPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	if h.DeployID != "fail-id" || h.Status != StatusFailed || h.Failure == nil || h.Failure.Code != "deploy_health_timeout" {
		t.Fatalf("%+v", h)
	}
	if h.LastOKPath == "" || h.NoPriorOK {
		t.Fatalf("last ok: %+v", h)
	}
	if h.Drift == nil || !*h.Drift {
		t.Fatalf("expected drift: %+v", h)
	}
	if len(h.NextActions) < 3 || !strings.Contains(h.NextActions[0], "logs") || !strings.Contains(h.NextActions[1], "explain") || !strings.Contains(h.NextActions[2], "down") {
		t.Fatalf("next_actions=%v", h.NextActions)
	}
	if !strings.Contains(h.NextActions[0], "web") {
		t.Fatalf("expected service in logs hint: %v", h.NextActions)
	}
	if strings.Contains(strings.ToLower(h.RemediationNote), "automatic remediation") == false {
		t.Fatalf("note should clarify not automatic remediation: %q", h.RemediationNote)
	}
}

func TestBuildHandoff_noPriorOK(t *testing.T) {
	dir := t.TempDir()
	fail := &Receipt{
		FormatVersion: CurrentFormatVersion,
		GeneratedAt:   time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		ContractPath:  "/c.yaml",
		Project:       "demo",
		Status:        StatusFailed,
		Failure:       &FailureSummary{Code: "deploy_health_timeout", Service: "api", Class: "timeout"},
		Services:      []ServiceRecord{{Service: "api", ContainerName: "c"}},
		DeployID:      "fail-id",
	}
	path := filepath.Join(dir, "fail.json")
	raw, err := Encode(fail)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := BuildHandoff(fail, path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !h.NoPriorOK || h.LastOKPath != "" {
		t.Fatalf("%+v", h)
	}
	if h.Drift != nil {
		t.Fatalf("false drift: %+v", h)
	}
}

func TestBuildHandoff_okCurrent(t *testing.T) {
	ok := &Receipt{
		FormatVersion: CurrentFormatVersion,
		GeneratedAt:   time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		ContractPath:  "/c.yaml",
		Project:       "demo",
		Status:        StatusOK,
		Services:      []ServiceRecord{{Service: "web", ContainerName: "c"}},
		DeployID:      "ok-id",
	}
	h, err := BuildHandoff(ok, "/tmp/ok.json", "")
	if err != nil {
		t.Fatal(err)
	}
	if h.Failure != nil || h.NoPriorOK {
		t.Fatalf("%+v", h)
	}
	if len(h.NextActions) < 1 || !strings.Contains(h.NextActions[0], "diff") {
		t.Fatalf("next_actions=%v", h.NextActions)
	}
}
