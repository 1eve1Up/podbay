package clijson

import (
	"errors"
	"fmt"
	"strings"

	"github.com/1eve1Up/podbay/internal/composefile"
	"github.com/1eve1Up/podbay/internal/spec"
	"github.com/1eve1Up/podbay/internal/validate"
)

// Stable issue codes for init --json.
const (
	CodeInitTargetExists = "init_target_exists"
	CodeInitError        = "init_error"
)

// Init source_kind values for --from-codebase success documents.
const (
	InitSourceCompose    = "compose"
	InitSourceDockerfile = "dockerfile"
)

// InitTargetExistsError is returned when init refuses to overwrite an existing contract.
type InitTargetExistsError struct {
	Path string
}

func (e *InitTargetExistsError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s already exists", e.Path)
}

// InitHandTightenHint is appended when a Dockerfile stub still lacks published ports or health.
const InitHandTightenHint = "hand-tighten: add published ports and/or health on the first-pass contract"

// InitOrientNextActions returns onboard / validate CLI hints for a written contract path.
func InitOrientNextActions(contractPath string) []string {
	p := cleanPath(contractPath)
	return []string{
		fmt.Sprintf("podbay onboard -f %s --json", p),
		fmt.Sprintf("podbay validate -f %s --json", p),
	}
}

// InitDockerfileNextActions is InitOrientNextActions plus a hand-tighten hint when gaps remain.
func InitDockerfileNextActions(contractPath string, gaps []string) []string {
	actions := InitOrientNextActions(contractPath)
	if dockerfileNeedsHandTighten(gaps) {
		actions = append(actions, InitHandTightenHint)
	}
	return actions
}

func dockerfileNeedsHandTighten(gaps []string) bool {
	for _, g := range gaps {
		if g == InitFieldHealth || g == InitFieldPublishedPorts {
			return true
		}
	}
	return false
}

// FromInitFromCodebaseSuccess builds a successful init document after Compose --from-codebase.
// contractPath is the written podbay.yaml; composeSource is the Compose file used.
func FromInitFromCodebaseSuccess(contractPath, composeSource string, c *spec.Contract) *Document {
	n := 0
	proj := ""
	if c != nil {
		n = len(c.Services)
		proj = strings.TrimSpace(c.Project)
	}
	doc := &Document{
		FormatVersion:      FormatVersion,
		Kind:               KindInit,
		Status:             StatusOK,
		ContractPath:       cleanPath(contractPath),
		ComposeSource:      cleanPath(composeSource),
		SourceKind:         InitSourceCompose,
		ImportServiceCount: n,
		NextActions:        InitOrientNextActions(contractPath),
	}
	if proj != "" {
		doc.Project = proj
	}
	return doc
}

// FromInitDockerfileSuccess builds a successful init document after Dockerfile --from-codebase.
func FromInitDockerfileSuccess(contractPath, dockerfileSource string, c *spec.Contract) *Document {
	n := 0
	proj := ""
	if c != nil {
		n = len(c.Services)
		proj = strings.TrimSpace(c.Project)
	}
	extracted, gaps := dockerfileFieldReport(c)
	doc := &Document{
		FormatVersion:      FormatVersion,
		Kind:               KindInit,
		Status:             StatusOK,
		ContractPath:       cleanPath(contractPath),
		DockerfileSource:   cleanPath(dockerfileSource),
		SourceKind:         InitSourceDockerfile,
		ImportServiceCount: n,
		NextActions:        InitDockerfileNextActions(contractPath, gaps),
		Extracted:          extracted,
		Gaps:               gaps,
	}
	if proj != "" {
		doc.Project = proj
	}
	return doc
}

const (
	InitFieldExpose         = "expose"
	InitFieldHealth         = "health"
	InitFieldPublishedPorts = "published_ports"
)

// DockerfileGaps returns still-missing operational fields for a Dockerfile stub.
func DockerfileGaps(c *spec.Contract) []string {
	_, gaps := dockerfileFieldReport(c)
	return gaps
}

func dockerfileFieldReport(c *spec.Contract) (extracted, gaps []string) {
	svc := dockerfileStubService(c)
	if len(svc.Expose) > 0 {
		extracted = append(extracted, InitFieldExpose)
	} else {
		gaps = append(gaps, InitFieldExpose)
	}
	if svc.Health.HasProbe() {
		extracted = append(extracted, InitFieldHealth)
	} else {
		gaps = append(gaps, InitFieldHealth)
	}
	if len(svc.Ports) > 0 {
		extracted = append(extracted, InitFieldPublishedPorts)
	} else {
		gaps = append(gaps, InitFieldPublishedPorts)
	}
	return extracted, gaps
}

func dockerfileStubService(c *spec.Contract) spec.Service {
	if c == nil || len(c.Services) == 0 {
		return spec.Service{}
	}
	if svc, ok := c.Services["app"]; ok {
		return svc
	}
	for _, name := range spec.ServiceNamesSorted(c.Services) {
		return c.Services[name]
	}
	return spec.Service{}
}

// FromInitGreenfieldSuccess builds a successful init document for the nginx template path.
func FromInitGreenfieldSuccess(contractPath string) *Document {
	return &Document{
		FormatVersion: FormatVersion,
		Kind:          KindInit,
		Status:        StatusOK,
		ContractPath:  cleanPath(contractPath),
		Project:       "myapp",
		NextActions:   InitOrientNextActions(contractPath),
	}
}

// FromInitError builds a failed init document. Prefer *InitTargetExistsError and
// *composefile.ImportFailure for stable codes.
func FromInitError(contractPath, composeSource, dockerfileSource string, err error) *Document {
	code := CodeInitError
	msg := "init failed"
	if err != nil {
		msg = err.Error()
		var exists *InitTargetExistsError
		if errors.As(err, &exists) && exists != nil {
			code = CodeInitTargetExists
			msg = exists.Error()
			if contractPath == "" {
				contractPath = exists.Path
			}
		} else {
			var inf *composefile.ImportFailure
			if errors.As(err, &inf) && inf != nil && inf.Code != "" {
				code = inf.Code
				msg = inf.Msg
			}
		}
	}
	doc := &Document{
		FormatVersion:    FormatVersion,
		Kind:             KindInit,
		Status:           StatusFailed,
		ContractPath:     cleanPath(contractPath),
		ComposeSource:    cleanPath(composeSource),
		DockerfileSource: cleanPath(dockerfileSource),
		Issues: []Issue{{
			Level:   validate.LevelFail,
			Code:    code,
			Message: msg,
		}},
	}
	return doc
}
