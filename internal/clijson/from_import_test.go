package clijson

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/1eve1Up/podbay/internal/composefile"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestFromImportComposeError_usesImportFailure(t *testing.T) {
	inner := composefile.NewImportFailure(composefile.CodeImportIncludeCycle, "composefile: include cycle involving \"/x\"")
	doc := FromImportComposeError("/app/compose.yml", inner)
	if doc.Kind != KindImportCompose || doc.Status != StatusFailed {
		t.Fatalf("doc=%+v", doc)
	}
	if doc.ContractPath == "" {
		t.Fatal("expected contract_path")
	}
	if len(doc.Issues) != 1 || doc.Issues[0].Code != composefile.CodeImportIncludeCycle {
		t.Fatalf("issues=%+v", doc.Issues)
	}
}

func TestFromImportComposeError_genericWrapped(t *testing.T) {
	doc := FromImportComposeError("/app/c.yml", errors.New("composeimport: nope"))
	if doc.Issues[0].Code != "import_contract_error" {
		t.Fatalf("code=%q", doc.Issues[0].Code)
	}
}

func TestFromImportComposeError_nilErr(t *testing.T) {
	doc := FromImportComposeError("/x/a.yml", nil)
	if doc.Status != StatusOK || len(doc.Issues) != 0 {
		t.Fatalf("%+v", doc)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["issues"] != nil {
		t.Fatalf("expected no issues key or empty: %s", string(raw))
	}
}

func TestFromImportComposeSuccess_shape(t *testing.T) {
	yamlBody := "version: \"1\"\nservices:\n  web:\n    image: nginx\n"
	c := &spec.Contract{
		Version:  "1",
		Project:  "demo",
		Services: map[string]spec.Service{"web": {Image: "nginx"}},
	}
	doc := FromImportComposeSuccess("/abs/compose.yml", []byte(yamlBody), c, "/out/podbay.yaml")
	if doc.Kind != KindImportCompose || doc.Status != StatusOK {
		t.Fatalf("doc=%+v", doc)
	}
	if doc.ContractPath == "" || doc.ImportContractYAML != yamlBody {
		t.Fatalf("paths/body wrong: %+v", doc)
	}
	if doc.Project != "demo" || doc.ImportServiceCount != 1 {
		t.Fatalf("project/count: %+v", doc)
	}
	if doc.ImportOutputPath == "" {
		t.Fatal("expected output_path")
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["issues"] != nil {
		t.Fatalf("issues should be absent: %s", string(raw))
	}
	if m["contract_yaml"] != yamlBody {
		t.Fatalf("contract_yaml=%v", m["contract_yaml"])
	}
}
