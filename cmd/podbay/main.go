package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/deploy"
	"github.com/1eve1Up/podbay/internal/diff"
	"github.com/1eve1Up/podbay/internal/explain"
	"github.com/1eve1Up/podbay/internal/ps"
	"github.com/1eve1Up/podbay/internal/receipt"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/runtimestate"
	"github.com/1eve1Up/podbay/internal/spec"
	"github.com/1eve1Up/podbay/internal/teardown"
	"github.com/1eve1Up/podbay/internal/validate"
)

const initTemplate = `# Podbay contract — https://github.com/1eve1Up/podbay
version: "1"
project: myapp

requirements:
  - type: command_exists
    command: podman

services:
  web:
    image: docker.io/library/nginx:alpine
    ports:
      - "8080:80"
    health:
      http:
        url: http://127.0.0.1:8080/
        timeout: 15s
    requirements:
      - type: port_available
        port: 8080

volumes: {}
networks: {}
`

func main() {
	root := &cobra.Command{
		Use:           "podbay",
		Short:         "Runtime contract layer for multi-agent delivery (Podman)",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	fileFlag := ""
	cwd, _ := os.Getwd()
	defaultFile := filepath.Join(cwd, spec.DefaultFilename)

	root.PersistentFlags().StringVarP(&fileFlag, "file", "f", "", "path to podbay.yaml or its directory, relative to the current working directory unless absolute (or pass the same path as a subcommand argument)")

	root.AddCommand(initCmd(&fileFlag, defaultFile))
	root.AddCommand(validateCmd(&fileFlag, defaultFile))
	root.AddCommand(deployCmd(&fileFlag, defaultFile))
	root.AddCommand(teardownCmd(&fileFlag, defaultFile))
	root.AddCommand(downCmd(&fileFlag, defaultFile))
	root.AddCommand(psCmd(&fileFlag, defaultFile))
	root.AddCommand(logsCmd(&fileFlag, defaultFile))
	root.AddCommand(explainCmd(&fileFlag, defaultFile))
	root.AddCommand(diffCmd(&fileFlag, defaultFile))
	root.AddCommand(receiptCmd())
	root.AddCommand(importCmd())
	root.AddCommand(versionCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func receiptCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "receipt <path-to-receipt.json>",
		Args:  cobra.ExactArgs(1),
		Short: "Read and summarize a deploy receipt JSON file",
		Long: `Load a receipt produced by podbay deploy --receipt.

Without --json: print a short human summary (project, contract path, services).

With --json: print a versioned envelope (kind receipt_read, format_version) containing the validated receipt.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := args[0]
			data, err := os.ReadFile(p)
			if err != nil {
				return fmt.Errorf("receipt: read %s: %w", p, err)
			}
			abs, err := filepath.Abs(p)
			if err != nil {
				abs = p
			}
			rec, err := receipt.Decode(data)
			if err != nil {
				if jsonOut {
					doc := clijson.ReceiptReadFailure(abs, err)
					raw, mErr := clijson.MarshalIndent(doc)
					if mErr != nil {
						return mErr
					}
					fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
					os.Exit(1)
				}
				return fmt.Errorf("receipt: %w", err)
			}
			enc, err := receipt.Encode(rec)
			if err != nil {
				return err
			}
			if jsonOut {
				doc := clijson.ReceiptReadSuccess(abs, enc)
				raw, err := clijson.MarshalIndent(doc)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Receipt format_version=%d\n", rec.FormatVersion)
			fmt.Fprintf(cmd.OutOrStdout(), "Project:     %s\n", rec.Project)
			fmt.Fprintf(cmd.OutOrStdout(), "Contract:    %s\n", rec.ContractPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Generated:   %s\n", rec.GeneratedAt.UTC().Format(time.RFC3339))
			if len(rec.Profiles) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Profiles:    %s\n", strings.Join(rec.Profiles, ", "))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Services:    %d\n", len(rec.Services))
			for _, s := range rec.Services {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", s.Service, s.ContainerName)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON envelope (format_version) for agents and CI")
	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build metadata and host-gateway rewrite behavior",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("podbay")
			if bi, ok := debug.ReadBuildInfo(); ok {
				fmt.Printf("module: %s\n", bi.Main.Path)
				fmt.Printf("go: %s\n", bi.GoVersion)
				for _, s := range bi.Settings {
					switch s.Key {
					case "vcs.revision":
						fmt.Printf("revision: %s\n", s.Value)
					case "vcs.time":
						fmt.Printf("vcs.time: %s\n", s.Value)
					case "vcs.modified":
						fmt.Printf("vcs.modified: %s\n", s.Value)
					}
				}
			}
			rewrite := "no (Linux: pass host-gateway through to Podman)"
			switch runtime.GOOS {
			case "darwin", "windows":
				rewrite = fmt.Sprintf("yes → %s (override with PODBAY_HOST_GATEWAY_IP)", runner.DefaultPodmanMachineHostIP)
			}
			fmt.Printf("this binary: GOOS/GOARCH=%s/%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Printf("host-gateway rewrite: %s\n", rewrite)
		},
	}
}

func resolvePath(fileFlag, defaultFile string) string {
	if fileFlag != "" {
		return fileFlag
	}
	return defaultFile
}

// contractPathOrArg resolves the contract path: -f/--file wins; otherwise first positional arg; else defaultFile.
func contractPathOrArg(fileFlag string, args []string, defaultFile string) (string, error) {
	if fileFlag != "" && len(args) > 0 {
		return "", fmt.Errorf("use either --file / -f or a path argument, not both")
	}
	if fileFlag != "" {
		return fileFlag, nil
	}
	if len(args) > 0 {
		return args[0], nil
	}
	return defaultFile, nil
}

// contractPathAndDeployServices resolves the contract file path and optional deploy/validate service roots.
// With --file / -f: every positional argument is a service name (zero means full stack).
// Without --file: [path] validates/deploys the full profile-active set; [path svc [svc...]] selects partial deploy roots.
func contractPathAndDeployServices(fileFlag string, args []string, defaultFile string) (path string, deployServices []string, err error) {
	if fileFlag != "" {
		return fileFlag, append([]string(nil), args...), nil
	}
	switch len(args) {
	case 0:
		return defaultFile, nil, nil
	case 1:
		arg := args[0]
		if contractLocationExists(arg) {
			return arg, nil, nil
		}
		c, _, err := spec.Load(defaultFile)
		if err == nil {
			if _, ok := c.Services[arg]; ok {
				return defaultFile, []string{arg}, nil
			}
		}
		return arg, nil, nil
	default:
		return args[0], append([]string(nil), args[1:]...), nil
	}
}

func loadContractWithDeployServices(fileFlag string, args []string, defaultFile string) (*spec.Contract, string, []string, error) {
	p, deployServices, err := contractPathAndDeployServices(fileFlag, args, defaultFile)
	if err != nil {
		return nil, "", nil, err
	}
	c, path, err := spec.Load(p)
	if err != nil {
		return nil, "", nil, augmentContractLoadError(p, err)
	}
	return c, path, deployServices, nil
}

// diffArgsDecodeAsReceiptPair reports whether args has two paths that both read and decode as deploy receipts.
func diffArgsDecodeAsReceiptPair(args []string) (pathA, pathB string, ok bool) {
	if len(args) != 2 {
		return "", "", false
	}
	data0, err0 := os.ReadFile(args[0])
	data1, err1 := os.ReadFile(args[1])
	if err0 != nil || err1 != nil {
		return "", "", false
	}
	if _, err := receipt.Decode(data0); err != nil {
		return "", "", false
	}
	if _, err := receipt.Decode(data1); err != nil {
		return "", "", false
	}
	return args[0], args[1], true
}

func loadContract(fileFlag string, args []string, defaultFile string) (*spec.Contract, string, error) {
	p, err := contractPathOrArg(fileFlag, args, defaultFile)
	if err != nil {
		return nil, "", err
	}
	c, path, err := spec.Load(p)
	if err != nil {
		return nil, "", augmentContractLoadError(p, err)
	}
	return c, path, nil
}

// expectedContractPath is the yaml file we would read for this user-supplied path (file, or dir + podbay.yaml).
func expectedContractPath(userPath string) string {
	abs, err := filepath.Abs(userPath)
	if err != nil {
		return userPath
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return abs
	}
	if fi.IsDir() {
		return filepath.Join(abs, spec.DefaultFilename)
	}
	return abs
}

// contractLocationExists reports whether userPath resolves to an existing contract file
// (a YAML file or a directory that contains podbay.yaml).
func contractLocationExists(userPath string) bool {
	p := expectedContractPath(userPath)
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func augmentContractLoadError(userPath string, loadErr error) error {
	if !errors.Is(loadErr, os.ErrNotExist) {
		return fmt.Errorf("load contract: %w", loadErr)
	}
	wd, _ := os.Getwd()
	tried := expectedContractPath(userPath)
	return fmt.Errorf("load contract: %w\n  cwd:           %s\n  expected file: %s\n  hint: -f is relative to cwd (above); cd to your app root or use e.g. -f myrepo/%s if the file lives in a subdirectory",
		loadErr, wd, tried, spec.DefaultFilename)
}

func projectName(c *spec.Contract, contractPath string) string {
	base := filepath.Base(filepath.Dir(contractPath))
	if base == "." || base == "" {
		wd, _ := os.Getwd()
		base = filepath.Base(wd)
	}
	return c.ProjectName(base)
}

func initCmd(fileFlag *string, defaultFile string) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a baseline podbay.yaml in the current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := filepath.Join(filepath.Dir(defaultFile), spec.DefaultFilename)
			if *fileFlag != "" {
				p := *fileFlag
				if st, err := os.Stat(p); err == nil && st.IsDir() {
					target = filepath.Join(p, spec.DefaultFilename)
				} else {
					target = p
				}
			}
			if _, err := os.Stat(target); err == nil {
				return fmt.Errorf("%s already exists", target)
			}
			if err := os.WriteFile(target, []byte(initTemplate), 0o644); err != nil {
				return err
			}
			fmt.Printf("Wrote %s\n", target)
			return nil
		},
	}
}

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

After a fully successful deploy, --receipt PATH writes a versioned JSON receipt (atomic write; no partial file if deploy fails).

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
				absReceipt := ""
				if strings.TrimSpace(receiptPath) != "" {
					if rp, err := filepath.Abs(receiptPath); err == nil {
						absReceipt = rp
					}
				}
				depQuiet := quiet || jsonOut
				depErr := deploy.Deploy(c, path, proj, deploy.Options{
					Profiles:         profiles,
					DeployServices:   deployServices,
					DeployDependents: dependents,
					SkipHealthWait:   skipHealth,
					HealthTimeout:    healthTimeout,
					Quiet:            depQuiet,
					Out:              cmd.OutOrStdout(),
					ReceiptPath:      receiptPath,
				})
				doc := clijson.DeployOutcome(path, proj, profiles, deployServices, absReceipt, depErr, dependents)
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
	cmd.Flags().StringVar(&receiptPath, "receipt", "", "after success, write deploy receipt JSON to this path (atomic; omitted on failure)")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "with partial targets, include transitive dependents within the profile-active set")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON (format_version) for agents and CI")
	return cmd
}

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

// logContractPathAndService: with --file, args must be exactly [service]. Otherwise [service] or [contract] [service].
func logContractPathAndService(fileFlag string, args []string, defaultFile string) (contractPath, service string, err error) {
	if fileFlag != "" {
		if len(args) != 1 {
			return "", "", fmt.Errorf("with --file / -f, pass exactly one argument: the service name")
		}
		return fileFlag, args[0], nil
	}
	switch len(args) {
	case 1:
		return defaultFile, args[0], nil
	case 2:
		return args[0], args[1], nil
	default:
		return "", "", fmt.Errorf("expected <service> or <contract-path> <service>")
	}
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
	cmd := &cobra.Command{
		Use:   "logs [podbay.yaml|directory] <service>",
		Short: "Show logs for a service container (podman logs)",
		Long: `Stream or print logs for the Podman container backing a single <service>.

With one argument: <service> (contract path defaults to ./podbay.yaml).
With two arguments: <contract-path-or-dir> then <service>.
With --file / -f on this command: pass only the service name as the sole argument (the contract file comes from the root -f/--file flag).

Use the same --profile flags as validate, deploy, ps, and teardown so <service> is evaluated against the same active profile slice (including after partial deploy).

This command always tails one container; multi-service aggregation is not implemented.

Note: use --follow on this command to stream logs (the root -f/--file flag selects the contract file, not log follow).`,
		Args:         cobra.RangeArgs(1, 2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if *fileFlag != "" && len(args) != 1 {
				return fmt.Errorf("with --file / -f, pass exactly one argument: the service name")
			}
			if *fileFlag == "" && len(args) != 1 && len(args) != 2 {
				return fmt.Errorf("expected <service> or <contract-path> <service>")
			}
			if err := runner.EnsurePodman(); err != nil {
				return err
			}
			path, svc, err := logContractPathAndService(*fileFlag, args, defaultFile)
			if err != nil {
				return err
			}
			c, contractPath, err := spec.Load(path)
			if err != nil {
				return augmentContractLoadError(path, err)
			}
			proj := projectName(c, contractPath)
			active := c.ServicesForProfiles(profiles)
			if _, ok := active[svc]; !ok {
				return fmt.Errorf("service %q is not active for this profile set (check --profile and spelling)", svc)
			}
			r := runner.New(proj)
			cname := r.ContainerName(svc)
			return runner.Logs(cmd.OutOrStdout(), cname, follow, tailN, since)
		},
	}
	cmd.Flags().StringSliceVar(&profiles, "profile", nil, "enable services with this Compose-style profile (repeatable)")
	cmd.Flags().BoolVar(&follow, "follow", false, "follow log output (podman logs -f)")
	cmd.Flags().IntVar(&tailN, "tail", 0, "only show the last N lines (0 means all lines)")
	cmd.Flags().StringVar(&since, "since", "", "only show logs since timestamp (podman logs --since), e.g. 10m or 2024-01-01")
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
			if pathA, pathB, ok := diffArgsDecodeAsReceiptPair(args); ok {
				return runReceiptPairDiff(cmd, pathA, pathB, jsonOut, profiles, receiptDiffShowEnv)
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
