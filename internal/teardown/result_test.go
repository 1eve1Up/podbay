package teardown

import (
	"errors"
	"testing"
)

func TestExitCode_nil(t *testing.T) {
	if got := ExitCode(nil); got != 0 {
		t.Fatalf("ExitCode(nil) = %d, want 0", got)
	}
}

func TestExitCode_error(t *testing.T) {
	if got := ExitCode(errors.New("boom")); got != 1 {
		t.Fatalf("ExitCode(err) = %d, want 1", got)
	}
}

func TestIssueCodes_nonEmpty(t *testing.T) {
	for _, c := range []string{
		CodePodmanError,
		CodeListError,
		CodeNetworkWarn,
		CodeVolumeError,
	} {
		if c == "" {
			t.Fatal("empty code constant")
		}
	}
}
