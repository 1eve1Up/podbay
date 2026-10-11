package explain

import (
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
	sess, err := prepareExplain(c, contractPath, project, profiles, deployRoots, expandDependents, false)
	if err != nil {
		orientation.AttachRuntime(doc, false, nil)
		return doc, nil
	}
	orientation.AttachRuntime(doc, true, RuntimeRowsFromStatus(sess.statuses))
	return doc, nil
}
