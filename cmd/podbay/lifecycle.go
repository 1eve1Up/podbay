package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/spec"
	"github.com/1eve1Up/podbay/internal/teardown"
)

func teardownCmd(fileFlag *string, defaultFile string) *cobra.Command {
	return newTeardownLikeCmd("teardown", "Remove Podman containers, network, and optionally volumes for this contract", fileFlag, defaultFile)
}

func downCmd(fileFlag *string, defaultFile string) *cobra.Command {
	return newTeardownLikeCmd("down", "Compose-style alias for teardown: remove containers, network, and optionally volumes", fileFlag, defaultFile)
}

func newTeardownLikeCmd(use, short string, fileFlag *string, defaultFile string) *cobra.Command {
	var profiles []string
	quiet := false
	rmVolumes := false
	keepNetwork := false
	jsonOut := false
	dependents := false
	cmd := &cobra.Command{
		Use:   use + " [podbay.yaml|directory] [service ...]",
		Args:  cobra.ArbitraryArgs,
		Short: short,
		Long: fmt.Sprintf(`%s removes Podman containers, the project network, and optionally named volumes.

With -f / --file: optional trailing arguments are service names for partial teardown (same rules as validate/deploy/ps). Use --dependents to include transitive dependents within the profile-active map. With no extra arguments, all project-labeled containers are removed (full teardown).

Without -f: use "%s path [service ...]" — a single argument is either a contract path or a service name when ./podbay.yaml exists; additional arguments are partial teardown roots.

Partial teardown skips removing the project network while any project-labeled container remains, and rejects --volumes until you run a full teardown (omit service names).

With --json: print one versioned JSON document (format_version, kind teardown) on stdout instead of plain text.

Exit codes:
  0  Teardown finished without a fatal error (network removal warnings still yield 0).
  1  Contract load failure, Podman unavailable, list/remove failure, or volume removal error.`, use, use),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, path, deployServices, err := loadContractWithDeployServices(*fileFlag, args, defaultFile)
			if err != nil {
				if jsonOut {
					emitTeardownLoadJSON(cmd, "", "", profiles, err.Error())
					os.Exit(1)
				}
				return err
			}
			proj := projectName(c, path)
			profileActive := c.ServicesForProfiles(profiles)
			var partialServices []string
			if len(deployServices) > 0 || dependents {
				active, err2 := spec.ObservabilityActiveServices(profileActive, deployServices, dependents)
				if err2 != nil {
					if jsonOut {
						emitTeardownLoadJSON(cmd, path, proj, profiles, err2.Error())
						os.Exit(1)
					}
					return err2
				}
				partialServices = spec.ServiceNamesSorted(active)
			}
			opt := teardown.Options{
				Quiet:           jsonOut || quiet,
				Out:             cmd.OutOrStdout(),
				Volumes:         rmVolumes,
				KeepNetwork:     keepNetwork,
				PartialServices: partialServices,
			}
			if jsonOut {
				res, runErr := teardown.Execute(c, proj, opt)
				emitTeardownJSON(cmd, path, proj, profiles, deployServices, dependents, res, runErr)
				if teardown.ExitCode(runErr) != 0 {
					os.Exit(1)
				}
				return nil
			}
			return teardown.Run(c, proj, opt)
		},
	}
	cmd.Flags().StringSliceVar(&profiles, "profile", nil, "enable services with this Compose-style profile (repeatable)")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress progress output")
	cmd.Flags().BoolVarP(&rmVolumes, "volumes", "v", false, "also remove named volumes declared in the contract (destructive; not allowed with partial service selection)")
	cmd.Flags().BoolVar(&keepNetwork, "keep-network", false, "do not run podman network rm for the project network")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "with partial service targets, include transitive dependents within the profile-active set")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON (format_version) for agents and CI")
	return cmd
}

func emitTeardownJSON(cmd *cobra.Command, contractPath, project string, profiles []string, deployServices []string, dependentsExpand bool, res teardown.TeardownResult, runErr error) {
	doc := clijson.FromTeardown(contractPath, project, profiles, deployServices, dependentsExpand, res, runErr)
	raw, err := clijson.MarshalIndent(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
}

func emitTeardownLoadJSON(cmd *cobra.Command, contractPath, project string, profiles []string, msg string) {
	doc := clijson.TeardownLoadError(contractPath, project, profiles, msg)
	raw, err := clijson.MarshalIndent(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
}
