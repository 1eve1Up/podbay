package deploy

import (
	"testing"

	"github.com/1eve1Up/podbay/internal/spec"
)

func TestBridgeDNSForContract_networkDNS(t *testing.T) {
	c := &spec.Contract{
		Podman: &spec.PodmanSection{NetworkDNS: []string{" 1.1.1.1 ", "8.8.8.8"}},
	}
	got := bridgeDNSForContract(c, nil)
	if len(got) != 2 || got[0] != "1.1.1.1" || got[1] != "8.8.8.8" {
		t.Fatalf("got %#v", got)
	}
}

func TestBridgeDNSForContract_disableDefault(t *testing.T) {
	c := &spec.Contract{
		Podman: &spec.PodmanSection{DisableDefaultBridgeDNS: true},
	}
	if got := bridgeDNSForContract(c, nil); len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestBridgeDNSForContract_customOverridesDisable(t *testing.T) {
	c := &spec.Contract{
		Podman: &spec.PodmanSection{
			DisableDefaultBridgeDNS: true,
			NetworkDNS:              []string{"9.9.9.9"},
		},
	}
	got := bridgeDNSForContract(c, nil)
	if len(got) != 1 || got[0] != "9.9.9.9" {
		t.Fatalf("got %#v", got)
	}
}
