package receipt

import (
	"strings"
	"testing"
)

func TestFormatReceiptDiff_noDrift(t *testing.T) {
	a := receiptFixture("demo", "/app/p.yaml", []string{"a", "b"}, []ServiceRecord{
		{Service: "api", ContainerName: "podbay_demo_api", Image: "img:v1"},
	})
	b := receiptFixture("demo", "/app/p.yaml", []string{"b", "a"}, []ServiceRecord{
		{Service: "api", ContainerName: "podbay_demo_api", Image: "img:v1"},
	})
	res := CompareReceipts(a, b)
	out := FormatReceiptDiff(res)
	if res.Drift {
		t.Fatal("CompareReceipts drift")
	}
	if !strings.Contains(out, "No drift: compared fields match") {
		t.Fatalf("missing no-drift line:\n%s", out)
	}
	if strings.Contains(out, "Drift detected") {
		t.Fatalf("unexpected drift line:\n%s", out)
	}
	if !strings.Contains(out, "Project: demo\n") {
		t.Fatalf("project line:\n%s", out)
	}
	if !strings.Contains(out, "Profiles: a, b\n") {
		t.Fatalf("profiles should sort for display:\n%s", out)
	}
	if !strings.Contains(out, "(no service-level differences)") {
		t.Fatalf("services section:\n%s", out)
	}
}

func TestFormatReceiptDiff_globalAndServiceDrift(t *testing.T) {
	a := receiptFixture("p1", "/a.yaml", []string{"dev"}, []ServiceRecord{
		{Service: "api", ContainerName: "n1", Image: "i1", ContainerID: "id1"},
	})
	b := receiptFixture("p2", "/b.yaml", []string{"prod"}, []ServiceRecord{
		{Service: "web", ContainerName: "n2", Image: "i2"},
	})
	res := CompareReceipts(a, b)
	out := FormatReceiptDiff(res)
	if !strings.Contains(out, "Drift detected") {
		t.Fatalf("want drift footer:\n%s", out)
	}
	if !strings.Contains(out, "Project: mismatch") || !strings.Contains(out, "p1") || !strings.Contains(out, "p2") {
		t.Fatalf("project mismatch block:\n%s", out)
	}
	if !strings.Contains(out, "Contract: mismatch") {
		t.Fatalf("contract mismatch:\n%s", out)
	}
	if !strings.Contains(out, "Profiles: mismatch") {
		t.Fatalf("profiles mismatch:\n%s", out)
	}
	if !strings.Contains(out, "[api] removed in second receipt") {
		t.Fatalf("api removed:\n%s", out)
	}
	if !strings.Contains(out, "[web] added in second receipt") {
		t.Fatalf("web added:\n%s", out)
	}
}

func TestFormatReceiptDiff_fieldChanges(t *testing.T) {
	a := receiptFixture("p", "/x.yaml", nil, []ServiceRecord{
		{Service: "api", ContainerName: "n1", Image: "i1", ContainerID: "c1"},
	})
	b := receiptFixture("p", "/x.yaml", nil, []ServiceRecord{
		{Service: "api", ContainerName: "n2", Image: "i2", ContainerID: "c2"},
	})
	out := FormatReceiptDiff(CompareReceipts(a, b))
	for _, sub := range []string{
		"image changed",
		`first:  "i1"`,
		`second: "i2"`,
		"container_name changed",
		`first:  "n1"`,
		"container_id changed",
		`first:  "c1"`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in:\n%s", sub, out)
		}
	}
}
