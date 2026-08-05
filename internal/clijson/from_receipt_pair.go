package clijson

import (
	"fmt"

	"github.com/1eve1Up/podbay/internal/receipt"
	"github.com/1eve1Up/podbay/internal/validate"
)

// Stable issue codes when a receipt pair diff cannot run (compare FromDiff DiffError).
const (
	CodeReceiptDiffLoadError   = "receipt_diff_load_error"
	CodeReceiptDiffDecodeError = "receipt_diff_decode_error"
	CodeReceiptDiffUsageError  = "receipt_diff_usage_error"
	CodeReceiptNoLastOK        = "receipt_no_last_ok"
)

// ReceiptPairDiffOptions configures receipt-pair JSON emission (two-arg diff).
type ReceiptPairDiffOptions struct {
	// ShowRawEnv when true includes raw env values in receipt_pair snapshots (unsafe for CI logs).
	ShowRawEnv bool
}

// FromReceiptDiff maps receipt.CompareReceipts output to a KindDiff document (sprint name).
// It is identical to FromReceiptPairDiff (default options: redacted env values).
func FromReceiptDiff(res receipt.ReceiptDiffResult) *Document {
	return FromReceiptPairDiff(res)
}

// FromReceiptPairDiff builds a KindDiff document from receipt.CompareReceipts output.
//
// Status is ok when Drift is false, failed when true. Drift is always set explicitly
// so consumers can branch like contract-runtime diff JSON.
//
// Issues list global mismatches (project, contract_path, profiles) and each
// per-service code from receipt diff, with Service set when applicable, so CI can
// grep issues[] the same way as diff_inspect_error.
func FromReceiptPairDiff(res receipt.ReceiptDiffResult) *Document {
	return FromReceiptPairDiffWithOptions(res, ReceiptPairDiffOptions{})
}

// FromReceiptPairDiffWithOptions is like FromReceiptPairDiff but allows env display policy.
func FromReceiptPairDiffWithOptions(res receipt.ReceiptDiffResult, opts ReceiptPairDiffOptions) *Document {
	st := StatusOK
	if res.Drift {
		st = StatusFailed
	}
	drift := res.Drift

	envPolicy := "redacted"
	if opts.ShowRawEnv {
		envPolicy = "raw"
	}

	pair := &ReceiptPairDiff{
		ProjectA:                 res.ProjectA,
		ProjectB:                 res.ProjectB,
		ContractPathA:            res.ContractPathA,
		ContractPathB:            res.ContractPathB,
		ContractDigestA:          res.ContractDigestA,
		ContractDigestB:          res.ContractDigestB,
		ProjectMatch:             res.ProjectMatch,
		ContractPathMatch:        res.ContractPathMatch,
		ProfilesMatch:            res.ProfilesMatch,
		ContractDigestMatch:      res.ContractDigestMatch,
		ContractDigestComparable: res.ContractDigestComparable,
	}
	if len(res.ProfilesA) > 0 {
		pair.ProfilesA = append([]string(nil), res.ProfilesA...)
	}
	if len(res.ProfilesB) > 0 {
		pair.ProfilesB = append([]string(nil), res.ProfilesB...)
	}
	if len(res.Services) > 0 {
		pair.Services = make([]ReceiptPairService, 0, len(res.Services))
		for _, s := range res.Services {
			row := ReceiptPairService{
				Service: s.Service,
				Codes:   append([]string(nil), s.Codes...),
			}
			if sideNonEmpty(s.RecordA) {
				fs := receiptSideFrom(s.RecordA, opts.ShowRawEnv)
				row.First = &fs
			}
			if sideNonEmpty(s.RecordB) {
				ss := receiptSideFrom(s.RecordB, opts.ShowRawEnv)
				row.Second = &ss
			}
			pair.Services = append(pair.Services, row)
		}
	}

	doc := &Document{
		FormatVersion:          FormatVersion,
		Kind:                   KindDiff,
		Status:                 st,
		Drift:                  &drift,
		ReceiptPair:            pair,
		ReceiptPairDiffVersion: ReceiptPairDiffPayloadVersion,
		EnvValueDisplayPolicy:  envPolicy,
	}

	if !res.ProjectMatch {
		doc.Issues = append(doc.Issues, Issue{
			Level:   validate.LevelFail,
			Code:    receipt.CodeProjectMismatch,
			Message: fmt.Sprintf("project mismatch: first=%q second=%q", res.ProjectA, res.ProjectB),
		})
	}
	if !res.ContractPathMatch {
		doc.Issues = append(doc.Issues, Issue{
			Level:   validate.LevelFail,
			Code:    receipt.CodeContractPathMismatch,
			Message: fmt.Sprintf("contract_path mismatch: first=%q second=%q", res.ContractPathA, res.ContractPathB),
		})
	}
	if !res.ProfilesMatch {
		doc.Issues = append(doc.Issues, Issue{
			Level:   validate.LevelFail,
			Code:    receipt.CodeProfilesMismatch,
			Message: fmt.Sprintf("profiles mismatch: first=%v second=%v", res.ProfilesA, res.ProfilesB),
		})
	}
	if !res.ContractDigestComparable {
		doc.Issues = append(doc.Issues, Issue{
			Level: validate.LevelWarn,
			Code:  receipt.CodeContractDigestIncomparable,
			Message: fmt.Sprintf("contract_digest incomparable (recorded on one receipt only): first=%q second=%q",
				res.ContractDigestA, res.ContractDigestB),
		})
	} else if !res.ContractDigestMatch {
		doc.Issues = append(doc.Issues, Issue{
			Level: validate.LevelFail,
			Code:  receipt.CodeContractDigestMismatch,
			Message: fmt.Sprintf("contract_digest mismatch: first=%q second=%q",
				res.ContractDigestA, res.ContractDigestB),
		})
	}
	for _, s := range res.Services {
		for _, code := range s.Codes {
			lvl := validate.LevelFail
			if code == receipt.CodeEnvIncomparable || code == receipt.CodeMountsIncomparable {
				lvl = validate.LevelWarn
			}
			doc.Issues = append(doc.Issues, Issue{
				Level:   lvl,
				Code:    code,
				Message: receiptPairIssueMessage(code, s),
				Service: s.Service,
			})
		}
	}

	return doc
}

