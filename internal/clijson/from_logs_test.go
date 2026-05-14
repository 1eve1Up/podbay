package clijson

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/validate"
)

func TestKindLogs_constant(t *testing.T) {
	if KindLogs != "logs" {
		t.Fatalf("KindLogs = %q, want logs", KindLogs)
	}
}

func TestFromLogsSuccess_shape(t *testing.T) {
	d := FromLogsSuccess("/app/podbay.yaml", "demo", []string{"dev"}, "web", "podbay_demo_web", 50, "10m", "hello\n")
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["kind"] != KindLogs || m["status"] != StatusOK {
		t.Fatalf("envelope: %+v", m)
	}
	if m["service"] != "web" {
		t.Fatalf("service = %v", m["service"])
	}
	if m["container_name"] != "podbay_demo_web" {
		t.Fatalf("container_name = %v", m["container_name"])
	}
	if int(m["tail"].(float64)) != 50 {
		t.Fatalf("tail = %v", m["tail"])
	}
	if m["since"] != "10m" {
		t.Fatalf("since = %v", m["since"])
	}
	if m["log_body"] != "hello\n" {
		t.Fatalf("log_body = %q", m["log_body"])
	}
}

func TestLogsFailure_codes(t *testing.T) {
	for _, tc := range []struct {
		code string
	}{
		{CodeLogsUsageJSONFollow},
		{CodeLogsLoadError},
		{CodeLogsServiceNotActive},
		{CodeLogsPodmanUnavailable},
		{CodeLogsRuntimeError},
	} {
		d := LogsFailure("/x.yaml", "p", nil, "web", tc.code, "msg")
		if d.Status != StatusFailed || d.Kind != KindLogs {
			t.Fatalf("%s: %+v", tc.code, d)
		}
		if len(d.Issues) != 1 || d.Issues[0].Code != tc.code {
			t.Fatalf("%s issues: %+v", tc.code, d.Issues)
		}
		if d.Issues[0].Level != validate.LevelFail {
			t.Fatalf("%s level", tc.code)
		}
	}
}

func TestFromLogsSuccess_omitsZeroTail(t *testing.T) {
	d := FromLogsSuccess("/x.yaml", "p", nil, "web", "c", 0, "", "")
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"tail"`) {
		t.Fatalf("expected tail omitted: %s", raw)
	}
}
