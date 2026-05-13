package diff

import (
	"errors"
	"testing"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
)

func TestCompute_noDrift(t *testing.T) {
	r := runner.New("demo")
	running := &runtimestate.ContainerState{State: "running"}
	inspect := func(name string) (*runtimestate.ContainerState, error) {
		if name == "podbay_demo_api" {
			return running, nil
		}
		return nil, nil
	}

	got, err := Compute(r, []string{"api"}, inspect, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Drift {
		t.Fatalf("expected no drift, got drift=true: %+v", got)
	}
	if got.Project != "demo" {
		t.Fatalf("project=%q, want demo", got.Project)
	}
	if len(got.Services) != 1 {
		t.Fatalf("services len=%d, want 1", len(got.Services))
	}
	svc := got.Services[0]
	if svc.Name != "api" || svc.ContainerName != "podbay_demo_api" {
		t.Fatalf("unexpected service identity: %+v", svc)
	}
	if svc.Status != StatusOK {
		t.Fatalf("status=%q, want %q", svc.Status, StatusOK)
	}
	if svc.State != "running" {
		t.Fatalf("state=%q, want running", svc.State)
	}
	if len(got.Extras) != 0 {
		t.Fatalf("extras=%v, want empty", got.Extras)
	}
}

func TestCompute_missing(t *testing.T) {
	r := runner.New("demo")
	inspect := func(string) (*runtimestate.ContainerState, error) { return nil, nil }

	got, err := Compute(r, []string{"api"}, inspect, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Drift {
		t.Fatal("expected drift for missing container")
	}
	if got.Services[0].Status != StatusMissing {
		t.Fatalf("status=%q, want %q", got.Services[0].Status, StatusMissing)
	}
	if got.Services[0].State != "" {
		t.Fatalf("state=%q, want empty for missing", got.Services[0].State)
	}
}

func TestCompute_wrongState(t *testing.T) {
	r := runner.New("demo")
	exited := &runtimestate.ContainerState{State: "exited", ExitCode: 2, Error: "boom"}
	inspect := func(string) (*runtimestate.ContainerState, error) { return exited, nil }

	got, err := Compute(r, []string{"web"}, inspect, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Drift {
		t.Fatal("expected drift for wrong-state container")
	}
	svc := got.Services[0]
	if svc.Status != StatusWrongState {
		t.Fatalf("status=%q, want %q", svc.Status, StatusWrongState)
	}
	if svc.State != "exited" || svc.ExitCode != 2 || svc.Error != "boom" {
		t.Fatalf("wrong-state details mismatch: %+v", svc)
	}
}

func TestCompute_inspectError(t *testing.T) {
	r := runner.New("demo")
	inspect := func(string) (*runtimestate.ContainerState, error) {
		return nil, errors.New("inspect failed")
	}

	got, err := Compute(r, []string{"api"}, inspect, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Drift {
		t.Fatal("expected drift on inspect error")
	}
	svc := got.Services[0]
	if svc.Status != StatusInspectError {
		t.Fatalf("status=%q, want %q", svc.Status, StatusInspectError)
	}
	if svc.Error != "inspect failed" {
		t.Fatalf("error=%q, want %q", svc.Error, "inspect failed")
	}
}

func TestCompute_extras(t *testing.T) {
	r := runner.New("demo")
	running := &runtimestate.ContainerState{State: "running"}
	inspect := func(string) (*runtimestate.ContainerState, error) { return running, nil }

	got, err := Compute(r, []string{"api"}, inspect, []string{"podbay_demo_debug"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Drift {
		t.Fatal("expected drift due to extras")
	}
	if len(got.Extras) != 1 || got.Extras[0] != "podbay_demo_debug" {
		t.Fatalf("extras=%v, want [podbay_demo_debug]", got.Extras)
	}
	if got.Services[0].Status != StatusOK {
		t.Fatalf("service status=%q, want %q", got.Services[0].Status, StatusOK)
	}
}

func TestCompute_extrasErrorPropagates(t *testing.T) {
	r := runner.New("demo")
	extrasErr := errors.New("ps failed")
	_, err := Compute(r, nil, func(string) (*runtimestate.ContainerState, error) { return nil, nil }, nil, extrasErr)
	if !errors.Is(err, extrasErr) {
		t.Fatalf("expected extrasErr, got %v", err)
	}
}

func TestCompute_emptyInputs(t *testing.T) {
	r := runner.New("demo")
	inspect := func(string) (*runtimestate.ContainerState, error) { return nil, nil }

	got, err := Compute(r, nil, inspect, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Drift {
		t.Fatalf("expected no drift with empty inputs, got: %+v", got)
	}
	if len(got.Services) != 0 || len(got.Extras) != 0 {
		t.Fatalf("expected empty services/extras, got services=%v extras=%v", got.Services, got.Extras)
	}
	if got.Project != "demo" {
		t.Fatalf("project=%q, want demo", got.Project)
	}
}

func TestCompute_doesNotMutateExtras(t *testing.T) {
	r := runner.New("demo")
	inspect := func(string) (*runtimestate.ContainerState, error) { return nil, nil }
	src := []string{"podbay_demo_extra"}

	got, err := Compute(r, nil, inspect, src, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Extras) != 1 || got.Extras[0] != "podbay_demo_extra" {
		t.Fatalf("got.Extras=%v, want [podbay_demo_extra]", got.Extras)
	}
	if &got.Extras[0] == &src[0] {
		t.Fatal("Compute returned aliased Extras slice; want defensive copy")
	}
}

func TestServiceStatus_constantValues(t *testing.T) {
	cases := []struct {
		got  ServiceStatus
		want string
	}{
		{StatusOK, "ok"},
		{StatusMissing, "missing"},
		{StatusWrongState, "wrong_state"},
		{StatusInspectError, "inspect_error"},
	}
	for _, c := range cases {
		if string(c.got) != c.want {
			t.Errorf("constant %q != %q", string(c.got), c.want)
		}
	}
}
