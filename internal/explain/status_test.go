package explain

import (
	"fmt"
	"testing"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestCollectServiceStatus_missing(t *testing.T) {
	r := runner.New("demo")
	st := collectServiceStatus(r, "web", spec.Service{Image: "nginx"}, nil, nil, nil)
	if !st.Missing || st.InspectErr != "" || st.Running {
		t.Fatalf("%+v", st)
	}
	if st.Container != "podbay_demo_web" {
		t.Fatalf("container=%q", st.Container)
	}
}

func TestCollectServiceStatus_inspectError(t *testing.T) {
	r := runner.New("demo")
	st := collectServiceStatus(r, "web", spec.Service{}, nil, nil, fmt.Errorf("boom"))
	if st.InspectErr != "boom" || st.Missing || st.Running {
		t.Fatalf("%+v", st)
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
	st := collectServiceStatus(r, "web", svc, nil, cst, nil)
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
	st := collectServiceStatus(r, "web", spec.Service{Image: "nginx"}, nil, cst, nil)
	if st.Missing || !st.Running || st.State != "running" || st.ExecRan || st.HTTPURL != "" {
		t.Fatalf("%+v", st)
	}
}
