package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/diff"
	"github.com/1eve1Up/podbay/internal/explain"
	"github.com/1eve1Up/podbay/internal/logs"
	"github.com/1eve1Up/podbay/internal/ps"
	"github.com/1eve1Up/podbay/internal/receipt"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
)

func emitLogsJSON(cmd *cobra.Command, doc *clijson.Document) {
	raw, err := clijson.MarshalIndent(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
}

func emitLogsJSONFailure(cmd *cobra.Command, contractPath, project string, profiles, deployServices []string, dependents bool, service, code, msg string) {
	emitLogsJSON(cmd, clijson.LogsFailurePartial(contractPath, project, profiles, deployServices, dependents, service, code, msg))
	os.Exit(1)
}

func psCmd(fileFlag *string, defaultFile string) *cobra.Command {
	var profiles []string
	jsonOut := false
	dependents := false
	cmd := &cobra.Command{
		Use:   "ps [podbay.yaml|directory] [service ...]",
		Args:  cobra.ArbitraryArgs,
		Short: "List Podman containers for active services (profile-aware)",
		Long: `Print one line per resolved service: service name, container name, state, and image when known.

With -f / --file: optional trailing arguments are service names for partial ps (explicit targets only by default).
Use --dependents to include the transitive closure of services that depend_on any explicit target, within the profile-active map.
Without -f: use "ps path [service ...]" — a single argument is either a contract path or a service name when ./podbay.yaml exists; additional arguments are partial roots.

Uses the same container naming, --profile, and partial selection rules as validate and deploy.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runner.EnsurePodman(); err != nil {
				return err
			}
			c, path, deployServices, err := loadContractWithDeployServices(*fileFlag, args, defaultFile)
			if err != nil {
				return err
			}
			proj := projectName(c, path)
			inspect := runtimestate.InspectContainer
			if jsonOut {
				raw, err := ps.ReportJSON(c, path, proj, profiles, deployServices, dependents, inspect)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
				return nil
			}
			rows, err := ps.ListRows(c, proj, profiles, deployServices, dependents, inspect)
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "SERVICE\tCONTAINER\tSTATE\tIMAGE")
			for _, rw := range rows {
				stateCol := rw.State
				if rw.Error != "" {
					stateCol = rw.State + ": " + rw.Error
				}
				img := rw.Image
				if img == "" && (rw.Missing || rw.State == "error") {
					img = "—"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", rw.Service, rw.Container, stateCol, img)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringSliceVar(&profiles, "profile", nil, "enable services with this Compose-style profile (repeatable)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON (format_version) for agents and CI")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "with partial service targets, include transitive dependents within the profile-active set")
	return cmd
}

func logsCmd(fileFlag *string, defaultFile string) *cobra.Command {
	var profiles []string
	follow := false
	tailN := 0
	since := ""
	jsonOut := false
	dependents := false
	cmd := &cobra.Command{
		Use:   "logs [podbay.yaml|directory] [service ...]",
		Args:  cobra.ArbitraryArgs,
		Short: "Show logs for service containers (podman logs)",
		Long: `Print or stream logs for Podman containers backing resolved services.

With -f / --file: optional trailing arguments are service names for partial logs (explicit targets only by default).
Use --dependents to include the transitive closure of services that depend_on any explicit target, within the profile-active map.
Without -f: use "logs path [service ...]" — same partial rules as validate, deploy, ps, and explain.
With no service arguments after the contract path, all profile-active services are included.

--follow streams logs for a single resolved service only (not with --json).

With --json: print one versioned JSON document (format_version, kind logs) on stdout. Cannot be used with --follow.
Success includes log_entries[] for each resolved service; when exactly one service is resolved, top-level service and log_body are also set.

Exit codes:
  0  Logs printed or JSON success.
  1  Contract load failure, resolution error, podman unavailable, podman logs error, --json with --follow, or --follow with multiple services.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOut && follow {
				emitLogsJSONFailure(cmd, "", "", profiles, nil, false, "", clijson.CodeLogsUsageJSONFollow, "logs: --json cannot be used with --follow")
			}
			p, deployServices, err := contractPathAndDeployServices(*fileFlag, args, defaultFile)
			if err != nil {
				if jsonOut {
					emitLogsJSONFailure(cmd, "", "", profiles, deployServices, dependents, "", clijson.CodeLogsUsageArgs, err.Error())
				}
				return err
			}
			c, path, err := spec.Load(p)
			if err != nil {
				failPath := path
				if failPath == "" {
					failPath = p
				}
				pabs, _ := filepath.Abs(failPath)
				loadErr := augmentContractLoadError(p, err)
				if jsonOut {
					emitLogsJSONFailure(cmd, pabs, "", profiles, deployServices, dependents, "", clijson.CodeLogsLoadError, loadErr.Error())
				}
				return loadErr
			}
			absContract, _ := filepath.Abs(path)
			proj := projectName(c, path)
			active, err := logs.ActiveServices(c, profiles, deployServices, dependents)
			if err != nil {
				msg := logs.ResolveErrorMessage(err)
				if jsonOut {
					emitLogsJSONFailure(cmd, absContract, proj, profiles, deployServices, dependents, "", clijson.CodeLogsResolveError, msg)
				}
				return err
			}
			if len(active) == 0 {
				msg := "no services selected (check --profile and service names)"
				if jsonOut {
					emitLogsJSONFailure(cmd, absContract, proj, profiles, deployServices, dependents, "", clijson.CodeLogsResolveError, msg)
				}
				return fmt.Errorf("logs: %s", msg)
			}
			if follow && len(active) > 1 {
				msg := fmt.Sprintf("logs: --follow supports only one service (resolved %d)", len(active))
				code := clijson.CodeLogsFollowMulti
				if jsonOut {
					emitLogsJSONFailure(cmd, absContract, proj, profiles, deployServices, dependents, "", code, msg)
				}
				return fmt.Errorf("%s", msg)
			}
			if err := runner.EnsurePodman(); err != nil {
				if jsonOut {
					emitLogsJSONFailure(cmd, absContract, proj, profiles, deployServices, dependents, "", clijson.CodeLogsPodmanUnavailable, err.Error())
				}
				return err
			}
			if jsonOut {
				captured, err := logs.CaptureBytes(proj, active, tailN, since)
				if err != nil {
					emitLogsJSONFailure(cmd, absContract, proj, profiles, deployServices, dependents, "", clijson.CodeLogsRuntimeError, err.Error())
				}
				entries := make([]clijson.LogEntry, len(captured))
				for i, e := range captured {
					entries[i] = clijson.LogEntry{
						Service:       e.Service,
						ContainerName: e.Container,
						LogBody:       e.Body,
					}
				}
				emitLogsJSON(cmd, clijson.FromLogsBatchSuccess(absContract, proj, profiles, deployServices, dependents, tailN, since, entries))
				return nil
			}
			if err := logs.PrintHuman(cmd.OutOrStdout(), proj, active, follow, tailN, since); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&profiles, "profile", nil, "enable services with this Compose-style profile (repeatable)")
	cmd.Flags().BoolVar(&follow, "follow", false, "follow log output (podman logs -f); single resolved service only")
	cmd.Flags().IntVar(&tailN, "tail", 0, "only show the last N lines (0 means all lines)")
	cmd.Flags().StringVar(&since, "since", "", "only show logs since timestamp (podman logs --since), e.g. 10m or 2024-01-01")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON (format_version, kind logs) for agents and CI")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "with partial service targets, include transitive dependents within the profile-active set")
	return cmd
}

func explainCmd(fileFlag *string, defaultFile string) *cobra.Command {
	var profiles []string
	jsonOut := false
	dependents := false
	cmd := &cobra.Command{
		Use:   "explain [podbay.yaml|directory] [service ...]",
		Short: "Describe expected vs actual runtime (Podman + health probes)",
		Long: `Without partial service roots: prints every profile-active service (historical full-list behavior).

With partial roots (same rules as validate/deploy): trailing service names narrow the explained set.
Use --dependents to expand to transitive dependents within the profile-active map.
With -f / --file: optional trailing arguments are partial roots; zero extras means the full profile-active set.

When partial roots select exactly one service, dependency context (depends_on, dependents, deploy order) is included (same as the old single-service positional form).

With --json: print one JSON document (format_version) for agents and CI instead of plain text.`,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, path, deployServices, err := loadContractWithDeployServices(*fileFlag, args, defaultFile)
			if err != nil {
				return err
			}
			proj := projectName(c, path)
			if jsonOut {
				raw, err := explain.ReportJSON(c, path, proj, profiles, deployServices, dependents)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(raw))
				return nil
			}
			s, err := explain.Report(c, path, proj, profiles, deployServices, dependents)
			if err != nil {
				return err
			}
			fmt.Print(s)
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&profiles, "profile", nil, "enable services with this profile (repeatable)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON (format_version) instead of text")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "with partial service targets, include transitive dependents within the profile-active set")
	return cmd
}

