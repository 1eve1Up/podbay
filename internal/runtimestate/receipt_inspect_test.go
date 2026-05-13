package runtimestate

import (
	"testing"

	"github.com/1eve1Up/podbay/internal/receipt"
)

func TestParseContainerForReceiptJSON_sample(t *testing.T) {
	raw := `[
  {
    "Id": "abc123",
    "Image": "docker.io/library/nginx:latest",
    "Config": {
      "Env": ["PATH=/usr/bin", "FOO=bar", "BAZ=qux"]
    },
    "Mounts": [
      {"Type": "bind", "Source": "/host/x", "Destination": "/ctr/x"}
    ]
  }
]`
	id, img, env, mounts, err := ParseContainerForReceiptJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if id != "abc123" || img != "docker.io/library/nginx:latest" {
		t.Fatalf("id/img: %q %q", id, img)
	}
	if env == nil || mounts == nil {
		t.Fatal("expected non-nil env and mounts pointers")
	}
	if len(*env) != 3 {
		t.Fatalf("env len %d", len(*env))
	}
	if (*env)[0].Name != "BAZ" || (*env)[0].Value != "qux" {
		t.Fatalf("sorted env[0] = %+v", (*env)[0])
	}
	if len(*mounts) != 1 || (*mounts)[0].Source != "/host/x" {
		t.Fatalf("mounts %+v", *mounts)
	}
}

func TestParseContainerForReceiptJSON_emptySlices(t *testing.T) {
	raw := `[{"Id":"x","Image":"i","Config":{"Env":[]},"Mounts":[]}]`
	_, _, env, mounts, err := ParseContainerForReceiptJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(*env) != 0 || len(*mounts) != 0 {
		t.Fatalf("want empty, got env=%d mounts=%d", len(*env), len(*mounts))
	}
}

func TestParseContainerForReceiptJSON_roundTripMatchesReceiptCompare(t *testing.T) {
	raw := `[{"Id":"id1","Image":"img1","Config":{"Env":["Z=1","A=2"]},"Mounts":[]}]`
	_, _, env, mounts, err := ParseContainerForReceiptJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	a := receiptFixtureR("p", "/c.yaml", []receipt.ServiceRecord{
		{Service: "s", ContainerName: "c", Env: env, Mounts: mounts},
	})
	b := receiptFixtureR("p", "/c.yaml", []receipt.ServiceRecord{
		{Service: "s", ContainerName: "c", Env: env, Mounts: mounts},
	})
	res := receipt.CompareReceipts(a, b)
	if res.Drift || len(res.Services) != 0 {
		t.Fatalf("compare same: %+v", res)
	}
}

func receiptFixtureR(project, contract string, svcs []receipt.ServiceRecord) *receipt.Receipt {
	return &receipt.Receipt{
		FormatVersion: receipt.CurrentFormatVersion,
		ContractPath:  contract,
		Project:       project,
		Services:      svcs,
	}
}
