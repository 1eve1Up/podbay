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

type inspectReceiptElem struct {
	Name   string `json:"Name"`
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

// ReceiptInspect is identity plus env/mount snapshots for one container.
type ReceiptInspect struct {
	ID     string
	Image  string
	Env    *[]receipt.EnvVar
	Mounts *[]receipt.MountSpec
}

func receiptInspectFromElem(o inspectReceiptElem) ReceiptInspect {
	ev := normalizeEnvVars(parseEnvStrings(o.Config.Env))
	ms := normalizeMountSpecs(parseMounts(o.Mounts))
	return ReceiptInspect{
		ID:     strings.TrimSpace(o.ID),
		Image:  strings.TrimSpace(o.Image),
		Env:    &ev,
		Mounts: &ms,
	}
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

// InspectContainersForReceipt returns receipt inspect data for each name in one
// podman inspect when possible. On batch failure it falls back to per-name
// InspectContainerForReceipt (same pattern as InspectContainers).
func InspectContainersForReceipt(names []string) (map[string]ReceiptInspect, error) {
	if len(names) == 0 {
		return map[string]ReceiptInspect{}, nil
	}
	args := append([]string{"inspect"}, names...)
	out, err := exec.Command("podman", args...).Output()
	if err == nil {
		batch, parseErr := ParseContainersForReceiptJSON(out)
		if parseErr != nil {
			return nil, parseErr
		}
		return mergeReceiptInspectResults(names, batch), nil
	}
	result := make(map[string]ReceiptInspect, len(names))
	for _, name := range names {
		id, image, env, mounts, inspectErr := InspectContainerForReceipt(name)
		if inspectErr != nil {
			return nil, inspectErr
		}
		result[name] = ReceiptInspect{ID: id, Image: image, Env: env, Mounts: mounts}
	}
	return result, nil
}

func mergeReceiptInspectResults(names []string, batch map[string]ReceiptInspect) map[string]ReceiptInspect {
	out := make(map[string]ReceiptInspect, len(names))
	for _, name := range names {
		if ri, ok := batch[name]; ok {
			out[name] = ri
			continue
		}
		// Also accept Name-keyed batch entries under normalized form.
		if ri, ok := batch[normalizeInspectName(name)]; ok {
			out[name] = ri
		}
	}
	return out
}

// ParseContainersForReceiptJSON parses a multi-element podman inspect JSON array
// into a map keyed by container name (leading slash stripped).
func ParseContainersForReceiptJSON(out []byte) (map[string]ReceiptInspect, error) {
	var data []inspectReceiptElem
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, fmt.Errorf("parse inspect many receipt: %w", err)
	}
	result := make(map[string]ReceiptInspect, len(data))
	for _, elem := range data {
		key := normalizeInspectName(elem.Name)
		if key == "" {
			// Single-element inspect without Name: leave empty key unused; caller uses ParseContainerForReceiptJSON.
			continue
		}
		result[key] = receiptInspectFromElem(elem)
	}
	return result, nil
}

// ParseContainerForReceiptJSON parses podman inspect JSON (array of objects).
func ParseContainerForReceiptJSON(out []byte) (id, image string, env *[]receipt.EnvVar, mounts *[]receipt.MountSpec, err error) {
	var data []inspectReceiptElem
	if err := json.Unmarshal(out, &data); err != nil {
		return "", "", nil, nil, fmt.Errorf("parse inspect: %w", err)
	}
	if len(data) == 0 {
		return "", "", nil, nil, fmt.Errorf("parse inspect: empty array")
	}
	ri := receiptInspectFromElem(data[0])
	return ri.ID, ri.Image, ri.Env, ri.Mounts, nil
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