func diffCmd(fileFlag *string, defaultFile string) *cobra.Command {
	var profiles []string
	jsonOut := false
	receiptDiffShowEnv := false
	dependents := false
	mode := ""
	cmd := &cobra.Command{
		Use:   "diff [contract-path] [service ...]|diff <receipt-a> <receipt-b>",
		Short: "Compare contract to Podman or compare two deploy receipts (drift detection)",
		Long: `Two modes:

(1) Contract vs Podman — zero or one contract path (directory or podbay.yaml), or use -f / default podbay.yaml in cwd.
    With -f / --file: optional trailing arguments are service names for partial diff (explicit targets only by default).
    Use --dependents to include the transitive closure of services that depend_on any explicit target, within the profile-active map.
    Without -f: use "diff path [service ...]" — a single argument is either a contract path or a service name when ./podbay.yaml exists; additional arguments are partial roots.
    Compare expected containers for the selected set to Podman; project-labeled extras are detected against the full profile-active set.
    Use --profile to select Compose-style profiles (repeatable).

(2) Receipt pair — exactly two file arguments that both decode as deploy receipt JSON (from deploy --receipt).
    Compares the two recorded snapshots. --profile is not valid in this mode.
    Environment values in JSON output are redacted by default; --receipt-diff-show-env includes raw values (unsafe for CI logs).

With --json: print one versioned JSON document (format_version, kind diff) on stdout instead of plain text.
Contract mode: inspect errors per service appear under issues[] with code diff_inspect_error.
Receipt mode: structured payload under receipt_pair; load/decode failures use receipt_diff_load_error or receipt_diff_decode_error.

Exit codes:
  0  No drift — contract mode: every expected service is running and no extra project containers; receipt mode: compared receipt fields match.
  1  Drift, invalid inputs, or (contract mode) Podman/contract unavailable.`,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch strings.ToLower(strings.TrimSpace(mode)) {
			case "", "auto":
				// auto-detect below
			case "receipt", "receipt-pair":
				if len(args) != 2 {
					return fmt.Errorf("--mode receipt requires exactly two file arguments")
				}
				return runReceiptPairDiff(cmd, args[0], args[1], jsonOut, profiles, receiptDiffShowEnv)
			case "contract":
				// fall through to contract path below
			default:
				return fmt.Errorf("--mode must be auto, contract, or receipt (got %q)", mode)
			}
			if mode == "" || strings.EqualFold(mode, "auto") {
				if pathA, pathB, ok := diffArgsDecodeAsReceiptPair(args); ok {
					return runReceiptPairDiff(cmd, pathA, pathB, jsonOut, profiles, receiptDiffShowEnv)
				}
			}

			c, path, deployServices, err := loadContractWithDeployServices(*fileFlag, args, defaultFile)
			if err != nil {
				if jsonOut {
					emitDiffErrorJSON(cmd, "", "", profiles, "diff_load_error", err.Error())
					os.Exit(1)
				}
				return err
			}
			proj := projectName(c, path)

			if jsonOut {
				res, runErr := diff.ReportContractResult(c, path, proj, profiles, deployServices, dependents)
				if runErr != nil {
					emitDiffErrorJSON(cmd, path, proj, profiles, "diff_runtime_error", runErr.Error())
					os.Exit(1)
				}
				emitDiffJSON(cmd, path, proj, profiles, deployServices, dependents, res)
				if code := diffJSONExitCode(res); code != 0 {
					os.Exit(code)
				}
				return nil
			}

			s, drift, err := diff.Report(c, path, proj, profiles, deployServices, dependents)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), s)
			if drift {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&profiles, "profile", nil, "enable services with this Compose-style profile (repeatable; contract diff only)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON (format_version) for agents and CI")
	cmd.Flags().BoolVar(&receiptDiffShowEnv, "receipt-diff-show-env", false, "receipt pair only: include raw env values in JSON (default redacts; unsafe for logs)")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "with partial service targets, include transitive dependents within the profile-active set (contract diff only)")
	cmd.Flags().StringVar(&mode, "mode", "", "force diff mode: 'contract' (compare contract vs Podman) or 'receipt' (compare two receipt files); default auto-detects from arguments")
	return cmd
}

