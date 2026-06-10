package diff

import (
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
)

// ServiceStatus is a per-service drift outcome on a DriftResult entry.
//
// Values are stable strings: callers (including future JSON renderers) may
// compare them as plain text. Add new values only via additive changes.
type ServiceStatus string

const (
	// StatusOK means the expected container exists and is running.
	StatusOK ServiceStatus = "ok"
	// StatusMissing means the expected container was not found.
	StatusMissing ServiceStatus = "missing"
	// StatusWrongState means the container exists but is not running.
	StatusWrongState ServiceStatus = "wrong_state"
	// StatusInspectError means inspecting the expected container failed.
	StatusInspectError ServiceStatus = "inspect_error"
)

// ServiceDrift records one expected service's runtime state vs the contract.
//
// Optional fields (State, ExitCode, Error) are populated when the runtime
// supplies them; renderers must tolerate empty zero values.
type ServiceDrift struct {
	Name          string
	ContainerName string
	Status        ServiceStatus
	State         string
	ExitCode      int
	Error         string
}

// DriftResult is the structured outcome of comparing a contract's active
// services to the runtime. It is intended as the single source of truth
// that text and JSON renderers can both consume.
//
// Drift is true when any service has Status != StatusOK or when Extras
// is non-empty.
type DriftResult struct {
	Project  string
	Services []ServiceDrift
	Extras   []string
	Drift    bool
}

// Compute builds a DriftResult by inspecting each expected service and
// recording extras.
//
// extrasErr, when non-nil, short-circuits the computation and is returned
// to the caller (mirrors Analyze semantics).
func Compute(r *runner.Runner, serviceNames []string, inspect InspectFunc, extras []string, extrasErr error) (DriftResult, error) {
	if extrasErr != nil {
		return DriftResult{}, extrasErr
	}

	res := DriftResult{Project: r.Project}
	if len(extras) > 0 {
		res.Extras = append(make([]string, 0, len(extras)), extras...)
	}

	for _, name := range serviceNames {
		cname := r.ContainerName(name)
		st, err := inspect(cname)
		sd := serviceDriftForContainer(name, cname, st, err)
		if sd.Status != StatusOK {
			res.Drift = true
		}
		res.Services = append(res.Services, sd)
	}

	if len(res.Extras) > 0 {
		res.Drift = true
	}
	return res, nil
}

// ComputeWithContainerStates builds a DriftResult from a pre-fetched container
// state map (for example runtimestate.InspectContainers). Names missing from
// states are treated as missing containers.
func ComputeWithContainerStates(r *runner.Runner, serviceNames []string, states map[string]*runtimestate.ContainerState, extras []string, extrasErr error) (DriftResult, error) {
	if extrasErr != nil {
		return DriftResult{}, extrasErr
	}

	res := DriftResult{Project: r.Project}
	if len(extras) > 0 {
		res.Extras = append(make([]string, 0, len(extras)), extras...)
	}

	for _, name := range serviceNames {
		cname := r.ContainerName(name)
		var st *runtimestate.ContainerState
		if states != nil {
			st = states[cname]
		}
		sd := serviceDriftForContainer(name, cname, st, nil)
		if sd.Status != StatusOK {
			res.Drift = true
		}
		res.Services = append(res.Services, sd)
	}

	if len(res.Extras) > 0 {
		res.Drift = true
	}
	return res, nil
}

func serviceDriftForContainer(name, cname string, st *runtimestate.ContainerState, inspectErr error) ServiceDrift {
	sd := ServiceDrift{Name: name, ContainerName: cname}
	switch {
	case inspectErr != nil:
		sd.Status = StatusInspectError
		sd.Error = inspectErr.Error()
	case st == nil:
		sd.Status = StatusMissing
	case st.State != "running":
		sd.Status = StatusWrongState
		sd.State = st.State
		sd.ExitCode = st.ExitCode
		sd.Error = st.Error
	default:
		sd.Status = StatusOK
		sd.State = st.State
	}
	return sd
}
