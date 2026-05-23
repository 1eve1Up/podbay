package deploy

import (
	"fmt"
	"strings"
)

// ProbeKind identifies the health probe mechanism that failed.
type ProbeKind string

const (
	ProbeHTTP ProbeKind = "http"
	ProbeExec ProbeKind = "exec"
)

// HealthFailureClass distinguishes timeout from probe errors.
type HealthFailureClass string

const (
	HealthFailureTimeout    HealthFailureClass = "timeout"
	HealthFailureProbeError HealthFailureClass = "probe_error"
)

// HealthGateFailure is a structured deploy health-gate outcome for agents and clijson.
type HealthGateFailure struct {
	Service      string
	ProbeKind    ProbeKind
	FailureClass HealthFailureClass
	Message      string
	// ExternalDep is true when the failure is for a dependency outside the partial deploy set.
	ExternalDep bool
	// RequestedBy is the partial-deploy service that triggered an external dependency health wait.
	RequestedBy string
}

func (e *HealthGateFailure) Error() string {
	if e == nil {
		return "health gate failure"
	}
	svc := strings.TrimSpace(e.Service)
	if e.ExternalDep && strings.TrimSpace(e.RequestedBy) != "" {
		return fmt.Sprintf("partial deploy service %q: waiting for dependency %q: %s", e.RequestedBy, svc, e.detail())
	}
	if svc != "" {
		return fmt.Sprintf("service %q: %s", svc, e.detail())
	}
	return e.detail()
}

func (e *HealthGateFailure) detail() string {
	msg := strings.TrimSpace(e.Message)
	if msg != "" {
		return msg
	}
	switch e.FailureClass {
	case HealthFailureTimeout:
		return "health check timed out"
	default:
		return "health check failed"
	}
}

// BoundMessage caps issue message length for JSON output.
func BoundMessage(msg string, max int) string {
	msg = strings.TrimSpace(msg)
	if max <= 0 || len(msg) <= max {
		return msg
	}
	return msg[:max] + "…"
}
