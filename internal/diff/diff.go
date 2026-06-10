package diff

import (
	"fmt"
	"strings"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

// InspectFunc returns container state for a Podman container name, matching runtimestate.InspectContainer semantics.
type InspectFunc func(containerName string) (*runtimestate.ContainerState, error)

// Render formats a DriftResult as the human-readable diff text.
//
// The output is byte-stable: the legacy Analyze format is preserved so
// existing operator scripts and tests do not break. Renderers are the only
// place that turns DriftResult into text; Compute is the only place that
// produces DriftResult.
func Render(res DriftResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Project: %s\n", res.Project)
	fmt.Fprintf(&b, "Expected services (active): %d\n", len(res.Services))
	if len(res.Services) > 0 {
		names := make([]string, 0, len(res.Services))
		for _, s := range res.Services {
			names = append(names, s.Name)
		}
		b.WriteString(strings.Join(names, ", "))
		b.WriteString("\n\n")
	}

	for _, s := range res.Services {
		switch s.Status {
		case StatusInspectError:
			fmt.Fprintf(&b, "[%s] inspect error: %s\n", s.Name, s.Error)
		case StatusMissing:
			fmt.Fprintf(&b, "[%s] missing container %s\n", s.Name, s.ContainerName)
		case StatusWrongState:
			fmt.Fprintf(&b, "[%s] container %s state=%s exit=%d err=%s\n", s.Name, s.ContainerName, s.State, s.ExitCode, s.Error)
		case StatusOK:
			fmt.Fprintf(&b, "[%s] ok (running)\n", s.Name)
		}
	}

	if len(res.Extras) > 0 {
		b.WriteString("\nUnexpected containers (podbay.project label, not in contract):\n")
		for _, x := range res.Extras {
			fmt.Fprintf(&b, "  + %s\n", x)
		}
	}

	if !res.Drift {
		b.WriteString("\nNo drift: every expected service has a running container; no extra project containers.\n")
	} else {
		b.WriteString("\nDrift detected.\n")
	}

	return b.String()
}

// Analyze compares expected services to inspect results and extra container names (testable core).
//
// Implementation note: Analyze is now a thin wrapper around Compute + Render
// so the structured DriftResult and the human text share a single source of
// truth. The (string, bool, error) signature is preserved for callers
// (cmd/podbay diff, existing tests) so behavior is unchanged.
func Analyze(r *runner.Runner, serviceNames []string, inspect InspectFunc, extraNames []string, extraErr error) (string, bool, error) {
	res, err := Compute(r, serviceNames, inspect, extraNames, extraErr)
	if err != nil {
		return "", false, err
	}
	return Render(res), res.Drift, nil
}

// ReportResult performs the runtime setup that Report does, then returns the
// structured DriftResult. The JSON --json path consumes this so it
// can render a versioned envelope without re-running inspect.
func ReportResult(c *spec.Contract, contractPath, project string, profiles []string) (DriftResult, error) {
	return ReportContractResult(c, contractPath, project, profiles, nil, false)
}

// ReportContractResult compares the contract to Podman like ReportResult, but
// when deployRoots is non-empty uses spec.ObservabilityActiveServices so the
// expected service set matches validate/deploy partial semantics. Unexpected
// project containers are still detected against the full profile-active set so
// co-running services are not reported as extras.
func ReportContractResult(c *spec.Contract, contractPath, project string, profiles []string, deployRoots []string, expandDependents bool) (DriftResult, error) {
	if err := runner.EnsurePodman(); err != nil {
		return DriftResult{}, err
	}

	names, extrasSeed, err := contractDiffNameSets(c, profiles, deployRoots, expandDependents)
	if err != nil {
		return DriftResult{}, err
	}

	r := runner.New(project)
	projectNames, err := runtimestate.ProjectContainerNames(r)
	if err != nil {
		return DriftResult{}, err
	}
	extras := runtimestate.ExtraContainerNamesWithProjectList(r, extrasSeed, projectNames)

	containerNames := make([]string, len(names))
	for i, name := range names {
		containerNames[i] = r.ContainerName(name)
	}
	states, err := runtimestate.InspectContainers(containerNames)
	if err != nil {
		return DriftResult{}, err
	}
	return ComputeWithContainerStates(r, names, states, extras, nil)
}

// contractDiffNameSets returns sorted service names to inspect vs sorted names
// used as the "expected" set for detecting extra project containers (always
// full profile-active). Unit-tested without Podman.
func contractDiffNameSets(c *spec.Contract, profiles []string, deployRoots []string, expandDependents bool) (observe []string, extrasSeed []string, err error) {
	profileActive := c.ServicesForProfiles(profiles)
	if len(c.Services) > 0 && len(profileActive) == 0 {
		return nil, nil, fmt.Errorf("no services selected for this profile set (check --profile)")
	}
	if len(profileActive) == 0 && len(c.Services) == 0 {
		return nil, nil, fmt.Errorf("no services defined")
	}
	active, err := spec.ObservabilityActiveServices(profileActive, deployRoots, expandDependents)
	if err != nil {
		return nil, nil, err
	}
	return spec.ServiceNamesSorted(active), spec.ServiceNamesSorted(profileActive), nil
}

// Report compares the contract's active or partial-observability services to Podman.
func Report(c *spec.Contract, contractPath, project string, profiles []string, deployRoots []string, expandDependents bool) (string, bool, error) {
	res, err := ReportContractResult(c, contractPath, project, profiles, deployRoots, expandDependents)
	if err != nil {
		return "", false, err
	}
	return Render(res), res.Drift, nil
}
