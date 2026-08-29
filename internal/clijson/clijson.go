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
	KindValidate       = "validate"
	KindDeploy         = "deploy"
	KindReceiptRead    = "receipt_read"
	KindReceiptList    = "receipt_list"
	KindReceiptLastOK  = "receipt_last_ok"
	KindReceiptHandoff = "receipt_handoff"
	KindDiff           = "diff"
	KindTeardown       = "teardown"
	KindLogs           = "logs"
	KindInit           = "init"
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
// KindImportCompose is used for import compose --json (success and failure).
// On success, contract_path is the Compose source file; contract_yaml holds the
// generated Podbay contract when using --json. Non-JSON success still emits YAML only.
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
	LogsService       string     `json:"service,omitempty"`
	LogsContainerName string     `json:"container_name,omitempty"`
	LogsTail          int        `json:"tail,omitempty"`
	LogsSince         string     `json:"since,omitempty"`
	LogsBody          *string    `json:"log_body,omitempty"`
	LogEntries        []LogEntry `json:"log_entries,omitempty"`
	// Import compose success (KindImportCompose, status ok): generated contract YAML as UTF-8 string.
	ImportContractYAML string `json:"contract_yaml,omitempty"`
	// ImportOutputPath is the absolute -o/--output path when set (file written in addition to stdout JSON).
	ImportOutputPath string `json:"output_path,omitempty"`
	// ImportServiceCount is len(services) in the generated contract (also used by KindInit --from-codebase).
	ImportServiceCount int `json:"service_count,omitempty"`
	// ComposeSource is the Compose file path for KindInit --from-codebase (contract_path is the written podbay.yaml).
	ComposeSource string `json:"compose_source,omitempty"`
	// DockerfileSource is the Dockerfile path for KindInit --from-codebase Dockerfile fallback.
	DockerfileSource string `json:"dockerfile_source,omitempty"`
	// SourceKind is "compose" or "dockerfile" for KindInit --from-codebase success; omitted for greenfield.
	SourceKind string `json:"source_kind,omitempty"`
	// NextActions lists ordered CLI hints for KindInit success (onboard / validate); omitted when empty.
	NextActions []string `json:"next_actions,omitempty"`
	// Extracted lists Dockerfile instructions copied onto the stub (expose, health).
	// KindInit Dockerfile success only; omitted when empty.
	Extracted []string `json:"extracted,omitempty"`
	// Gaps lists still-missing operational fields after Dockerfile stub fill
	// (expose, health, published_ports). KindInit Dockerfile success only.
	Gaps []string `json:"gaps,omitempty"`
	// Receipt list (KindReceiptList): directory inventoried and newest-first entries.
	ReceiptListDir string             `json:"receipt_list_dir,omitempty"`
	Receipts       []ReceiptListEntry `json:"receipts,omitempty"`
	// ReceiptListSkipped lists paths that were not valid receipts (best-effort).
	ReceiptListSkipped []string `json:"receipt_list_skipped,omitempty"`
	// Receipt handoff (KindReceiptHandoff): structured next-steps summary payload.
	Handoff *receipt.HandoffSummary `json:"handoff,omitempty"`
}

// ReceiptListEntry is one inventory row for kind receipt_list.
type ReceiptListEntry struct {
	Path         string `json:"path"`
	DeployID     string `json:"deploy_id,omitempty"`
	GeneratedAt  string `json:"generated_at,omitempty"`
	Project      string `json:"project,omitempty"`
	Status       string `json:"status,omitempty"`
	ServiceCount int    `json:"service_count"`
}

// DiffServiceStatus is one expected service's runtime outcome inside a
// KindDiff document. Status mirrors the diff package's per-service status
// strings (ok, missing, wrong_state, inspect_error). State, ExitCode, and
// Error are populated when the runtime supplied them; they omit when zero
// so the JSON stays compact for the common ok / missing cases.
// LogEntry is one service's captured log output inside a KindLogs success document.
type LogEntry struct {
	Service       string `json:"service"`
	ContainerName string `json:"container_name"`
	LogBody       string `json:"log_body"`
}

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
	ProjectA        string   `json:"project_a"`
	ProjectB        string   `json:"project_b"`
	ContractPathA   string   `json:"contract_path_a,omitempty"`
	ContractPathB   string   `json:"contract_path_b,omitempty"`
	ContractDigestA string   `json:"contract_digest_a,omitempty"`
	ContractDigestB string   `json:"contract_digest_b,omitempty"`
	ProfilesA       []string `json:"profiles_a,omitempty"`
	ProfilesB       []string `json:"profiles_b,omitempty"`

	ProjectMatch             bool `json:"project_match"`
	ContractPathMatch        bool `json:"contract_path_match"`
	ProfilesMatch            bool `json:"profiles_match"`
	ContractDigestMatch      bool `json:"contract_digest_match"`
	ContractDigestComparable bool `json:"contract_digest_comparable"`

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
			ReceiptPath:      receiptPath,
			Issues:           IssuesFromDeployError(deployErr),
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

