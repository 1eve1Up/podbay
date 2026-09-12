package teardown

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizePartialServices_dedupAndTrim(t *testing.T) {
	got := normalizePartialServices([]string{" web ", "web", "  ", "api"})
	want := []string{"web", "api"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	if len(normalizePartialServices(nil)) != 0 {
		t.Fatalf("nil input should yield empty")
	}
}

func TestContainersForPartialRemoval(t *testing.T) {
	project := "demo"
	all := []string{
		"podbay_demo_web",
		"podbay_demo_api",
		"podbay_demo_other",
	}
	got := containersForPartialRemoval(project, all, []string{"web", "api"})
	want := []string{"podbay_demo_web", "podbay_demo_api"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	if len(containersForPartialRemoval(project, all, []string{"missing"})) != 0 {
		t.Fatalf("unknown service should match no containers")
	}
}

func TestSkipNetworkAfterPartialTeardown(t *testing.T) {
	if !skipNetworkAfterPartialTeardown(true, false, 0) {
		t.Fatal("keep-network always skips rm")
	}
	if !skipNetworkAfterPartialTeardown(false, true, 1) {
		t.Fatal("partial with remaining containers skips rm")
	}
	if skipNetworkAfterPartialTeardown(false, true, 0) {
		t.Fatal("partial with no remaining should not skip")
	}
	if skipNetworkAfterPartialTeardown(false, false, 3) {
		t.Fatal("full teardown with leftovers should not skip — caller removes all first")
	}
}

func TestFinishUnavailable_exitZeroAndMessage(t *testing.T) {
	var buf strings.Builder
	res, err := finishUnavailable(TeardownResult{Project: "pinionize", Network: "podbay_pinionize"}, Options{}, &buf, errors.New("podman ps: exit status 125: Cannot connect to Podman"))
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !res.NetworkKept || res.EngineWarning == "" {
		t.Fatalf("res = %#v", res)
	}
	got := buf.String()
	if !strings.Contains(got, "Podman is not running; nothing to tear down") || !strings.Contains(got, "Done.") {
		t.Fatalf("output = %q", got)
	}
	quiet, err := finishUnavailable(TeardownResult{}, Options{Quiet: true, Volumes: true}, io.Discard, errors.New("unavailable"))
	if err != nil || quiet.EngineWarning == "" {
		t.Fatalf("quiet = %#v err=%v", quiet, err)
	}
}
