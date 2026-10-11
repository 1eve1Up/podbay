package explain

import (
	"fmt"
	"testing"
	"time"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestCollectServiceStatus_missing(t *testing.T) {
	r := runner.New("demo")
	st := collectServiceStatus(r, "web", spec.Service{Image: "nginx"}, nil, nil, nil, false)
	if !st.Missing || st.InspectErr != "" || st.Running {
		t.Fatalf("%+v", st)
	}
	if st.Container != "podbay_demo_web" {
		t.Fatalf("container=%q", st.Container)
	}
}

func TestCollectServiceStatus_inspectError(t *testing.T) {
	r := runner.New("demo")
	probed := 0
	orig := httpProbeOnceFn
	httpProbeOnceFn = func(string, bool, time.Duration) (int, error) {
		probed++
		return 200, nil
	}
	t.Cleanup(func() { httpProbeOnceFn = orig })

	svc := spec.Service{Health: &spec.HealthCheck{HTTP: &spec.HTTPHealth{URL: "http://127.0.0.1/"}}}
	st := collectServiceStatus(r, "web", svc, nil, nil, fmt.Errorf("boom"), false)
	if st.InspectErr != "boom" || st.Missing || st.Running {
		t.Fatalf("%+v", st)
	}
	if probed != 0 {
		t.Fatalf("inspect error must skip probes, calls=%d", probed)
	}
}

func TestCollectServiceStatus_exitedSkipsExecProbe(t *testing.T) {
	r := runner.New("demo")
	svc := spec.Service{
		Health: &spec.HealthCheck{
			Exec: &spec.ExecHealth{Command: []string{"true"}},
		},
	}
	cst := &runtimestate.ContainerState{State: "exited", ExitCode: 2, Error: "died"}
	st := collectServiceStatus(r, "web", svc, nil, cst, nil, false)
	if st.Missing || st.Running || st.State != "exited" || st.ExitCode != 2 {
		t.Fatalf("%+v", st)
	}
	if !st.ExecRan || st.ExecErr != "container not running" {
		t.Fatalf("exec: ran=%v err=%q", st.ExecRan, st.ExecErr)
	}
}

func TestCollectServiceStatus_runningNoHealth(t *testing.T) {
	r := runner.New("demo")
	cst := &runtimestate.ContainerState{State: "running", Image: "nginx"}
	st := collectServiceStatus(r, "web", spec.Service{Image: "nginx"}, nil, cst, nil, false)
	if st.Missing || !st.Running || st.State != "running" || st.ExecRan || st.HTTPURL != "" {
		t.Fatalf("%+v", st)
	}
}

func TestCollectServiceStatus_noProbesSkipsHTTPAndExec(t *testing.T) {
	r := runner.New("demo")
	svc := spec.Service{
		Health: &spec.HealthCheck{
			HTTP: &spec.HTTPHealth{URL: "http://127.0.0.1:1/"},
			Exec: &spec.ExecHealth{Command: []string{"true"}},
		},
	}
	cst := &runtimestate.ContainerState{State: "running"}
	st := collectServiceStatus(r, "web", svc, nil, cst, nil, true)
	if st.Missing || !st.Running || st.State != "running" {
		t.Fatalf("inspect should remain: %+v", st)
	}
	if st.HTTPURL != "" || st.HTTPStatus != 0 || st.HTTPProbeErr != "" {
		t.Fatalf("HTTP probe should stay unset: %+v", st)
	}
	if st.ExecRan || st.ExecOutput != "" || st.ExecErr != "" {
		t.Fatalf("exec probe should stay unset: %+v", st)
	}
}
