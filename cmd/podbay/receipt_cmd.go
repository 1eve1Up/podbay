package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/1eve1Up/podbay/internal/clijson"
	"github.com/1eve1Up/podbay/internal/receipt"
)

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
			abs, absErr := filepath.Abs(p)
			if absErr != nil {
				abs = p
			}
			data, err := os.ReadFile(p)
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
				return fmt.Errorf("receipt: read %s: %w", p, err)
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