// ReceiptPairDiffError builds KindDiff when receipt files could not be read or decoded.
// Drift is nil (comparison did not run). Typical codes: CodeReceiptDiffLoadError,
// CodeReceiptDiffDecodeError (align with receipt.Decode / os.ReadFile failures).
func ReceiptPairDiffError(code, msg string) *Document {
	return &Document{
		FormatVersion: FormatVersion,
		Kind:          KindDiff,
		Status:        StatusFailed,
		Issues: []Issue{{
			Level:   validate.LevelFail,
			Code:    code,
			Message: msg,
		}},
	}
}

func sideNonEmpty(r receipt.ServiceRecord) bool {
	if r.ContainerName != "" || r.ContainerID != "" || r.Image != "" {
		return true
	}
	return r.Env != nil || r.Mounts != nil
}

func receiptSideFrom(r receipt.ServiceRecord, rawEnv bool) ReceiptPairSide {
	s := ReceiptPairSide{
		ContainerName: r.ContainerName,
		ContainerID:   r.ContainerID,
		Image:         r.Image,
	}
	if r.Env != nil {
		cp := append([]receipt.EnvVar(nil), (*r.Env)...)
		if !rawEnv {
			for i := range cp {
				if cp[i].Value != "" {
					cp[i].Value = "(redacted)"
				}
			}
		}
		s.Env = &cp
	}
	if r.Mounts != nil {
		cp := append([]receipt.MountSpec(nil), (*r.Mounts)...)
		s.Mounts = &cp
	}
	return s
}

func receiptPairIssueMessage(code string, s receipt.ServiceReceiptDiff) string {
	switch code {
	case receipt.CodeServiceAdded:
		return fmt.Sprintf("service %q present only in second receipt", s.Service)
	case receipt.CodeServiceRemoved:
		return fmt.Sprintf("service %q present only in first receipt", s.Service)
	case receipt.CodeImageChanged:
		return fmt.Sprintf("service %q image changed", s.Service)
	case receipt.CodeContainerNameChanged:
		return fmt.Sprintf("service %q container_name changed", s.Service)
	case receipt.CodeContainerIDChanged:
		return fmt.Sprintf("service %q container_id changed", s.Service)
	case receipt.CodeEnvChanged:
		return fmt.Sprintf("service %q env changed", s.Service)
	case receipt.CodeMountsChanged:
		return fmt.Sprintf("service %q mounts changed", s.Service)
	case receipt.CodeEnvIncomparable:
		return fmt.Sprintf("service %q env incomparable (recorded on one receipt only)", s.Service)
	case receipt.CodeMountsIncomparable:
		return fmt.Sprintf("service %q mounts incomparable (recorded on one receipt only)", s.Service)
	default:
		return code
	}
}
