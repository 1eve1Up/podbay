package composefile

import (
	"errors"
	"os"
	"testing"
)

func TestReadError_notFound(t *testing.T) {
	err := ReadError("/no/such/file.yml", os.ErrNotExist)
	if CodeOrEmpty(err) != CodeImportComposeFileNotFound {
		t.Fatalf("code=%q", CodeOrEmpty(err))
	}
}

func TestReadError_other(t *testing.T) {
	err := ReadError("/tmp/x", errors.New("boom"))
	if CodeOrEmpty(err) != CodeImportComposeRead {
		t.Fatalf("code=%q", CodeOrEmpty(err))
	}
}

func TestParseError(t *testing.T) {
	err := ParseError(errors.New("bad yaml"))
	if CodeOrEmpty(err) != CodeImportComposeParse {
		t.Fatalf("code=%q", CodeOrEmpty(err))
	}
}
