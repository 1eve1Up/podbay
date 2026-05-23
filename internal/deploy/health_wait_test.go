package deploy

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewHealthGateFailure_timeoutClass(t *testing.T) {
	f := newHealthGateFailure("api", ProbeHTTP, fmt.Errorf("health check failed: timeout"))
	if f.FailureClass != HealthFailureTimeout {
		t.Fatalf("class = %v", f.FailureClass)
	}
	if f.Service != "api" || f.ProbeKind != ProbeHTTP {
		t.Fatalf("got %+v", f)
	}
}

func TestNewHealthGateFailure_probeErrorClass(t *testing.T) {
	f := newHealthGateFailure("api", ProbeExec, fmt.Errorf("exec health failed: exit 1"))
	if f.FailureClass != HealthFailureProbeError {
		t.Fatalf("class = %v", f.FailureClass)
	}
}

func TestNewExternalDepHealthFailure(t *testing.T) {
	inner := newHealthGateFailure("db", ProbeHTTP, errors.New("health check failed: HTTP 500"))
	f := newExternalDepHealthFailure("api", "db", ProbeHTTP, inner)
	if !f.ExternalDep || f.RequestedBy != "api" || f.Service != "db" {
		t.Fatalf("got %+v", f)
	}
}
