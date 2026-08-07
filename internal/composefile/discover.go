package composefile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WellKnownComposeNames is the documented first-match order for directory discovery.
// Explicit paths passed to Discover bypass this list.
var WellKnownComposeNames = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yaml",
	"docker-compose.yml",
}

// Discover resolves a Compose file path for brownfield init / import helpers.
//
// If explicit is non-empty, it wins: the path is absolutized and must exist as a
// regular file (directories are rejected). Otherwise dir (default ".") is scanned
// for WellKnownComposeNames in order; the first existing regular file wins.
//
// When no candidate exists, returns an *ImportFailure with CodeComposeDiscoveryNotFound.
func Discover(dir, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", NewImportFailure(CodeComposeDiscoveryNotFound, fmt.Sprintf("composefile: compose path: %v", err))
		}
		st, err := os.Stat(abs)
		if err != nil {
			return "", ReadError(abs, err)
		}
		if st.IsDir() {
			return "", NewImportFailure(CodeComposeDiscoveryNotFound, fmt.Sprintf("composefile: compose path is a directory: %s", abs))
		}
		return abs, nil
	}

	if strings.TrimSpace(dir) == "" {
		dir = "."
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", NewImportFailure(CodeComposeDiscoveryNotFound, fmt.Sprintf("composefile: directory: %v", err))
	}
	st, err := os.Stat(absDir)
	if err != nil {
		return "", ReadError(absDir, err)
	}
	if !st.IsDir() {
		return "", NewImportFailure(CodeComposeDiscoveryNotFound, fmt.Sprintf("composefile: not a directory: %s", absDir))
	}

	var tried []string
	for _, name := range WellKnownComposeNames {
		p := filepath.Join(absDir, name)
		tried = append(tried, name)
		fi, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", ReadError(p, err)
		}
		if fi.IsDir() {
			continue
		}
		return p, nil
	}
	return "", NewImportFailure(
		CodeComposeDiscoveryNotFound,
		fmt.Sprintf("composefile: no compose file found in %s (tried %s)", absDir, strings.Join(tried, ", ")),
	)
}
