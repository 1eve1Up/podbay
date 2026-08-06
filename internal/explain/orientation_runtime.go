package explain

import (
	"github.com/1eve1Up/podbay/internal/orientation"
)

// RuntimeRowsFromStatus maps explain ServiceStatus facts into orientation runtime rows.
// It does not probe or inspect — callers pass already-collected statuses.
func RuntimeRowsFromStatus(statuses []ServiceStatus) []orientation.RuntimeService {
	out := make([]orientation.RuntimeService, 0, len(statuses))
	for _, st := range statuses {
		out = append(out, runtimeRowFromStatus(st))
	}
	return out
}

func runtimeRowFromStatus(st ServiceStatus) orientation.RuntimeService {
	row := orientation.RuntimeService{
		Name:    st.Name,
		Missing: st.Missing || st.InspectErr != "",
		Running: st.Running,
		State:   st.State,
	}
	if st.InspectErr != "" {
		row.Missing = true
		row.Running = false
	}
	if h := healthyFromStatus(st); h != nil {
		row.Healthy = h
	}
	return row
}

// healthyFromStatus derives an optional healthy flag from probe facts.
// Returns nil when no health probes ran (no HTTP URL and no exec).
func healthyFromStatus(st ServiceStatus) *bool {
	if st.Missing || st.InspectErr != "" {
		v := false
		return &v
	}
	hasHTTP := st.HTTPURL != ""
	hasExec := st.ExecRan
	if !hasHTTP && !hasExec {
		if !st.Running {
			v := false
			return &v
		}
		return nil
	}
	ok := st.Running
	if hasHTTP {
		ok = ok && st.HTTPProbeErr == "" && st.HTTPStatus >= 200 && st.HTTPStatus < 400
	}
	if hasExec {
		ok = ok && st.ExecErr == ""
	}
	return &ok
}
