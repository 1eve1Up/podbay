package main

import (
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
	jsonOut := false
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a baseline podbay.yaml or import one from Compose in a codebase",
		Long: `Write a baseline podbay.yaml (greenfield), or with --from-codebase discover a
Compose file and write a first-pass contract via the same import pipeline as
podbay import compose. Then print orientation next steps (onboard / validate).
Does not require Podman. Refuses to overwrite an existing file.

Compose discovery order: compose.yaml, compose.yml, docker-compose.yaml,
docker-compose.yml. Pass --compose to override discovery.

With --json: one init document (format_version 1, kind init) on stdout for
success or failure; exit 0 or 1. Human next-step lines are omitted in JSON mode.`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := initTargetPath(*fileFlag, defaultFile)
			emitFail := func(composeSrc string, err error) error {
				if !jsonOut {
					return err
				}
				doc := clijson.FromInitError(target, composeSrc, err)
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
				return emitFail("", &clijson.InitTargetExistsError{Path: target})
			}

			if fromCodebase {
				dir := "."
				if len(args) > 0 {
					dir = args[0]
				}
				return initFromCodebase(cmd, dir, composePath, target, jsonOut, emitFail)
			}
			if composePath != "" {
				return emitFail("", fmt.Errorf("--compose requires --from-codebase"))
			}
			if len(args) > 0 {
				return emitFail("", fmt.Errorf("unexpected argument %q (did you mean --from-codebase?)", args[0]))
			}
			return initGreenfield(cmd, target, jsonOut)
		},
	}
	cmd.Flags().BoolVar(&fromCodebase, "from-codebase", false, "discover Compose under [dir] (default .) and write a first-pass podbay.yaml")
	cmd.Flags().StringVar(&composePath, "compose", "", "explicit Compose file path (implies discovery override; requires --from-codebase)")
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
	dir, composePath, target string,
	jsonOut bool,
	emitFail func(composeSrc string, err error) error,
) error {
	absCompose, err := composefile.Discover(dir, composePath)
	if err != nil {
		return emitFail("", fmt.Errorf("init --from-codebase: %w", err))
	}
	f, err := composefile.Load(absCompose)
	if err != nil {
		return emitFail(absCompose, fmt.Errorf("init --from-codebase: read %s: %w", absCompose, err))
	}
	c, err := composeimport.ToContract(f, filepath.Dir(absCompose))
	if err != nil {
		return emitFail(absCompose, fmt.Errorf("init --from-codebase: %w", err))
	}
	raw, err := composeimport.MarshalContract(c)
	if err != nil {
		return emitFail(absCompose, fmt.Errorf("init --from-codebase: encode: %w", err))
	}
	if err := os.WriteFile(target, raw, 0o644); err != nil {
		return emitFail(absCompose, err)
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

func printInitNextSteps(cmd *cobra.Command, target string) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Wrote %s\n", target)
	printInitOrientHints(out, target)
}

func printInitOrientHints(out io.Writer, target string) {
	fmt.Fprintln(out, "Next steps:")
	for _, a := range clijson.InitOrientNextActions(target) {
		fmt.Fprintf(out, "  %s\n", a)
	}
}
