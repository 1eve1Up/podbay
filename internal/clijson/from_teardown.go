package clijson

import (
	"path/filepath"

	"github.com/1eve1Up/podbay/internal/teardown"
	"github.com/1eve1Up/podbay/internal/validate"
)

// FromTeardown builds a KindTeardown Document from teardown.Execute results.
//
// Status is ok when runErr is nil, failed otherwise. A non-fatal network rm
// warning is recorded as issues[] with level warn and code
// teardown_network_warning; status remains ok.
//
// deployServices and dependentsExpand mirror validate/deploy JSON when partial
// teardown roots were used (additive fields; omitted when unset).
//
// Inputs are not mutated; slices on the document are copies.
func FromTeardown(contractPath, project string, profiles []string, deployServices []string, dependentsExpand bool, res teardown.TeardownResult, runErr error) *Document {
	cp := contractPath
	if cp != "" {
		cp = filepath.Clean(cp)
	}

	doc := &Document{
		FormatVersion: FormatVersion,
		Kind:          KindTeardown,
		ContractPath:  cp,
		Project:       project,
		Profiles:      profiles,
	}
	if len(deployServices) > 0 {
		doc.DeployServices = append(make([]string, 0, len(deployServices)), deployServices...)
	}
	if dependentsExpand {
		doc.DependentsExpand = true
	}

	if len(res.ContainerNames) > 0 {
		doc.ContainersRemoved = append(make([]string, 0, len(res.ContainerNames)), res.ContainerNames...)
	}
	if len(res.VolumeNames) > 0 {
		doc.VolumesRemoved = append(make([]string, 0, len(res.VolumeNames)), res.VolumeNames...)
	}

	if runErr != nil {
		doc.Status = StatusFailed
		code := teardown.FatalCode(runErr)
		if code == "" {
			code = "teardown_error"
		}
		doc.Issues = append(doc.Issues, Issue{
			Level:   validate.LevelFail,
			Code:    code,
			Message: runErr.Error(),
		})
		return doc
	}

	doc.Status = StatusOK
	if res.NetworkWarning != "" {
		doc.Issues = append(doc.Issues, Issue{
			Level:   validate.LevelWarn,
			Code:    teardown.CodeNetworkWarn,
			Message: res.NetworkWarning,
		})
	}
	return doc
}

// TeardownLoadError is emitted when the contract cannot be loaded (same spirit as diff_load_error).
func TeardownLoadError(contractPath, project string, profiles []string, msg string) *Document {
	cp := contractPath
	if cp != "" {
		cp = filepath.Clean(cp)
	}
	return &Document{
		FormatVersion: FormatVersion,
		Kind:          KindTeardown,
		Status:        StatusFailed,
		ContractPath:  cp,
		Project:       project,
		Profiles:      profiles,
		Issues: []Issue{{
			Level:   validate.LevelFail,
			Code:    "teardown_load_error",
			Message: msg,
		}},
	}
}
