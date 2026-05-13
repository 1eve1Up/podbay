package diff

import (
	"errors"
	"testing"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
)

// These tests lock the byte-exact text format produced by Analyze. They are
// the safety net for the PIN-033 refactor (Analyze => Compute + Render): if
// the text drifts, drift output via the existing CLI changes, which would
// break operator scripts that grep these lines.

func TestRender_noDrift_singleService(t *testing.T) {
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
		t.Fatalf("expected no drift, got drift=true:\n%s", out)
	}
	want := "Project: demo\n" +
		"Expected services (active): 1\n" +
		"api\n\n" +
		"[api] ok (running)\n" +
		"\nNo drift: every expected service has a running container; no extra project containers.\n"
	if out != want {
		t.Fatalf("text mismatch.\n got:\n%q\nwant:\n%q", out, want)
	}
}

func TestRender_missing_singleService(t *testing.T) {
	r := runner.New("demo")
	inspect := func(string) (*runtimestate.ContainerState, error) { return nil, nil }
	out, drift, err := Analyze(r, []string{"api"}, inspect, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !drift {
		t.Fatal("expected drift")
	}
	want := "Project: demo\n" +
		"Expected services (active): 1\n" +
		"api\n\n" +
		"[api] missing container podbay_demo_api\n" +
		"\nDrift detected.\n"
	if out != want {
		t.Fatalf("text mismatch.\n got:\n%q\nwant:\n%q", out, want)
	}
}

func TestRender_wrongState_singleService(t *testing.T) {
	r := runner.New("demo")
	exited := &runtimestate.ContainerState{State: "exited", ExitCode: 1, Error: "boom"}
	inspect := func(string) (*runtimestate.ContainerState, error) { return exited, nil }
	out, drift, err := Analyze(r, []string{"web"}, inspect, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !drift {
		t.Fatal("expected drift")
	}
	want := "Project: demo\n" +
		"Expected services (active): 1\n" +
		"web\n\n" +
		"[web] container podbay_demo_web state=exited exit=1 err=boom\n" +
		"\nDrift detected.\n"
	if out != want {
		t.Fatalf("text mismatch.\n got:\n%q\nwant:\n%q", out, want)
	}
}

func TestRender_inspectError_singleService(t *testing.T) {
	r := runner.New("demo")
	inspect := func(string) (*runtimestate.ContainerState, error) {
		return nil, errors.New("inspect failed")
	}
	out, drift, err := Analyze(r, []string{"api"}, inspect, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !drift {
		t.Fatal("expected drift")
	}
	want := "Project: demo\n" +
		"Expected services (active): 1\n" +
		"api\n\n" +
		"[api] inspect error: inspect failed\n" +
		"\nDrift detected.\n"
	if out != want {
		t.Fatalf("text mismatch.\n got:\n%q\nwant:\n%q", out, want)
	}
}

func TestRender_extras_runningServicePlusExtra(t *testing.T) {
	r := runner.New("demo")
	running := &runtimestate.ContainerState{State: "running"}
	inspect := func(string) (*runtimestate.ContainerState, error) { return running, nil }
	out, drift, err := Analyze(r, []string{"api"}, inspect, []string{"podbay_demo_debug"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !drift {
		t.Fatal("expected drift due to extras")
	}
	want := "Project: demo\n" +
		"Expected services (active): 1\n" +
		"api\n\n" +
		"[api] ok (running)\n" +
		"\nUnexpected containers (podbay.project label, not in contract):\n" +
		"  + podbay_demo_debug\n" +
		"\nDrift detected.\n"
	if out != want {
		t.Fatalf("text mismatch.\n got:\n%q\nwant:\n%q", out, want)
	}
}

func TestRender_zeroServices_zeroExtras(t *testing.T) {
	r := runner.New("demo")
	inspect := func(string) (*runtimestate.ContainerState, error) { return nil, nil }
	out, drift, err := Analyze(r, nil, inspect, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if drift {
		t.Fatalf("expected no drift, got:\n%s", out)
	}
	want := "Project: demo\n" +
		"Expected services (active): 0\n" +
		"\nNo drift: every expected service has a running container; no extra project containers.\n"
	if out != want {
		t.Fatalf("text mismatch.\n got:\n%q\nwant:\n%q", out, want)
	}
}

func TestRender_multipleServicesMixedStatus(t *testing.T) {
	r := runner.New("demo")
	running := &runtimestate.ContainerState{State: "running"}
	exited := &runtimestate.ContainerState{State: "exited", ExitCode: 137, Error: "OOM"}
	inspect := func(name string) (*runtimestate.ContainerState, error) {
		switch name {
		case "podbay_demo_api":
			return running, nil
		case "podbay_demo_worker":
			return exited, nil
		case "podbay_demo_db":
			return nil, nil
		case "podbay_demo_metrics":
			return nil, errors.New("inspect failed")
		}
		return nil, nil
	}
	out, drift, err := Analyze(r, []string{"api", "worker", "db", "metrics"}, inspect, []string{"podbay_demo_debug"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !drift {
		t.Fatal("expected drift")
	}
	want := "Project: demo\n" +
		"Expected services (active): 4\n" +
		"api, worker, db, metrics\n\n" +
		"[api] ok (running)\n" +
		"[worker] container podbay_demo_worker state=exited exit=137 err=OOM\n" +
		"[db] missing container podbay_demo_db\n" +
		"[metrics] inspect error: inspect failed\n" +
		"\nUnexpected containers (podbay.project label, not in contract):\n" +
		"  + podbay_demo_debug\n" +
		"\nDrift detected.\n"
	if out != want {
		t.Fatalf("text mismatch.\n got:\n%q\nwant:\n%q", out, want)
	}
}

func TestRender_directOnDriftResult_matchesAnalyze(t *testing.T) {
	r := runner.New("demo")
	running := &runtimestate.ContainerState{State: "running"}
	inspect := func(name string) (*runtimestate.ContainerState, error) {
		if name == "podbay_demo_api" {
			return running, nil
		}
		return nil, nil
	}
	res, err := Compute(r, []string{"api"}, inspect, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	rendered := Render(res)

	out, _, err := Analyze(r, []string{"api"}, inspect, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rendered != out {
		t.Fatalf("Render(Compute()) and Analyze produced different text.\nrender:\n%q\nanalyze:\n%q", rendered, out)
	}
}
