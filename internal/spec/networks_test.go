package spec

import "testing"

func TestEffectiveServiceNetworks_legacy(t *testing.T) {
	t.Parallel()
	c := &Contract{Networks: nil, Services: map[string]Service{"a": {}}}
	got, err := EffectiveServiceNetworks(c, "a", c.Services["a"])
	if err != nil || got != nil {
		t.Fatalf("got %v err %v", got, err)
	}
}

func TestEffectiveServiceNetworks_serviceNetworksWithoutTopLevel(t *testing.T) {
	t.Parallel()
	c := &Contract{Services: map[string]Service{"a": {Networks: []string{"x"}}}}
	_, err := EffectiveServiceNetworks(c, "a", c.Services["a"])
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEffectiveServiceNetworks_singleImplicit(t *testing.T) {
	t.Parallel()
	c := &Contract{
		Networks: map[string]Network{"n1": {}},
		Services: map[string]Service{"a": {}},
	}
	got, err := EffectiveServiceNetworks(c, "a", c.Services["a"])
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "n1" {
		t.Fatalf("got %v", got)
	}
}

func TestEffectiveServiceNetworks_multiRequiresExplicit(t *testing.T) {
	t.Parallel()
	c := &Contract{
		Networks: map[string]Network{"n1": {}, "n2": {}},
		Services: map[string]Service{"a": {}},
	}
	_, err := EffectiveServiceNetworks(c, "a", c.Services["a"])
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateNetworkDrivers(t *testing.T) {
	t.Parallel()
	if err := ValidateNetworkDrivers(map[string]Network{"a": {Driver: "bridge"}}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateNetworkDrivers(map[string]Network{"a": {Driver: "overlay"}}); err == nil {
		t.Fatal("expected error")
	}
}
