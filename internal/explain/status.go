package explain

import (
	"strings"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

// ServiceStatus is runtime and health-probe state for one service (shared by text and JSON explain).
type ServiceStatus struct {
	Name       string
	Container  string
	InspectErr string
	Missing    bool
	State      string
	ExitCode   int
	StateError string
	Running    bool

	HTTPURL      string
	HTTPStatus   int
	HTTPProbeErr string

	ExecRan    bool
	ExecOutput string
	ExecErr    string
}

func collectServiceStatus(r *runner.Runner, svcName string, svc spec.Service, hostSubst map[string]string) ServiceStatus {
	svc = expandService(svc, hostSubst)
	cname := r.ContainerName(svcName)
	out := ServiceStatus{Name: svcName, Container: cname}

	st, err := runtimestate.InspectContainer(cname)
	if err != nil {
		out.InspectErr = err.Error()
		return out
	}
	if st == nil {
		out.Missing = true
		return out
	}
	out.State = st.State
	out.ExitCode = st.ExitCode
	out.StateError = st.Error
	out.Running = st.State == "running"

	if svc.Health != nil && svc.Health.HTTP != nil && svc.Health.HTTP.URL != "" {
		out.HTTPURL = svc.Health.HTTP.URL
		code, err := httpCode(svc.Health.HTTP.URL, svc.Health.HTTP.Insecure)
		out.HTTPStatus = code
		if err != nil {
			out.HTTPProbeErr = err.Error()
		}
	}
	if svc.Health != nil && svc.Health.Exec != nil && len(svc.Health.Exec.Command) > 0 {
		out.ExecRan = true
		bout, err := runner.ExecOnce(cname, svc.Health.Exec.Command)
		out.ExecOutput = strings.TrimSpace(string(bout))
		if err != nil {
			out.ExecErr = err.Error()
		}
	}
	return out
}
