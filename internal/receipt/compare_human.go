package receipt

import (
	"fmt"
	"slices"
	"strings"
)

// FormatReceiptDiff renders a ReceiptDiffResult as multi-line text for stdout.
// Semantics match CompareReceipts: "first" is receipt A and "second" is receipt B.
func FormatReceiptDiff(res ReceiptDiffResult) string {
	var b strings.Builder
	b.WriteString("Receipt diff (first vs second)\n\n")

	b.WriteString("Project: ")
	if res.ProjectMatch {
		fmt.Fprintf(&b, "%s\n", res.ProjectA)
	} else {
		fmt.Fprintf(&b, "mismatch\n  first:  %s\n  second: %s\n", res.ProjectA, res.ProjectB)
	}

	b.WriteString("Contract: ")
	if res.ContractPathMatch {
		fmt.Fprintf(&b, "%s\n", res.ContractPathA)
	} else {
		fmt.Fprintf(&b, "mismatch\n  first:  %s\n  second: %s\n", res.ContractPathA, res.ContractPathB)
	}

	b.WriteString("Contract digest: ")
	switch {
	case !res.ContractDigestComparable:
		fmt.Fprintf(&b, "incomparable (recorded on one receipt only)\n  first:  %s\n  second: %s\n",
			digestOrNone(res.ContractDigestA), digestOrNone(res.ContractDigestB))
	case res.ContractDigestMatch:
		fmt.Fprintf(&b, "%s\n", digestOrNone(res.ContractDigestA))
	default:
		fmt.Fprintf(&b, "mismatch\n  first:  %s\n  second: %s\n", res.ContractDigestA, res.ContractDigestB)
	}

	b.WriteString("Profiles: ")
	if res.ProfilesMatch {
		fmt.Fprintf(&b, "%s\n", formatProfileLine(res.ProfilesA))
	} else {
		fmt.Fprintf(&b, "mismatch\n  first:  %s\n  second: %s\n",
			formatProfileLine(res.ProfilesA), formatProfileLine(res.ProfilesB))
	}

	b.WriteString("\nServices:\n")
	if len(res.Services) == 0 {
		b.WriteString("  (no service-level differences)\n")
	} else {
		for _, s := range res.Services {
			appendServiceHumanLines(&b, s)
		}
	}

	b.WriteString("\n")
	if !res.Drift {
		b.WriteString("No drift: compared fields match between the two receipts.\n")
	} else {
		b.WriteString("Drift detected.\n")
	}
	return b.String()
}

func digestOrNone(d string) string {
	if d == "" {
		return "(none)"
	}
	return d
}

func formatProfileLine(p []string) string {
	if len(p) == 0 {
		return "(none)"
	}
	c := append([]string(nil), p...)
	slices.Sort(c)
	return strings.Join(c, ", ")
}

func appendServiceHumanLines(b *strings.Builder, s ServiceReceiptDiff) {
	for _, code := range s.Codes {
		switch code {
		case CodeServiceAdded:
			fmt.Fprintf(b, "  [%s] added in second receipt (container=%q image=%q)\n",
				s.Service, s.RecordB.ContainerName, s.RecordB.Image)
		case CodeServiceRemoved:
			fmt.Fprintf(b, "  [%s] removed in second receipt (was container=%q image=%q)\n",
				s.Service, s.RecordA.ContainerName, s.RecordA.Image)
		case CodeImageChanged:
			fmt.Fprintf(b, "  [%s] image changed\n    first:  %q\n    second: %q\n",
				s.Service, s.RecordA.Image, s.RecordB.Image)
		case CodeContainerNameChanged:
			fmt.Fprintf(b, "  [%s] container_name changed\n    first:  %q\n    second: %q\n",
				s.Service, s.RecordA.ContainerName, s.RecordB.ContainerName)
		case CodeContainerIDChanged:
			fmt.Fprintf(b, "  [%s] container_id changed\n    first:  %q\n    second: %q\n",
				s.Service, s.RecordA.ContainerID, s.RecordB.ContainerID)
		case CodeEnvChanged:
			keys := EnvDiffChangedKeyList(s.RecordA, s.RecordB)
			if len(keys) == 0 {
				fmt.Fprintf(b, "  [%s] env changed (see JSON; values redacted by default)\n", s.Service)
			} else {
				fmt.Fprintf(b, "  [%s] env changed (keys: %s; values redacted — use --receipt-diff-show-env with --json for raw)\n",
					s.Service, strings.Join(keys, ", "))
			}
		case CodeMountsChanged:
			fmt.Fprintf(b, "  [%s] mounts changed\n    first:  %s\n    second:  %s\n",
				s.Service, formatMountsOneLine(s.RecordA.Mounts), formatMountsOneLine(s.RecordB.Mounts))
		case CodeEnvIncomparable:
			fmt.Fprintf(b, "  [%s] env: incomparable (recorded on one receipt only)\n", s.Service)
		case CodeMountsIncomparable:
			fmt.Fprintf(b, "  [%s] mounts: incomparable (recorded on one receipt only)\n", s.Service)
		default:
			fmt.Fprintf(b, "  [%s] %s\n", s.Service, code)
		}
	}
}

func formatMountsOneLine(m *[]MountSpec) string {
	if m == nil {
		return "(not recorded)"
	}
	if len(*m) == 0 {
		return "(none)"
	}
	var b strings.Builder
	for i, x := range *m {
		if i > 0 {
			b.WriteString("; ")
		}
		fmt.Fprintf(&b, "%s→%s", x.Source, x.Destination)
	}
	return b.String()
}
