package ps

import (
	"fmt"
	"testing"

	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestListRows_missingAndRunning(t *testing.T) {
	c := &spec.Contract{
		Version: "1",
		Services: map[string]spec.Service{
			"web": {Image: "nginx:alpine"},
			"api": {Image: "alpine:latest"},
		},
	}
	inspect := func(name string) (*runtimestate.ContainerState, error) {
		switch name {
		case "podbay_demo_web":
			return &runtimestate.ContainerState{State: "running", Image: "docker.io/library/nginx:alpine"}, nil
		case "podbay_demo_api":
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected %q", name)
		}
	}
	rows, err := ListRows(c, "demo", nil, nil, false, inspect)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("len=%d", len(rows))
	}
	if rows[0].Service != "api" || !rows[0].Missing || rows[0].State != "missing" {
		t.Fatalf("api row: %+v", rows[0])
	}
	if rows[1].Service != "web" || rows[1].Missing || rows[1].State != "running" {
		t.Fatalf("web row: %+v", rows[1])
	}
}

func TestListRows_inspectError(t *testing.T) {
	c := &spec.Contract{
		Services: map[string]spec.Service{"a": {Image: "x"}},
	}
	inspect := func(string) (*runtimestate.ContainerState, error) {
		return nil, fmt.Errorf("boom")
	}
	rows, err := ListRows(c, "p", nil, nil, false, inspect)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].State != "error" || rows[0].Error != "boom" {
		t.Fatalf("%+v", rows[0])
	}
}

func TestListRows_noServicesForProfile(t *testing.T) {
	c := &spec.Contract{
		Services: map[string]spec.Service{
			"web": {Profiles: []string{"x"}},
		},
	}
	_, err := ListRows(c, "p", []string{"other"}, nil, false, func(string) (*runtimestate.ContainerState, error) { return nil, nil })
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListRows_partialRoots(t *testing.T) {
	c := &spec.Contract{
		Services: map[string]spec.Service{
			"web": {},
			"api": {},
		},
	}
	inspect := func(name string) (*runtimestate.ContainerState, error) {
		if name == "podbay_demo_web" {
			return &runtimestate.ContainerState{State: "running"}, nil
		}
		return nil, fmt.Errorf("unexpected inspect %q", name)
	}
	rows, err := ListRows(c, "demo", nil, []string{"web"}, false, inspect)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Service != "web" {
		t.Fatalf("got %+v", rows)
	}
}
