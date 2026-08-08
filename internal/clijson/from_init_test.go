package clijson

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/1eve1Up/podbay/internal/composefile"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestFromInitFromCodebaseSuccess_shape(t *testing.T) {
	c := &spec.Contract{
		Version:  "1",
		Project:  "demo",
		Services: map[string]spec.Service{"web": {Image: "nginx"}},
	}
	doc := FromInitFromCodebaseSuccess("/out/podbay.yaml", "/src/compose.yaml", c)
	if doc.Kind != KindInit || doc.Status != StatusOK {
		t.Fatalf("%+v", doc)
	}
	if doc.ContractPath != "/out/podbay.yaml" || doc.ComposeSource != "/src/compose.yaml" {
		t.Fatalf("paths=%+v", doc)
	}
	if doc.SourceKind != InitSourceCompose || doc.DockerfileSource != "" {
		t.Fatalf("source=%+v", doc)
	}
	if doc.ImportServiceCount != 1 || doc.Project != "demo" {
		t.Fatalf("meta=%+v", doc)
	}
	if len(doc.NextActions) != 2 {
		t.Fatalf("next_actions=%v", doc.NextActions)
	}
	raw, err := MarshalIndent(doc)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["kind"] != KindInit || m["compose_source"] == nil || m["source_kind"] != InitSourceCompose {
		t.Fatalf("%s", raw)
	}
	if _, ok := m["dockerfile_source"]; ok {
		t.Fatalf("compose success must omit dockerfile_source: %s", raw)
	}
}

func TestFromInitDockerfileSuccess_shape(t *testing.T) {
	c := &spec.Contract{
		Version:  "1",
		Project:  "demo",
		Services: map[string]spec.Service{"app": {Image: "localhost/demo:local"}},
	}
	doc := FromInitDockerfileSuccess("/out/podbay.yaml", "/src/Dockerfile", c)
	if doc.Kind != KindInit || doc.Status != StatusOK {
		t.Fatalf("%+v", doc)
	}
	if doc.DockerfileSource != "/src/Dockerfile" || doc.ComposeSource != "" {
		t.Fatalf("paths=%+v", doc)
	}
	if doc.SourceKind != InitSourceDockerfile {
		t.Fatalf("source_kind=%q", doc.SourceKind)
	}
	if len(doc.NextActions) != 2 {
		t.Fatalf("next_actions=%v", doc.NextActions)
	}
	raw, err := MarshalIndent(doc)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["dockerfile_source"] == nil || m["source_kind"] != InitSourceDockerfile {
		t.Fatalf("%s", raw)
	}
	if _, ok := m["compose_source"]; ok {
		t.Fatalf("dockerfile success must omit compose_source: %s", raw)
	}
}

func TestFromInitError_targetExists(t *testing.T) {
	doc := FromInitError("/x/podbay.yaml", "", "", &InitTargetExistsError{Path: "/x/podbay.yaml"})
	if doc.Issues[0].Code != CodeInitTargetExists {
		t.Fatalf("%+v", doc.Issues)
	}
}

func TestFromInitError_discovery(t *testing.T) {
	inner := composefile.NewImportFailure(composefile.CodeComposeDiscoveryNotFound, "no compose")
	doc := FromInitError("/x/podbay.yaml", "", "", inner)
	if doc.Issues[0].Code != composefile.CodeComposeDiscoveryNotFound {
		t.Fatalf("%+v", doc.Issues)
	}
}

func TestFromInitError_codebaseNotFound(t *testing.T) {
	inner := composefile.NewImportFailure(composefile.CodeCodebaseDiscoveryNotFound, "neither")
	doc := FromInitError("/x/podbay.yaml", "", "", inner)
	if doc.Issues[0].Code != composefile.CodeCodebaseDiscoveryNotFound {
		t.Fatalf("%+v", doc.Issues)
	}
}

func TestFromInitError_generic(t *testing.T) {
	doc := FromInitError("", "", "", errors.New("boom"))
	if doc.Issues[0].Code != CodeInitError {
		t.Fatalf("%+v", doc.Issues)
	}
}
