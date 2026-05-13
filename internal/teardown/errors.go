package teardown

import (
	"errors"
)

// FatalError is returned by Execute for failures that should yield CLI exit code 1
// and a structured clijson issue (Code identifies the teardown_* code).
type FatalError struct {
	Code string
	Err  error
}

func (e *FatalError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "teardown failed"
}

func (e *FatalError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// FatalCode returns the teardown issue code if err wraps *FatalError, else empty.
func FatalCode(err error) string {
	var fe *FatalError
	if errors.As(err, &fe) && fe != nil {
		return fe.Code
	}
	return ""
}

// NewFatalError wraps err with a stable issue code (use Code* constants).
func NewFatalError(code string, err error) error {
	if err == nil {
		return nil
	}
	return &FatalError{Code: code, Err: err}
}
