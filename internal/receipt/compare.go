// Receipt-to-receipt comparison (v1 stack identity + optional v2 env/mount snapshots).
//
// CompareReceipts compares two receipts that have already passed Validate or Decode.
// Compared fields: project, contract_path, profiles (order-independent), and per
// service name: presence in each receipt, image, container_name, container_id, and
// optional env / mounts when both receipts record those slices for the service.
// Not compared: generated_at (intentionally ignored for “what changed in deploy
// identity”), and format_version (callers must already enforce v1 via Validate).
//
// If the same service name appears more than once in one receipt, the last entry
// in that receipt’s services slice wins for comparison (Validate does not forbid duplicates).
package receipt

import (
	"cmp"
	"slices"
)

// Stable semantic codes for receipt diffs (JSON / human renderers may key off these strings).
const (
	CodeProjectMismatch      = "receipt_diff_project_mismatch"
	CodeContractPathMismatch = "receipt_diff_contract_path_mismatch"
	CodeProfilesMismatch     = "receipt_diff_profiles_mismatch"
	CodeServiceAdded         = "receipt_diff_service_added"
	CodeServiceRemoved       = "receipt_diff_service_removed"
	CodeImageChanged         = "receipt_diff_image_changed"
	CodeContainerNameChanged = "receipt_diff_container_name_changed"
	CodeContainerIDChanged   = "receipt_diff_container_id_changed"
	CodeEnvChanged           = "receipt_diff_env_changed"
	CodeMountsChanged        = "receipt_diff_mounts_changed"
	CodeEnvIncomparable      = "receipt_diff_env_incomparable"
	CodeMountsIncomparable   = "receipt_diff_mounts_incomparable"
)

// ReceiptDiffResult is the structured outcome of comparing two validated v1 receipts.
// Services lists only rows with at least one delta code; identical services are omitted.
type ReceiptDiffResult struct {
	ProjectA      string
	ProjectB      string
	ContractPathA string
	ContractPathB string
	ProfilesA     []string
	ProfilesB     []string

	ProjectMatch      bool
	ContractPathMatch bool
	ProfilesMatch     bool

	Services []ServiceReceiptDiff
	Drift    bool
}

// ServiceReceiptDiff records per-service changes between two receipts.
type ServiceReceiptDiff struct {
	Service string
	Codes   []string
	RecordA ServiceRecord
	RecordB ServiceRecord
}

// CompareReceipts compares a and b. Both must be non-nil and already validated.
func CompareReceipts(a, b *Receipt) ReceiptDiffResult {
	if a == nil || b == nil {
		panic("receipt: CompareReceipts: nil receipt")
	}

	res := ReceiptDiffResult{
		ProjectA:      a.Project,
		ProjectB:      b.Project,
		ContractPathA: a.ContractPath,
		ContractPathB: b.ContractPath,
		ProfilesA:     append([]string(nil), a.Profiles...),
		ProfilesB:     append([]string(nil), b.Profiles...),
	}

	res.ProjectMatch = a.Project == b.Project
	if !res.ProjectMatch {
		res.Drift = true
	}
	res.ContractPathMatch = a.ContractPath == b.ContractPath
	if !res.ContractPathMatch {
		res.Drift = true
	}
	res.ProfilesMatch = profilesEqual(a.Profiles, b.Profiles)
	if !res.ProfilesMatch {
		res.Drift = true
	}

	ma := serviceMap(a.Services)
	mb := serviceMap(b.Services)
	for _, name := range unionSorted(ma, mb) {
		sa, aOK := ma[name]
		sb, bOK := mb[name]
		var sd ServiceReceiptDiff
		sd.Service = name
		sd.RecordA = sa
		sd.RecordB = sb

		switch {
		case !aOK && bOK:
			sd.Codes = append(sd.Codes, CodeServiceAdded)
			res.Drift = true
		case aOK && !bOK:
			sd.Codes = append(sd.Codes, CodeServiceRemoved)
			res.Drift = true
		default:
			if sa.Image != sb.Image {
				sd.Codes = append(sd.Codes, CodeImageChanged)
				res.Drift = true
			}
			if sa.ContainerName != sb.ContainerName {
				sd.Codes = append(sd.Codes, CodeContainerNameChanged)
				res.Drift = true
			}
			if sa.ContainerID != sb.ContainerID {
				sd.Codes = append(sd.Codes, CodeContainerIDChanged)
				res.Drift = true
			}
			appendEnvMountDiff(&sd, sa, sb, &res.Drift)
		}
		if len(sd.Codes) > 0 {
			res.Services = append(res.Services, sd)
		}
	}
	return res
}

func profilesEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	ca := append([]string(nil), a...)
	cb := append([]string(nil), b...)
	slices.Sort(ca)
	slices.Sort(cb)
	return slices.Equal(ca, cb)
}

func serviceMap(recs []ServiceRecord) map[string]ServiceRecord {
	m := make(map[string]ServiceRecord, len(recs))
	for _, s := range recs {
		m[s.Service] = s
	}
	return m
}

func unionSorted(a, b map[string]ServiceRecord) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func appendEnvMountDiff(sd *ServiceReceiptDiff, sa, sb ServiceRecord, drift *bool) {
	switch {
	case sa.Env == nil && sb.Env == nil:
	case sa.Env == nil || sb.Env == nil:
		sd.Codes = append(sd.Codes, CodeEnvIncomparable)
	default:
		if !envVarsEqual(*sa.Env, *sb.Env) {
			sd.Codes = append(sd.Codes, CodeEnvChanged)
			*drift = true
		}
	}
	switch {
	case sa.Mounts == nil && sb.Mounts == nil:
	case sa.Mounts == nil || sb.Mounts == nil:
		sd.Codes = append(sd.Codes, CodeMountsIncomparable)
	default:
		if !mountSpecsEqual(*sa.Mounts, *sb.Mounts) {
			sd.Codes = append(sd.Codes, CodeMountsChanged)
			*drift = true
		}
	}
}

func envVarsEqual(a, b []EnvVar) bool {
	na := normalizeEnv(a)
	nb := normalizeEnv(b)
	return slices.EqualFunc(na, nb, func(x, y EnvVar) bool { return x.Name == y.Name && x.Value == y.Value })
}

func normalizeEnv(e []EnvVar) []EnvVar {
	out := append([]EnvVar(nil), e...)
	slices.SortFunc(out, func(a, b EnvVar) int {
		if c := cmp.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		return cmp.Compare(a.Value, b.Value)
	})
	return out
}

func mountSpecsEqual(a, b []MountSpec) bool {
	na := normalizeMounts(a)
	nb := normalizeMounts(b)
	return slices.EqualFunc(na, nb, func(x, y MountSpec) bool {
		return x.Type == y.Type && x.Source == y.Source && x.Destination == y.Destination
	})
}

// EnvDiffChangedKeyList returns sorted env names where values differ between a and b.
// Both sides must have non-nil Env; otherwise it returns nil.
func EnvDiffChangedKeyList(sa, sb ServiceRecord) []string {
	if sa.Env == nil || sb.Env == nil {
		return nil
	}
	ma := make(map[string]string)
	for _, e := range *sa.Env {
		ma[e.Name] = e.Value
	}
	mb := make(map[string]string)
	for _, e := range *sb.Env {
		mb[e.Name] = e.Value
	}
	names := make(map[string]struct{})
	for k := range ma {
		names[k] = struct{}{}
	}
	for k := range mb {
		names[k] = struct{}{}
	}
	var out []string
	for n := range names {
		if ma[n] != mb[n] {
			out = append(out, n)
		}
	}
	slices.Sort(out)
	return out
}

func normalizeMounts(m []MountSpec) []MountSpec {
	out := append([]MountSpec(nil), m...)
	slices.SortFunc(out, func(a, b MountSpec) int {
		if c := cmp.Compare(a.Destination, b.Destination); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Source, b.Source); c != 0 {
			return c
		}
		return cmp.Compare(a.Type, b.Type)
	})
	return out
}