func runReceiptPairDiff(cmd *cobra.Command, pathA, pathB string, jsonOut bool, profiles []string, receiptDiffShowEnv bool) error {
	if len(profiles) > 0 {
		msg := "diff: --profile applies only to contract diff (zero or one contract path); omit --profile when comparing two receipt files"
		if jsonOut {
			emitReceiptPairErrorJSON(cmd, clijson.CodeReceiptDiffUsageError, msg)
			os.Exit(1)
		}
		return fmt.Errorf("%s", msg)
	}

	dataA, err := os.ReadFile(pathA)
	if err != nil {
		if jsonOut {
			emitReceiptPairErrorJSON(cmd, clijson.CodeReceiptDiffLoadError, fmt.Sprintf("read %s: %v", pathA, err))
			os.Exit(1)
		}
		return fmt.Errorf("diff receipts: read %s: %w", pathA, err)
	}
	dataB, err := os.ReadFile(pathB)
	if err != nil {
		if jsonOut {
			emitReceiptPairErrorJSON(cmd, clijson.CodeReceiptDiffLoadError, fmt.Sprintf("read %s: %v", pathB, err))
			os.Exit(1)
		}
		return fmt.Errorf("diff receipts: read %s: %w", pathB, err)
	}

	recA, err := receipt.Decode(dataA)
	if err != nil {
		if jsonOut {
			emitReceiptPairErrorJSON(cmd, clijson.CodeReceiptDiffDecodeError, fmt.Sprintf("%s: %v", pathA, err))
			os.Exit(1)
		}
		return fmt.Errorf("diff receipts: %s: %w", pathA, err)
	}
	recB, err := receipt.Decode(dataB)
	if err != nil {
		if jsonOut {
			emitReceiptPairErrorJSON(cmd, clijson.CodeReceiptDiffDecodeError, fmt.Sprintf("%s: %v", pathB, err))
			os.Exit(1)
		}
		return fmt.Errorf("diff receipts: %s: %w", pathB, err)
	}

	res := receipt.CompareReceipts(recA, recB)
	if jsonOut {
		if err := emitReceiptPairResultJSON(cmd, res, receiptDiffShowEnv); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if code := receiptPairJSONExitCode(res); code != 0 {
			os.Exit(code)
		}
		return nil
	}

	fmt.Fprint(cmd.OutOrStdout(), receipt.FormatReceiptDiff(res))
	if res.Drift {
		os.Exit(1)
	}
	return nil
}

