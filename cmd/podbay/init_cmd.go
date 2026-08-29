package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/composefile"
	"github.com/1eve1Up/podbay/internal/composeimport"
	"github.com/1eve1Up/podbay/internal/spec"
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

func initCmd(fileFlag *string, defaultFile string) *cobra.Command {
	var fromCodebase bool
	var composePath string
	var dockerfilePath string
	jsonOut := false
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a baseline podbay.yaml or import one from a codebase",
		Long: `Write a baseline podbay.yaml (greenfield), or with --from-codebase discover a
Compose file (preferred) or Dockerfile and write a first-pass contract. Compose
uses the same import pipeline as podbay import compose; Dockerfile yields a
single-service build stub. Then print orientation next steps (onboard / validate).
Does not require Podman. Refuses to overwrite an existing file.

Compose discovery order: compose.yaml, compose.yml, docker-compose.yaml,
docker-compose.yml. Pass --compose to override discovery.

Dockerfile discovery order (Compose miss only): Dockerfile, dockerfile.
Pass --dockerfile to force the Dockerfile path (skips Compose discovery).
--compose and --dockerfile together are rejected.

With --json: one init document (format_version 1, kind init) on stdout for
success or failure; exit 0 or 1. Human next-step lines are omitted in JSON mode.`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := initTargetPath(*fileFlag, defaultFile)
			emitFail := func(composeSrc, dockerfileSrc string, err error) error {
				if !jsonOut {
					return err
				}
				doc := clijson.FromInitError(target, composeSrc, dockerfileSrc, err)
				raw, mErr := clijson.MarshalIndent(doc)
				if mErr != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", mErr.Error())
					os.Exit(1)
				}
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
				os.Exit(1)
				return nil
			}

			if _, err := os.Stat(target); err == nil {
				return emitFail("", "", &clijson.InitTargetExistsError{Path: target})
			}

			if fromCodebase {
				dir := "."
				if len(args) > 0 {
					dir = args[0]
				}
				return initFromCodebase(cmd, dir, composePath, dockerfilePath, target, jsonOut, emitFail)
			}
			if composePath != "" {
				return emitFail("", "", fmt.Errorf("--compose requires --from-codebase"))
			}
			if dockerfilePath != "" {
				return emitFail("", "", fmt.Errorf("--dockerfile requires --from-codebase"))
			}
			if len(args) > 0 {
				return emitFail("", "", fmt.Errorf("unexpected argument %q (did you mean --from-codebase?)", args[0]))
			}
			return initGreenfield(cmd, target, jsonOut)
		},
	}
	cmd.Flags().BoolVar(&fromCodebase, "from-codebase", false, "discover Compose or Dockerfile under [dir] (default .) and write a first-pass podbay.yaml")
	cmd.Flags().StringVar(&composePath, "compose", "", "explicit Compose file path (requires --from-codebase; mutually exclusive with --dockerfile)")
	cmd.Flags().StringVar(&dockerfilePath, "dockerfile", "", "explicit Dockerfile path; forces Dockerfile stub path (requires --from-codebase; mutually exclusive with --compose)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "stdout is one init JSON document: success (status ok) or failure (issues[]); exit 0 or 1")
	return cmd
}

func initTargetPath(fileFlag, defaultFile string) string {
	target := filepath.Join(filepath.Dir(defaultFile), spec.DefaultFilename)
	if fileFlag != "" {
		p := fileFlag
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			target = filepath.Join(p, spec.DefaultFilename)
		} else {
			target = p
		}
	}
	return target
}

func initGreenfield(cmd *cobra.Command, target string, jsonOut bool) error {
	if err := os.WriteFile(target, []byte(initTemplate), 0o644); err != nil {
		return err
	}
	if jsonOut {
		doc := clijson.FromInitGreenfieldSuccess(target)
		raw, err := clijson.MarshalIndent(doc)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
		return nil
	}
	printInitNextSteps(cmd, target)
	return nil
}

