// Package orientation builds structured arrive/mid-loop context for agents and humans.
// Orientation is rule-based next-steps packaging — not causal diagnosis or auto-remediation.
package orientation

import (
	"fmt"

	"github.com/1eve1Up/podbay/internal/spec"
)

// FormatVersion is the orientation document shape version.
const FormatVersion = 1

// Kind is the JSON kind for orientation documents.
const Kind = "orientation"

// BoundaryNote always clarifies structured next-steps only.
const BoundaryNote = "structured context and next-steps only; not automatic remediation or root-cause diagnosis"

// Document is a versioned orientation object shared by onboard and explain packaging.
type Document struct {
	FormatVersion    int             `json:"format_version"`
	Kind             string          `json:"kind"`
	Project          string          `json:"project,omitempty"`
	ContractPath     string          `json:"contract_path"`
	Profiles         []string        `json:"profiles,omitempty"`
	DeployServices   []string        `json:"deploy_services,omitempty"`
	DependentsExpand bool            `json:"dependents_expand,omitempty"`
	ActiveServices   []string        `json:"active_services"`
	Graph            []GraphService  `json:"graph"`
	Runtime          *RuntimeSummary `json:"runtime,omitempty"`
	NextActions      []string        `json:"next_actions"`
	Note             string          `json:"note"`
}

// GraphService is a compact depends_on skim for one service in scope.
type GraphService struct {
	Name      string      `json:"name"`
	DependsOn []GraphEdge `json:"depends_on,omitempty"`
}

// GraphEdge is one depends_on edge in the graph skim.
type GraphEdge struct {
	Service   string `json:"service"`
	Condition string `json:"condition"`
}

// RuntimeSummary is optional live state (filled by later targets; omitted offline).
type RuntimeSummary struct {
	Available bool             `json:"available"`
	Services  []RuntimeService `json:"services,omitempty"`
}

// RuntimeService is a compact per-service runtime row for orientation.
type RuntimeService struct {
	Name    string `json:"name"`
	Missing bool   `json:"missing,omitempty"`
	Running bool   `json:"running,omitempty"`
	State   string `json:"state,omitempty"`
	Healthy *bool  `json:"healthy,omitempty"`
}

// BuildOptions controls offline orientation scope.
type BuildOptions struct {
	Profiles         []string
	DeployRoots      []string
	ExpandDependents bool
}

// Build builds an offline orientation document from a loaded contract.
// It does not call Podman. c must be non-nil.
func Build(c *spec.Contract, contractPath string, opts BuildOptions) (*Document, error) {
	if c == nil {
		return nil, fmt.Errorf("orientation: nil contract")
	}
	if contractPath == "" {
		return nil, fmt.Errorf("orientation: empty contract path")
	}

	profileActive := c.ServicesForProfiles(opts.Profiles)
	if len(c.Services) > 0 && len(profileActive) == 0 {
		return nil, fmt.Errorf("orientation: no services selected for this profile set (check --profile)")
	}
	if len(profileActive) == 0 && len(c.Services) == 0 {
		return nil, fmt.Errorf("orientation: no services defined")
	}

	obs, err := spec.ObservabilityActiveServices(profileActive, opts.DeployRoots, opts.ExpandDependents)
	if err != nil {
		return nil, fmt.Errorf("orientation: %w", err)
	}

	iterate := spec.ServiceNamesSorted(profileActive)
	if len(opts.DeployRoots) > 0 {
		iterate = spec.ServiceNamesSorted(obs)
	}

	doc := &Document{
		FormatVersion:  FormatVersion,
		Kind:           Kind,
		Project:        c.Project,
		ContractPath:   contractPath,
		Profiles:       append([]string(nil), opts.Profiles...),
		ActiveServices: iterate,
		Graph:          buildGraphSkim(profileActive, iterate),
		NextActions:    idleNextActions(contractPath, opts.DeployRoots),
		Note:           BoundaryNote,
	}
	if len(opts.DeployRoots) > 0 {
		doc.DeployServices = append([]string(nil), opts.DeployRoots...)
		doc.DependentsExpand = opts.ExpandDependents
	}
	return doc, nil
}

func buildGraphSkim(active map[string]spec.Service, names []string) []GraphService {
	out := make([]GraphService, 0, len(names))
	for _, name := range names {
		svc := active[name]
		gs := GraphService{Name: name}
		for _, d := range svc.DependsOn {
			cond := d.Condition
			if cond == "" {
				cond = spec.ConditionStarted
			}
			gs.DependsOn = append(gs.DependsOn, GraphEdge{
				Service:   d.Service,
				Condition: cond,
			})
		}
		out = append(out, gs)
	}
	return out
}

// AttachRuntime fills doc.Runtime from precomputed rows (e.g. explain service-status facts)
// and refreshes NextActions for the observed live state. Does not call Podman.
// When rows is nil/empty and available is false, Runtime is marked unavailable and idle next actions are kept.
func AttachRuntime(doc *Document, available bool, rows []RuntimeService) {
	if doc == nil {
		return
	}
	if !available {
		doc.Runtime = &RuntimeSummary{Available: false}
		doc.NextActions = idleNextActions(doc.ContractPath, doc.DeployServices)
		return
	}
	doc.Runtime = &RuntimeSummary{
		Available: true,
		Services:  append([]RuntimeService(nil), rows...),
	}
	doc.NextActions = liveNextActions(doc.ContractPath, doc.DeployServices, rows)
}

// idleNextActions returns ordered agent-loop gates for a cold/idle contract (no live runtime).
func idleNextActions(contractPath string, deployRoots []string) []string {
	f := "-f " + contractPath
	roots := rootSuffix(deployRoots)
	return []string{
		fmt.Sprintf("podbay validate %s%s --json", f, roots),
		fmt.Sprintf("podbay deploy %s%s --json", f, roots),
		fmt.Sprintf("podbay diff %s%s --json", f, roots),
		fmt.Sprintf("podbay explain %s%s --json", f, roots),
	}
}

func liveNextActions(contractPath string, deployRoots []string, rows []RuntimeService) []string {
	f := "-f " + contractPath
	roots := rootSuffix(deployRoots)
	focus := ""
	anyBad := false
	allMissing := len(rows) > 0
	for _, r := range rows {
		if !r.Missing {
			allMissing = false
		}
		bad := r.Missing || !r.Running || (r.Healthy != nil && !*r.Healthy)
		if bad {
			anyBad = true
			if focus == "" {
				focus = r.Name
			}
		}
	}
	if allMissing {
		return []string{
			fmt.Sprintf("podbay validate %s%s --json", f, roots),
			fmt.Sprintf("podbay deploy %s%s --json", f, roots),
			fmt.Sprintf("podbay explain %s%s --json", f, roots),
		}
	}
	if anyBad {
		svc := ""
		if focus != "" {
			svc = " " + focus
		}
		return []string{
			fmt.Sprintf("podbay logs%s --json", svc),
			fmt.Sprintf("podbay explain%s --json", svc),
			"podbay down --json",
		}
	}
	return []string{
		fmt.Sprintf("podbay diff %s%s --json", f, roots),
		fmt.Sprintf("podbay logs%s --json", roots),
		fmt.Sprintf("podbay explain %s%s --json", f, roots),
	}
}

func rootSuffix(deployRoots []string) string {
	if len(deployRoots) == 0 {
		return ""
	}
	roots := ""
	for _, r := range deployRoots {
		roots += " " + r
	}
	return roots
}
