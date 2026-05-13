package spec

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExtraHostListMapForm(t *testing.T) {
	const y = `
extra_hosts:
  host.docker.internal: host-gateway
`
	var s struct {
		H ExtraHostList `yaml:"extra_hosts"`
	}
	if err := yaml.Unmarshal([]byte(y), &s); err != nil {
		t.Fatal(err)
	}
	if len(s.H) != 1 || s.H[0] != "host.docker.internal:host-gateway" {
		t.Fatalf("got %#v", []string(s.H))
	}
}