func emitReceiptPairErrorJSON(cmd *cobra.Command, code, msg string) {
	doc := clijson.ReceiptPairDiffError(code, msg)
	raw, err := clijson.MarshalIndent(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
}

// emitReceiptPairResultJSON writes the KindDiff receipt_pair document for a completed comparison.
func emitReceiptPairResultJSON(cmd *cobra.Command, res receipt.ReceiptDiffResult, showRawEnv bool) error {
	doc := clijson.FromReceiptPairDiffWithOptions(res, clijson.ReceiptPairDiffOptions{ShowRawEnv: showRawEnv})
	raw, err := clijson.MarshalIndent(doc)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
	return nil
}

// receiptPairJSONExitCode matches runReceiptPairDiff --json: 1 when drift, else 0.
func receiptPairJSONExitCode(res receipt.ReceiptDiffResult) int {
	if res.Drift {
		return 1
	}
	return 0
}

// diffJSONExitCode mirrors the existing podbay diff exit-code contract for
// the --json path: 0 when no drift was detected, 1 otherwise. Load and
// runtime failures use os.Exit(1) directly in diffCmd; this helper covers
// only the success path so tests can assert the contract without spawning
// a subprocess.
func diffJSONExitCode(res diff.DriftResult) int {
	if res.Drift {
		return 1
	}
	return 0
}

func emitDiffJSON(cmd *cobra.Command, contractPath, project string, profiles []string, deployServices []string, dependents bool, res diff.DriftResult) {
	doc := clijson.FromDiffWithPartial(contractPath, project, profiles, deployServices, dependents, res)
	raw, err := clijson.MarshalIndent(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
}

func emitDiffErrorJSON(cmd *cobra.Command, contractPath, project string, profiles []string, code, msg string) {
	doc := clijson.DiffError(contractPath, project, profiles, code, msg)
	raw, err := clijson.MarshalIndent(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
}
