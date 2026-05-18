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
	CodeLogsResolveError      = "logs_resolve_error"
	CodeLogsFollowMulti       = "logs_follow_multi_service"
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

// FromLogsBatchSuccess builds a KindLogs document after capturing one or more services.
// When len(entries)==1, top-level service, container_name, and log_body mirror that entry for backward compatibility.
func FromLogsBatchSuccess(contractPath, project string, profiles, deployServices []string, dependentsExpand bool, tail int, since string, entries []LogEntry) *Document {
	cp := contractPath
	if cp != "" {
		cp = filepath.Clean(cp)
	}
	cpCopy := append([]LogEntry(nil), entries...)
	doc := &Document{
		FormatVersion:    FormatVersion,
		Kind:             KindLogs,
		Status:           StatusOK,
		ContractPath:     cp,
		Project:          project,
		Profiles:         append([]string(nil), profiles...),
		LogsTail:         tail,
		LogsSince:        since,
		LogEntries:       cpCopy,
		DependentsExpand: dependentsExpand,
	}
	if len(deployServices) > 0 {
		doc.DeployServices = append([]string(nil), deployServices...)
	}
	if len(deployServices) > 0 && dependentsExpand {
		doc.DependentsExpand = true
	}
	if len(entries) == 1 {
		e := entries[0]
		body := e.LogBody
		doc.LogsService = e.Service
		doc.LogsContainerName = e.ContainerName
		doc.LogsBody = &body
	}
	return doc
}

// LogsFailure builds a failed KindLogs document (contract_path / project / profiles may be partial).
func LogsFailure(contractPath, project string, profiles []string, service, code, msg string) *Document {
	return LogsFailurePartial(contractPath, project, profiles, nil, false, service, code, msg)
}

// LogsFailurePartial adds deploy_services / dependents_expand when partial roots were used.
func LogsFailurePartial(contractPath, project string, profiles, deployServices []string, dependentsExpand bool, service, code, msg string) *Document {
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
	doc := &Document{
		FormatVersion: FormatVersion,
		Kind:          KindLogs,
		Status:        StatusFailed,
		ContractPath:  cp,
		Project:       project,
		Profiles:      append([]string(nil), profiles...),
		LogsService:   service,
		Issues:        []Issue{iss},
	}
	if len(deployServices) > 0 {
		doc.DeployServices = append([]string(nil), deployServices...)
	}
	if len(deployServices) > 0 && dependentsExpand {
		doc.DependentsExpand = true
	}
	return doc
}
