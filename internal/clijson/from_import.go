package clijson

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/1eve1Up/podbay/internal/composefile"
	"github.com/1eve1Up/podbay/internal/spec"
	"github.com/1eve1Up/podbay/internal/validate"
)

// KindImportCompose identifies import compose --json output (success and failure).
const KindImportCompose = "import_compose"

// FromImportComposeSuccess builds a successful import_compose document for --json mode.
// absComposePath is the absolute path to the Compose file; contractYAML is the encoded Podbay contract;
// c supplies project and service count when non-nil; outputAbs is the absolute -o path when set (file already written or about to be).
func FromImportComposeSuccess(absComposePath string, contractYAML []byte, c *spec.Contract, outputAbs string) *Document {
	n := 0
	proj := ""
	if c != nil {
		n = len(c.Services)
		proj = strings.TrimSpace(c.Project)
	}
	doc := &Document{
		FormatVersion:      FormatVersion,
		Kind:               KindImportCompose,
		Status:             StatusOK,
		ContractPath:       cleanPath(absComposePath),
		ImportContractYAML: string(contractYAML),
		ImportServiceCount: n,
	}
	if proj != "" {
		doc.Project = proj
	}
	if outputAbs != "" {
		doc.ImportOutputPath = cleanPath(outputAbs)
	}
	return doc
}

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
