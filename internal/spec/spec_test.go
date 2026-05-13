package spec

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTopologicalOrder(t *testing.T) {
	services := map[string]Service{
		"api": {DependsOn: Dependencies{{Service: "db", Condition: ConditionStarted}}},
		"db":  {},
	}
	order, err := TopologicalOrder(services)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "db" || order[1] != "api" {
		t.Fatalf("got %v want [db api]", order)
	}
}

func TestTopologicalCycle(t *testing.T) {
	services := map[string]Service{
		"a": {DependsOn: Dependencies{{Service: "b", Condition: ConditionStarted}}},
		"b": {DependsOn: Dependencies{{Service: "a", Condition: ConditionStarted}}},
	}
	_, err := TopologicalOrder(services)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestTopologicalOrderSubset_ignoresExternalDeps(t *testing.T) {
	full := map[string]Service{
		"web": {DependsOn: Dependencies{{Service: "api", Condition: ConditionStarted}}},
		"api": {DependsOn: Dependencies{{Service: "db", Condition: ConditionStarted}}},
		"db":  {},
	}
	sub := map[string]Service{"web": full["web"]}
	order, err := TopologicalOrderSubset(sub)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 1 || order[0] != "web" {
		t.Fatalf("got %v want [web]", order)
	}
}

func TestTopologicalOrderSubset_ordersInternalEdges(t *testing.T) {
	full := map[string]Service{
		"web": {DependsOn: Dependencies{{Service: "api", Condition: ConditionStarted}}},
		"api": {},
	}
	sub := map[string]Service{"web": full["web"], "api": full["api"]}
	order, err := TopologicalOrderSubset(sub)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "api" || order[1] != "web" {
		t.Fatalf("got %v want [api web]", order)
	}
}

func TestTopologicalOrderSubset_cycleWithinSubset(t *testing.T) {
	sub := map[string]Service{
		"a": {DependsOn: Dependencies{{Service: "b", Condition: ConditionStarted}}},
		"b": {DependsOn: Dependencies{{Service: "a", Condition: ConditionStarted}}},
	}
	_, err := TopologicalOrderSubset(sub)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestDependsOnUnmarshalMap(t *testing.T) {
	const y = `
depends_on:
  api:
    condition: service_healthy
  web:
    condition: service_started
`
	var s struct {
		D Dependencies `yaml:"depends_on"`
	}
	if err := yaml.Unmarshal([]byte(y), &s); err != nil {
		t.Fatal(err)
	}
	if len(s.D) != 2 {
		t.Fatalf("len %d", len(s.D))
	}
	want := map[string]string{}
	for _, d := range s.D {
		want[d.Service] = d.Condition
	}
	if want["api"] != ConditionHealthy || want["web"] != ConditionStarted {
		t.Fatalf("got %#v", want)
	}
}

func TestProfilesIncluded(t *testing.T) {
	s := Service{Profiles: []string{"observability"}}
	if s.IsIncluded(nil) {
		t.Fatal("expected excluded with no profiles")
	}
	if s.IsIncluded([]string{"other"}) {
		t.Fatal("expected excluded")
	}
	if !s.IsIncluded([]string{"observability"}) {
		t.Fatal("expected included")
	}
	var noProfile Service
	if !noProfile.IsIncluded(nil) {
		t.Fatal("no profiles = always on")
	}
}

func TestServicesForDeployTargets_emptyRoots_returnsBase(t *testing.T) {
	base := map[string]Service{"a": {}, "b": {}}
	out, err := ServicesForDeployTargets(base, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("got %d", len(out))
	}
	out2, err := ServicesForDeployTargets(base, []string{"", "  "})
	if err != nil {
		t.Fatal(err)
	}
	if len(out2) != 2 {
		t.Fatalf("got %d", len(out2))
	}
}

func TestServicesForDeployTargets_onlyNamedServices(t *testing.T) {
	base := map[string]Service{
		"web": {DependsOn: Dependencies{{Service: "api", Condition: ConditionStarted}}},
		"api": {DependsOn: Dependencies{{Service: "db", Condition: ConditionStarted}}},
		"db":  {},
	}
	out, err := ServicesForDeployTargets(base, []string{"web"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("want single web only, got %v", ServiceNamesSorted(out))
	}
	if _, ok := out["web"]; !ok {
		t.Fatal("missing web")
	}
	out2, err := ServicesForDeployTargets(base, []string{"web", "web", "api"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out2) != 2 {
		t.Fatalf("dedupe roots: got %d names %v", len(out2), ServiceNamesSorted(out2))
	}
}

func TestServicesForDeployTargets_unknownRoot(t *testing.T) {
	base := map[string]Service{"a": {}}
	_, err := ServicesForDeployTargets(base, []string{"missing"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServicesForDeployTargets_depOutsideBaseStillOk(t *testing.T) {
	base := map[string]Service{
		"web": {DependsOn: Dependencies{{Service: "api", Condition: ConditionStarted}}},
	}
	out, err := ServicesForDeployTargets(base, []string{"web"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d", len(out))
	}
}

func TestDependsOnContains(t *testing.T) {
	s := Service{DependsOn: Dependencies{{Service: "api", Condition: ConditionHealthy}}}
	if !DependsOnContains(s, "api") || DependsOnContains(s, "web") {
		t.Fatal("DependsOnContains mismatch")
	}
}

func TestExpandDependentsTransitive_seedsEmpty(t *testing.T) {
	base := map[string]Service{"a": {}}
	if got := ExpandDependentsTransitive(base, nil); got != nil {
		t.Fatalf("got %#v", got)
	}
	if got := ExpandDependentsTransitive(base, map[string]Service{}); len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestExpandDependentsTransitive_seedPullsDependent(t *testing.T) {
	base := map[string]Service{
		"web": {DependsOn: Dependencies{{Service: "api", Condition: ConditionStarted}}},
		"api": {Image: "a"},
	}
	seeds := map[string]Service{"api": base["api"]}
	out := ExpandDependentsTransitive(base, seeds)
	if got := strings.Join(ServiceNamesSorted(out), ","); got != "api,web" {
		t.Fatalf("want api+web, got %q", got)
	}
}

func TestExpandDependentsTransitive_transitiveChain(t *testing.T) {
	// c -> b -> a (each depends_on parent to the right)
	base := map[string]Service{
		"a": {},
		"b": {DependsOn: Dependencies{{Service: "a", Condition: ConditionStarted}}},
		"c": {DependsOn: Dependencies{{Service: "b", Condition: ConditionStarted}}},
	}
	seeds := map[string]Service{"a": base["a"]}
	out := ExpandDependentsTransitive(base, seeds)
	if got := strings.Join(ServiceNamesSorted(out), ","); got != "a,b,c" {
		t.Fatalf("want full chain, got %q", got)
	}
}

func TestExpandDependentsTransitive_diamond(t *testing.T) {
	base := map[string]Service{
		"base": {},
		"l":    {DependsOn: Dependencies{{Service: "base", Condition: ConditionStarted}}},
		"r":    {DependsOn: Dependencies{{Service: "base", Condition: ConditionStarted}}},
		"top":  {DependsOn: Dependencies{{Service: "l", Condition: ConditionStarted}, {Service: "r", Condition: ConditionStarted}}},
	}
	seeds := map[string]Service{"base": base["base"]}
	out := ExpandDependentsTransitive(base, seeds)
	if got := strings.Join(ServiceNamesSorted(out), ","); got != "base,l,r,top" {
		t.Fatalf("want diamond closure, got %q", got)
	}
}

func TestExpandDependentsTransitive_twoSeeds(t *testing.T) {
	base := map[string]Service{
		"mid":   {},
		"leaf1": {DependsOn: Dependencies{{Service: "mid", Condition: ConditionStarted}}},
		"leaf2": {DependsOn: Dependencies{{Service: "mid", Condition: ConditionStarted}}},
	}
	seeds := map[string]Service{"mid": base["mid"], "leaf1": base["leaf1"]}
	out := ExpandDependentsTransitive(base, seeds)
	if got := strings.Join(ServiceNamesSorted(out), ","); got != "leaf1,leaf2,mid" {
		t.Fatalf("want mid+leaf1+leaf2, got %q", got)
	}
}

func TestExpandDependentsTransitive_dependentOutsideBaseIgnored(t *testing.T) {
	base := map[string]Service{
		"web": {DependsOn: Dependencies{{Service: "api", Condition: ConditionStarted}}},
	}
	seeds := map[string]Service{"web": base["web"]}
	out := ExpandDependentsTransitive(base, seeds)
	if len(out) != 1 || out["web"].Image != base["web"].Image {
		t.Fatalf("want web only, got %#v", ServiceNamesSorted(out))
	}
}

func TestObservabilityActiveServices_noRootsFullProfile(t *testing.T) {
	base := map[string]Service{
		"web": {},
		"api": {},
	}
	got, err := ObservabilityActiveServices(base, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want full profile map, got %v", ServiceNamesSorted(got))
	}
	got2, err := ObservabilityActiveServices(base, []string{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got2) != 2 {
		t.Fatalf("empty roots slice: want full map, got %v", ServiceNamesSorted(got2))
	}
}

func TestObservabilityActiveServices_unknownRoot(t *testing.T) {
	base := map[string]Service{"a": {}}
	_, err := ObservabilityActiveServices(base, []string{"nope"}, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestObservabilityActiveServices_explicitOnly(t *testing.T) {
	base := map[string]Service{
		"web": {DependsOn: Dependencies{{Service: "api", Condition: ConditionStarted}}},
		"api": {},
	}
	got, err := ObservabilityActiveServices(base, []string{"web"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(ServiceNamesSorted(got), ","); got != "web" {
		t.Fatalf("want web only, got %q", got)
	}
}

func TestObservabilityActiveServices_dependentsExpand(t *testing.T) {
	base := map[string]Service{
		"db":   {},
		"api":  {DependsOn: Dependencies{{Service: "db", Condition: ConditionStarted}}},
		"web":  {DependsOn: Dependencies{{Service: "api", Condition: ConditionStarted}}},
		"jobs": {DependsOn: Dependencies{{Service: "api", Condition: ConditionStarted}}},
	}
	got, err := ObservabilityActiveServices(base, []string{"db"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(ServiceNamesSorted(got), ","); got != "api,db,jobs,web" {
		t.Fatalf("want transitive dependents of db, got %q", got)
	}
}
