package deploy

import (
	"errors"
	"testing"
)

func TestHealthGateFailure_Error(t *testing.T) {
	f := &HealthGateFailure{
		Service:      "api",
		ProbeKind:    ProbeHTTP,
		FailureClass: HealthFailureTimeout,
		Message:      "health check failed: timeout",
	}
	if got := f.Error(); got != `service "api": health check failed: timeout` {
		t.Fatalf("Error() = %q", got)
	}
}

func TestHealthGateFailure_ErrorExternalDep(t *testing.T) {
	f := &HealthGateFailure{
		Service:      "db",
		ProbeKind:    ProbeExec,
		FailureClass: HealthFailureProbeError,
		Message:      "exec health failed",
		ExternalDep:  true,
		RequestedBy:  "api",
	}
	want := `partial deploy service "api": waiting for dependency "db": exec health failed`
	if got := f.Error(); got != want {
		t.Fatalf("Error() = %q want %q", got, want)
	}
}

func TestHealthGateFailure_errorsAs(t *testing.T) {
	f := &HealthGateFailure{Service: "web", Message: "boom"}
	var hg *HealthGateFailure
	if !errors.As(f, &hg) {
		t.Fatal("expected errors.As to match HealthGateFailure")
	}
	if hg.Service != "web" {
		t.Fatalf("service = %q", hg.Service)
	}
}

func TestBoundMessage(t *testing.T) {
	if got := BoundMessage("short", 10); got != "short" {
		t.Fatalf("got %q", got)
	}
	long := "0123456789abcdef"
	if got := BoundMessage(long, 10); got != "0123456789…" {
		t.Fatalf("got %q", got)
	}
}
