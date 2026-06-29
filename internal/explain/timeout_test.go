package explain

import (
	"testing"
	"time"

	"github.com/1eve1Up/podbay/internal/spec"
)

func TestExplainProbeTimeout_capsLongContract(t *testing.T) {
	svc := spec.Service{
		Health: &spec.HealthCheck{
			Timeout: "15s",
			HTTP:    &spec.HTTPHealth{URL: "http://127.0.0.1:8080/"},
		},
	}
	if got := explainProbeTimeout(svc, true); got != ExplainProbeMax {
		t.Fatalf("HTTP: got %v want %v", got, ExplainProbeMax)
	}
	if got := explainProbeTimeout(svc, false); got != ExplainProbeMax {
		t.Fatalf("exec: got %v want %v", got, ExplainProbeMax)
	}
}

func TestExplainProbeTimeout_honorsShortContract(t *testing.T) {
	svc := spec.Service{
		Health: &spec.HealthCheck{
			Timeout: "1s",
			HTTP:    &spec.HTTPHealth{URL: "http://127.0.0.1:8080/", Timeout: "2s"},
		},
	}
	if got := explainProbeTimeout(svc, true); got != 2*time.Second {
		t.Fatalf("HTTP override: got %v want 2s", got)
	}
	if got := explainProbeTimeout(svc, false); got != time.Second {
		t.Fatalf("exec: got %v want 1s", got)
	}
}

func TestExplainProbeTimeout_capsHugeContract(t *testing.T) {
	svc := spec.Service{
		Health: &spec.HealthCheck{Timeout: "600s"},
	}
	if got := explainProbeTimeout(svc, false); got != ExplainProbeMax {
		t.Fatalf("got %v want %v", got, ExplainProbeMax)
	}
}
