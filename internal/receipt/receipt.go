// Package receipt defines the deploy receipt JSON schema (v1) and atomic writes after deploy.
package receipt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CurrentFormatVersion is the only format version this codebase emits.
const CurrentFormatVersion = 1

// StatusOK is the status written on successful deploy receipts (evidence foundation).
const StatusOK = "ok"

// StatusFailed is the status written on failure/attempt receipts (health-gate and related).
const StatusFailed = "failed"

// Receipt records a Podbay deploy attempt for agents and CI (v1).
// Evidence fields (DeployID, ContractDigest, Status, DeployServices, DependentsExpand)
// are optional on decode so pre-Sprint-34 receipts remain valid; new writers should set them.
type Receipt struct {
	FormatVersion int             `json:"format_version"`
	GeneratedAt   time.Time       `json:"generated_at"`
	ContractPath  string          `json:"contract_path"`
	Project       string          `json:"project"`
	Profiles      []string        `json:"profiles,omitempty"`
	Services      []ServiceRecord `json:"services"`
	// DeployID correlates a receipt with a deploy invocation (optional on legacy receipts).
	DeployID string `json:"deploy_id,omitempty"`
	// ContractDigest is sha256: hex of the contract file bytes used for deploy (optional on legacy).
	ContractDigest string `json:"contract_digest,omitempty"`
	// Status is "ok" on successful writes, "failed" on attempt receipts; empty on legacy receipts.
	Status string `json:"status,omitempty"`
	// DeployServices lists partial-deploy roots when selection applied (same as deploy --json).
	DeployServices []string `json:"deploy_services,omitempty"`
	// DependentsExpand is true when partial roots were set and --dependents was used.
	DependentsExpand bool `json:"dependents_expand,omitempty"`
	// Failure holds attempt-receipt summary fields when Status is failed (optional on decode).
	Failure *FailureSummary `json:"failure,omitempty"`
}

// FailureSummary records why an attempt receipt was written (health-gate aligned).
// Fields are optional on decode; writers should set Code and Message when known.
type FailureSummary struct {
	// Service is the service that failed the health gate (or external dep name).
	Service string `json:"service,omitempty"`
	// Code is a stable issue code (e.g. deploy_health_timeout, deploy_health_probe_failed).
	Code string `json:"code,omitempty"`
	// Class is timeout or probe_error when known.
	Class string `json:"class,omitempty"`
	// ProbeKind is http or exec when known.
	ProbeKind string `json:"probe_kind,omitempty"`
	// Message is a human/agent-readable failure detail.
	Message string `json:"message,omitempty"`
	// ExternalDep is true when the failure is for a dependency outside the partial deploy set.
	ExternalDep bool `json:"external_dep,omitempty"`
	// RequestedBy is the partial-deploy service that triggered an external dependency health wait.
	RequestedBy string `json:"requested_by,omitempty"`
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
	// Evidence fields are optional (legacy receipts omit them). When present, constrain shape.
	if r.Status != "" && r.Status != StatusOK && r.Status != StatusFailed {
		return fmt.Errorf("receipt: unsupported status %q (want %q, %q, or empty)", r.Status, StatusOK, StatusFailed)
	}
	if r.Failure != nil {
		if err := validateFailureSummary(r.Failure); err != nil {
			return err
		}
	}
	if r.ContractDigest != "" && !strings.HasPrefix(r.ContractDigest, "sha256:") {
		return fmt.Errorf("receipt: contract_digest must start with sha256:")
	}
	if r.DeployID != "" {
		trimmed := strings.TrimSpace(r.DeployID)
		if trimmed == "" || trimmed != r.DeployID {
			return fmt.Errorf("receipt: deploy_id must be non-empty without surrounding whitespace")
		}
	}
	return nil
}

func validateFailureSummary(f *FailureSummary) error {
	if f == nil {
		return nil
	}
	if f.Code != "" {
		trimmed := strings.TrimSpace(f.Code)
		if trimmed == "" || trimmed != f.Code {
			return fmt.Errorf("receipt: failure.code must be non-empty without surrounding whitespace")
		}
	}
	if f.Class != "" && f.Class != "timeout" && f.Class != "probe_error" {
		return fmt.Errorf("receipt: failure.class unsupported %q (want timeout, probe_error, or empty)", f.Class)
	}
	if f.ProbeKind != "" && f.ProbeKind != "http" && f.ProbeKind != "exec" {
		return fmt.Errorf("receipt: failure.probe_kind unsupported %q (want http, exec, or empty)", f.ProbeKind)
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
	// Sync the file body before close so a crash between rename and writeback cannot
	// leave the target path pointing at a 0-byte receipt. The directory fsync below
	// likewise persists the rename itself.
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("receipt: sync: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("receipt: close: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("receipt: rename to %s: %w", path, err)
	}
	cleanup = false
	if dirF, err := os.Open(dir); err == nil {
		_ = dirF.Sync()
		_ = dirF.Close()
	}
	return nil
}
