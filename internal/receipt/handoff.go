package receipt

import (
	"errors"
	"fmt"
	"os"
)

// HandoffFormatVersion is the handoff summary document shape version.
const HandoffFormatVersion = 1

// HandoffSummary is a structured agent handoff object (not automatic remediation).
type HandoffSummary struct {
	FormatVersion   int             `json:"format_version"`
	CurrentPath     string          `json:"current_path"`
	DeployID        string          `json:"deploy_id,omitempty"`
	ContractDigest  string          `json:"contract_digest,omitempty"`
	Status          string          `json:"status,omitempty"`
	Project         string          `json:"project,omitempty"`
	Failure         *FailureSummary `json:"failure,omitempty"`
	LastOKPath      string          `json:"last_ok_path,omitempty"`
	NoPriorOK       bool            `json:"no_prior_ok,omitempty"`
	Drift           *bool           `json:"drift,omitempty"`
	DiffCodes       []string        `json:"diff_codes,omitempty"`
	NextActions     []string        `json:"next_actions"`
	RemediationNote string          `json:"note"` // always clarifies structured handoff only
}

// BuildHandoff builds a handoff summary from a current receipt and optional store directory.
// storeDir may be empty to skip last-ok resolution (NoPriorOK stays false; LastOKPath empty).
// When storeDir is set and no ok receipt exists, NoPriorOK is true and Drift is omitted.
func BuildHandoff(current *Receipt, currentPath, storeDir string) (*HandoffSummary, error) {
	if current == nil {
		return nil, fmt.Errorf("receipt handoff: nil current receipt")
	}
	if err := Validate(current); err != nil {
		return nil, err
	}
	h := &HandoffSummary{
		FormatVersion:   HandoffFormatVersion,
		CurrentPath:     currentPath,
		DeployID:        current.DeployID,
		ContractDigest:  current.ContractDigest,
		Status:          current.Status,
		Project:         current.Project,
		RemediationNote: "structured next-steps only; not automatic remediation or root-cause diagnosis",
	}
	if current.Failure != nil {
		cp := *current.Failure
		h.Failure = &cp
	}

	if storeDir != "" {
		entry, err := LastOK(storeDir)
		if err != nil {
			if errors.Is(err, ErrNoLastOK) {
				h.NoPriorOK = true
			} else {
				return nil, err
			}
		} else {
			h.LastOKPath = entry.Path
			data, readErr := os.ReadFile(entry.Path)
			if readErr != nil {
				return nil, fmt.Errorf("receipt handoff: read last ok: %w", readErr)
			}
			lastOK, decErr := Decode(data)
			if decErr != nil {
				return nil, fmt.Errorf("receipt handoff: decode last ok: %w", decErr)
			}
			res := CompareReceipts(lastOK, current)
			drift := res.Drift
			h.Drift = &drift
			h.DiffCodes = compactDiffCodes(res)
		}
	}

	h.NextActions = nextActionHints(current)
	return h, nil
}

func compactDiffCodes(res ReceiptDiffResult) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(code string) {
		if code == "" {
			return
		}
		if _, ok := seen[code]; ok {
			return
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	if !res.ProjectMatch {
		add(CodeProjectMismatch)
	}
	if !res.ContractPathMatch {
		add(CodeContractPathMismatch)
	}
	if res.ContractDigestComparable && !res.ContractDigestMatch {
		add(CodeContractDigestMismatch)
	}
	if !res.ContractDigestComparable && (res.ContractDigestA != "" || res.ContractDigestB != "") {
		add(CodeContractDigestIncomparable)
	}
	if !res.ProfilesMatch {
		add(CodeProfilesMismatch)
	}
	for _, s := range res.Services {
		for _, c := range s.Codes {
			add(c)
		}
	}
	return out
}

func nextActionHints(current *Receipt) []string {
	svc := ""
	if current.Failure != nil {
		svc = current.Failure.Service
	}
	if current.Status == StatusFailed || current.Failure != nil {
		logsHint := "podbay logs --json"
		explainHint := "podbay explain --json"
		if svc != "" {
			logsHint = fmt.Sprintf("podbay logs %s --json", svc)
			explainHint = fmt.Sprintf("podbay explain %s --json", svc)
		}
		return []string{
			logsHint,
			explainHint,
			"podbay down --json",
		}
	}
	return []string{
		"podbay diff --json",
		"podbay logs --json",
	}
}
