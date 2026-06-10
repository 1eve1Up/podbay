package diff

import (
	"errors"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
)

func TestAnalyze_noDrift(t *testing.T) {
	r := runner.New("demo")
	running := &runtimestate.ContainerState{State: "running"}
	inspect := func(name string) (*runtimestate.ContainerState, error) {
		if name == "podbay_demo_api" {
			return running, nil
		}
		return nil, nil
	}
	out, drift, err := Analyze(r, []string{"api"}, inspect, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if drift {
		t.Fatalf("expected no drift, got:\n%s", out)
	}
	if !strings.Contains(out, "No drift") {
		t.Fatalf("expected success message, got:\n%s", out)
	}
}

func TestAnalyze_missing(t *testing.T) {
	r := runner.New("demo")
	inspect := func(string) (*runtimestate.ContainerState, error) { return nil, nil }
	out, drift, err := Analyze(r, []string{"api"}, inspect, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !drift {
		t.Fatal("expected drift for missing container")
	}
	if !strings.Contains(out, "missing container") {
		t.Fatalf("expected missing line, got:\n%s", out)
	}
}

func TestAnalyze_notRunning(t *testing.T) {
	r := runner.New("demo")
	exited := &runtimestate.ContainerState{State: "exited", ExitCode: 1, Error: "boom"}
	inspect := func(string) (*runtimestate.ContainerState, error) { return exited, nil }
	out, drift, err := Analyze(r, []string{"web"}, inspect, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !drift || !strings.Contains(out, "state=exited") {
		t.Fatalf("expected not-running drift, got:\n%s", out)
	}
}

func TestAnalyze_extras(t *testing.T) {
	r := runner.New("demo")
	running := &runtimestate.ContainerState{State: "running"}
	inspect := func(string) (*runtimestate.ContainerState, error) { return running, nil }
	out, drift, err := Analyze(r, []string{"api"}, inspect, []string{"podbay_demo_debug"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !drift || !strings.Contains(out, "Unexpected containers") {
		t.Fatalf("expected extra drift, got:\n%s", out)
	}
}

func TestAnalyze_inspectError(t *testing.T) {
	r := runner.New("demo")
	inspect := func(string) (*runtimestate.ContainerState, error) {
		return nil, errors.New("inspect failed")
	}
	_, drift, err := Analyze(r, []string{"api"}, inspect, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !drift {
		t.Fatal("expected drift on inspect error")
	}
}

func TestComputeWithContainerStates_batch(t *testing.T) {
	r := runner.New("demo")
	states := map[string]*runtimestate.ContainerState{
		"podbay_demo_api": {State: "running"},
	}
	res, err := ComputeWithContainerStates(r, []string{"api", "worker"}, states, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Drift {
		t.Fatal("expected drift for missing worker")
	}
	if len(res.Services) != 2 {
		t.Fatalf("got %d services", len(res.Services))
	}
	if res.Services[0].Status != StatusOK || res.Services[1].Status != StatusMissing {
		t.Fatalf("got %+v", res.Services)
	}
}

func TestComputeWithContainerStates_exitedAndExtras(t *testing.T) {
	r := runner.New("demo")
	states := map[string]*runtimestate.ContainerState{
		"podbay_demo_web": {State: "exited", ExitCode: 3, Error: "boom"},
	}
	res, err := ComputeWithContainerStates(r, []string{"web"}, states, []string{"podbay_demo_debug"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Drift {
		t.Fatal("expected drift")
	}
	if res.Services[0].Status != StatusWrongState || res.Services[0].ExitCode != 3 {
		t.Fatalf("got %+v", res.Services[0])
	}
	if len(res.Extras) != 1 || res.Extras[0] != "podbay_demo_debug" {
		t.Fatalf("extras: %v", res.Extras)
	}
}

func TestServiceDriftForContainer_inspectError(t *testing.T) {
	sd := serviceDriftForContainer("api", "podbay_demo_api", nil, errors.New("inspect failed"))
	if sd.Status != StatusInspectError || sd.Error != "inspect failed" {
		t.Fatalf("got %+v", sd)
	}
}

func TestAnalyze_extraListError(t *testing.T) {
	r := runner.New("demo")
	extrasErr := errors.New("ps failed")
	_, _, err := Analyze(r, nil, func(string) (*runtimestate.ContainerState, error) { return nil, nil }, nil, extrasErr)
	if !errors.Is(err, extrasErr) {
		t.Fatalf("expected extrasErr, got %v", err)
	}
}
