package receipt

import (
	"errors"
)

// ErrNoLastOK is returned when a receipt directory has no ok receipt
// (including legacy empty status). Callers must not invent a path.
var ErrNoLastOK = errors.New("receipt: no last ok")

// LastOK returns the newest ok receipt from dir (legacy empty status counts as ok).
// ListDir errors are returned as-is. When the directory has no ok receipts,
// LastOK returns (nil, ErrNoLastOK).
func LastOK(dir string) (*ListEntry, error) {
	entries, _, err := ListDir(dir)
	if err != nil {
		return nil, err
	}
	return LastOKFromEntries(entries)
}

// LastOKFromEntries returns the newest ok entry from a ListDir result
// (entries must already be newest-first). Returns (nil, ErrNoLastOK) when none match.
func LastOKFromEntries(entries []ListEntry) (*ListEntry, error) {
	ok := FilterEntries(entries, StatusOK)
	if len(ok) == 0 {
		return nil, ErrNoLastOK
	}
	e := ok[0]
	return &e, nil
}
