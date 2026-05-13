package runner

import (
	"strings"
	"testing"
)

func TestExecHealthFailureSuffix_runningContainer(t *testing.T) {
	s := execHealthFailureSuffix("podbay_x_api", "running exit=0")
	if !strings.Contains(s, "hint:") {
		t.Fatalf("expected hint, got %q", s)
	}
	if strings.Contains(s, "--- podman logs") {
		t.Fatalf("should not embed logs when not exited: %q", s)
	}
}
