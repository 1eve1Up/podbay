package clijson

import (
	"errors"
	"path/filepath"

	"github.com/1eve1Up/podbay/internal/composefile"
	"github.com/1eve1Up/podbay/internal/validate"
)

// KindImportCompose identifies import compose JSON output (failure path; success stays YAML-only).
const KindImportCompose = "import_compose"

// FromImportComposeError builds a failed import_compose document from any compose load error.
// When err wraps *composefile.ImportFailure, its Code and message are used; otherwise code is import_contract_error.
func FromImportComposeError(absComposePath string, err error) *Document {
	if err == nil {
		return &Document{
			FormatVersion: FormatVersion,
			Kind:          KindImportCompose,
			Status:        StatusOK,
			ContractPath:  cleanPath(absComposePath),
		}
	}
	code := "import_contract_error"
	msg := err.Error()
	var inf *composefile.ImportFailure
	if errors.As(err, &inf) && inf != nil && inf.Code != "" {
		code = inf.Code
		msg = inf.Msg
	}
	return &Document{
		FormatVersion: FormatVersion,
		Kind:          KindImportCompose,
		Status:        StatusFailed,
		ContractPath:  cleanPath(absComposePath),
		Issues: []Issue{{
			Level:   validate.LevelFail,
			Code:    code,
			Message: msg,
		}},
	}
}

func cleanPath(p string) string {
	if p == "" {
		return ""
	}
	return filepath.Clean(p)
}
