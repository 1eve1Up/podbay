package validate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/spec"
)

func TestRun_deployRootsUnknownService(t *testing.T) {
	c := &spec.Contract{
		Version: "1",
		Services: map[string]spec.Service{
			"web": {Image: "nginx:alpine"},
		},
	}
	p := filepath.Join(t.TempDir(), "podbay.yaml")
	res := Run(c, p, nil, []string{"missing"}, false)
	if len(res) == 0 {
		t.Fatal("expected at least one result")
	}
	r := res[0]
	if r.OK || r.Code != "deploy_service_selection" {
		t.Fatalf("got ok=%v code=%q msg=%q", r.OK, r.Code, r.Message)
	}
}

func TestRun_deployRoots_explicitTargetsAcyclic(t *testing.T) {
	c := &spec.Contract{
		Version: "1",
		Services: map[string]spec.Service{
			"web": {Image: "w", DependsOn: spec.Dependencies{{Service: "api", Condition: spec.ConditionStarted}}},
			"api": {Image: "a"},
		},
	}
	p := filepath.Join(t.TempDir(), "podbay.yaml")
	res := Run(c, p, nil, []string{"web"}, false)
	var depOK bool
	for _, r := range res {
		if strings.Contains(r.Message, "explicit targets only") && r.OK {
			depOK = true
		}
	}
	if !depOK {
		t.Fatalf("expected explicit-targets dependency ok in results: %#v", res)
	}
}

func TestRun_deployRoots_externalDepNotRequiredInActive(t *testing.T) {
	c := &spec.Contract{
		Version: "1",
		Services: map[string]spec.Service{
			"web": {Image: "w", DependsOn: spec.Dependencies{{Service: "api", Condition: spec.ConditionStarted}}},
			"api": {Image: "a"},
		},
	}
	p := filepath.Join(t.TempDir(), "podbay.yaml")
	res := Run(c, p, nil, []string{"web"}, false)
	for _, r := range res {
		if strings.Contains(r.Message, "which is not active") {
			t.Fatalf("external dep should be allowed in partial mode: %q", r.Message)
		}
	}
}

func TestRun_deployRoots_cycleInTargets(t *testing.T) {
	c := &spec.Contract{
		Version: "1",
		Services: map[string]spec.Service{
			"a": {Image: "x", DependsOn: spec.Dependencies{{Service: "b", Condition: spec.ConditionStarted}}},
			"b": {Image: "y", DependsOn: spec.Dependencies{{Service: "a", Condition: spec.ConditionStarted}}},
		},
	}
	p := filepath.Join(t.TempDir(), "podbay.yaml")
	res := Run(c, p, nil, []string{"a", "b"}, false)
	found := false
	for _, r := range res {
		if r.Code == "dependency_invalid" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected cycle failure: %#v", res)
	}
}
