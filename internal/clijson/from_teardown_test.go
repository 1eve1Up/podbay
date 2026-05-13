package clijson

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/1eve1Up/podbay/internal/teardown"
	"github.com/1eve1Up/podbay/internal/validate"
)

func TestFromTeardown_successNoContainers(t *testing.T) {
	res := teardown.TeardownResult{Project: "demo", Network: "podbay_demo"}
	d := FromTeardown("/app/podbay.yaml", "demo", nil, nil, false, res, nil)
	if d.Kind != KindTeardown || d.Status != StatusOK {
		t.Fatalf("kind/status = %s/%s", d.Kind, d.Status)
	}
	if len(d.Issues) != 0 {
		t.Fatalf("issues = %v", d.Issues)
	}
	raw, _ := json.Marshal(d)
	if strings.Contains(string(raw), "containers_removed") {
		t.Fatalf("expected omit containers_removed: %s", raw)
	}
}

func TestFromTeardown_successWithNetworkWarn(t *testing.T) {
	res := teardown.TeardownResult{
		Project:        "demo",
		Network:        "podbay_demo",
		NetworkWarning: "no such network",
		ContainerNames: []string{"podbay_demo_api"},
		NetworkRemoved: false,
		NetworkKept:    false,
	}
	d := FromTeardown("/app/podbay.yaml", "demo", []string{"dev"}, nil, false, res, nil)
	if d.Status != StatusOK {
		t.Fatalf("status = %s", d.Status)
	}
	if len(d.Issues) != 1 || d.Issues[0].Code != teardown.CodeNetworkWarn || d.Issues[0].Level != validate.LevelWarn {
		t.Fatalf("issues = %#v", d.Issues)
	}
	if len(d.ContainersRemoved) != 1 || d.ContainersRemoved[0] != "podbay_demo_api" {
		t.Fatalf("containers_removed = %v", d.ContainersRemoved)
	}
}

func TestFromTeardown_fatalPodman(t *testing.T) {
	res := teardown.TeardownResult{Project: "demo"}
	err := teardown.NewFatalError(teardown.CodePodmanError, errors.New("missing podman"))
	d := FromTeardown("/x.yaml", "demo", nil, nil, false, res, err)
	if d.Status != StatusFailed || len(d.Issues) != 1 {
		t.Fatalf("doc = %#v", d)
	}
	if d.Issues[0].Code != teardown.CodePodmanError {
		t.Fatalf("code = %q", d.Issues[0].Code)
	}
}

func TestTeardownLoadError(t *testing.T) {
	d := TeardownLoadError("", "demo", nil, "no file")
	if d.Kind != KindTeardown || d.Status != StatusFailed {
		t.Fatal(d)
	}
	if d.Issues[0].Code != "teardown_load_error" {
		t.Fatal(d.Issues)
	}
}

func TestFromTeardown_partialDeployMeta(t *testing.T) {
	res := teardown.TeardownResult{Project: "demo", Network: "podbay_demo"}
	d := FromTeardown("/c.yaml", "demo", []string{"dev"}, []string{"web"}, true, res, nil)
	if len(d.DeployServices) != 1 || d.DeployServices[0] != "web" {
		t.Fatalf("deploy_services = %#v", d.DeployServices)
	}
	if !d.DependentsExpand {
		t.Fatal("expected dependents_expand")
	}
}

func TestFromTeardown_pathNormalization(t *testing.T) {
	d := FromTeardown("/app/./podbay.yaml", "demo", nil, nil, false, teardown.TeardownResult{}, nil)
	if d.ContractPath != "/app/podbay.yaml" {
		t.Fatalf("contract_path = %q", d.ContractPath)
	}
}
