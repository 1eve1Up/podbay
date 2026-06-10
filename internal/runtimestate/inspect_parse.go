package runtimestate

import (
	"encoding/json"
	"fmt"
	"strings"
)

type inspectElem struct {
	Name      string `json:"Name"`
	Image     string `json:"Image"`
	ImageName string `json:"ImageName"`
	State     struct {
		Status   string `json:"Status"`
		ExitCode int    `json:"ExitCode"`
		Error    string `json:"Error"`
	} `json:"State"`
}

func containerStateFromElem(elem inspectElem) *ContainerState {
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
	}
}

func normalizeInspectName(name string) string {
	return strings.TrimPrefix(strings.TrimSpace(name), "/")
}

// ParseInspectMany decodes a multi-element podman inspect JSON array into a map
// keyed by container name (leading slash stripped). Names absent from the output
// are simply missing from the map; callers treat that as a missing container.
func ParseInspectMany(out []byte) (map[string]*ContainerState, error) {
	var data []inspectElem
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, fmt.Errorf("parse inspect many: %w", err)
	}
	result := make(map[string]*ContainerState, len(data))
	for _, elem := range data {
		key := normalizeInspectName(elem.Name)
		if key == "" {
			continue
		}
		result[key] = containerStateFromElem(elem)
	}
	return result, nil
}
