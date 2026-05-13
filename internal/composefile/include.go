package composefile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxIncludeDepth = 16

// IncludeEntry is one Compose `include:` list item (subset: local path only).
type IncludeEntry struct {
	Path string
}

// UnmarshalYAML accepts a bare path string or a mapping with `path:` only.
func (e *IncludeEntry) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		p := strings.TrimSpace(n.Value)
		if p == "" {
			return NewImportFailure(CodeImportComposeParse, "composefile: include: empty path")
		}
		e.Path = p
		return nil
	case yaml.MappingNode:
		type raw struct {
			Path    string   `yaml:"path"`
			EnvFile []string `yaml:"env_file"`
		}
		var r raw
		if err := n.Decode(&r); err != nil {
			return NewImportFailure(CodeImportComposeParse, fmt.Sprintf("composefile: include: %v", err))
		}
		if len(r.EnvFile) > 0 {
			return NewImportFailure(CodeImportIncludeUnsupported, "composefile: include: env_file on include entries is not supported")
		}
		r.Path = strings.TrimSpace(r.Path)
		if r.Path == "" {
			return NewImportFailure(CodeImportComposeParse, "composefile: include: path is required")
		}
		e.Path = r.Path
		return nil
	default:
		return NewImportFailure(CodeImportComposeParse, "composefile: include: expected string or mapping")
	}
}

func newEmptyFile() *File {
	return &File{
		Services: map[string]ServiceSpec{},
		Volumes:  map[string]VolumeSpec{},
		Networks: map[string]NetworkSpec{},
		Configs:  map[string]ConfigSpec{},
		Secrets:  map[string]SecretSpec{},
	}
}

// mergeComposeOverlay merges src into dst; for maps, src keys overwrite dst (later wins).
func mergeComposeOverlay(dst, src *File) {
	if dst == nil || src == nil {
		return
	}
	for k, v := range src.Services {
		dst.Services[k] = v
	}
	for k, v := range src.Volumes {
		dst.Volumes[k] = v
	}
	for k, v := range src.Networks {
		dst.Networks[k] = v
	}
	for k, v := range src.Configs {
		dst.Configs[k] = v
	}
	for k, v := range src.Secrets {
		dst.Secrets[k] = v
	}
}

func assertIncludePathUnderComposeRoot(rootDir, joined string) error {
	rootDir = filepath.Clean(rootDir)
	joined = filepath.Clean(joined)
	rel, err := filepath.Rel(rootDir, joined)
	if err != nil {
		return NewImportFailure(CodeImportIncludePathEscape, fmt.Sprintf("composefile: include: resolve path: %v", err))
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return NewImportFailure(CodeImportIncludePathEscape, fmt.Sprintf("composefile: include: path %q escapes compose directory", joined))
	}
	return nil
}

// loadIncludedMergedAndExtends reads abs, merges its nested includes, then runs ResolveExtends.
// ancestorStack is the absolute paths of compose files from the root primary down to the parent of abs (inclusive of parent, exclusive of abs).
func loadIncludedMergedAndExtends(abs string, ancestorStack []string) (*File, error) {
	abs = filepath.Clean(abs)
	for _, s := range ancestorStack {
		if s == abs {
			return nil, NewImportFailure(CodeImportIncludeCycle, fmt.Sprintf("composefile: include cycle involving %q", abs))
		}
	}
	if len(ancestorStack) >= maxIncludeDepth {
		return nil, NewImportFailure(CodeImportIncludeDepth, fmt.Sprintf("composefile: include depth exceeds %d", maxIncludeDepth))
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, ReadError(abs, err)
	}
	sub, err := Parse(data)
	if err != nil {
		return nil, err
	}
	if err := ResolveIncludeMergeOnly(abs, sub, ancestorStack); err != nil {
		return nil, err
	}
	if err := ResolveExtends(abs, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// ResolveIncludeMergeOnly merges Compose `include:` files into f (in place), then clears f.Include.
// It does not run ResolveExtends on f; the caller should run ResolveExtends on the primary file after includes.
// ancestorStack is absolute paths of compose files from the root primary down to but not including fileAbs.
func ResolveIncludeMergeOnly(fileAbs string, f *File, ancestorStack []string) error {
	if f == nil || len(f.Include) == 0 {
		return nil
	}
	fileAbs = filepath.Clean(fileAbs)
	rootDir := filepath.Dir(fileAbs)

	merged := newEmptyFile()
	merged.Version = f.Version

	for _, inc := range f.Include {
		p := strings.TrimSpace(inc.Path)
		if p == "" {
			return NewImportFailure(CodeImportComposeParse, "composefile: include: empty path in list")
		}
		if strings.Contains(p, "://") {
			return NewImportFailure(CodeImportIncludeUnsupported, fmt.Sprintf("composefile: include: URL includes are not supported (%q)", p))
		}
		if filepath.IsAbs(p) {
			return NewImportFailure(CodeImportIncludeUnsupported, fmt.Sprintf("composefile: include: absolute paths are not supported (%q)", p))
		}
		joined := filepath.Clean(filepath.Join(rootDir, p))
		if err := assertIncludePathUnderComposeRoot(rootDir, joined); err != nil {
			return err
		}
		childAncestors := append(append([]string(nil), ancestorStack...), fileAbs)
		for _, s := range childAncestors {
			if s == joined {
				return NewImportFailure(CodeImportIncludeCycle, fmt.Sprintf("composefile: include cycle involving %q", joined))
			}
		}
		if len(childAncestors) >= maxIncludeDepth {
			return NewImportFailure(CodeImportIncludeDepth, fmt.Sprintf("composefile: include depth exceeds %d", maxIncludeDepth))
		}
		sub, err := loadIncludedMergedAndExtends(joined, childAncestors)
		if err != nil {
			return fmt.Errorf("include %q: %w", p, err)
		}
		mergeComposeOverlay(merged, sub)
	}

	rootPart := *f
	rootPart.Include = nil
	mergeComposeOverlay(merged, &rootPart)
	*f = *merged
	f.Include = nil
	return nil
}
