package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
)

// New commands belong in group files (validate.go, deploy.go, etc.), not main.go.

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
