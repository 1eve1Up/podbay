package explain

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

// Report is human-oriented system state vs contract.
// When deployRoots is non-empty, only services in spec.ObservabilityActiveServices are listed;
// use expandDependents like validate/deploy. When exactly one service appears in that resolved set,
// a dependency summary is printed (same as historical single-focus explain).
// When deployRoots is empty, all profile-active services are listed (no dependency section unless one service).
func Report(c *spec.Contract, contractPath, project string, profiles []string, deployRoots []string, expandDependents bool) (string, error) {
	if err := runner.EnsurePodman(); err != nil {
		return "", err
	}
	hostSubst, err := expand.LoadHostSubst(filepath.Dir(contractPath), c.HostEnvFiles)
	if err != nil {
		return "", err
	}

	r := runner.New(project)
	var b strings.Builder

	profileActive := c.ServicesForProfiles(profiles)
	if len(c.Services) > 0 && len(profileActive) == 0 {
		return "", fmt.Errorf("no services selected for this profile set (check --profile)")
	}
	if len(profileActive) == 0 && len(c.Services) == 0 {
		return "", fmt.Errorf("no services defined")
	}

	obs, err := spec.ObservabilityActiveServices(profileActive, deployRoots, expandDependents)
	if err != nil {
		return "", err
	}

	iterate := spec.ServiceNamesSorted(profileActive)
	if len(deployRoots) > 0 {
		iterate = spec.ServiceNamesSorted(obs)
	}

	b.WriteString(fmt.Sprintf("Project: %s\n", project))
	b.WriteString(fmt.Sprintf("Active services: %d\n", len(iterate)))
	if len(iterate) > 0 {
		b.WriteString(strings.Join(iterate, ", "))
		b.WriteString("\n\n")
	}

	if len(deployRoots) > 0 && len(iterate) == 1 {
		b.WriteString(DependencySummary(profileActive, iterate[0]))
		b.WriteString("\n")
	}

	for _, svcName := range iterate {
		st := collectServiceStatus(r, svcName, profileActive[svcName], hostSubst)
		writeServiceDetail(&b, st)
	}

	extrasSeed := spec.ServiceNamesSorted(profileActive)
	extra, err := runtimestate.ExtraContainerNames(r, extrasSeed)
	if err == nil && len(extra) > 0 {
		b.WriteString("Unexpected containers (same project label, unknown service):\n")
		for _, x := range extra {
			b.WriteString("  - " + x + "\n")
		}
	}

	return b.String(), nil
}

func writeServiceDetail(b *strings.Builder, st ServiceStatus) {
	svcName := st.Name
	if st.InspectErr != "" {
		b.WriteString(fmt.Sprintf("[%s] cannot inspect: %s\n", svcName, st.InspectErr))
		b.WriteString("\n")
		return
	}
	if st.Missing {
		b.WriteString(fmt.Sprintf("[%s] not running (no container %s)\n", svcName, st.Container))
		b.WriteString("\n")
		return
	}
	b.WriteString(fmt.Sprintf("[%s] state=%s\n", svcName, st.State))
	if st.State != "running" {
		b.WriteString(fmt.Sprintf("  exit: %d  error: %s\n", st.ExitCode, st.StateError))
	}
	if st.HTTPURL != "" {
		if st.HTTPProbeErr != "" {
			b.WriteString(fmt.Sprintf("  health URL %s: error: %s\n", st.HTTPURL, st.HTTPProbeErr))
		} else {
			b.WriteString(fmt.Sprintf("  health URL %s: HTTP %d\n", st.HTTPURL, st.HTTPStatus))
		}
	}
	if st.ExecRan {
		if st.ExecErr != "" {
			b.WriteString(fmt.Sprintf("  health exec: error: %s\n", st.ExecErr))
		} else {
			b.WriteString(fmt.Sprintf("  health exec: ok (%s)\n", st.ExecOutput))
		}
	}
	b.WriteString("\n")
}

func expandService(svc spec.Service, host map[string]string) spec.Service {
	svc.Ports = expandStrs(svc.Ports, host)
	svc.Volumes = expandStrs(svc.Volumes, host)
	svc.Environment = expandMap(svc.Environment, host)
	svc.User = expand.String(svc.User, host)
	svc.DNS = expandStrs(svc.DNS, host)
	svc.ExtraHosts = spec.ExtraHostList(expandStrs([]string(svc.ExtraHosts), host))
	if svc.Health != nil && svc.Health.HTTP != nil {
		svc.Health.HTTP.URL = expand.String(svc.Health.HTTP.URL, host)
	}
	return svc
}

func expandStrs(in []string, m map[string]string) []string {
	if len(in) == 0 {
		return in
	}
	o := make([]string, len(in))
	for i, s := range in {
		o[i] = expand.String(s, m)
	}
	return o
}

func expandMap(in map[string]string, m map[string]string) map[string]string {
	if len(in) == 0 {
		return in
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = expand.String(v, m)
	}
	return out
}

func httpCode(url string, insecure bool) (int, error) {
	args := []string{"-s", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "5"}
	if insecure {
		args = append(args, "-k")
	}
	args = append(args, url)
	cmd := exec.Command("curl", args...)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var code int
	_, err = fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &code)
	return code, err
}
