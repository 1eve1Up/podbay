package runtimestate

import (
	"os/exec"
)

// mergeInspectResults maps requested container names to states from a batch parse.
// Names absent from batch are nil (missing container).
func mergeInspectResults(names []string, batch map[string]*ContainerState) map[string]*ContainerState {
	out := make(map[string]*ContainerState, len(names))
	for _, name := range names {
		if st, ok := batch[name]; ok {
			out[name] = st
		} else {
			out[name] = nil
		}
	}
	return out
}

// InspectContainers returns container state for each name in one podman inspect when
// possible. When batch inspect fails (for example a missing name), it falls back to
// per-name InspectContainer calls so semantics match the single-container path.
func InspectContainers(names []string) (map[string]*ContainerState, error) {
	if len(names) == 0 {
		return map[string]*ContainerState{}, nil
	}
	args := append([]string{"inspect"}, names...)
	out, err := exec.Command("podman", args...).Output()
	if err == nil {
		batch, parseErr := ParseInspectMany(out)
		if parseErr != nil {
			return nil, parseErr
		}
		return mergeInspectResults(names, batch), nil
	}
	result := make(map[string]*ContainerState, len(names))
	for _, name := range names {
		st, inspectErr := InspectContainer(name)
		if inspectErr != nil {
			return nil, inspectErr
		}
		result[name] = st
	}
	return result, nil
}
