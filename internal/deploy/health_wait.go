package deploy

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
)

func waitServiceHealth(out io.Writer, quiet bool, service, container string, svc spec.Service, cliMax time.Duration) error {
	kind := probeKind(svc)
	if kind == "" {
		return nil
	}
	h := svc.Health
	start, interval, probe := healthTiming(svc, cliMax)
	total := start + probe
	if cliMax > 0 && total > cliMax {
		total = cliMax
	}
	if total < 5*time.Second {
		total = 5 * time.Second
	}
	if !quiet {
		_, _ = fmt.Fprintf(out, "  Waiting for service %q health (%s), up to %v ...\n", service, kind, total)
	}
	var err error
	if h.HTTP != nil && h.HTTP.URL != "" {
		err = runner.WaitHTTPHealth(h.HTTP.URL, total, h.HTTP.Insecure, interval)
	} else if h.Exec != nil && len(h.Exec.Command) > 0 {
		err = runner.WaitExecHealth(container, h.Exec.Command, interval, total)
	}
	if err != nil {
		return newHealthGateFailure(service, ProbeKind(kind), err)
	}
	if !quiet {
		_, _ = fmt.Fprintf(out, "  Service %q is healthy\n", service)
	}
	return nil
}

func newHealthGateFailure(service string, kind ProbeKind, err error) *HealthGateFailure {
	msg := BoundMessage(err.Error(), 512)
	cls := HealthFailureProbeError
	low := strings.ToLower(msg)
	if strings.Contains(low, "timeout") {
		cls = HealthFailureTimeout
	}
	return &HealthGateFailure{
		Service:      service,
		ProbeKind:    kind,
		FailureClass: cls,
		Message:      msg,
	}
}

func newExternalDepHealthFailure(requestedBy, depService string, kind ProbeKind, err error) *HealthGateFailure {
	f := newHealthGateFailure(depService, kind, err)
	f.ExternalDep = true
	f.RequestedBy = requestedBy
	return f
}

func probeKind(svc spec.Service) string {
	if svc.Health == nil {
		return ""
	}
	if svc.Health.HTTP != nil && strings.TrimSpace(svc.Health.HTTP.URL) != "" {
		return "http"
	}
	if svc.Health.Exec != nil && len(svc.Health.Exec.Command) > 0 {
		return "exec"
	}
	return ""
}