func initFromCodebase(
	cmd *cobra.Command,
	dir, composePath, dockerfilePath, target string,
	jsonOut bool,
	emitFail func(composeSrc, dockerfileSrc string, err error) error,
) error {
	if strings.TrimSpace(composePath) != "" && strings.TrimSpace(dockerfilePath) != "" {
		return emitFail("", "", fmt.Errorf("--compose and --dockerfile are mutually exclusive"))
	}

	forceDockerfile := strings.TrimSpace(dockerfilePath) != ""
	if !forceDockerfile {
		absCompose, err := composefile.Discover(dir, composePath)
		if err == nil {
			return initFromCompose(cmd, absCompose, target, jsonOut, emitFail)
		}
		if strings.TrimSpace(composePath) != "" || !isComposeDiscoveryNotFound(err) {
			return emitFail("", "", fmt.Errorf("init --from-codebase: %w", err))
		}
		// Compose miss → Dockerfile fallback.
	}

	absDockerfile, err := composefile.DiscoverDockerfile(dir, dockerfilePath)
	if err != nil {
		if !forceDockerfile && isDockerfileDiscoveryNotFound(err) {
			return emitFail("", "", composefile.NewImportFailure(
				composefile.CodeCodebaseDiscoveryNotFound,
				fmt.Sprintf("init --from-codebase: no Compose file or Dockerfile found in %s", dirOrDot(dir)),
			))
		}
		return emitFail("", "", fmt.Errorf("init --from-codebase: %w", err))
	}
	return initFromDockerfile(cmd, dir, absDockerfile, target, jsonOut, emitFail)
}

func initFromCompose(
	cmd *cobra.Command,
	absCompose, target string,
	jsonOut bool,
	emitFail func(composeSrc, dockerfileSrc string, err error) error,
) error {
	f, err := composefile.Load(absCompose)
	if err != nil {
		return emitFail(absCompose, "", fmt.Errorf("init --from-codebase: read %s: %w", absCompose, err))
	}
	c, err := composeimport.ToContract(f, filepath.Dir(absCompose))
	if err != nil {
		return emitFail(absCompose, "", fmt.Errorf("init --from-codebase: %w", err))
	}
	raw, err := composeimport.MarshalContract(c)
	if err != nil {
		return emitFail(absCompose, "", fmt.Errorf("init --from-codebase: encode: %w", err))
	}
	if err := os.WriteFile(target, raw, 0o644); err != nil {
		return emitFail(absCompose, "", err)
	}
	if jsonOut {
		doc := clijson.FromInitFromCodebaseSuccess(target, absCompose, c)
		rawJSON, mErr := clijson.MarshalIndent(doc)
		if mErr != nil {
			return mErr
		}
		fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(rawJSON)))
		return nil
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Wrote %s (from %s)\n", target, absCompose)
	printInitOrientHints(out, target)
	return nil
}

func initFromDockerfile(
	cmd *cobra.Command,
	dir, absDockerfile, target string,
	jsonOut bool,
	emitFail func(composeSrc, dockerfileSrc string, err error) error,
) error {
	absDir, err := filepath.Abs(dirOrDot(dir))
	if err != nil {
		return emitFail("", absDockerfile, fmt.Errorf("init --from-codebase: %w", err))
	}
	project := filepath.Base(absDir)
	dfRel := composeimport.DockerfileRelForStub(absDir, absDockerfile)
	scan, scanErr := composeimport.ScanDockerfile(absDockerfile)
	if scanErr != nil {
		scan = composeimport.DockerfileScan{}
	}
	c := composeimport.StubFromDockerfileScan(project, dfRel, scan)
	raw, err := composeimport.MarshalContract(c)
	if err != nil {
		return emitFail("", absDockerfile, fmt.Errorf("init --from-codebase: encode: %w", err))
	}
	if err := os.WriteFile(target, raw, 0o644); err != nil {
		return emitFail("", absDockerfile, err)
	}
	if jsonOut {
		doc := clijson.FromInitDockerfileSuccess(target, absDockerfile, c)
		rawJSON, mErr := clijson.MarshalIndent(doc)
		if mErr != nil {
			return mErr
		}
		fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(rawJSON)))
		return nil
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Wrote %s (from %s)\n", target, absDockerfile)
	printInitNextActionList(out, clijson.InitDockerfileNextActions(target, clijson.DockerfileGaps(c)))
	return nil
}

func isComposeDiscoveryNotFound(err error) bool {
	var inf *composefile.ImportFailure
	return errors.As(err, &inf) && inf != nil && inf.Code == composefile.CodeComposeDiscoveryNotFound
}

func isDockerfileDiscoveryNotFound(err error) bool {
	var inf *composefile.ImportFailure
	return errors.As(err, &inf) && inf != nil && inf.Code == composefile.CodeDockerfileDiscoveryNotFound
}

func dirOrDot(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return "."
	}
	return dir
}

func printInitNextSteps(cmd *cobra.Command, target string) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Wrote %s\n", target)
	printInitOrientHints(out, target)
}

func printInitOrientHints(out io.Writer, target string) {
	printInitNextActionList(out, clijson.InitOrientNextActions(target))
}

func printInitNextActionList(out io.Writer, actions []string) {
	fmt.Fprintln(out, "Next steps:")
	for _, a := range actions {
		fmt.Fprintf(out, "  %s\n", a)
	}
}
