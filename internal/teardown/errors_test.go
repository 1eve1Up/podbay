package teardown

import (
	"errors"
	"testing"
)

func TestFatalCode_fatalError(t *testing.T) {
	err := NewFatalError(CodeListError, errors.New("x"))
	if got := FatalCode(err); got != CodeListError {
		t.Fatalf("FatalCode = %q, want %q", got, CodeListError)
	}
}

func TestFatalCode_plain(t *testing.T) {
	if got := FatalCode(errors.New("plain")); got != "" {
		t.Fatalf("FatalCode(plain) = %q, want empty", got)
	}
}

func TestFatalError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	err := NewFatalError(CodePodmanError, inner)
	if !errors.Is(err, inner) {
		t.Fatal("errors.Is should find inner")
	}
}
