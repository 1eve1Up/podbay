// Package clijson provides a shared versioned JSON envelope for podbay --json CLI output
// (validate, deploy, receipt read), aligned with format_version conventions used by ps and explain.
package clijson

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/1eve1Up/podbay/internal/receipt"
	"github.com/1eve1Up/podbay/internal/validate"
)

// FormatVersion is bumped when breaking the shared CLI outcome document shape.
const FormatVersion = 1

// Kind identifies which command produced the document.
const (
	KindValidate    = "validate"
	KindDeploy      = "deploy"
	KindReceiptRead = "receipt_read"
	KindDiff        = "diff"
	KindTeardown    = "teardown"
	KindLogs        = "logs"
)

// Status is a coarse outcome for automation.
const (
	StatusOK     = "ok"
	StatusFailed = "failed"
)

// Document is the v1 top-level JSON object for validate/deploy/receipt/diff/teardown/logs --json.
//
// added the diff payload fields (Drift, ServicesStatus, Extras) as
// additive optional fields. They serialize only when the document represents
// a diff result, so existing validate/deploy/receipt JSON bytes are
// unchanged. Drift is a *bool so a KindDiff document can emit an explicit
// false (no drift) while non-diff documents omit the key entirely.
//
// dependents_expand (validate/deploy) is set only when partial CLI roots
// were given and --dependents was used.
//
// KindTeardown uses containers_removed and volumes_removed when present;
// other kinds leave them omitted.
//
// KindImportCompose is used for import compose --json failure envelopes only;
// success imports still emit YAML to stdout or -o (contract_path names the Compose source file).
//
// KindLogs is used for logs --json (success or failure); success includes log_body.
//
// Receipt-pair diff (KindDiff): when comparing two deploy receipts, set
// receipt_pair with compared fields; contract-vs-runtime diff omits it.
// Load/decode failures before CompareReceipts runs use ReceiptPairDiffError
// with stable issue codes (e.g. receipt_diff_load_error, receipt_diff_decode_error).
//
// receipt_pair_diff_version documents the receipt_pair payload shape (additive v2 fields).
// env_value_display_policy applies to receipt_pair env snapshots ("raw" vs "redacted").
const ReceiptPairDiffPayloadVersion = 2

type Document struct {
	FormatVersion int      `json:"format_version"`
	Kind          string   `json:"kind"`
	Status        string   `json:"status"`
	ContractPath  string   `json:"contract_path,omitempty"`
	Project       string   `json:"project,omitempty"`
	Profiles      []string `json:"profiles,omitempty"`
	// DeployServices lists explicit partial-deploy roots from the CLI (additive; omitted when empty).
	DeployServices []string `json:"deploy_services,omitempty"`
	// DependentsExpand is true when partial roots were set and --dependents was used (validate/deploy --json).
	DependentsExpand bool                `json:"dependents_expand,omitempty"`
	Issues           []Issue             `json:"issues,omitempty"`
	ReceiptPath      string              `json:"receipt_path,omitempty"`
	Receipt          json.RawMessage     `json:"receipt,omitempty"`
	Drift            *bool               `json:"drift,omitempty"`
	ServicesStatus   []DiffServiceStatus `json:"services_status,omitempty"`
	Extras           []string            `json:"extras,omitempty"`
	// EnvValueDisplayPolicy documents how env values appear in receipt_pair ("raw", "redacted").
	EnvValueDisplayPolicy string `json:"env_value_display_policy,omitempty"`
	// ReceiptPairDiffVersion is set with receipt_pair success payloads (shape v2+).
	ReceiptPairDiffVersion int `json:"receipt_pair_diff_version,omitempty"`
	// ReceiptPair is set for KindDiff receipt-vs-receipt comparisons only.
	ReceiptPair *ReceiptPairDiff `json:"receipt_pair,omitempty"`
	// Teardown-only (KindTeardown): container names removed; volume Podman names removed when -v.
	ContainersRemoved []string `json:"containers_removed,omitempty"`
	VolumesRemoved    []string `json:"volumes_removed,omitempty"`
	// Logs-only (KindLogs): one-shot podman logs capture when using logs --json without --follow.
	LogsService       string  `json:"service,omitempty"`
	LogsContainerName string  `json:"container_name,omitempty"`
	LogsTail          int     `json:"tail,omitempty"`
	LogsSince         string  `json:"since,omitempty"`
	LogsBody          *string `json:"log_body,omitempty"`
}

// DiffServiceStatus is one expected service's runtime outcome inside a
// KindDiff document. Status mirrors the diff package's per-service status
// strings (ok, missing, wrong_state, inspect_error). State, ExitCode, and
// Error are populated when the runtime supplied them; they omit when zero
// so the JSON stays compact for the common ok / missing cases.
type DiffServiceStatus struct {
	Name          string `json:"name"`
	ContainerName string `json:"container_name"`
	Status        string `json:"status"`
	State         string `json:"state,omitempty"`
	ExitCode      int    `json:"exit_code,omitempty"`
	Error         string `json:"error,omitempty"`
}

// ReceiptPairDiff is the structured receipt-vs-receipt comparison payload
// (KindDiff, format_version 1). "first" / project_a align with CompareReceipts(a, b).
type ReceiptPairDiff struct {
	ProjectA      string   `json:"project_a"`
	ProjectB      string   `json:"project_b"`
	ContractPathA string   `json:"contract_path_a,omitempty"`
	ContractPathB string   `json:"contract_path_b,omitempty"`
	ProfilesA     []string `json:"profiles_a,omitempty"`
	ProfilesB     []string `json:"profiles_b,omitempty"`

	ProjectMatch      bool `json:"project_match"`
	ContractPathMatch bool `json:"contract_path_match"`
	ProfilesMatch     bool `json:"profiles_match"`

	Services []ReceiptPairService `json:"services,omitempty"`
}

