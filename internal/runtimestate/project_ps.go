package runtimestate

import (
	"bytes"
	"strings"

	"github.com/1eve1Up/podbay/internal/runner"
)

// ProjectContainerNames returns all Podman container names for the project label (one ps call).
func ProjectContainerNames(r *runner.Runner) ([]string, error) {
	return r.ListProjectContainers()
}

// ListProjectContainerStates returns a best-effort state map from one podman ps -a call.
// ExitCode and Error are unset for ps-derived rows; use InspectContainers when full inspect
// fields are required (for example exited container error text).
func ListProjectContainerStates(r *runner.Runner) (map[string]*ContainerState, error) {
	out, err := r.ProjectPs("{{.Names}}\t{{.State}}")
	if err != nil {
		return nil, err
	}
	states := make(map[string]*ContainerState)
	for _, line := range strings.Split(string(bytes.TrimSpace(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		name := strings.TrimSpace(parts[0])
		state := ""
		if len(parts) > 1 {
			state = strings.ToLower(strings.TrimSpace(parts[1]))
		}
		for _, n := range splitPsNames(name) {
			if n == "" {
				continue
			}
			states[n] = &ContainerState{State: state}
		}
	}
	return states, nil
}

// ExtraContainerNamesWithProjectList returns project container names not in the expected service set.
func ExtraContainerNamesWithProjectList(r *runner.Runner, serviceNames []string, projectNames []string) []string {
	want := map[string]struct{}{}
	for _, n := range serviceNames {
		want[r.ContainerName(n)] = struct{}{}
	}
	var extra []string
	for _, name := range projectNames {
		if _, ok := want[name]; !ok {
			extra = append(extra, name)
		}
	}
	return extra
}

func parsePsNameLines(out []byte) []string {
	var names []string
	for _, line := range strings.Split(string(bytes.TrimSpace(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		names = append(names, splitPsNames(line)...)
	}
	return names
}

func splitPsNames(line string) []string {
	var names []string
	for _, name := range strings.Split(line, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}
