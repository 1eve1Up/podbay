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
		Use:   "receipt [path-to-receipt.json]",
		Args:  cobra.MaximumNArgs(1),
		Short: "Read, summarize, or list deploy receipt JSON",
		Long: `Load a receipt produced by podbay deploy --receipt, or list a receipt store directory.

Without --json: print a short human summary (project, contract path, evidence fields, services).

With --json: print a versioned envelope (kind receipt_read, format_version) containing the validated receipt.

Use "receipt list <dir>" to inventory receipts in a directory (newest first).`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("receipt: path required (or use: podbay receipt list <dir>)")
			}
			return runReceiptRead(cmd, args[0], jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON envelope (format_version) for agents and CI")
	cmd.AddCommand(receiptListCmd())
	return cmd
}

func receiptListCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list <dir>",
		Args:  cobra.ExactArgs(1),
		Short: "List deploy receipts in a directory (newest first)",
		Long: `Inventory *.json files under a directory that decode as deploy receipts.

Newest generated_at first. Non-receipt JSON files are skipped.

With --json: kind receipt_list with receipts[] rows (path, deploy_id, generated_at, project, status, service_count).`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			abs, absErr := filepath.Abs(dir)
			if absErr != nil {
				abs = dir
			}
			entries, skipped, err := receipt.ListDir(dir)
			if err != nil {
				if jsonOut {
					doc := clijson.ReceiptListFailure(abs, err)
					raw, mErr := clijson.MarshalIndent(doc)
					if mErr != nil {
						return mErr
					}
					fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
					os.Exit(1)
				}
				return err
			}
			if jsonOut {
				doc := clijson.ReceiptListSuccess(abs, entries, skipped)
				raw, mErr := clijson.MarshalIndent(doc)
				if mErr != nil {
					return mErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Receipt list: %s (%d)\n", abs, len(entries))
			for _, e := range entries {
				status := e.Status
				if status == "" {
					status = "(none)"
				}
				id := e.DeployID
				if id == "" {
					id = "(none)"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %s  project=%s  status=%s  deploy_id=%s  services=%d\n",
					e.GeneratedAt, e.Project, status, id, e.ServiceCount)
				fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", e.Path)
			}
			if len(skipped) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Skipped %d non-receipt file(s)\n", len(skipped))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON (kind receipt_list)")
	return cmd
}

func runReceiptRead(cmd *cobra.Command, p string, jsonOut bool) error {
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
	if rec.DeployID != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Deploy ID:   %s\n", rec.DeployID)
	}
	if rec.ContractDigest != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Digest:      %s\n", rec.ContractDigest)
	}
	if rec.Status != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Status:      %s\n", rec.Status)
	}
	if len(rec.DeployServices) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Deploy svcs: %s\n", strings.Join(rec.DeployServices, ", "))
		if rec.DependentsExpand {
			fmt.Fprintf(cmd.OutOrStdout(), "Dependents:  expanded\n")
		}
	}
	if len(rec.Profiles) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Profiles:    %s\n", strings.Join(rec.Profiles, ", "))
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Services:    %d\n", len(rec.Services))
	for _, s := range rec.Services {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", s.Service, s.ContainerName)
	}
	return nil
}
