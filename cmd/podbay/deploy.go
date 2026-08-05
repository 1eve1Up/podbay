package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/deploy"
	"github.com/1eve1Up/podbay/internal/validate"
)

func deployCmd(fileFlag *string, defaultFile string) *cobra.Command {
	skipHealth := false
	quiet := false
	jsonOut := false
	healthTimeout := 120 * time.Second
	var profiles []string
	receiptPath := ""
	dependents := false
	cmd := &cobra.Command{
		Use:   "deploy [podbay.yaml|directory] [service ...]",
		Args:  cobra.ArbitraryArgs,
		Short: "Compile contract to Podman: network, volumes, containers, validation gates",
		Long: `Apply the contract to Podman (network, volumes, containers, health gates).

With -f / --file: optional extra arguments are service names for partial deploy (explicit targets only by default). Use --dependents to expand the deployed set to the transitive closure of services that depend on any explicit target, within the profile-active map. Zero extras deploys the full active set.

Without -f: use "deploy path [service ...]" — a single argument is either a contract path (if that path exists) or a service name when ./podbay.yaml exists and defines that service; additional arguments select partial-deploy roots.

After a fully successful deploy, --receipt PATH writes a versioned JSON receipt (atomic write).
On health-gate failure with --receipt set, writes an attempt receipt (status failed) the same way.
If PATH is a directory (or ends with /), writes <dir>/<UTC>-<deploy_id>.json instead.
No receipt on preflight validate failure or when --receipt is unset.

With --json: print one versioned JSON document (format_version, kind deploy) on stdout; progress output is
suppressed for machine-readable runs. Exit code 1 on preflight or deploy failure.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, path, deployServices, err := loadContractWithDeployServices(*fileFlag, args, defaultFile)
			if err != nil {
				return err
			}
			vout := deploy.PreflightOutcome(c, path, profiles, deployServices, dependents)
			proj := projectName(c, path)

			if jsonOut {
				for _, r := range vout.Results {
					if r.Level == validate.LevelFail && !r.OK {
						doc := clijson.DeployFromValidateResults(path, proj, profiles, deployServices, vout.Results, dependents)
						raw, mErr := clijson.MarshalIndent(doc)
						if mErr != nil {
							return mErr
						}
						fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
						os.Exit(1)
					}
				}
				var writtenReceipt string
				depQuiet := quiet || jsonOut
				depErr := deploy.Deploy(c, path, proj, deploy.Options{
					Profiles:           profiles,
					DeployServices:     deployServices,
					DeployDependents:   dependents,
					SkipHealthWait:     skipHealth,
					HealthTimeout:      healthTimeout,
					Quiet:              depQuiet,
					Out:                cmd.OutOrStdout(),
					ReceiptPath:        receiptPath,
					WrittenReceiptPath: &writtenReceipt,
				})
				doc := clijson.DeployOutcome(path, proj, profiles, deployServices, writtenReceipt, depErr, dependents)
				raw, mErr := clijson.MarshalIndent(doc)
				if mErr != nil {
					return mErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
				if depErr != nil {
					os.Exit(1)
				}
				return nil
			}

			for _, r := range vout.Results {
				if r.Level == validate.LevelFail && !r.OK {
					return fmt.Errorf("validate failed: %s", r.Message)
				}
			}
			return deploy.Deploy(c, path, proj, deploy.Options{
				Profiles:         profiles,
				DeployServices:   deployServices,
				DeployDependents: dependents,
				SkipHealthWait:   skipHealth,
				HealthTimeout:    healthTimeout,
				Quiet:            quiet,
				Out:              cmd.OutOrStdout(),
				ReceiptPath:      receiptPath,
			})
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress progress output (no build stream or status lines)")
	cmd.Flags().BoolVar(&skipHealth, "skip-health-wait", false, "do not wait for health gates after start")
	cmd.Flags().DurationVar(&healthTimeout, "health-timeout", 120*time.Second, "max wall time per service health gate (HTTP or exec)")
	cmd.Flags().StringSliceVar(&profiles, "profile", nil, "enable services with this profile (repeatable)")
	cmd.Flags().StringVar(&receiptPath, "receipt", "", "after success, write deploy receipt JSON to this path (or directory for <UTC>-<deploy_id>.json; atomic; omitted on failure)")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "with partial targets, include transitive dependents within the profile-active set")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON (format_version) for agents and CI")
	return cmd
}
