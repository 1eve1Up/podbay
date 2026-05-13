// Package receipt defines the deploy receipt JSON schema (v1) and atomic writes after deploy.
package receipt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CurrentFormatVersion is the only format version this codebase emits.
const CurrentFormatVersion = 1

// Receipt records a successful Podbay deploy for agents and CI (v1).
type Receipt struct {
	FormatVersion int             `json:"format_version"`
	GeneratedAt   time.Time       `json:"generated_at"`
	ContractPath  string          `json:"contract_path"`
	Project       string          `json:"project"`
	Profiles      []string        `json:"profiles,omitempty"`
	Services      []ServiceRecord `json:"services"`
}

// EnvVar is one container environment variable captured on the receipt (optional).
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// MountSpec is one mount or volume attachment captured on the receipt (optional).
type MountSpec struct {
	Type        string `json:"type,omitempty"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// ServiceRecord is one active service’s container identity after deploy.
type ServiceRecord struct {
	Service       string `json:"service"`
	ContainerName string `json:"container_name"`
	ContainerID   string `json:"container_id,omitempty"`
	Image         string `json:"image,omitempty"`
	// Env is nil when omitted from JSON (v1 receipts). Non-nil points to recorded env (possibly empty).
	Env *[]EnvVar `json:"env,omitempty"`
	// Mounts is nil when omitted. Non-nil points to recorded mounts (possibly empty).
	Mounts *[]MountSpec `json:"mounts,omitempty"`
}

// Validate checks required v1 fields (after JSON decode, time fields parsed).
func Validate(r *Receipt) error {
	if r == nil {
		return fmt.Errorf("receipt: nil")
	}
	if r.FormatVersion != CurrentFormatVersion {
		return fmt.Errorf("receipt: unsupported format_version %d (want %d)", r.FormatVersion, CurrentFormatVersion)
	}
	if r.GeneratedAt.IsZero() {
		return fmt.Errorf("receipt: generated_at required")
	}
	if r.ContractPath == "" {
		return fmt.Errorf("receipt: contract_path required")
	}
	if r.Project == "" {
		return fmt.Errorf("receipt: project required")
	}
	for i, s := range r.Services {
		if s.Service == "" {
			return fmt.Errorf("receipt: services[%d].service required", i)
		}
		if s.ContainerName == "" {
			return fmt.Errorf("receipt: services[%d].container_name required", i)
		}
	}
	return nil
}

// Encode returns pretty-printed JSON for a validated receipt.
func Encode(r *Receipt) ([]byte, error) {
	if err := Validate(r); err != nil {
		return nil, err
	}
	return json.MarshalIndent(r, "", "  ")
}

// Decode parses and validates receipt JSON.
func Decode(data []byte) (*Receipt, error) {
	var r Receipt
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("receipt: decode: %w", err)
	}
	if err := Validate(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// New builds a v1 receipt with the current UTC timestamp and format version.
func New(contractPath, project string, profiles []string, services []ServiceRecord) *Receipt {
	return &Receipt{
		FormatVersion: CurrentFormatVersion,
		GeneratedAt:   time.Now().UTC(),
		ContractPath:  contractPath,
		Project:       project,
		Profiles:      profiles,
		Services:      services,
	}
}

// WriteAtomic writes validated receipt JSON to path using a temp file in the same directory and rename
// (no partial file on failure).
func WriteAtomic(path string, r *Receipt) error {
	data, err := Encode(r)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("receipt: mkdir %s: %w", dir, err)
	}
	f, err := os.CreateTemp(dir, ".podbay-receipt-*")
	if err != nil {
		return fmt.Errorf("receipt: temp file: %w", err)
	}
	tmpName := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("receipt: write: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("receipt: close: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("receipt: rename to %s: %w", path, err)
	}
	cleanup = false
	return nil
}
