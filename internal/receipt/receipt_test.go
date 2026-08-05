package receipt

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEncodeDecode_roundTrip(t *testing.T) {
	fixed := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	r := &Receipt{
		FormatVersion: CurrentFormatVersion,
		GeneratedAt:   fixed,
		ContractPath:  "/app/podbay.yaml",
		Project:       "demo",
		Profiles:      []string{"obs"},
		Services: []ServiceRecord{
			{Service: "api", ContainerName: "podbay_demo_api", ContainerID: "abc123", Image: "demo:latest"},
			{Service: "web", ContainerName: "podbay_demo_web"},
		},
	}
	data, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatalf("decode: %v\n%s", err, string(data))
	}
	if got.Project != r.Project || got.ContractPath != r.ContractPath {
		t.Fatalf("metadata mismatch %+v vs %+v", got, r)
	}
	if len(got.Services) != 2 || got.Services[0].ContainerID != "abc123" {
		t.Fatalf("services mismatch %+v", got.Services)
	}
	if got.FormatVersion != CurrentFormatVersion {
		t.Fatalf("format_version %d", got.FormatVersion)
	}
}

func TestDecode_wrongVersion(t *testing.T) {
	_, err := Decode([]byte(`{"format_version":99,"generated_at":"2026-05-08T00:00:00Z","contract_path":"/x","project":"p","services":[]}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_missingServiceName(t *testing.T) {
	r := New("/x.yaml", "p", nil, []ServiceRecord{{ContainerName: "n"}})
	err := Validate(r)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNew_setsVersionAndTime(t *testing.T) {
	r := New("/c.yaml", "proj", nil, nil)
	if r.FormatVersion != CurrentFormatVersion {
		t.Fatal(r.FormatVersion)
	}
	if r.GeneratedAt.IsZero() {
		t.Fatal("timestamp")
	}
}

func TestDecode_legacyWithoutEvidenceFields(t *testing.T) {
	raw := []byte(`{
  "format_version": 1,
  "generated_at": "2026-05-08T12:00:00Z",
  "contract_path": "/app/podbay.yaml",
  "project": "demo",
  "services": [
    {"service": "api", "container_name": "podbay_demo_api"}
  ]
}`)
	got, err := Decode(raw)
	if err != nil {
		t.Fatalf("legacy decode: %v", err)
	}
	if got.DeployID != "" || got.ContractDigest != "" || got.Status != "" {
		t.Fatalf("expected empty evidence fields, got deploy_id=%q digest=%q status=%q", got.DeployID, got.ContractDigest, got.Status)
	}
	if got.DependentsExpand || len(got.DeployServices) != 0 {
		t.Fatalf("unexpected selection fields: services=%v expand=%v", got.DeployServices, got.DependentsExpand)
	}
}

func TestEncodeDecode_withEvidenceFields(t *testing.T) {
	fixed := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	r := &Receipt{
		FormatVersion:    CurrentFormatVersion,
		GeneratedAt:      fixed,
		ContractPath:     "/app/podbay.yaml",
		Project:          "demo",
		Services:         []ServiceRecord{{Service: "api", ContainerName: "podbay_demo_api"}},
		DeployID:         "11111111-2222-3333-4444-555555555555",
		ContractDigest:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Status:           StatusOK,
		DeployServices:   []string{"api"},
		DependentsExpand: true,
	}
	data, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatalf("decode: %v\n%s", err, string(data))
	}
	if got.DeployID != r.DeployID || got.ContractDigest != r.ContractDigest || got.Status != StatusOK {
		t.Fatalf("evidence mismatch %+v", got)
	}
	if len(got.DeployServices) != 1 || got.DeployServices[0] != "api" || !got.DependentsExpand {
		t.Fatalf("selection mismatch %+v", got)
	}
}

func TestValidate_acceptsOKAndFailedStatus(t *testing.T) {
	r := New("/x.yaml", "p", nil, []ServiceRecord{{Service: "s", ContainerName: "c"}})
	for _, st := range []string{"", StatusOK, StatusFailed} {
		r.Status = st
		if err := Validate(r); err != nil {
			t.Fatalf("status %q: %v", st, err)
		}
	}
	if StatusFailed != "failed" {
		t.Fatalf("StatusFailed = %q, want failed", StatusFailed)
	}
}

func TestValidate_rejectsUnsupportedStatus(t *testing.T) {
	r := New("/x.yaml", "p", nil, []ServiceRecord{{Service: "s", ContainerName: "c"}})
	r.Status = "pending"
	if err := Validate(r); err == nil {
		t.Fatal("expected unsupported status error")
	}
}

func TestEncodeDecode_attemptFailureSummary(t *testing.T) {
	fixed := time.Date(2026, 8, 4, 18, 0, 0, 0, time.UTC)
	r := &Receipt{
		FormatVersion:  CurrentFormatVersion,
		GeneratedAt:    fixed,
		ContractPath:   "/app/podbay.yaml",
		Project:        "demo",
		Services:       []ServiceRecord{{Service: "api", ContainerName: "podbay_demo_api"}},
		DeployID:       "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		ContractDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Status:         StatusFailed,
		DeployServices: []string{"api"},
		Failure: &FailureSummary{
			Service:     "api",
			Code:        "deploy_health_timeout",
			Class:       "timeout",
			ProbeKind:   "http",
			Message:     `service "api": health check timed out`,
			ExternalDep: false,
		},
	}
	data, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatalf("decode: %v\n%s", err, string(data))
	}
	if got.Status != StatusFailed || got.Failure == nil {
		t.Fatalf("attempt mismatch %+v", got)
	}
	if got.Failure.Code != "deploy_health_timeout" || got.Failure.Class != "timeout" || got.Failure.Service != "api" {
		t.Fatalf("failure summary mismatch %+v", got.Failure)
	}
}

func TestValidate_rejectsBadFailureClass(t *testing.T) {
	r := New("/x.yaml", "p", nil, []ServiceRecord{{Service: "s", ContainerName: "c"}})
	r.Status = StatusFailed
	r.Failure = &FailureSummary{Class: "boom"}
	if err := Validate(r); err == nil {
		t.Fatal("expected bad failure.class error")
	}
}

func TestValidate_successWithoutFailureSummary(t *testing.T) {
	r := New("/x.yaml", "p", nil, []ServiceRecord{{Service: "s", ContainerName: "c"}})
	r.Status = StatusOK
	if err := Validate(r); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_rejectsBadContractDigest(t *testing.T) {
	r := New("/x.yaml", "p", nil, []ServiceRecord{{Service: "s", ContainerName: "c"}})
	r.ContractDigest = "md5:deadbeef"
	if err := Validate(r); err == nil {
		t.Fatal("expected bad digest error")
	}
}

func TestWriteAtomic_roundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "r.json")
	fixed := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	r := &Receipt{
		FormatVersion: CurrentFormatVersion,
		GeneratedAt:   fixed,
		ContractPath:  "/x/podbay.yaml",
		Project:       "p",
		Services:      []ServiceRecord{{Service: "s", ContainerName: "podbay_p_s"}},
	}
	if err := WriteAtomic(path, r); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Project != "p" || len(got.Services) != 1 {
		t.Fatalf("%+v", got)
	}
}
