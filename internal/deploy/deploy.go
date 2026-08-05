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
	// ReceiptPath, if non-empty, is the path for a deploy receipt JSON.
	// After a fully successful deploy, a status=ok receipt is written.
	// On HealthGateFailure (after deploy has started), a status=failed attempt receipt is written.
	// When ReceiptPath is an existing directory or ends with "/", a unique file
	// <dir>/<UTC>-<deploy_id>.json is written instead.
	ReceiptPath string
	// WrittenReceiptPath, when non-nil, is set to the absolute path of the receipt file after a successful write
	// (success or in-scope attempt receipt).
	WrittenReceiptPath *string
}

// Deploy applies the contract in phases:
//  1. Resolve active services and host substitution (setup)
//  2. prepareNetworks
//  3. prepareVolumes
//  4. deployServicesInOrder
//  5. writeDeployReceipt (when --receipt set) or attempt receipt on HealthGateFailure
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
		_ = maybeWriteAttemptReceipt(ctx, order, err)
		return err
	}

	return writeDeployReceipt(ctx, order)
}
