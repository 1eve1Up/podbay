package deploy

import (
	"io"
	"time"

	"github.com/1eve1Up/podbay/internal/spec"
)

// Options for Deploy.
type Options struct {
	Profiles []string
	// DeployServices, when non-empty, selects a partial deploy: each name must be in the profile-active set.
	// The effective set is explicit targets unless DeployDependents is true (see spec.ExpandDependentsTransitive).
	DeployServices []string
	// DeployDependents expands partial deploy to include transitive downstream services (depends_on closure).
	DeployDependents bool
	SkipHealthWait   bool
	HealthTimeout    time.Duration
	// Quiet suppresses progress output (similar to docker compose --quiet).
	Quiet bool
	// Out is the writer for progress when Quiet is false. If nil, os.Stdout is used.
	Out io.Writer
	// ReceiptPath, if non-empty, is the path for a deploy receipt JSON written only after a fully successful deploy.
	ReceiptPath string
}

// Deploy applies the contract in phases:
//  1. Resolve active services and host substitution (setup)
//  2. prepareNetworks
//  3. prepareVolumes
//  4. deployServicesInOrder
//  5. writeDeployReceipt (when --receipt set)
func Deploy(c *spec.Contract, contractFile string, project string, opt Options) error {
	ctx, err := newDeployContext(c, contractFile, project, opt)
	if err != nil {
		return err
	}

	if err := prepareNetworks(ctx); err != nil {
		return err
	}

	volMap, err := prepareVolumes(ctx)
	if err != nil {
		return err
	}

	order, err := deployServicesInOrder(ctx, volMap)
	if err != nil {
		return err
	}

	return writeDeployReceipt(ctx, order)
}
