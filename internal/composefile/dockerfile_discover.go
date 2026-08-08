package composefile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WellKnownDockerfileNames is the documented first-match order for directory discovery.
// Explicit paths passed to DiscoverDockerfile bypass this list.
var WellKnownDockerfileNames = []string{
	"Dockerfile",
	"dockerfile",
}

// DiscoverDockerfile resolves a Dockerfile path for brownfield init helpers.
//
// If explicit is non-empty, it wins: the path is absolutized and must exist as a
// regular file (directories are rejected). Otherwise dir (default ".") is scanned
// for WellKnownDockerfileNames in order; the first existing regular file wins.
//
// When no candidate exists, returns an *ImportFailure with CodeDockerfileDiscoveryNotFound.
func DiscoverDockerfile(dir, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", NewImportFailure(CodeDockerfileDiscoveryNotFound, fmt.Sprintf("composefile: dockerfile path: %v", err))
		}
		st, err := os.Stat(abs)
		if err != nil {
			return "", dockerfileReadError(abs, err)
		}
		if st.IsDir() {
			return "", NewImportFailure(CodeDockerfileDiscoveryNotFound, fmt.Sprintf("composefile: dockerfile path is a directory: %s", abs))
		}
		return abs, nil
	}

	if strings.TrimSpace(dir) == "" {
		dir = "."
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", NewImportFailure(CodeDockerfileDiscoveryNotFound, fmt.Sprintf("composefile: directory: %v", err))
	}
	st, err := os.Stat(absDir)
	if err != nil {
		return "", dockerfileReadError(absDir, err)
	}
	if !st.IsDir() {
		return "", NewImportFailure(CodeDockerfileDiscoveryNotFound, fmt.Sprintf("composefile: not a directory: %s", absDir))
	}

	var tried []string
	for _, name := range WellKnownDockerfileNames {
		p := filepath.Join(absDir, name)
		tried = append(tried, name)
		fi, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", dockerfileReadError(p, err)
		}
		if fi.IsDir() {
			continue
		}
		return p, nil
	}
	return "", NewImportFailure(
		CodeDockerfileDiscoveryNotFound,
		fmt.Sprintf("composefile: no Dockerfile found in %s (tried %s)", absDir, strings.Join(tried, ", ")),
	)
}

// dockerfileReadError maps OS read failures for Dockerfile discovery without
// reusing Compose import_compose_* codes for missing Dockerfiles.
func dockerfileReadError(path string, err error) error {
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return NewImportFailure(CodeDockerfileDiscoveryNotFound, fmt.Sprintf("composefile: dockerfile read %s: %v", path, err))
	}
	return NewImportFailure(CodeImportComposeRead, fmt.Sprintf("composefile: dockerfile read %s: %v", path, err))
}
