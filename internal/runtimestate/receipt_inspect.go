package runtimestate

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"cmp"
	"slices"

	"github.com/1eve1Up/podbay/internal/receipt"
)

type inspectReceiptObj []struct {
	ID     string `json:"Id"`
	Image  string `json:"Image"`
	Config struct {
		Env []string `json:"Env"`
	} `json:"Config"`
	Mounts []struct {
		Type        string `json:"Type"`
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
	} `json:"Mounts"`
}

// InspectContainerForReceipt runs podman inspect once and returns identity plus
// normalized env and mount snapshots for deploy receipts.
func InspectContainerForReceipt(containerName string) (id, image string, env *[]receipt.EnvVar, mounts *[]receipt.MountSpec, err error) {
	cmd := exec.Command("podman", "inspect", containerName)
	out, err := cmd.Output()
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("podman inspect %q: %w", containerName, err)
	}
	return ParseContainerForReceiptJSON(out)
}

// ParseContainerForReceiptJSON parses podman inspect JSON (array of objects).
func ParseContainerForReceiptJSON(out []byte) (id, image string, env *[]receipt.EnvVar, mounts *[]receipt.MountSpec, err error) {
	var data inspectReceiptObj
	if err := json.Unmarshal(out, &data); err != nil {
		return "", "", nil, nil, fmt.Errorf("parse inspect: %w", err)
	}
	if len(data) == 0 {
		return "", "", nil, nil, fmt.Errorf("parse inspect: empty array")
	}
	o := data[0]
	id = strings.TrimSpace(o.ID)
	image = strings.TrimSpace(o.Image)
	ev := normalizeEnvVars(parseEnvStrings(o.Config.Env))
	ms := normalizeMountSpecs(parseMounts(o.Mounts))
	return id, image, &ev, &ms, nil
}

func parseEnvStrings(lines []string) []receipt.EnvVar {
	out := make([]receipt.EnvVar, 0, len(lines))
	for _, line := range lines {
		name, val, _ := strings.Cut(line, "=")
		out = append(out, receipt.EnvVar{Name: name, Value: val})
	}
	return out
}

func normalizeEnvVars(e []receipt.EnvVar) []receipt.EnvVar {
	out := append([]receipt.EnvVar(nil), e...)
	slices.SortFunc(out, func(a, b receipt.EnvVar) int {
		if c := cmp.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		return cmp.Compare(a.Value, b.Value)
	})
	return out
}

func parseMounts(in []struct {
	Type        string `json:"Type"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
}) []receipt.MountSpec {
	out := make([]receipt.MountSpec, 0, len(in))
	for _, m := range in {
		out = append(out, receipt.MountSpec{
			Type:        strings.TrimSpace(m.Type),
			Source:      strings.TrimSpace(m.Source),
			Destination: strings.TrimSpace(m.Destination),
		})
	}
	return out
}

func normalizeMountSpecs(m []receipt.MountSpec) []receipt.MountSpec {
	out := append([]receipt.MountSpec(nil), m...)
	slices.SortFunc(out, func(a, b receipt.MountSpec) int {
		if c := cmp.Compare(a.Destination, b.Destination); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Source, b.Source); c != 0 {
			return c
		}
		return cmp.Compare(a.Type, b.Type)
	})
	return out
}
