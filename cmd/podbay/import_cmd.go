package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/composefile"
	"github.com/1eve1Up/podbay/internal/composeimport"
)

func importCmd() *cobra.Command {
	var outPath string
	jsonOut := false
	compose := &cobra.Command{
		Use:   "compose <file>",
		Short: "Convert a Docker Compose file to podbay.yaml on stdout",
		Long: `Read a Compose file (e.g. docker-compose.yml, compose.yaml) and emit a Podbay contract.

Unsupported Compose features for import v1 return a clear error (for example top-level networks,
long-form port mappings, or build without an image tag).

With --json: on failure print one versioned JSON document (format_version, kind import_compose) to stdout
and exit 1; success behavior is unchanged (YAML still goes to stdout or -o).

Examples:
  podbay import compose docker-compose.yml
  podbay import compose compose.yaml -o podbay.yaml`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			inPath := args[0]
			absIn, err := filepath.Abs(inPath)
			if err != nil {
				absIn = inPath
			}
			emitImportJSONFail := func(err error) {
				doc := clijson.FromImportComposeError(absIn, err)
				raw, mErr := clijson.MarshalIndent(doc)
				if mErr != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", mErr.Error())
					os.Exit(1)
				}
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
				os.Exit(1)
			}

			f, err := composefile.Load(absIn)
			if err != nil {
				if jsonOut {
					emitImportJSONFail(err)
				}
				return fmt.Errorf("import compose: read %s: %w", absIn, err)
			}
			composeDir := filepath.Dir(absIn)
			c, err := composeimport.ToContract(f, composeDir)
			if err != nil {
				if jsonOut {
					emitImportJSONFail(err)
				}
				return fmt.Errorf("import compose: %w", err)
			}
			raw, err := composeimport.MarshalContract(c)
			if err != nil {
				if jsonOut {
					emitImportJSONFail(err)
				}
				return fmt.Errorf("import compose: encode: %w", err)
			}
			if outPath == "" {
				_, err = cmd.OutOrStdout().Write(raw)
				return err
			}
			outAbs, err := filepath.Abs(outPath)
			if err != nil {
				if jsonOut {
					emitImportJSONFail(err)
				}
				return fmt.Errorf("import compose: output path: %w", err)
			}
			if err := os.WriteFile(outAbs, raw, 0o644); err != nil {
				if jsonOut {
					emitImportJSONFail(err)
				}
				return fmt.Errorf("import compose: write %s: %w", outAbs, err)
			}
			return nil
		},
	}
	compose.Flags().StringVarP(&outPath, "output", "o", "", "write podbay contract to this file instead of stdout")
	compose.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON (format_version) on failure for agents and CI")

	root := &cobra.Command{
		Use:   "import",
		Short: "Import external definitions into a Podbay contract",
	}
	root.AddCommand(compose)
	return root
}