// ReceiptPairService is one service row with delta codes and optional first/second snapshots.
type ReceiptPairService struct {
	Service string           `json:"service"`
	Codes   []string         `json:"codes"`
	First   *ReceiptPairSide `json:"first,omitempty"`
	Second  *ReceiptPairSide `json:"second,omitempty"`
}

// ReceiptPairSide mirrors receipt.ServiceRecord fields for JSON (omitempty when empty).
type ReceiptPairSide struct {
	ContainerName string               `json:"container_name,omitempty"`
	ContainerID   string               `json:"container_id,omitempty"`
	Image         string               `json:"image,omitempty"`
	Env           *[]receipt.EnvVar    `json:"env,omitempty"`
	Mounts        *[]receipt.MountSpec `json:"mounts,omitempty"`
}

// Issue is one validation- or deploy-style finding for agents and CI.
type Issue struct {
	Level   string `json:"level"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Service string `json:"service,omitempty"`
}

// MarshalIndent returns pretty-printed JSON for stdout.
func MarshalIndent(d *Document) ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

// FromValidate builds a validate-kind document from validation results.
// When dependentsExpand is true and deployServices is non-empty, dependents_expand is set in JSON.
func FromValidate(contractPath, project string, profiles, deployServices []string, results []validate.Result, dependentsExpand bool) *Document {
	st := StatusOK
	issues := IssuesFromResults(results)
	for _, r := range results {
		if !r.OK && r.Level == validate.LevelFail {
			st = StatusFailed
			break
		}
	}
	cp := contractPath
	if cp != "" {
		cp = filepath.Clean(cp)
	}
	var ds []string
	if len(deployServices) > 0 {
		ds = append([]string(nil), deployServices...)
	}
	depExpand := len(ds) > 0 && dependentsExpand
	return &Document{
		FormatVersion:    FormatVersion,
		Kind:             KindValidate,
		Status:           st,
		ContractPath:     cp,
		Project:          project,
		Profiles:         profiles,
		DeployServices:   ds,
		DependentsExpand: depExpand,
		Issues:           issues,
	}
}

// IssuesFromResults maps validate.Result rows to Issue (level preserved).
func IssuesFromResults(results []validate.Result) []Issue {
	if len(results) == 0 {
		return nil
	}
	out := make([]Issue, 0, len(results))
	for _, r := range results {
		out = append(out, IssueFromResult(r))
	}
	return out
}

// IssueFromResult converts one validation result.
func IssueFromResult(r validate.Result) Issue {
	code := strings.TrimSpace(r.Code)
	if code == "" {
		switch r.Level {
		case validate.LevelFail:
			if !r.OK {
				code = "validation_fail"
			}
		case validate.LevelWarn:
			code = "validation_warn"
		case validate.LevelOK:
			code = "validation_ok"
		}
	}
	return Issue{
		Level:   r.Level,
		Code:    code,
		Message: r.Message,
		Service: r.Service,
	}
}

// DeployFromValidateResults builds a deploy-kind document from failed preflight (validate results).
// Use when deploy never started because validation had fail-level issues.
func DeployFromValidateResults(contractPath, project string, profiles, deployServices []string, results []validate.Result, dependentsExpand bool) *Document {
	d := FromValidate(contractPath, project, profiles, deployServices, results, dependentsExpand)
	d.Kind = KindDeploy
	return d
}

// DeployOutcome builds a deploy-kind document (success or failure).
func DeployOutcome(contractPath, project string, profiles, deployServices []string, receiptPath string, deployErr error, dependentsExpand bool) *Document {
	cp := contractPath
	if cp != "" {
		cp = filepath.Clean(cp)
	}
	var ds []string
	if len(deployServices) > 0 {
		ds = append([]string(nil), deployServices...)
	}
	depExpand := len(ds) > 0 && dependentsExpand
	if deployErr != nil {
		return &Document{
			FormatVersion:    FormatVersion,
			Kind:             KindDeploy,
			Status:           StatusFailed,
			ContractPath:     cp,
			Project:          project,
			Profiles:         profiles,
			DeployServices:   ds,
			DependentsExpand: depExpand,
			Issues: []Issue{{
				Level:   validate.LevelFail,
				Code:    "deploy_error",
				Message: deployErr.Error(),
			}},
		}
	}
	return &Document{
		FormatVersion:    FormatVersion,
		Kind:             KindDeploy,
		Status:           StatusOK,
		ContractPath:     cp,
		Project:          project,
		Profiles:         profiles,
		DeployServices:   ds,
		DependentsExpand: depExpand,
		ReceiptPath:      receiptPath,
	}
}

// ReceiptReadSuccess wraps an already-validated receipt JSON payload.
func ReceiptReadSuccess(absReceiptPath string, receiptJSON []byte) *Document {
	return &Document{
		FormatVersion: FormatVersion,
		Kind:          KindReceiptRead,
		Status:        StatusOK,
		ReceiptPath:   absReceiptPath,
		Receipt:       json.RawMessage(receiptJSON),
	}
}

// ReceiptReadFailure returns a failed receipt_read document.
func ReceiptReadFailure(absReceiptPath string, err error) *Document {
	msg := "receipt read failed"
	if err != nil {
		msg = err.Error()
	}
	return &Document{
		FormatVersion: FormatVersion,
		Kind:          KindReceiptRead,
		Status:        StatusFailed,
		ReceiptPath:   absReceiptPath,
		Issues: []Issue{{
			Level:   validate.LevelFail,
			Code:    "receipt_read_error",
			Message: msg,
		}},
	}
}
