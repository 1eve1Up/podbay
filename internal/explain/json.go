package explain

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

// ExplainJSONFormatVersion is bumped when breaking the explain --json document shape.
const ExplainJSONFormatVersion = 1

type explainJSONV1 struct {
	FormatVersion        int            `json:"format_version"`
	Project              string         `json:"project"`
	ContractPath         string         `json:"contract_path"`
	Profiles             []string       `json:"profiles,omitempty"`
	DeployServices       []string       `json:"deploy_services,omitempty"`
	DependentsExpand     bool           `json:"dependents_expand,omitempty"`
	ActiveServices       []string       `json:"active_services"`
	Focus                string         `json:"focus,omitempty"`
	Dependencies         *focusDepsJSON `json:"dependencies,omitempty"`
	Services             []serviceJSON  `json:"services"`
	UnexpectedContainers []string       `json:"unexpected_containers,omitempty"`
}

type focusDepsJSON struct {
	DependsOn   []depEdgeJSON `json:"depends_on"`
	Dependents  []depEdgeJSON `json:"dependents"`
	DeployOrder []string      `json:"deploy_order,omitempty"`
}

type depEdgeJSON struct {
	Service           string `json:"service"`
	Condition         string `json:"condition"`
	InactiveInProfile bool   `json:"inactive_in_profile,omitempty"`
}

type serviceJSON struct {
	Name         string             `json:"name"`
	Container    string             `json:"container"`
	Running      bool               `json:"running"`
	Missing      bool               `json:"missing,omitempty"`
	InspectError string             `json:"inspect_error,omitempty"`
	State        string             `json:"state,omitempty"`
	ExitCode     int                `json:"exit_code,omitempty"`
	StateError   string             `json:"state_error,omitempty"`
	Health       *serviceHealthJSON `json:"health,omitempty"`
}

type serviceHealthJSON struct {
	HTTP *httpHealthJSON `json:"http,omitempty"`
	Exec *execHealthJSON `json:"exec,omitempty"`
}

type httpHealthJSON struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

type execHealthJSON struct {
	Ok     bool   `json:"ok"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ReportJSON returns indented JSON for agents and CI (same inputs as Report).
func ReportJSON(c *spec.Contract, contractPath, project string, profiles []string, deployRoots []string, expandDependents bool) ([]byte, error) {
	if err := runner.EnsurePodman(); err != nil {
		return nil, err
	}
	hostSubst, err := expand.LoadHostSubst(filepath.Dir(contractPath), c.HostEnvFiles)
	if err != nil {
		return nil, err
	}

	r := runner.New(project)
	profileActive := c.ServicesForProfiles(profiles)
	if len(c.Services) > 0 && len(profileActive) == 0 {
		return nil, fmt.Errorf("no services selected for this profile set (check --profile)")
	}
	if len(profileActive) == 0 && len(c.Services) == 0 {
		return nil, fmt.Errorf("no services defined")
	}

	obs, err := spec.ObservabilityActiveServices(profileActive, deployRoots, expandDependents)
	if err != nil {
		return nil, err
	}

	iterate := spec.ServiceNamesSorted(profileActive)
	if len(deployRoots) > 0 {
		iterate = spec.ServiceNamesSorted(obs)
	}

	focusOut := ""
	if len(deployRoots) > 0 && len(iterate) == 1 {
		focusOut = iterate[0]
	}

	svcs := make([]serviceJSON, 0, len(iterate))
	for _, svcName := range iterate {
		st := collectServiceStatus(r, svcName, profileActive[svcName], hostSubst)
		svcs = append(svcs, serviceStatusToJSON(st))
	}

	out := explainJSONV1{
		FormatVersion:  ExplainJSONFormatVersion,
		Project:        project,
		ContractPath:   contractPath,
		Profiles:       profiles,
		ActiveServices: iterate,
		Focus:          focusOut,
		Services:       svcs,
	}
	if len(deployRoots) > 0 {
		out.DeployServices = append([]string(nil), deployRoots...)
		out.DependentsExpand = expandDependents && len(out.DeployServices) > 0
	}
	if focusOut != "" {
		out.Dependencies = buildFocusDepsJSON(profileActive, focusOut)
	}

	extrasSeed := spec.ServiceNamesSorted(profileActive)
	extra, err := runtimestate.ExtraContainerNames(r, extrasSeed)
	if err != nil {
		return nil, err
	}
	out.UnexpectedContainers = extra

	return json.MarshalIndent(out, "", "  ")
}

func buildFocusDepsJSON(active map[string]spec.Service, focus string) *focusDepsJSON {
	svc, ok := active[focus]
	if !ok {
		return nil
	}
	f := &focusDepsJSON{}
	for _, d := range svc.DependsOn {
		_, in := active[d.Service]
		f.DependsOn = append(f.DependsOn, depEdgeJSON{
			Service:           d.Service,
			Condition:         conditionLabel(d.Condition),
			InactiveInProfile: !in,
		})
	}
	for name, s := range active {
		if name == focus {
			continue
		}
		for _, d := range s.DependsOn {
			if d.Service == focus {
				f.Dependents = append(f.Dependents, depEdgeJSON{
					Service:   name,
					Condition: conditionLabel(d.Condition),
				})
				break
			}
		}
	}
	sort.Slice(f.Dependents, func(i, j int) bool { return f.Dependents[i].Service < f.Dependents[j].Service })

	if order, err := spec.TopologicalOrder(active); err == nil && len(order) > 0 {
		f.DeployOrder = order
	}
	return f
}

func serviceStatusToJSON(st ServiceStatus) serviceJSON {
	j := serviceJSON{
		Name:         st.Name,
		Container:    st.Container,
		Running:      st.Running,
		Missing:      st.Missing,
		InspectError: st.InspectErr,
		State:        st.State,
		ExitCode:     st.ExitCode,
		StateError:   st.StateError,
	}
	health := &serviceHealthJSON{}
	hasHealth := false
	if st.HTTPURL != "" {
		hasHealth = true
		h := httpHealthJSON{URL: st.HTTPURL, StatusCode: st.HTTPStatus}
		if st.HTTPProbeErr != "" {
			h.Error = st.HTTPProbeErr
		}
		health.HTTP = &h
	}
	if st.ExecRan {
		hasHealth = true
		e := execHealthJSON{Ok: st.ExecErr == "", Output: st.ExecOutput}
		if st.ExecErr != "" {
			e.Error = st.ExecErr
		}
		health.Exec = &e
	}
	if hasHealth {
		j.Health = health
	}
	return j
}
