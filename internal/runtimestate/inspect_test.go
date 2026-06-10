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

func TestParseInspectMany_twoContainers(t *testing.T) {
	raw := `[
		{"Name":"/podbay_demo_api","State":{"Status":"running","ExitCode":0,"Error":""}},
		{"Name":"podbay_demo_worker","State":{"Status":"exited","ExitCode":2,"Error":"fail"}}
	]`
	got, err := ParseInspectMany([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	api := got["podbay_demo_api"]
	if api == nil || api.State != "running" {
		t.Fatalf("api: %+v", api)
	}
	worker := got["podbay_demo_worker"]
	if worker == nil || worker.State != "exited" || worker.ExitCode != 2 || worker.Error != "fail" {
		t.Fatalf("worker: %+v", worker)
	}
}

func TestParseInspectMany_missingNameSkipped(t *testing.T) {
	raw := `[{"State":{"Status":"running","ExitCode":0,"Error":""}}]`
	got, err := ParseInspectMany([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %+v", got)
	}
}

func TestParseInspectMany_invalidJSON(t *testing.T) {
	_, err := ParseInspectMany([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMergeInspectResults(t *testing.T) {
	batch := map[string]*ContainerState{
		"podbay_demo_api": {State: "running"},
	}
	got := mergeInspectResults([]string{"podbay_demo_api", "podbay_demo_missing"}, batch)
	if got["podbay_demo_api"] == nil || got["podbay_demo_api"].State != "running" {
		t.Fatalf("api: %+v", got["podbay_demo_api"])
	}
	if got["podbay_demo_missing"] != nil {
		t.Fatalf("expected nil missing, got %+v", got["podbay_demo_missing"])
	}
}

func TestInspectContainers_empty(t *testing.T) {
	got, err := InspectContainers(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %+v", got)
	}
}

func TestParseInspectMany_exitedWithImage(t *testing.T) {
	raw := `[{"Name":"/c1","ImageName":"img:tag","State":{"Status":"exited","ExitCode":7,"Error":"e"}}]`
	got, err := ParseInspectMany([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	st := got["c1"]
	if st == nil || st.State != "exited" || st.ExitCode != 7 || st.Image != "img:tag" {
		t.Fatalf("got %+v", st)
	}
}
