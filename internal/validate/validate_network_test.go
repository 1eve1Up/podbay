package validate

import (
	"path/filepath"
	"testing"

	"github.com/1eve1Up/podbay/internal/spec"
)

func TestRun_networkInvalidMultiMissing(t *testing.T) {
	t.Parallel()
	c := &spec.Contract{
		Version: "1",
		Services: map[string]spec.Service{
			"a": {Image: "x", Networks: []string{"n1"}},
			"b": {Image: "y"},
		},
		Networks: map[string]spec.Network{"n1": {}, "n2": {}},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	res := Run(c, path, nil, nil, false)
	var found bool
	for _, r := range res {
		if r.Code == "network_invalid" && r.Service == "b" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected network_invalid for service b: %#v", res)
	}
}

func TestRun_networkDriverUnsupported(t *testing.T) {
	t.Parallel()
	c := &spec.Contract{
		Version:  "1",
		Services: map[string]spec.Service{"a": {Image: "x"}},
		Networks: map[string]spec.Network{"n1": {Driver: "overlay"}},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	res := Run(c, path, nil, nil, false)
	var found bool
	for _, r := range res {
		if r.Code == "network_driver_unsupported" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected network_driver_unsupported: %#v", res)
	}
}

func TestRun_externalNetworkRejectsNonBridgeDriver(t *testing.T) {
	t.Parallel()
	c := &spec.Contract{
		Version: "1",
		Services: map[string]spec.Service{
			"a": {Image: "x", Networks: []string{"ext"}},
		},
		Networks: map[string]spec.Network{"ext": {External: true, Driver: "overlay"}},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "podbay.yaml")
	res := Run(c, path, nil, nil, false)
	var found bool
	for _, r := range res {
		if r.Code == "network_driver_unsupported" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected network_driver_unsupported for external+overlay: %#v", res)
	}
}
