package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

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
	return &cobra.Command{
		Use:   "init",
		Short: "Create a baseline podbay.yaml in the current directory",
		Long: `Write a baseline podbay.yaml, then print orientation next steps
(onboard / validate). Does not require Podman. Refuses to overwrite an existing file.`,
		SilenceUsage: true,
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
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Wrote %s\n", target)
			fmt.Fprintln(out, "Next steps:")
			fmt.Fprintf(out, "  podbay onboard -f %s --json\n", target)
			fmt.Fprintf(out, "  podbay validate -f %s --json\n", target)
			return nil
		},
	}
}
