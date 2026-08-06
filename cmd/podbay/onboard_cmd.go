package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/explain"
	"github.com/1eve1Up/podbay/internal/orientation"
)

func onboardCmd(fileFlag *string, defaultFile string) *cobra.Command {
	var profiles []string
	jsonOut := false
	dependents := false
	cmd := &cobra.Command{
		Use:   "onboard [podbay.yaml|directory] [service ...]",
		Short: "Orient agents and humans: project shape, graph skim, next CLI steps",
		Long: `Print structured orientation for a contract: identity, service graph skim, and
ordered next-step CLI hints aligned with the agent-loop playbook.

Offline-first: identity and graph skim do not require Podman.
When Podman is available, a compact live runtime summary is attached.

With --json: emit the shared orientation document (kind orientation).
Without --json: print a short human summary.

Partial roots and --dependents / --profile match validate / explain selection rules.

Orientation is structured context and next-steps only — not diagnosis or remediation.`,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, path, deployServices, err := loadContractWithDeployServices(*fileFlag, args, defaultFile)
			if err != nil {
				if jsonOut {
					return writeOnboardFailure(cmd, pathHint(fileFlag, args, defaultFile), err)
				}
				return err
			}
			proj := projectName(c, path)
			doc, err := explain.OrientationReport(c, path, proj, profiles, deployServices, dependents)
			if err != nil {
				if jsonOut {
					return writeOnboardFailure(cmd, path, err)
				}
				return err
			}
			if jsonOut {
				raw, mErr := json.MarshalIndent(doc, "", "  ")
				if mErr != nil {
					return mErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
				return nil
			}
			writeOnboardHuman(cmd, doc)
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&profiles, "profile", nil, "enable services with this profile (repeatable)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit orientation JSON (format_version, kind orientation)")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "with partial service targets, include transitive dependents within the profile-active set")
	return cmd
}

func pathHint(fileFlag *string, args []string, defaultFile string) string {
	p, _, err := contractPathAndDeployServices(*fileFlag, args, defaultFile)
	if err != nil || p == "" {
		return defaultFile
	}
	return expectedContractPath(p)
}

type onboardFailDoc struct {
	FormatVersion int                `json:"format_version"`
	Kind          string             `json:"kind"`
	Status        string             `json:"status"`
	ContractPath  string             `json:"contract_path,omitempty"`
	Issues        []onboardFailIssue `json:"issues"`
	NextActions   []string           `json:"next_actions"`
	Note          string             `json:"note"`
}

type onboardFailIssue struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeOnboardFailure(cmd *cobra.Command, contractPath string, err error) error {
	doc := onboardFailDoc{
		FormatVersion: orientation.FormatVersion,
		Kind:          orientation.Kind,
		Status:        "failed",
		ContractPath:  contractPath,
		Issues: []onboardFailIssue{{
			Level:   "fail",
			Code:    "orientation_load_error",
			Message: err.Error(),
		}},
		NextActions: []string{},
		Note:        orientation.BoundaryNote,
	}
	raw, mErr := json.MarshalIndent(doc, "", "  ")
	if mErr != nil {
		return mErr
	}
	fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
	return err
}

func writeOnboardHuman(cmd *cobra.Command, doc *orientation.Document) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Project: %s\n", doc.Project)
	fmt.Fprintf(out, "Contract: %s\n", doc.ContractPath)
	fmt.Fprintf(out, "Active services (%d): %s\n", len(doc.ActiveServices), strings.Join(doc.ActiveServices, ", "))
	if doc.Runtime != nil && doc.Runtime.Available {
		fmt.Fprintf(out, "Runtime: available (%d services)\n", len(doc.Runtime.Services))
	} else {
		fmt.Fprintln(out, "Runtime: offline / unavailable")
	}
	fmt.Fprintln(out, "Next steps (structured hints, not diagnosis):")
	for _, a := range doc.NextActions {
		fmt.Fprintf(out, "  - %s\n", a)
	}
	fmt.Fprintf(out, "Note: %s\n", doc.Note)
}
