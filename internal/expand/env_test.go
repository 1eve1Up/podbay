package expand

import (
	"errors"
	"testing"
)

func TestStringSubst(t *testing.T) {
	m := map[string]string{"A": "1", "EMPTY": ""}
	if got := String("x${A}y", m); got != "x1y" {
		t.Fatalf("braced: got %q", got)
	}
	if got := String("${MISSING:-d}", m); got != "d" {
		t.Fatalf(":- on missing: got %q", got)
	}
	if got := String("${EMPTY:-d}", m); got != "d" {
		t.Fatalf(":- on empty: got %q", got)
	}
}

func TestStringSubstExtendedForms(t *testing.T) {
	m := map[string]string{"A": "1", "EMPTY": ""}

	if got := String("$A", m); got != "1" {
		t.Fatalf("bare $VAR: got %q", got)
	}
	if got := String("$MISSING", m); got != "$MISSING" {
		t.Fatalf("bare missing should pass through: got %q", got)
	}

	if got := String("${EMPTY-d}", m); got != "" {
		t.Fatalf("- on empty-but-set should keep empty value, got %q", got)
	}
	if got := String("${MISSING-d}", m); got != "d" {
		t.Fatalf("- on unset should use default, got %q", got)
	}

	if got := String("${A:+alt}", m); got != "alt" {
		t.Fatalf(":+ on non-empty: got %q", got)
	}
	if got := String("${EMPTY:+alt}", m); got != "" {
		t.Fatalf(":+ on empty should be empty: got %q", got)
	}
	if got := String("${EMPTY+alt}", m); got != "alt" {
		t.Fatalf("+ on set-but-empty: got %q", got)
	}
	if got := String("${MISSING+alt}", m); got != "" {
		t.Fatalf("+ on unset: got %q", got)
	}

	if got := String("$$LITERAL", m); got != "$LITERAL" {
		t.Fatalf("$$ escape: got %q", got)
	}
	if got := String("price=$$5", m); got != "price=$5" {
		t.Fatalf("$$ in mixed string: got %q", got)
	}
}

func TestSubstituteRequiredVar(t *testing.T) {
	m := map[string]string{"A": "1", "EMPTY": ""}

	if _, err := Substitute("${MISSING:?must-set}", m); err == nil {
		t.Fatalf(":? on missing should error")
	} else {
		var subErr *SubstitutionError
		if !errors.As(err, &subErr) || subErr.Var != "MISSING" {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if _, err := Substitute("${A:?must-set}", m); err != nil {
		t.Fatalf(":? on set should not error: %v", err)
	}

	if _, err := Substitute("${EMPTY:?must-set}", m); err == nil {
		t.Fatalf(":? on empty should error")
	}
	if _, err := Substitute("${EMPTY?must-set}", m); err != nil {
		t.Fatalf("? on set-but-empty should not error: %v", err)
	}
}
