package expand

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var substPattern = regexp.MustCompile(`\$\{([^}:]+)(:-([^}]*))?\}`)

// MapFromEnviron parses KEY=VAL pairs (e.g. os.Environ).
func MapFromEnviron(environ []string) map[string]string {
	m := make(map[string]string, len(environ))
	for _, e := range environ {
		k, v, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		m[k] = v
	}
	return m
}

// MergeFile parses dotenv-style lines into dst. If overrideExisting is false, existing keys in dst are kept.
func MergeFile(dst map[string]string, path string, overrideExisting bool) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			v = v[1 : len(v)-1]
		}
		if k == "" {
			continue
		}
		if _, exists := dst[k]; exists && !overrideExisting {
			continue
		}
		dst[k] = v
	}
	return sc.Err()
}

// ServiceEnvFile is a single env_file entry (mirrors Compose path + required).
type ServiceEnvFile struct {
	Path     string
	Required bool
}

// MergeServiceEnvFiles loads env files for a container (paths relative to contractDir).
// Later files override earlier keys.
func MergeServiceEnvFiles(contractDir string, files []ServiceEnvFile) (map[string]string, error) {
	out := map[string]string{}
	for _, e := range files {
		p := filepath.Join(contractDir, e.Path)
		if _, err := os.Stat(p); err != nil {
			if os.IsNotExist(err) && !e.Required {
				continue
			}
			if os.IsNotExist(err) && e.Required {
				return nil, fmt.Errorf("required env file missing: %s", e.Path)
			}
			return nil, fmt.Errorf("env file %s: %w", e.Path, err)
		}
		if err := MergeFile(out, p, true); err != nil {
			return nil, fmt.Errorf("env file %s: %w", e.Path, err)
		}
	}
	return out, nil
}

// LoadHostSubst builds a map for ${VAR} substitution: os env, then project env files (relative to contractDir).
// If hostEnvFiles is nil, loads .env.example then .env when each exists. If hostEnvFiles is non-nil,
// only those paths are loaded in order (missing files are skipped without error).
func LoadHostSubst(contractDir string, hostEnvFiles []string) (map[string]string, error) {
	m := MapFromEnviron(os.Environ())
	var files []string
	if hostEnvFiles == nil {
		for _, name := range []string{".env.example", ".env"} {
			p := filepath.Join(contractDir, name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				files = append(files, name)
			}
		}
	} else {
		files = append(files, hostEnvFiles...)
	}
	for _, name := range files {
		p := filepath.Join(contractDir, name)
		if err := MergeFile(m, p, true); err != nil {
			return nil, fmt.Errorf("env file %q: %w", name, err)
		}
	}
	return m, nil
}

// String replaces ${VAR} and ${VAR:-default} using m.
// Empty value uses default when :- is present; unset key with no default leaves the token unchanged.
func String(s string, m map[string]string) string {
	return substPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := substPattern.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		key := sub[1]
		hasDefault := sub[2] != ""
		def := sub[3]
		if v, ok := m[key]; ok {
			if v != "" {
				return v
			}
			if hasDefault {
				return def
			}
			return v
		}
		if hasDefault {
			return def
		}
		return match
	})
}
