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
