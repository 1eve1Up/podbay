package logs

import (
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/spec"
)

func TestActiveServices_matchesObservability(t *testing.T) {
	c := &spec.Contract{
		Services: map[string]spec.Service{
			"web": {DependsOn: spec.Dependencies{{Service: "api", Condition: spec.ConditionStarted}}},
			"api": {},
		},
	}
	got, err := ActiveServices(c, nil, []string{"api"}, true)
	if err != nil {
		t.Fatal(err)
	}
	want := "api,web"
	if s := strings.Join(spec.ServiceNamesSorted(got), ","); s != want {
		t.Fatalf("got %q want %q", s, want)
	}
}
