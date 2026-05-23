package clijson

import (
	"errors"

	"github.com/1eve1Up/podbay/internal/deploy"
	"github.com/1eve1Up/podbay/internal/validate"
)

// Stable issue codes for deploy --json health-gate failures.
const (
	CodeDeployHealthTimeout        = "deploy_health_timeout"
	CodeDeployHealthProbeFailed    = "deploy_health_probe_failed"
	CodeDeployExternalDepUnhealthy = "deploy_external_dep_unhealthy"
	CodeDeployError                = "deploy_error"
)

// IssueFromHealthGateFailure maps a structured health failure to a deploy issue.
func IssueFromHealthGateFailure(h *deploy.HealthGateFailure) Issue {
	code := CodeDeployHealthProbeFailed
	if h.FailureClass == deploy.HealthFailureTimeout {
		code = CodeDeployHealthTimeout
	}
	if h.ExternalDep {
		code = CodeDeployExternalDepUnhealthy
	}
	return Issue{
		Level:   validate.LevelFail,
		Code:    code,
		Message: h.Error(),
		Service: h.Service,
	}
}

// IssuesFromDeployError returns structured issues for deploy failures.
func IssuesFromDeployError(err error) []Issue {
	if err == nil {
		return nil
	}
	var hg *deploy.HealthGateFailure
	if errors.As(err, &hg) && hg != nil {
		return []Issue{IssueFromHealthGateFailure(hg)}
	}
	msg := err.Error()
	return []Issue{{
		Level:   validate.LevelFail,
		Code:    CodeDeployError,
		Message: msg,
	}}
}
