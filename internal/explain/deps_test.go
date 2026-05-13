package explain

import (
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/spec"
)

func TestDependencySummary_chain(t *testing.T) {
	active := map[string]spec.Service{
		"api": {
			DependsOn: []spec.Dependency{
				{Service: "db", Condition: spec.ConditionHealthy},
			},
		},
		"db": {},
		"web": {
			DependsOn: []spec.Dependency{
				{Service: "api", Condition: spec.ConditionStarted},
			},
		},
	}
	out := DependencySummary(active, "api")
	if !strings.Contains(out, "Focus: api") {
		t.Fatalf("missing focus, got:\n%s", out)
	}
	if !strings.Contains(out, "db (healthy)") {
		t.Fatalf("missing dep, got:\n%s", out)
	}
	if !strings.Contains(out, "web (started)") {
		t.Fatalf("missing dependent, got:\n%s", out)
	}
	if !strings.Contains(out, "Deploy order (active):") {
		t.Fatalf("missing order, got:\n%s", out)
	}
}

func TestDependencySummary_noDeps(t *testing.T) {
	active := map[string]spec.Service{"solo": {}}
	out := DependencySummary(active, "solo")
	if !strings.Contains(out, "Depends on: (none)") || !strings.Contains(out, "Dependents: (none)") {
		t.Fatalf("got:\n%s", out)
	}
}
