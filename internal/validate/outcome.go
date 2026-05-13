package validate

import (
	"fmt"
	"io"

	"github.com/1eve1Up/podbay/internal/spec"
)

// RunOutcome is the structured output of a validation run (human or JSON renderers consume it).
type RunOutcome struct {
	Results []Result
}

// NewRunOutcome runs validation and wraps results for structured consumers.
// deployRoots selects a partial deploy subgraph when non-empty; expandDependents widens that subgraph; see Run.
func NewRunOutcome(c *spec.Contract, contractPath string, profiles []string, deployRoots []string, expandDependents bool) RunOutcome {
	return RunOutcome{Results: Run(c, contractPath, profiles, deployRoots, expandDependents)}
}

// HasFailure reports whether any fail-level check did not pass.
func (o RunOutcome) HasFailure() bool {
	for _, r := range o.Results {
		if !r.OK && r.Level == LevelFail {
			return true
		}
	}
	return false
}

// PrintText writes operator-style lines (✔ / ⚠ / ✖) matching historical validate output.
func (o RunOutcome) PrintText(w io.Writer) error {
	for _, r := range o.Results {
		symbol := "✔"
		if !r.OK {
			if r.Level == LevelWarn {
				symbol = "⚠"
			} else {
				symbol = "✖"
			}
		}
		if _, err := fmt.Fprintf(w, "%s %s\n", symbol, r.Message); err != nil {
			return err
		}
	}
	return nil
}
