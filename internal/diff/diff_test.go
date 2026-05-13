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

func TestAnalyze_extraListError(t *testing.T) {
	r := runner.New("demo")
	extrasErr := errors.New("ps failed")
	_, _, err := Analyze(r, nil, func(string) (*runtimestate.ContainerState, error) { return nil, nil }, nil, extrasErr)
	if !errors.Is(err, extrasErr) {
		t.Fatalf("expected extrasErr, got %v", err)
	}
}
