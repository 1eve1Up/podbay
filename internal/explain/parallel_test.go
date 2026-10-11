package explain

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestCollectStatuses_probesConcurrent(t *testing.T) {
	var started sync.WaitGroup
	started.Add(2)
	orig := httpProbeOnceFn
	httpProbeOnceFn = func(url string, _ bool, _ time.Duration) (int, error) {
		started.Done()
		started.Wait()
		return 200, nil
	}
	t.Cleanup(func() { httpProbeOnceFn = orig })

	r := runner.New("demo")
	iterate := []string{"api", "web"}
	active := map[string]spec.Service{
		"api": {Health: &spec.HealthCheck{HTTP: &spec.HTTPHealth{URL: "http://api/"}}},
		"web": {Health: &spec.HealthCheck{HTTP: &spec.HTTPHealth{URL: "http://web/"}}},
	}
	states := map[string]*runtimestate.ContainerState{
		r.ContainerName("api"): {State: "running"},
		r.ContainerName("web"): {State: "running"},
	}

	done := make(chan []ServiceStatus, 1)
	go func() {
		done <- collectStatuses(r, iterate, active, nil, states, false)
	}()
	select {
	case statuses := <-done:
		if len(statuses) != 2 {
			t.Fatalf("len=%d", len(statuses))
		}
		if statuses[0].Name != "api" || statuses[1].Name != "web" {
			t.Fatalf("cross-write or order: %+v", statuses)
		}
		if statuses[0].HTTPURL != "http://api/" || statuses[0].HTTPStatus != 200 {
			t.Fatalf("api: %+v", statuses[0])
		}
		if statuses[1].HTTPURL != "http://web/" || statuses[1].HTTPStatus != 200 {
			t.Fatalf("web: %+v", statuses[1])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP probes serialized; concurrent barrier did not release")
	}
}

func TestCollectStatuses_noProbesDoesNotProbe(t *testing.T) {
	var n atomic.Int32
	origHTTP := httpProbeOnceFn
	origExec := execOnceWithTimeoutFn
	httpProbeOnceFn = func(string, bool, time.Duration) (int, error) {
		n.Add(1)
		return 200, nil
	}
	execOnceWithTimeoutFn = func(string, []string, time.Duration) ([]byte, error) {
		n.Add(1)
		return []byte("ok"), nil
	}
	t.Cleanup(func() {
		httpProbeOnceFn = origHTTP
		execOnceWithTimeoutFn = origExec
	})

	r := runner.New("demo")
	iterate := []string{"api", "web"}
	active := map[string]spec.Service{
		"api": {Health: &spec.HealthCheck{HTTP: &spec.HTTPHealth{URL: "http://api/"}}},
		"web": {Health: &spec.HealthCheck{Exec: &spec.ExecHealth{Command: []string{"true"}}}},
	}
	states := map[string]*runtimestate.ContainerState{
		r.ContainerName("api"): {State: "running"},
		r.ContainerName("web"): {State: "running"},
	}
	statuses := collectStatuses(r, iterate, active, nil, states, true)
	if n.Load() != 0 {
		t.Fatalf("no-probes still invoked probe helpers: %d", n.Load())
	}
	for _, st := range statuses {
		if st.HTTPStatus != 0 || st.ExecRan {
			t.Fatalf("probe fields set under --no-probes: %+v", st)
		}
		if st.Missing || !st.Running {
			t.Fatalf("inspect should remain: %+v", st)
		}
	}
}

func TestCollectStatuses_parallelSkipAndTimeoutParity(t *testing.T) {
	type probeCall struct {
		url     string
		timeout time.Duration
	}
	var mu sync.Mutex
	var calls []probeCall
	orig := httpProbeOnceFn
	httpProbeOnceFn = func(url string, _ bool, timeout time.Duration) (int, error) {
		mu.Lock()
		calls = append(calls, probeCall{url, timeout})
		mu.Unlock()
		return 200, nil
	}
	t.Cleanup(func() { httpProbeOnceFn = orig })

	r := runner.New("demo")
	iterate := []string{"gone", "short", "capped"}
	active := map[string]spec.Service{
		"gone": {Health: &spec.HealthCheck{HTTP: &spec.HTTPHealth{URL: "http://gone/"}}},
		"short": {Health: &spec.HealthCheck{
			HTTP: &spec.HTTPHealth{URL: "http://short/", Timeout: "1s"},
		}},
		"capped": {Health: &spec.HealthCheck{
			Timeout: "30s",
			HTTP:    &spec.HTTPHealth{URL: "http://capped/"},
		}},
	}
	states := map[string]*runtimestate.ContainerState{
		r.ContainerName("short"):  {State: "running"},
		r.ContainerName("capped"): {State: "running"},
	}

	record := func(run func()) map[string]time.Duration {
		mu.Lock()
		calls = nil
		mu.Unlock()
		run()
		mu.Lock()
		defer mu.Unlock()
		out := map[string]time.Duration{}
		for _, c := range calls {
			out[c.url] = c.timeout
		}
		return out
	}

	serialTimeouts := record(func() {
		_ = collectServiceStatus(r, "gone", active["gone"], nil, states[r.ContainerName("gone")], nil, false)
		_ = collectServiceStatus(r, "short", active["short"], nil, states[r.ContainerName("short")], nil, false)
		_ = collectServiceStatus(r, "capped", active["capped"], nil, states[r.ContainerName("capped")], nil, false)
	})
	parallelTimeouts := record(func() {
		_ = collectStatuses(r, iterate, active, nil, states, false)
	})

	if _, ok := serialTimeouts["http://gone/"]; ok {
		t.Fatal("serial path probed missing service")
	}
	if _, ok := parallelTimeouts["http://gone/"]; ok {
		t.Fatal("parallel path probed missing service")
	}
	if serialTimeouts["http://short/"] != time.Second || parallelTimeouts["http://short/"] != time.Second {
		t.Fatalf("short timeout serial=%v parallel=%v", serialTimeouts["http://short/"], parallelTimeouts["http://short/"])
	}
	if serialTimeouts["http://capped/"] != ExplainProbeMax || parallelTimeouts["http://capped/"] != ExplainProbeMax {
		t.Fatalf("capped timeout serial=%v parallel=%v", serialTimeouts["http://capped/"], parallelTimeouts["http://capped/"])
	}

	inspectCalls := 0
	httpProbeOnceFn = func(string, bool, time.Duration) (int, error) {
		inspectCalls++
		return 200, nil
	}
	st := collectServiceStatus(r, "gone", active["gone"], nil, nil, fmt.Errorf("boom"), false)
	if st.InspectErr != "boom" || inspectCalls != 0 {
		t.Fatalf("inspect error skip: status=%+v calls=%d", st, inspectCalls)
	}
}
