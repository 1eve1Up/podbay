package explain

import (
	"testing"
)

func TestRuntimeRowsFromStatus_mapsProbeFacts(t *testing.T) {
	rows := RuntimeRowsFromStatus([]ServiceStatus{
		{Name: "missing", Missing: true},
		{Name: "okhttp", Running: true, State: "running", HTTPURL: "http://127.0.0.1/", HTTPStatus: 200},
		{Name: "badhttp", Running: true, State: "running", HTTPURL: "http://127.0.0.1/", HTTPStatus: 500},
		{Name: "execfail", Running: true, State: "running", ExecRan: true, ExecErr: "boom"},
	})
	if len(rows) != 4 {
		t.Fatalf("len=%d", len(rows))
	}
	if !rows[0].Missing {
		t.Fatal("missing row")
	}
	if rows[1].Healthy == nil || !*rows[1].Healthy {
		t.Fatalf("okhttp healthy: %+v", rows[1])
	}
	if rows[2].Healthy == nil || *rows[2].Healthy {
		t.Fatalf("badhttp should be unhealthy: %+v", rows[2])
	}
	if rows[3].Healthy == nil || *rows[3].Healthy {
		t.Fatalf("execfail should be unhealthy: %+v", rows[3])
	}
}

func TestRuntimeRowsFromStatus_inspectError(t *testing.T) {
	rows := RuntimeRowsFromStatus([]ServiceStatus{
		{Name: "x", InspectErr: "denied"},
	})
	if len(rows) != 1 || !rows[0].Missing || rows[0].Healthy == nil || *rows[0].Healthy {
		t.Fatalf("inspect error row: %+v", rows)
	}
}
