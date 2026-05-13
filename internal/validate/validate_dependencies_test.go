package validate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/spec"
)

func TestRun_dependentsInvalidDependent(t *testing.T) {
	c := &spec.Contract{
		Version: "1",
		Services: map[string]spec.Service{
			"web": {Image: "w", RedeployPeers: []string{"api"}},
			"api": {Image: "a"},
		},
	}
	p := filepath.Join(t.TempDir(), "p.yaml")
	res := Run(c, p, nil, nil, false)
	var found bool
	for _, r := range res {
		if r.Code == "dependents_invalid_dependent" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected dependents_invalid_dependent: %#v", res)
	}
}

func TestRun_dependentsUnknownService(t *testing.T) {
	c := &spec.Contract{
		Version: "1",
		Services: map[string]spec.Service{
			"api": {
				Image:         "a",
				RedeployPeers: []string{"missing"},
			},
			"web": {
				Image:     "w",
				DependsOn: spec.Dependencies{{Service: "api", Condition: spec.ConditionStarted}},
			},
		},
	}
	p := filepath.Join(t.TempDir(), "p.yaml")
	res := Run(c, p, nil, nil, false)
	var found bool
	for _, r := range res {
		if r.Code == "dependents_unknown_service" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected dependents_unknown_service: %#v", res)
	}
}

func TestRun_deployRoots_explicitTargetsOnly(t *testing.T) {
	c := &spec.Contract{
		Version: "1",
		Services: map[string]spec.Service{
			"web": {
				Image:     "w",
				DependsOn: spec.Dependencies{{Service: "api", Condition: spec.ConditionStarted}},
			},
			"api": {
				Image:         "a",
				RedeployPeers: []string{"web"},
			},
		},
	}
	p := filepath.Join(t.TempDir(), "p.yaml")
	res := Run(c, p, nil, []string{"web"}, false)
	var depOK bool
	for _, r := range res {
		if strings.Contains(r.Message, "explicit targets only") && r.OK {
			depOK = true
		}
	}
	if !depOK {
		t.Fatalf("expected partial acyclic ok with expansion wording: %#v", res)
	}
}

func TestRun_dependentsMissingInverse(t *testing.T) {
	c := &spec.Contract{
		Version: "1",
		Services: map[string]spec.Service{
			"web": {
				Image:     "w",
				DependsOn: spec.Dependencies{{Service: "api", Condition: spec.ConditionStarted}},
			},
			"api": {Image: "a"},
		},
	}
	p := filepath.Join(t.TempDir(), "p.yaml")
	res := Run(c, p, nil, nil, false)
	var found bool
	for _, r := range res {
		if r.Code == "dependents_missing_inverse" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected dependents_missing_inverse: %#v", res)
	}
}

func TestRun_deployRoots_expandDependentsMessage(t *testing.T) {
	c := &spec.Contract{
		Version: "1",
		Services: map[string]spec.Service{
			"web": {Image: "w", DependsOn: spec.Dependencies{{Service: "api", Condition: spec.ConditionStarted}}},
			"api": {Image: "a", RedeployPeers: []string{"web"}},
		},
	}
	p := filepath.Join(t.TempDir(), "p.yaml")
	res := Run(c, p, nil, []string{"api"}, true)
	var found bool
	for _, r := range res {
		if strings.Contains(r.Message, "dependents-expanded") && r.OK {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected dependents-expanded topo message: %#v", res)
	}
}
