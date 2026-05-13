package ps

import (
	"encoding/json"
	"path/filepath"
	"sort"

	"github.com/1eve1Up/podbay/internal/spec"
)

// JSONFormatVersion is bumped when breaking the ps --json document shape.
const JSONFormatVersion = 1

type jsonRow struct {
	Service   string `json:"service"`
	Container string `json:"container"`
	State     string `json:"state"`
	ExitCode  int    `json:"exit_code,omitempty"`
	Error     string `json:"error,omitempty"`
	Image     string `json:"image,omitempty"`
	Missing   bool   `json:"missing,omitempty"`
}

type jsonDoc struct {
	FormatVersion    int       `json:"format_version"`
	Project          string    `json:"project"`
	ContractPath     string    `json:"contract_path"`
	Profiles         []string  `json:"profiles,omitempty"`
	DeployServices   []string  `json:"deploy_services,omitempty"`
	DependentsExpand bool      `json:"dependents_expand,omitempty"`
	ActiveServices   []string  `json:"active_services"`
	Rows             []jsonRow `json:"containers"`
}

// ReportJSON returns indented JSON for agents (same row set as ListRows).
func ReportJSON(c *spec.Contract, contractPath, project string, profiles []string, deployRoots []string, expandDependents bool, inspect InspectFunc) ([]byte, error) {
	rows, err := ListRows(c, project, profiles, deployRoots, expandDependents, inspect)
	if err != nil {
		return nil, err
	}
	profileActive := c.ServicesForProfiles(profiles)
	active, err := spec.ObservabilityActiveServices(profileActive, deployRoots, expandDependents)
	if err != nil {
		return nil, err
	}
	names := spec.ServiceNamesSorted(active)
	sort.Strings(profiles)
	out := jsonDoc{
		FormatVersion:  JSONFormatVersion,
		Project:        project,
		ContractPath:   filepath.Clean(contractPath),
		Profiles:       profiles,
		ActiveServices: names,
		Rows:           make([]jsonRow, 0, len(rows)),
	}
	if len(deployRoots) > 0 {
		out.DeployServices = append([]string(nil), deployRoots...)
		out.DependentsExpand = expandDependents && len(out.DeployServices) > 0
	}
	for _, rw := range rows {
		out.Rows = append(out.Rows, jsonRow{
			Service:   rw.Service,
			Container: rw.Container,
			State:     rw.State,
			ExitCode:  rw.ExitCode,
			Error:     rw.Error,
			Image:     rw.Image,
			Missing:   rw.Missing,
		})
	}
	return json.MarshalIndent(out, "", "  ")
}
