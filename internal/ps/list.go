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
func ListRows(c *spec.Contract, project string, profiles []string, deployRoots []string, expandDependents bool, inspect InspectFunc) ([]Row, error) {
	profileActive := c.ServicesForProfiles(profiles)
	if len(c.Services) > 0 && len(profileActive) == 0 {
		return nil, fmt.Errorf("no services selected for this profile set (check --profile)")
	}
	if len(profileActive) == 0 && len(c.Services) == 0 {
		return nil, fmt.Errorf("no services defined")
	}

	active, err := spec.ObservabilityActiveServices(profileActive, deployRoots, expandDependents)
	if err != nil {
		return nil, err
	}

	r := runner.New(project)
	names := spec.ServiceNamesSorted(active)
	rows := make([]Row, 0, len(names))
	for _, svcName := range names {
		cname := r.ContainerName(svcName)
		st, err := inspect(cname)
		if err != nil {
			rows = append(rows, Row{
				Service:   svcName,
				Container: cname,
				State:     "error",
				Error:     err.Error(),
			})
			continue
		}
		if st == nil {
			rows = append(rows, Row{
				Service:   svcName,
				Container: cname,
				Missing:   true,
				State:     "missing",
			})
			continue
		}
		rows = append(rows, Row{
			Service:   svcName,
			Container: cname,
			State:     st.State,
			ExitCode:  st.ExitCode,
			Error:     st.Error,
			Image:     st.Image,
		})
	}
	return rows, nil
}
