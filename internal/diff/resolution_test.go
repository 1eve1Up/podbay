package diff

import (
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/spec"
)

func TestContractDiffNameSets_fullProfileWhenNoRoots(t *testing.T) {
	c := &spec.Contract{
		Services: map[string]spec.Service{
			"a": {}, "b": {},
		},
	}
	obs, seed, err := contractDiffNameSets(c, nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(obs, ",") != "a,b" || strings.Join(seed, ",") != "a,b" {
		t.Fatalf("obs=%v seed=%v", obs, seed)
	}
}

func TestContractDiffNameSets_partialExplicit(t *testing.T) {
	c := &spec.Contract{
		Services: map[string]spec.Service{
			"web": {DependsOn: spec.Dependencies{{Service: "api", Condition: spec.ConditionStarted}}},
			"api": {},
		},
	}
	obs, seed, err := contractDiffNameSets(c, nil, []string{"web"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(obs, ",") != "web" {
		t.Fatalf("want observe web only, got %v", obs)
	}
	if strings.Join(seed, ",") != "api,web" {
		t.Fatalf("extras seed should be full profile-active, got %v", seed)
	}
}

func TestContractDiffNameSets_unknownRoot(t *testing.T) {
	c := &spec.Contract{Services: map[string]spec.Service{"a": {}}}
	_, _, err := contractDiffNameSets(c, nil, []string{"x"}, false)
	if err == nil {
		t.Fatal("expected error")
	}
}
