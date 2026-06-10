// Package runtimestate centralizes Podman list/inspect helpers for contract vs runtime views
// (explain, diff, and similar).
package runtimestate

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/1eve1Up/podbay/internal/runner"
)

// ContainerState summarizes podman inspect for one container.
type ContainerState struct {
	State    string
	ExitCode int
	Error    string
	// Image is a display string (ImageName when present, else Image digest/id).
	Image string
}

type inspectOut []struct {
	Name      string `json:"Name"`
	Image     string `json:"Image"`
	ImageName string `json:"ImageName"`
	State     struct {
		Status   string `json:"Status"`
		ExitCode int    `json:"ExitCode"`
		Error    string `json:"Error"`
	} `json:"State"`
}

// ParseInspect decodes the first element of `podman inspect` JSON output.
func ParseInspect(out []byte) (*ContainerState, error) {
	var data inspectOut
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, fmt.Errorf("parse inspect: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("parse inspect: empty array")
	}
	elem := data[0]
	st := elem.State
	img := strings.TrimSpace(elem.ImageName)
	if img == "" {
		img = strings.TrimSpace(elem.Image)
	}
	return &ContainerState{
		State:    strings.ToLower(st.Status),
		ExitCode: st.ExitCode,
		Error:    st.Error,
		Image:    img,
	}, nil
}

// InspectContainer returns container state, or (nil, nil) if inspect fails
// (for example no object with that name).
func InspectContainer(containerName string) (*ContainerState, error) {
	cmd := exec.Command("podman", "inspect", containerName)
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	return ParseInspect(out)
}

// ExtraContainerNames returns container names for the project label that are not
// expected service containers (runner.ContainerName(service) for each service in serviceNames).
func ExtraContainerNames(r *runner.Runner, serviceNames []string) ([]string, error) {
	projectNames, err := ProjectContainerNames(r)
	if err != nil {
		return nil, err
	}
	return ExtraContainerNamesWithProjectList(r, serviceNames, projectNames), nil
}
