package clijson

import (
	"testing"

	"github.com/1eve1Up/podbay/internal/deploy"
)

func TestDeployOutcome_healthTimeout(t *testing.T) {
	err := &deploy.HealthGateFailure{
		Service:      "api",
		ProbeKind:    deploy.ProbeHTTP,
		FailureClass: deploy.HealthFailureTimeout,
		Message:      "health check failed: timeout",
	}
	d := DeployOutcome("/c/p.yaml", "x", nil, nil, "", err, false)
	if d.Status != StatusFailed || len(d.Issues) != 1 {
		t.Fatalf("%+v", d)
	}
	if d.Issues[0].Code != CodeDeployHealthTimeout || d.Issues[0].Service != "api" {
		t.Fatalf("issue %+v", d.Issues[0])
	}
}

func TestDeployOutcome_externalDepUnhealthy(t *testing.T) {
	err := &deploy.HealthGateFailure{
		Service:      "db",
		ProbeKind:    deploy.ProbeExec,
		FailureClass: deploy.HealthFailureProbeError,
		Message:      "exec health failed",
		ExternalDep:  true,
		RequestedBy:  "api",
	}
	d := DeployOutcome("/c/p.yaml", "x", nil, []string{"api"}, "", err, true)
	if d.Issues[0].Code != CodeDeployExternalDepUnhealthy || d.Issues[0].Service != "db" {
		t.Fatalf("issue %+v", d.Issues[0])
	}
	if !d.DependentsExpand {
		t.Fatalf("dependents_expand = %v", d.DependentsExpand)
	}
}

func TestDeployOutcome_genericDeployError(t *testing.T) {
	d := DeployOutcome("/c/p.yaml", "x", nil, nil, "", errFake("boom"), false)
	if d.Issues[0].Code != CodeDeployError {
		t.Fatalf("code = %q", d.Issues[0].Code)
	}
}
