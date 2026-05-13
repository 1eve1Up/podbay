package ps

import (
	"encoding/json"
	"testing"

	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestReportJSON_formatVersion(t *testing.T) {
	c := &spec.Contract{
		Services: map[string]spec.Service{"web": {Image: "nginx:alpine"}},
	}
	raw, err := ReportJSON(c, "/tmp/podbay.yaml", "demo", []string{"dev"}, nil, false, func(string) (*runtimestate.ContainerState, error) {
		return &runtimestate.ContainerState{State: "running"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if int(m["format_version"].(float64)) != JSONFormatVersion {
		t.Fatalf("format_version %v", m["format_version"])
	}
	if m["project"] != "demo" {
		t.Fatalf("project %v", m["project"])
	}
	rows, ok := m["containers"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("containers: %v", m["containers"])
	}
}
