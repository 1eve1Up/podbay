package validate

import (
	"bytes"
	"testing"
)

func TestRunOutcome_HasFailure(t *testing.T) {
	o := RunOutcome{Results: []Result{{OK: true, Level: LevelOK, Message: "x"}}}
	if o.HasFailure() {
		t.Fatal("unexpected failure")
	}
	o2 := RunOutcome{Results: []Result{{OK: false, Level: LevelFail, Message: "bad"}}}
	if !o2.HasFailure() {
		t.Fatal("expected failure")
	}
}

func TestRunOutcome_PrintText(t *testing.T) {
	var buf bytes.Buffer
	o := RunOutcome{Results: []Result{
		{OK: true, Level: LevelOK, Message: "ok1"},
		{OK: false, Level: LevelWarn, Message: "w"},
		{OK: false, Level: LevelFail, Message: "f"},
	}}
	if err := o.PrintText(&buf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if s != "✔ ok1\n⚠ w\n✖ f\n" {
		t.Fatalf("got %q", s)
	}
}
