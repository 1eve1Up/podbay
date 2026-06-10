package runtimestate

import (
	"testing"

	"github.com/1eve1Up/podbay/internal/runner"
)

func TestExtraContainerNamesWithProjectList(t *testing.T) {
	r := runner.New("demo")
	projectNames := []string{"podbay_demo_api", "podbay_demo_debug"}
	got := ExtraContainerNamesWithProjectList(r, []string{"api"}, projectNames)
	if len(got) != 1 || got[0] != "podbay_demo_debug" {
		t.Fatalf("got %v", got)
	}
}

func TestParsePsNameLines_commaSeparated(t *testing.T) {
	got := parsePsNameLines([]byte("a,b\n c \n"))
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("got %v", got)
	}
}

func TestListProjectContainerStates_parse(t *testing.T) {
	r := runner.New("demo")
	// Exercise parser via manual construction is covered by ListProjectContainerStates
	// integration when Podman is available; unit-test splitPsNames here.
	names := splitPsNames("foo, bar")
	if len(names) != 2 || names[0] != "foo" || names[1] != "bar" {
		t.Fatalf("got %v", names)
	}
	_ = r
}
