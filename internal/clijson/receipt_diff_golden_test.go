package clijson

import (
	_ "embed"
	"strings"
	"testing"
	"time"

	"github.com/1eve1Up/podbay/internal/receipt"
)

//go:embed testdata/receipt_pair_diff_no_drift.json
var goldenReceiptPairDiffNoDrift string

//go:embed testdata/receipt_pair_diff_drift.json
var goldenReceiptPairDiffDrift string

func TestFromReceiptDiff_matchesGoldenFiles(t *testing.T) {
	t.Run("no_drift", func(t *testing.T) {
		a := rec("demo", "/app/p.yaml", []string{"a", "b"}, []receipt.ServiceRecord{
			{Service: "api", ContainerName: "n1", Image: "img:v1"},
		})
		b := rec("demo", "/app/p.yaml", []string{"b", "a"}, []receipt.ServiceRecord{
			{Service: "api", ContainerName: "n1", Image: "img:v1"},
		})
		raw, err := MarshalIndent(FromReceiptDiff(receipt.CompareReceipts(a, b)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(string(raw)) != strings.TrimSpace(goldenReceiptPairDiffNoDrift) {
			t.Fatalf("golden mismatch no_drift.\n got:\n%s\nwant:\n%s", string(raw), goldenReceiptPairDiffNoDrift)
		}
	})
	t.Run("drift", func(t *testing.T) {
		a := &receipt.Receipt{
			FormatVersion: receipt.CurrentFormatVersion,
			GeneratedAt:   time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
			ContractPath:  "/a.yaml",
			Project:       "p1",
			Profiles:      []string{"dev"},
			Services:      []receipt.ServiceRecord{{Service: "api", ContainerName: "n1", Image: "i1"}},
		}
		b := &receipt.Receipt{
			FormatVersion: receipt.CurrentFormatVersion,
			GeneratedAt:   time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
			ContractPath:  "/b.yaml",
			Project:       "p2",
			Profiles:      []string{"prod"},
			Services:      []receipt.ServiceRecord{{Service: "web", ContainerName: "n2", Image: "i2"}},
		}
		raw, err := MarshalIndent(FromReceiptDiff(receipt.CompareReceipts(a, b)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(string(raw)) != strings.TrimSpace(goldenReceiptPairDiffDrift) {
			t.Fatalf("golden mismatch drift.\n got:\n%s\nwant:\n%s", string(raw), goldenReceiptPairDiffDrift)
		}
	})
}

func TestFromReceiptDiff_sameAsFromReceiptPairDiff(t *testing.T) {
	a := rec("p", "/x.yaml", nil, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n1", Image: "i1"},
	})
	b := rec("p", "/x.yaml", nil, []receipt.ServiceRecord{
		{Service: "api", ContainerName: "n2", Image: "i2"},
	})
	res := receipt.CompareReceipts(a, b)
	r1, err := MarshalIndent(FromReceiptDiff(res))
	if err != nil {
		t.Fatal(err)
	}
	r2, err := MarshalIndent(FromReceiptPairDiff(res))
	if err != nil {
		t.Fatal(err)
	}
	if string(r1) != string(r2) {
		t.Fatalf("FromReceiptDiff != FromReceiptPairDiff\n%s\nvs\n%s", r1, r2)
	}
}
