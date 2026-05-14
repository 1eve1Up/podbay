package expand

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Compose-compatible substitution forms understood by String:
//
//   - $$              → literal "$"
//   - $VAR            → value of VAR (or unchanged when absent)
//   - ${VAR}          → value of VAR (or unchanged when absent)
//   - ${VAR:-default} → value if set and non-empty, else default
//   - ${VAR-default}  → value if set (even empty), else default
//   - ${VAR:?error}   → value if set and non-empty, else SubstitutionError
//   - ${VAR?error}    → value if set (even empty), else SubstitutionError
//   - ${VAR:+alt}     → alt if VAR is set and non-empty, else ""
//   - ${VAR+alt}      → alt if VAR is set (even empty), else ""
//
// $$ must be matched before any $VAR form so the "escaped $" survives.
var (
	doubleDollar = regexp.MustCompile(`\$\$`)
	bareVar      = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	bracedVar    = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::?[-?+]([^}]*))?\}`)
	bracedOpKind = regexp.MustCompile(`\$\{[A-Za-z_][A-Za-z0-9_]*(:?[-?+])`)
)

// SubstitutionError is returned by Substitute when a ${VAR:?msg} form is used and VAR is unset/empty.
type SubstitutionError struct {
	Var     string
	Message string
}

func (e *SubstitutionError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("required substitution variable %q is unset", e.Var)
	}
	return fmt.Sprintf("required substitution variable %q: %s", e.Var, e.Message)
}

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

// String expands Compose-style substitution forms in s using m. Errors for the ${VAR:?msg}
// form are silently dropped (the token is left unchanged); use Substitute to surface them.
func String(s string, m map[string]string) string {
	out, _ := Substitute(s, m)
	return out
}

// Substitute expands Compose-style substitution forms in s using m. Returns a SubstitutionError
// if any ${VAR:?msg} / ${VAR?msg} form has no value.
func Substitute(s string, m map[string]string) (string, error) {
	// First handle $$ → \x00 placeholder so later passes do not see it as a sigil.
	const dollarPlaceholder = "\x00PODBAY_DOLLAR\x00"
	s = doubleDollar.ReplaceAllString(s, dollarPlaceholder)

	var firstErr error
	expand := func(match string) string {
		// Capture the operator kind ("", ":-", "-", ":?", "?", ":+", "+") to disambiguate.
		opMatch := bracedOpKind.FindStringSubmatch(match)
		op := ""
		if len(opMatch) == 2 {
			op = opMatch[1]
		}
		parts := bracedVar.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		key, arg := parts[1], parts[2]
		v, set := m[key]
		empty := !set || v == ""
		switch op {
		case "":
			if set {
				return v
			}
			return match
		case ":-":
			if empty {
				return arg
			}
			return v
		case "-":
			if !set {
				return arg
			}
			return v
		case ":?":
			if empty && firstErr == nil {
				firstErr = &SubstitutionError{Var: key, Message: arg}
			}
			if empty {
				return ""
			}
			return v
		case "?":
			if !set && firstErr == nil {
				firstErr = &SubstitutionError{Var: key, Message: arg}
			}
			if !set {
				return ""
			}
			return v
		case ":+":
			if !empty {
				return arg
			}
			return ""
		case "+":
			if set {
				return arg
			}
			return ""
		}
		return match
	}
	s = bracedVar.ReplaceAllStringFunc(s, expand)
	s = bareVar.ReplaceAllStringFunc(s, func(match string) string {
		key := match[1:]
		if v, ok := m[key]; ok {
			return v
		}
		return match
	})
	s = strings.ReplaceAll(s, dollarPlaceholder, "$")
	return s, firstErr
}
