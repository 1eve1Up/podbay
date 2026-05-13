package runtimestate

import (
	"encoding/json"
	"fmt"
	"strings"
)

type inspectIdentity []struct {
	ID    string `json:"Id"`
	Image string `json:"Image"`
}

// ParseContainerIdentityJSON extracts Id and Image from podman inspect JSON (first object).
func ParseContainerIdentityJSON(out []byte) (id, image string, err error) {
	var data inspectIdentity
	if err := json.Unmarshal(out, &data); err != nil {
		return "", "", fmt.Errorf("parse inspect identity: %w", err)
	}
	if len(data) == 0 {
		return "", "", fmt.Errorf("parse inspect identity: empty array")
	}
	return strings.TrimSpace(data[0].ID), strings.TrimSpace(data[0].Image), nil
}

// InspectContainerIdentity returns the container id and image digest/reference from podman inspect.
func InspectContainerIdentity(containerName string) (id, image string, err error) {
	id, image, _, _, err = InspectContainerForReceipt(containerName)
	return id, image, err
}
