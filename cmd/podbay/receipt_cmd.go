package main

import (
	"errors"
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

Use "receipt list <dir>" to inventory receipts in a directory (newest first).
Use "receipt last-ok <dir>" to resolve the newest successful receipt in a store.
Use "receipt handoff <current.json> --store <dir>" for a structured agent handoff summary.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("receipt: path required (or use: podbay receipt list|last-ok|handoff)")
			}
			return runReceiptRead(cmd, args[0], jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON envelope (format_version) for agents and CI")
	cmd.AddCommand(receiptListCmd())
	cmd.AddCommand(receiptLastOKCmd())
	cmd.AddCommand(receiptHandoffCmd())
	return cmd
}

func receiptListCmd() *cobra.Command {
	var jsonOut bool
	var statusFilter string
	cmd := &cobra.Command{
		Use:   "list <dir>",
		Args:  cobra.ExactArgs(1),
		Short: "List deploy receipts in a directory (newest first)",
		Long: `Inventory *.json files under a directory that decode as deploy receipts.

Newest generated_at first. Non-receipt JSON files are skipped.

--status ok|failed filters the inventory (ok also matches legacy receipts with empty status).

With --json: kind receipt_list with receipts[] rows (path, deploy_id, generated_at, project, status, service_count).`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			abs, absErr := filepath.Abs(dir)
			if absErr != nil {
				abs = dir
			}
			statusFilter = strings.TrimSpace(statusFilter)
			if statusFilter != "" && statusFilter != receipt.StatusOK && statusFilter != receipt.StatusFailed {
				return fmt.Errorf("receipt list: unsupported --status %q (want ok, failed, or empty)", statusFilter)
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
			entries = receipt.FilterEntries(entries, statusFilter)
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
	cmd.Flags().StringVar(&statusFilter, "status", "", "filter by status: ok or failed")
	return cmd
}

func receiptLastOKCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "last-ok <dir>",
		Args:  cobra.ExactArgs(1),
		Short: "Resolve the newest successful receipt in a directory",
		Long: `Resolve the newest deploy receipt with status ok (legacy empty status counts as ok).

Without --json: print the absolute receipt path.

With --json: kind receipt_last_ok. When found, status ok and receipt_path set.
When no prior ok exists, status failed with issue code receipt_no_last_ok (no fake path).`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			abs, absErr := filepath.Abs(dir)
			if absErr != nil {
				abs = dir
			}
			entry, err := receipt.LastOK(dir)
			if err != nil {
				code := "receipt_last_ok_error"
				if errors.Is(err, receipt.ErrNoLastOK) {
					code = "receipt_no_last_ok"
				}
				if jsonOut {
					doc := clijson.ReceiptLastOKFailure(abs, code, err)
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
				doc := clijson.ReceiptLastOKSuccess(abs, *entry)
				raw, mErr := clijson.MarshalIndent(doc)
				if mErr != nil {
					return mErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), entry.Path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON (kind receipt_last_ok)")
	return cmd
}

func receiptHandoffCmd() *cobra.Command {
	var jsonOut bool
	var storeDir string
	cmd := &cobra.Command{
		Use:   "handoff <current-receipt.json>",
		Args:  cobra.ExactArgs(1),
		Short: "Emit a structured agent handoff summary for a receipt",
		Long: `Build a handoff summary from a current receipt and optional receipt store directory.

With --store <dir>: resolve last ok, compare via pair-diff, and include drift / no_prior_ok.

With --json: kind receipt_handoff containing handoff{} (identity, failure, last_ok_path or no_prior_ok, next_actions).
Human output prints a short summary; --json is the agent contract.

Handoff is structured next-steps only — not automatic remediation.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			currentPath := args[0]
			abs, absErr := filepath.Abs(currentPath)
			if absErr != nil {
				abs = currentPath
			}
			data, err := os.ReadFile(currentPath)
			if err != nil {
				if jsonOut {
					doc := clijson.ReceiptHandoffFailure(abs, err)
					raw, mErr := clijson.MarshalIndent(doc)
					if mErr != nil {
						return mErr
					}
					fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
					os.Exit(1)
				}
				return fmt.Errorf("receipt handoff: read %s: %w", currentPath, err)
			}
			rec, err := receipt.Decode(data)
			if err != nil {
				if jsonOut {
					doc := clijson.ReceiptHandoffFailure(abs, err)
					raw, mErr := clijson.MarshalIndent(doc)
					if mErr != nil {
						return mErr
					}
					fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
					os.Exit(1)
				}
				return fmt.Errorf("receipt handoff: %w", err)
			}
			h, err := receipt.BuildHandoff(rec, abs, strings.TrimSpace(storeDir))
			if err != nil {
				if jsonOut {
					doc := clijson.ReceiptHandoffFailure(abs, err)
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
				doc := clijson.ReceiptHandoffSuccess(h)
				raw, mErr := clijson.MarshalIndent(doc)
				if mErr != nil {
					return mErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(raw)))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Handoff status=%s deploy_id=%s\n", h.Status, h.DeployID)
			if h.Failure != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Failure: %s (%s)\n", h.Failure.Code, h.Failure.Service)
			}
			if h.NoPriorOK {
				fmt.Fprintln(cmd.OutOrStdout(), "Last ok: (none)")
			} else if h.LastOKPath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Last ok: %s\n", h.LastOKPath)
				if h.Drift != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Drift:   %v\n", *h.Drift)
				}
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Next:")
			for _, a := range h.NextActions {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", a)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Note: %s\n", h.RemediationNote)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit versioned JSON (kind receipt_handoff)")
	cmd.Flags().StringVar(&storeDir, "store", "", "receipt store directory for last-ok resolve and pair-diff")
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
	if rec.Failure != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Failure:\n")
		if rec.Failure.Code != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Code:      %s\n", rec.Failure.Code)
		}
		if rec.Failure.Service != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Service:   %s\n", rec.Failure.Service)
		}
		if rec.Failure.Class != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Class:     %s\n", rec.Failure.Class)
		}
		if rec.Failure.ProbeKind != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Probe:     %s\n", rec.Failure.ProbeKind)
		}
		if rec.Failure.Message != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Message:   %s\n", rec.Failure.Message)
		}
		if rec.Failure.ExternalDep {
			fmt.Fprintf(cmd.OutOrStdout(), "  External:  true\n")
			if rec.Failure.RequestedBy != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Requested: %s\n", rec.Failure.RequestedBy)
			}
		}
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
