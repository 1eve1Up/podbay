package composeimport

import (
	"bytes"
	"fmt"

	"github.com/1eve1Up/podbay/internal/spec"
	"gopkg.in/yaml.v3"
)

// MarshalContract encodes a Podbay contract as YAML suitable for spec.Load.
// Indent is two spaces; a trailing newline is included.
func MarshalContract(c *spec.Contract) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("composeimport: nil contract")
	}
	root := contractToEmitRoot(c)
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		_ = enc.Close()
		return nil, fmt.Errorf("composeimport: encode yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
