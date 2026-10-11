package explain

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

// loadHostSubstFn is the single expand.LoadHostSubst call site for explain/onboard.
// Tests may wrap it to prove one load per invocation.
var loadHostSubstFn = expand.LoadHostSubst

func loadHostSubst(c *spec.Contract, contractPath string) (map[string]string, error) {
	return loadHostSubstFn(filepath.Dir(contractPath), c.HostEnvFiles)
}

type explainSession struct {
	hostSubst     map[string]string
	r             *runner.Runner
	profileActive map[string]spec.Service
	iterate       []string
	states        map[string]*runtimestate.ContainerState
	statuses      []ServiceStatus
	noProbes      bool
}

func prepareExplain(c *spec.Contract, contractPath, project string, profiles, deployRoots []string, expandDependents bool, noProbes bool) (*explainSession, error) {
	hostSubst, err := loadHostSubst(c, contractPath)
	if err != nil {
		return nil, err
	}
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
	r := runner.New(project)
	states, err := inspectServiceContainers(r, iterate)
	if err != nil {
		return nil, err
	}
	statuses := collectStatuses(r, iterate, profileActive, hostSubst, states, noProbes)
	return &explainSession{
		hostSubst:     hostSubst,
		r:             r,
		profileActive: profileActive,
		iterate:       iterate,
		states:        states,
		statuses:      statuses,
		noProbes:      noProbes,
	}, nil
}

// collectStatuses builds inspect+probe status for each in-scope service.
// When noProbes is false, services are probed concurrently; each probe still
// uses ExplainProbeMax. When noProbes is true, inspect-only status is gathered
// serially and HTTP/exec work is skipped.
func collectStatuses(r *runner.Runner, iterate []string, profileActive map[string]spec.Service, hostSubst map[string]string, states map[string]*runtimestate.ContainerState, noProbes bool) []ServiceStatus {
	statuses := make([]ServiceStatus, len(iterate))
	if noProbes || len(iterate) == 0 {
		for i, svcName := range iterate {
			cname := r.ContainerName(svcName)
			statuses[i] = collectServiceStatus(r, svcName, profileActive[svcName], hostSubst, states[cname], nil, true)
		}
		return statuses
	}
	var wg sync.WaitGroup
	for i, svcName := range iterate {
		wg.Add(1)
		go func(i int, svcName string) {
			defer wg.Done()
			cname := r.ContainerName(svcName)
			statuses[i] = collectServiceStatus(r, svcName, profileActive[svcName], hostSubst, states[cname], nil, false)
		}(i, svcName)
	}
	wg.Wait()
	return statuses
}
