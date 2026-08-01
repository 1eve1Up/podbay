// Package ps builds profile-aware container rows for podbay ps (contract vocabulary vs Podman).
package ps

import (
	"fmt"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

// InspectFunc matches runtimestate.InspectContainer semantics (nil state means missing).
type InspectFunc func(containerName string) (*runtimestate.ContainerState, error)

// Row is one active service's expected container and runtime snapshot.
type Row struct {
	Service   string
	Container string
	State     string
	ExitCode  int
	Error     string
	Image     string
	Missing   bool
}

// ListRows returns one row per service in the resolved observability set (profile-active
// by default, or spec.ObservabilityActiveServices when deployRoots is non-empty), in sorted order.
// Prefer ListRowsWithContainerStates with runtimestate.InspectContainers for the runtime path.
func ListRows(c *spec.Contract, project string, profiles []string, deployRoots []string, expandDependents bool, inspect InspectFunc) ([]Row, error) {
	r, names, err := resolveActiveServiceNames(c, project, profiles, deployRoots, expandDependents)
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(names))
	for _, svcName := range names {
		cname := r.ContainerName(svcName)
		st, err := inspect(cname)
		rows = append(rows, rowFromInspect(svcName, cname, st, err))
	}
	return rows, nil
}

// ListRowsWithContainerStates builds rows from a prefetched container state map
// (for example runtimestate.InspectContainers). Names missing from states, or mapped
// to nil, are treated as missing containers — same as InspectFunc returning (nil, nil).
func ListRowsWithContainerStates(c *spec.Contract, project string, profiles []string, deployRoots []string, expandDependents bool, states map[string]*runtimestate.ContainerState) ([]Row, error) {
	r, names, err := resolveActiveServiceNames(c, project, profiles, deployRoots, expandDependents)
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(names))
	for _, svcName := range names {
		cname := r.ContainerName(svcName)
		var st *runtimestate.ContainerState
		if states != nil {
			st = states[cname]
		}
		rows = append(rows, rowFromInspect(svcName, cname, st, nil))
	}
	return rows, nil
}

// ActiveContainerNames returns sorted service names and their expected container names
// for the resolved observability set. Useful for one-shot InspectContainers calls.
func ActiveContainerNames(c *spec.Contract, project string, profiles []string, deployRoots []string, expandDependents bool) (serviceNames []string, containerNames []string, err error) {
	r, names, err := resolveActiveServiceNames(c, project, profiles, deployRoots, expandDependents)
	if err != nil {
		return nil, nil, err
	}
	containers := make([]string, len(names))
	for i, svcName := range names {
		containers[i] = r.ContainerName(svcName)
	}
	return names, containers, nil
}

func resolveActiveServiceNames(c *spec.Contract, project string, profiles []string, deployRoots []string, expandDependents bool) (*runner.Runner, []string, error) {
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
	return runner.New(project), spec.ServiceNamesSorted(active), nil
}

func rowFromInspect(svcName, cname string, st *runtimestate.ContainerState, inspectErr error) Row {
	if inspectErr != nil {
		return Row{
			Service:   svcName,
			Container: cname,
			State:     "error",
			Error:     inspectErr.Error(),
		}
	}
	if st == nil {
		return Row{
			Service:   svcName,
			Container: cname,
			Missing:   true,
			State:     "missing",
		}
	}
	return Row{
		Service:   svcName,
		Container: cname,
		State:     st.State,
		ExitCode:  st.ExitCode,
		Error:     st.Error,
		Image:     st.Image,
	}
}
