package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// resolveReceiptWritePath chooses the absolute receipt file path.
// File paths are used as-is. Directory mode (path is an existing directory, or
// ends with / or the OS separator) writes <dir>/<UTC>-<deployID>.json.
func resolveReceiptWritePath(receiptArg, deployID string, at time.Time) (string, error) {
	rp := strings.TrimSpace(receiptArg)
	if rp == "" {
		return "", nil
	}
	if deployID == "" {
		return "", fmt.Errorf("receipt: deploy_id required for path resolution")
	}
	dirMode := strings.HasSuffix(rp, "/") || strings.HasSuffix(rp, string(os.PathSeparator))
	if !dirMode {
		if st, err := os.Stat(rp); err == nil && st.IsDir() {
			dirMode = true
		}
	}
	if dirMode {
		dir, err := filepath.Abs(rp)
		if err != nil {
			return "", fmt.Errorf("receipt: resolve directory: %w", err)
		}
		name := at.UTC().Format("20060102T150405Z") + "-" + deployID + ".json"
		return filepath.Join(dir, name), nil
	}
	abs, err := filepath.Abs(rp)
	if err != nil {
		return "", fmt.Errorf("receipt: resolve path: %w", err)
	}
	return abs, nil
}
