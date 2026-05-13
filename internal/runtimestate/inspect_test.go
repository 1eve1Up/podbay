package runtimestate

import (
	"strings"
	"testing"
)

func TestParseInspect_running(t *testing.T) {
	raw := `[{"State":{"Status":"running","ExitCode":0,"Error":""}}]`
	st, err := ParseInspect([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if st.State != "running" || st.ExitCode != 0 || st.Error != "" || st.Image != "" {
		t.Fatalf("got %+v", st)
	}
}

func TestParseInspect_imageNamePreferred(t *testing.T) {
	raw := `[{"Image":"sha256:abc","ImageName":"docker.io/library/nginx:alpine","State":{"Status":"running","ExitCode":0,"Error":""}}]`
	st, err := ParseInspect([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if st.Image != "docker.io/library/nginx:alpine" {
		t.Fatalf("image got %q", st.Image)
	}
}

func TestParseInspect_exited(t *testing.T) {
	raw := `[{"State":{"Status":"exited","ExitCode":1,"Error":"oops"}}]`
	st, err := ParseInspect([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if st.State != "exited" || st.ExitCode != 1 || st.Error != "oops" {
		t.Fatalf("got %+v", st)
	}
}

func TestParseInspect_emptyArray(t *testing.T) {
	_, err := ParseInspect([]byte(`[]`))
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty array error, got %v", err)
	}
}

func TestParseInspect_invalidJSON(t *testing.T) {
	_, err := ParseInspect([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error")
	}
}
