package expand

import "testing"

func TestStringSubst(t *testing.T) {
	m := map[string]string{"A": "1", "EMPTY": ""}
	if got := String("x${A}y", m); got != "x1y" {
		t.Fatalf("got %q", got)
	}
	if got := String("${MISSING:-d}", m); got != "d" {
		t.Fatalf("got %q", got)
	}
	if got := String("${EMPTY:-d}", m); got != "d" {
		t.Fatalf("empty should use default, got %q", got)
	}
}
