package composefile

import (
	"errors"
	"fmt"
	"os"
)

// Stable issue codes for import compose --json (documented in README).
const (
	CodeImportComposeFileNotFound = "import_compose_file_not_found"
	CodeImportComposeRead         = "import_compose_read"
	CodeImportComposeParse        = "import_compose_parse"
	CodeImportIncludeCycle        = "import_include_cycle"
	CodeImportIncludeDepth        = "import_include_depth"
	CodeImportIncludePathEscape   = "import_include_path_escape"
	CodeImportIncludeUnsupported  = "import_include_unsupported"
	// CodeComposeDiscoveryNotFound is returned by Discover when no well-known
	// Compose file exists in the directory (and no usable explicit path was given).
	CodeComposeDiscoveryNotFound = "compose_discovery_not_found"
	// CodeDockerfileDiscoveryNotFound is returned by DiscoverDockerfile when no
	// well-known Dockerfile exists in the directory (and no usable explicit path was given).
	CodeDockerfileDiscoveryNotFound = "dockerfile_discovery_not_found"
	// CodeCodebaseDiscoveryNotFound is returned by init --from-codebase when neither a
	// Compose file nor a Dockerfile can be discovered.
	CodeCodebaseDiscoveryNotFound = "codebase_discovery_not_found"
)

// ImportFailure is a compose load/parse/include error with a stable machine code.
type ImportFailure struct {
	Code string
	Msg  string
}

func (e *ImportFailure) Error() string {
	if e == nil {
		return ""
	}
	return e.Msg
}

// NewImportFailure returns an error with stable code and message text.
func NewImportFailure(code, msg string) *ImportFailure {
	return &ImportFailure{Code: code, Msg: msg}
}

// CodeOrEmpty returns the code or empty if err is not an ImportFailure.
func CodeOrEmpty(err error) string {
	var inf *ImportFailure
	if errors.As(err, &inf) && inf != nil {
		return inf.Code
	}
	return ""
}

// ReadError wraps an OS read failure with stable import codes.
func ReadError(path string, err error) error {
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return NewImportFailure(CodeImportComposeFileNotFound, fmt.Sprintf("composefile: read %s: %v", path, err))
	}
	return NewImportFailure(CodeImportComposeRead, fmt.Sprintf("composefile: read %s: %v", path, err))
}

// ParseError wraps YAML decode failures.
func ParseError(err error) error {
	return NewImportFailure(CodeImportComposeParse, fmt.Sprintf("composefile: parse yaml: %v", err))
}
