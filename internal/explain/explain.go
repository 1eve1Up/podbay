package explain

import (
	"fmt"
	"strings"

	"github.com/1eve1Up/podbay/internal/orientation"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

// Report is human-oriented system state vs contract.
// When deployRoots is non-empty, only services in spec.ObservabilityActiveServices are listed;
// use expandDependents like validate/deploy. When exactly one service appears in that resolved set,
// a dependency summary is printed (same as historical single-focus explain).
// When deployRoots is empty, all profile-active services are listed (no dependency section unless one service).
func Report(c *spec.Contract, contractPath, project string, profiles []string, deployRoots []string, expandDependents bool, noProbes bool) (string, error) {
	if err := runner.EnsurePodman(); err != nil {
		return "", err
	}
	sess, err := prepareExplain(c, contractPath, project, profiles, deployRoots, expandDependents, noProbes)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Project: %s\n", project))
	b.WriteString(fmt.Sprintf("Active services: %d\n", len(sess.iterate)))
	if len(sess.iterate) > 0 {
		b.WriteString(strings.Join(sess.iterate, ", "))
		b.WriteString("\n")
	}
	if sess.noProbes {
		b.WriteString("Probes: skipped (--no-probes)\n")
	}
	if orient, oerr := buildLiveOrientation(c, contractPath, profiles, deployRoots, expandDependents, sess.statuses); oerr == nil {
		writeOrientationPreamble(&b, orient)
	}
	b.WriteString("\n")

	if len(deployRoots) > 0 && len(sess.iterate) == 1 {
		b.WriteString(DependencySummary(sess.profileActive, sess.iterate[0]))
		b.WriteString("\n")
	}

	for _, st := range sess.statuses {
		writeServiceDetail(&b, st)
	}

	extrasSeed := spec.ServiceNamesSorted(sess.profileActive)
	extra, err := runtimestate.ExtraContainerNames(sess.r, extrasSeed)
	if err == nil && len(extra) > 0 {
		b.WriteString("Unexpected containers (same project label, unknown service):\n")
		for _, x := range extra {
			b.WriteString("  - " + x + "\n")
		}
	}

	return b.String(), nil
}

func writeOrientationPreamble(b *strings.Builder, doc *orientation.Document) {
	if doc == nil {
		return
	}
	b.WriteString("Orientation: structured next-steps (not diagnosis)\n")
	for _, a := range doc.NextActions {
		b.WriteString("  next: " + a + "\n")
	}
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
