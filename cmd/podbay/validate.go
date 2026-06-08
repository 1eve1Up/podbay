package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/validate"
)

func validateCmd(fileFlag *string, defaultFile string) *cobra.Command {
	var profiles []string
	jsonOut := false
	dependents := false
	cmd := &cobra.Command{
		Use:   "validate [podbay.yaml|directory] [service ...]",
		Args:  cobra.ArbitraryArgs,
		Short: "Validate contract and run preflight checks",
		Long: `Run preflight checks and contract validation.

With -f / --file: optional extra arguments are service names for partial validation (explicit targets only by default). Use --dependents to expand the checked set to the transitive closure of services that depend on any explicit target, within the profile-active map. Zero extras validates the full active set.

Without -f: use "validate path [service ...]" — a single argument is either a contract path (if that path or directory/podbay.yaml exists) or a service name when ./podbay.yaml exists and defines that service; additional arguments are partial-deploy service roots.

With --json: print one versioned JSON document (format_version, kind validate) for agents and CI;
exit code 1 if any fail-level check does not pass (same as text mode).`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, path, deployServices, err := loadContractWithDeployServices(*fileFlag, args, defaultFile)
			if err != nil {
				return err
			}
			out := validate.NewRunOutcome(c, path, profiles, deployServices, dependents)
			if jsonOut {
				proj := projectName(c, path)
				doc := clijson.FromValidate(path, proj, profiles, deployServices, out.Results, dependents)
				raw, err := clijson.MarshalIndent(doc)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
				if out.HasFailure() {
					os.Exit(1)
				}
				return nil
			}
			if err := out.PrintText(os.Stdout); err != nil {
				return err
			}
			if out.HasFailure() {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&profiles, "profile", nil, "enable services with this Compose-style profile (repeatable)")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "with partial targets, include transitive dependents within the profile-active set")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON (format_version) for agents and CI")
	return cmd
}
