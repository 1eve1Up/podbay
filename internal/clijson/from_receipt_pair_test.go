package clijson

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/1eve1Up/podbay/internal/diff"
	"github.com/1eve1Up/podbay/internal/receipt"
	"github.com/1eve1Up/podbay/internal/validate"
)

func rec(project, contract string, profiles []string, svcs []receipt.ServiceRecord) *receipt.Receipt {
	return &receipt.Receipt{
		FormatVersion: receipt.CurrentFormatVersion,
		GeneratedAt:   time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		ContractPath:  contract,
		Project:       project,
		Profiles:      profiles,
		Services:      svcs,
	}
}

func TestFromDiff_marshaledJSON_omitsReceiptPair(t *testing.T) {
	res := diff.DriftResult{
		Project: "demo",
		Services: []diff.ServiceDrift{
			{Name: "api", ContainerName: "podbay_demo_api", Status: diff.StatusOK, State: "running"},
		},
	}
	d := FromDiff("/app/podbay.yaml", "demo", nil, res)
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["receipt_pair"]; ok {
		t.Fatalf("contract-runtime FromDiff must omit receipt_pair:\n%s", string(raw))
	}
}

func TestFromReceiptPairDiff_noDrift_omitsServicesStatus(t *testing.T) {
	a := rec("p", "/x.yaml", nil, nil)
	b := rec("p", "/x.yaml", nil, nil)
	d := FromReceiptPairDiff(receipt.CompareReceipts(a, b))
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["services_status"]; ok {
		t.Errorf("omit services_status:\n%s", string(raw))
	}
	if _, ok := m["issues"]; ok {
		t.Errorf("omit issues when no drift:\n%s", string(raw))
	}
}

func TestFromReceiptPairDiff_drift_issuesAndPair(t *testing.T) {
	a := rec("p1", "/a.yaml", []string{"dev"}, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1", Image: "i1"},
	})
	b := rec("p2", "/b.yaml", []string{"prod"}, []receipt.ServiceRecord{
		{Service: "web", ContainerName: "n2", Image: "i2"},
	})
	d := FromReceiptPairDiff(receipt.CompareReceipts(a, b))
	if d.Status != StatusFailed || d.Drift == nil || !*d.Drift {
		t.Fatalf("want drift failed: %+v", d)
	}
	if d.ReceiptPair == nil || d.ReceiptPair.ProjectMatch {
		t.Fatalf("pair: %+v", d.ReceiptPair)
	}
	// 3 global + 2 service-level issues (api removed, web added)
	if len(d.Issues) != 5 {
		t.Fatalf("issues len=%d: %+v", len(d.Issues), d.Issues)
	}
}

func TestFromReceiptPairDiff_mountsSurfaceInReceiptPair(t *testing.T) {
	m1 := []receipt.MountSpec{{Type: "bind", Source: "/a", Destination: "/b"}}
	m2 := []receipt.MountSpec{{Type: "bind", Source: "/a", Destination: "/c"}}
	a := rec("p", "/x.yaml", nil, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1", Mounts: &m1},
	})
	b := rec("p", "/x.yaml", nil, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1", Mounts: &m2},
	})
	d := FromReceiptPairDiff(receipt.CompareReceipts(a, b))
	if d.ReceiptPair == nil || len(d.ReceiptPair.Services) != 1 {
		t.Fatalf("pair: %+v", d.ReceiptPair)
	}
	row := d.ReceiptPair.Services[0]
	if row.First == nil || row.First.Mounts == nil || len(*row.First.Mounts) != 1 {
		t.Fatalf("first mounts: %+v", row.First)
	}
	if (*row.First.Mounts)[0].Destination != "/b" {
		t.Fatalf("mount %+v", (*row.First.Mounts)[0])
	}
}

func TestFromReceiptPairDiff_envIncomparable_warnIssue(t *testing.T) {
	e := []receipt.EnvVar{{Name: "K", Value: "v"}}
	a := rec("p", "/x.yaml", nil, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1", Env: &e},
	})
	b := rec("p", "/x.yaml", nil, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1"},
	})
	d := FromReceiptPairDiff(receipt.CompareReceipts(a, b))
	if d.Status != StatusOK || d.Drift == nil || *d.Drift {
		t.Fatalf("want ok + no drift: %+v", d)
	}
	var warn *Issue
	for i := range d.Issues {
		if d.Issues[i].Code == receipt.CodeEnvIncomparable {
			warn = &d.Issues[i]
			break
		}
	}
	if warn == nil || warn.Level != validate.LevelWarn {
		t.Fatalf("issues: %+v", d.Issues)
	}
}

func TestReceiptPairDiffError_omitsDrift(t *testing.T) {
	d := ReceiptPairDiffError(CodeReceiptDiffLoadError, "open /no: no such file")
	raw, err := MarshalIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["drift"]; ok {
		t.Fatalf("load failure must omit drift:\n%s", string(raw))
	}
	if m["status"] != StatusFailed {
		t.Fatalf("status=%v", m["status"])
	}
	issues, _ := m["issues"].([]any)
	if len(issues) != 1 {
		t.Fatalf("issues=%v", issues)
	}
}
