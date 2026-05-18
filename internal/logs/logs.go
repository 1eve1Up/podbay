package logs

import (
	"fmt"
	"io"
	"strings"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
)

// Entry is one service's captured log output.
type Entry struct {
	Service   string
	Container string
	Body      string
}

// ActiveServices resolves the logs target set (same rules as diff/ps/explain).
func ActiveServices(c *spec.Contract, profiles, roots []string, expandDependents bool) (map[string]spec.Service, error) {
	profileActive := c.ServicesForProfiles(profiles)
	return spec.ObservabilityActiveServices(profileActive, roots, expandDependents)
}

// PrintHuman writes podman logs for each service in stable name order.
func PrintHuman(w io.Writer, project string, active map[string]spec.Service, follow bool, tail int, since string) error {
	if follow && len(active) != 1 {
		return fmt.Errorf("logs: --follow supports only one service (resolved %d)", len(active))
	}
	r := runner.New(project)
	names := spec.ServiceNamesSorted(active)
	for i, name := range names {
		cname := r.ContainerName(name)
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "=== %s (%s) ===\n", name, cname)
		if err := runner.Logs(w, cname, follow, tail, since); err != nil {
			return fmt.Errorf("logs %q: %w", name, err)
		}
	}
	return nil
}

// CaptureBytes runs one-shot podman logs per service (no --follow).
func CaptureBytes(project string, active map[string]spec.Service, tail int, since string) ([]Entry, error) {
	r := runner.New(project)
	var out []Entry
	for _, name := range spec.ServiceNamesSorted(active) {
		cname := r.ContainerName(name)
		body, err := runner.LogsBytes(cname, tail, since)
		if err != nil {
			return nil, fmt.Errorf("logs %q: %w", name, err)
		}
		out = append(out, Entry{
			Service:   name,
			Container: cname,
			Body:      string(body),
		})
	}
	return out, nil
}

// ResolveErrorMessage normalizes ObservabilityActiveServices / deploy target errors for CLI.
func ResolveErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}