// ReceiptListSuccess builds kind receipt_list from inventory rows.
func ReceiptListSuccess(absDir string, entries []receipt.ListEntry, skipped []string) *Document {
	out := make([]ReceiptListEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, ReceiptListEntry{
			Path:         e.Path,
			DeployID:     e.DeployID,
			GeneratedAt:  e.GeneratedAt,
			Project:      e.Project,
			Status:       e.Status,
			ServiceCount: e.ServiceCount,
		})
	}
	doc := &Document{
		FormatVersion:  FormatVersion,
		Kind:           KindReceiptList,
		Status:         StatusOK,
		ReceiptListDir: absDir,
		Receipts:       out,
	}
	if len(skipped) > 0 {
		doc.ReceiptListSkipped = append([]string(nil), skipped...)
	}
	return doc
}

// ReceiptListFailure builds a failed receipt_list document.
func ReceiptListFailure(absDir string, err error) *Document {
	msg := "receipt list failed"
	if err != nil {
		msg = err.Error()
	}
	return &Document{
		FormatVersion:  FormatVersion,
		Kind:           KindReceiptList,
		Status:         StatusFailed,
		ReceiptListDir: absDir,
		Issues: []Issue{{
			Level:   validate.LevelFail,
			Code:    "receipt_list_error",
			Message: msg,
		}},
	}
}

// ReceiptLastOKSuccess builds kind receipt_last_ok when a last-ok path was resolved.
func ReceiptLastOKSuccess(absDir string, entry receipt.ListEntry) *Document {
	return &Document{
		FormatVersion:  FormatVersion,
		Kind:           KindReceiptLastOK,
		Status:         StatusOK,
		ReceiptListDir: absDir,
		ReceiptPath:    entry.Path,
		Project:        entry.Project,
		Receipts: []ReceiptListEntry{{
			Path:         entry.Path,
			DeployID:     entry.DeployID,
			GeneratedAt:  entry.GeneratedAt,
			Project:      entry.Project,
			Status:       entry.Status,
			ServiceCount: entry.ServiceCount,
		}},
	}
}

// ReceiptLastOKFailure builds a failed receipt_last_ok document (list/IO errors or no prior ok).
func ReceiptLastOKFailure(absDir string, code string, err error) *Document {
	msg := "receipt last-ok failed"
	if err != nil {
		msg = err.Error()
	}
	if code == "" {
		code = "receipt_last_ok_error"
	}
	return &Document{
		FormatVersion:  FormatVersion,
		Kind:           KindReceiptLastOK,
		Status:         StatusFailed,
		ReceiptListDir: absDir,
		Issues: []Issue{{
			Level:   validate.LevelFail,
			Code:    code,
			Message: msg,
		}},
	}
}

// ReceiptHandoffSuccess builds kind receipt_handoff with a handoff summary payload.
func ReceiptHandoffSuccess(h *receipt.HandoffSummary) *Document {
	return &Document{
		FormatVersion: FormatVersion,
		Kind:          KindReceiptHandoff,
		Status:        StatusOK,
		ReceiptPath:   h.CurrentPath,
		Project:       h.Project,
		Handoff:       h,
	}
}

// ReceiptHandoffFailure builds a failed receipt_handoff document.
func ReceiptHandoffFailure(absCurrent string, err error) *Document {
	msg := "receipt handoff failed"
	if err != nil {
		msg = err.Error()
	}
	return &Document{
		FormatVersion: FormatVersion,
		Kind:          KindReceiptHandoff,
		Status:        StatusFailed,
		ReceiptPath:   absCurrent,
		Issues: []Issue{{
			Level:   validate.LevelFail,
			Code:    "receipt_handoff_error",
			Message: msg,
		}},
	}
}
