package explain

import (
	"path/filepath"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/orientation"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
)

// OrientationReport builds the shared orientation document.
// Offline fields always populate when the contract loads.
// When Podman is available, attaches live runtime summary from the same
// inspect/probe path as explain; otherwise leaves runtime omitted/unavailable.
func OrientationReport(c *spec.Contract, contractPath, project string, profiles, deployRoots []string, expandDependents bool) (*orientation.Document, error) {
	doc, err := orientation.Build(c, contractPath, orientation.BuildOptions{
		Profiles:         profiles,
		DeployRoots:      deployRoots,
		ExpandDependents: expandDependents,
	})
	if err != nil {
		return nil, err
	}
	if err := runner.EnsurePodman(); err != nil {
		return doc, nil
	}
	hostSubst, err := expand.LoadHostSubst(filepath.Dir(contractPath), c.HostEnvFiles)
	if err != nil {
		return doc, nil
	}
	profileActive := c.ServicesForProfiles(profiles)
	obs, err := spec.ObservabilityActiveServices(profileActive, deployRoots, expandDependents)
	if err != nil {
		return doc, nil
	}
	iterate := spec.ServiceNamesSorted(profileActive)
	if len(deployRoots) > 0 {
		iterate = spec.ServiceNamesSorted(obs)
	}
	r := runner.New(project)
	states, err := inspectServiceContainers(r, iterate)
	if err != nil {
		orientation.AttachRuntime(doc, false, nil)
		return doc, nil
	}
	statuses := make([]ServiceStatus, 0, len(iterate))
	for _, svcName := range iterate {
		cname := r.ContainerName(svcName)
		st := collectServiceStatus(r, svcName, profileActive[svcName], hostSubst, states[cname], nil)
		statuses = append(statuses, st)
	}
	orientation.AttachRuntime(doc, true, RuntimeRowsFromStatus(statuses))
	return doc, nil
}
