package deploy

import (
	"github.com/1eve1Up/podbay/internal/spec"
	"github.com/1eve1Up/podbay/internal/validate"
)

// PreflightOutcome runs the same validation checks as the deploy CLI preflight.
func PreflightOutcome(c *spec.Contract, contractPath string, profiles []string, deployServices []string, expandDependents bool) validate.RunOutcome {
	return validate.NewRunOutcome(c, contractPath, profiles, deployServices, expandDependents)
}
