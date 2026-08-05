package receipt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListEntry is one validated receipt found in a store directory.
type ListEntry struct {
	Path         string
	DeployID     string
	GeneratedAt  string // RFC3339 UTC
	Project      string
	Status       string
	ServiceCount int
}

// ListDir inventories *.json files under dir that Decode as receipts, newest first.
// Non-receipt or unreadable files are skipped (returned in skipped when non-nil).
func ListDir(dir string) (entries []ListEntry, skipped []string, err error) {
	st, err := os.Stat(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("receipt list: %w", err)
	}
	if !st.IsDir() {
		return nil, nil, fmt.Errorf("receipt list: %s is not a directory", dir)
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("receipt list: read dir: %w", err)
	}
	type row struct {
		entry ListEntry
		ts    int64
	}
	var rows []row
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}
		path := filepath.Join(dir, name)
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			skipped = append(skipped, path+": "+readErr.Error())
			continue
		}
		rec, decErr := Decode(raw)
		if decErr != nil {
			skipped = append(skipped, path+": "+decErr.Error())
			continue
		}
		abs := path
		if a, aErr := filepath.Abs(path); aErr == nil {
			abs = a
		}
		rows = append(rows, row{
			ts: rec.GeneratedAt.UTC().UnixNano(),
			entry: ListEntry{
				Path:         abs,
				DeployID:     rec.DeployID,
				GeneratedAt:  rec.GeneratedAt.UTC().Format("2006-01-02T15:04:05Z"),
				Project:      rec.Project,
				Status:       rec.Status,
				ServiceCount: len(rec.Services),
			},
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].ts != rows[j].ts {
			return rows[i].ts > rows[j].ts
		}
		return rows[i].entry.Path > rows[j].entry.Path
	})
	entries = make([]ListEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, r.entry)
	}
	return entries, skipped, nil
}

// MatchStatus reports whether entry status matches filter.
// filter "" matches everything. filter "ok" also matches legacy empty status.
// filter "failed" matches only StatusFailed.
func MatchStatus(entryStatus, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	switch filter {
	case StatusOK:
		return entryStatus == "" || entryStatus == StatusOK
	case StatusFailed:
		return entryStatus == StatusFailed
	default:
		return entryStatus == filter
	}
}

// FilterEntries returns entries whose Status matches filter (see MatchStatus).
func FilterEntries(entries []ListEntry, filter string) []ListEntry {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return entries
	}
	out := make([]ListEntry, 0, len(entries))
	for _, e := range entries {
		if MatchStatus(e.Status, filter) {
			out = append(out, e)
		}
	}
	return out
}
