package deploy

import (
	"testing"

	"github.com/1eve1Up/podbay/internal/receipt"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestBuildReceiptServiceRecords_mockInspect(t *testing.T) {
	prev := inspectContainerForReceipt
	t.Cleanup(func() { inspectContainerForReceipt = prev })
	inspectContainerForReceipt = func(containerName string) (string, string, *[]receipt.EnvVar, *[]receipt.MountSpec, error) {
		e := []receipt.EnvVar{}
		m := []receipt.MountSpec{}
		return "id:" + containerName, "registry/img:podman", &e, &m, nil
	}

	r := runner.New("demo")
	active := map[string]spec.Service{
		"web": {Image: "my:local"},
		"api": {},
	}
	order := []string{"api", "web"}
	recs, err := buildReceiptServiceRecords(r, order, active, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records", len(recs))
	}
	if recs[0].Service != "api" || recs[0].Image != "registry/img:podman" {
		t.Fatalf("api record %+v", recs[0])
	}
	if recs[1].Service != "web" || recs[1].Image != "my:local" {
		t.Fatalf("web record %+v", recs[1])
	}
	if recs[1].ContainerID == "" {
		t.Fatal("expected container id")
	}
}
