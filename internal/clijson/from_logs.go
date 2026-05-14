package clijson

import (
	"path/filepath"

	"github.com/1eve1Up/podbay/internal/validate"
)

// Stable issue codes for KindLogs documents (failure paths).
const (
	CodeLogsUsageJSONFollow   = "logs_usage_json_follow"
	CodeLogsUsageArgs         = "logs_usage_args"
	CodeLogsLoadError         = "logs_load_error"
	CodeLogsServiceNotActive  = "logs_service_not_active"
	CodeLogsPodmanUnavailable = "logs_podman_unavailable"
	CodeLogsRuntimeError      = "logs_runtime_error"
)

// FromLogsSuccess builds a KindLogs document after a successful podman logs capture.
// logBody is the raw combined stdout/stderr from podman logs (may be empty).
func FromLogsSuccess(contractPath, project string, profiles []string, service, containerName string, tail int, since, logBody string) *Document {
	cp := contractPath
	if cp != "" {
		cp = filepath.Clean(cp)
	}
	body := logBody
	doc := &Document{
		FormatVersion:     FormatVersion,
		Kind:              KindLogs,
		Status:            StatusOK,
		ContractPath:      cp,
		Project:           project,
		Profiles:          append([]string(nil), profiles...),
		LogsService:       service,
		LogsContainerName: containerName,
		LogsTail:          tail,
		LogsSince:         since,
		LogsBody:          &body,
	}
	return doc
}

// LogsFailure builds a failed KindLogs document (contract_path / project / profiles may be partial).
func LogsFailure(contractPath, project string, profiles []string, service, code, msg string) *Document {
	cp := contractPath
	if cp != "" {
		cp = filepath.Clean(cp)
	}
	iss := Issue{
		Level:   validate.LevelFail,
		Code:    code,
		Message: msg,
	}
	if service != "" {
		iss.Service = service
	}
	return &Document{
		FormatVersion: FormatVersion,
		Kind:          KindLogs,
		Status:        StatusFailed,
		ContractPath:  cp,
		Project:       project,
		Profiles:      append([]string(nil), profiles...),
		LogsService:   service,
		Issues:        []Issue{iss},
	}
}
