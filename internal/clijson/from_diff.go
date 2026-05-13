package clijson

import (
	"path/filepath"

	"github.com/1eve1Up/podbay/internal/diff"
	"github.com/1eve1Up/podbay/internal/validate"
)

// FromDiff builds a KindDiff Document from a diff.DriftResult (no partial roots metadata).
func FromDiff(contractPath, project string, profiles []string, res diff.DriftResult) *Document {
	return FromDiffWithPartial(contractPath, project, profiles, nil, false, res)
}

// FromDiffWithPartial builds a KindDiff Document from a diff.DriftResult and sets
// deploy_services / dependents_expand when partial roots were used (same semantics as validate/deploy JSON).
//
// Status semantics: ok when no drift, failed when drift is true (matches the
// existing FromValidate / DeployOutcome convention). The Drift pointer always
// emits explicit true/false on KindDiff documents so consumers can branch
// without inferring from Status.
//
// Inspect errors per service are recorded twice: once in services_status (so
// the service-level shape stays uniform) and once in issues with code
// diff_inspect_error (so CI can grep issues[] uniformly across kinds).
//
// Inputs are not mutated; the returned Document holds defensive copies of
// services and extras.
func FromDiffWithPartial(contractPath, project string, profiles, deployServices []string, dependentsExpand bool, res diff.DriftResult) *Document {
	cp := contractPath
	if cp != "" {
		cp = filepath.Clean(cp)
	}

	st := StatusOK
	if res.Drift {
		st = StatusFailed
	}

	driftCopy := res.Drift

	doc := &Document{
		FormatVersion: FormatVersion,
		Kind:          KindDiff,
		Status:        st,
		ContractPath:  cp,
		Project:       project,
		Profiles:      profiles,
		Drift:         &driftCopy,
	}

	var ds []string
	if len(deployServices) > 0 {
		ds = append([]string(nil), deployServices...)
		doc.DeployServices = ds
		doc.DependentsExpand = len(ds) > 0 && dependentsExpand
	}

	if len(res.Services) > 0 {
		doc.ServicesStatus = make([]DiffServiceStatus, 0, len(res.Services))
		for _, s := range res.Services {
			doc.ServicesStatus = append(doc.ServicesStatus, DiffServiceStatus{
				Name:          s.Name,
				ContainerName: s.ContainerName,
				Status:        string(s.Status),
				State:         s.State,
				ExitCode:      s.ExitCode,
				Error:         s.Error,
			})
		}
	}

	if len(res.Extras) > 0 {
		doc.Extras = append(make([]string, 0, len(res.Extras)), res.Extras...)
	}

	for _, s := range res.Services {
		if s.Status == diff.StatusInspectError {
			msg := s.Error
			if msg == "" {
				msg = "inspect error"
			}
			doc.Issues = append(doc.Issues, Issue{
				Level:   validate.LevelFail,
				Code:    "diff_inspect_error",
				Message: msg,
				Service: s.Name,
			})
		}
	}

	return doc
}

// DiffError builds a KindDiff Document for a load or runtime failure where
// drift detection could not run. Drift is left nil (unknown) so consumers
// can distinguish "comparison ran and saw no drift" (Drift=&false) from
// "comparison never ran" (Drift=nil); Status is failed in either failure
// case, and the issue carries a stable code (e.g. diff_load_error,
// diff_runtime_error) for CI to branch on.
func DiffError(contractPath, project string, profiles []string, code, msg string) *Document {
	cp := contractPath
	if cp != "" {
		cp = filepath.Clean(cp)
	}
	return &Document{
		FormatVersion: FormatVersion,
		Kind:          KindDiff,
		Status:        StatusFailed,
		ContractPath:  cp,
		Project:       project,
		Profiles:      profiles,
		Issues: []Issue{{
			Level:   validate.LevelFail,
			Code:    code,
			Message: msg,
		}},
	}
}
