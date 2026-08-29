package composeimport

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// DockerfileScan is the documented EXPOSE / HEALTHCHECK subset from a Dockerfile.
// Expose holds container ports only (never host:container). Health is a normalized
// exec argv (Compose CMD / CMD-SHELL / NONE rules); nil when absent or NONE.
type DockerfileScan struct {
	Expose []string
	Health []string
}

// ScanDockerfile reads path and returns last-wins EXPOSE ports and a normalized
// HEALTHCHECK argv. Comments, ENV, USER, WORKDIR, and other instructions are ignored.
// A missing or unreadable file is an error; a readable file with no matching
// instructions returns an empty scan (not an error).
func ScanDockerfile(path string) (DockerfileScan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DockerfileScan{}, fmt.Errorf("composeimport: dockerfile scan: %w", err)
	}
	return scanDockerfileBytes(data), nil
}

func scanDockerfileBytes(data []byte) DockerfileScan {
	var out DockerfileScan
	for _, line := range dockerfileLogicalLines(string(data)) {
		inst, rest, ok := splitDockerfileInstruction(line)
		if !ok {
			continue
		}
		switch inst {
		case "EXPOSE":
			out.Expose = parseExposePorts(rest)
		case "HEALTHCHECK":
			out.Health = parseHealthcheckArgv(rest)
		}
	}
	return out
}

func dockerfileLogicalLines(src string) []string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	raw := strings.Split(src, "\n")
	var out []string
	var buf string
	for _, line := range raw {
		trimmedRight := strings.TrimRight(line, " \t")
		if strings.HasSuffix(trimmedRight, "\\") && !strings.HasSuffix(trimmedRight, "\\\\") {
			buf += strings.TrimSuffix(trimmedRight, "\\")
			continue
		}
		buf += line
		out = append(out, buf)
		buf = ""
	}
	if buf != "" {
		out = append(out, buf)
	}
	return out
}

func splitDockerfileInstruction(line string) (inst, rest string, ok bool) {
	line = stripDockerfileComment(line)
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", false
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", "", false
	}
	inst = strings.ToUpper(fields[0])
	rest = strings.TrimSpace(line[len(fields[0]):])
	return inst, rest, true
}

func stripDockerfileComment(line string) string {
	inQuote := false
	for i, r := range line {
		switch r {
		case '"':
			inQuote = !inQuote
		case '#':
			if !inQuote {
				if i == 0 || unicode.IsSpace(rune(line[i-1])) {
					return line[:i]
				}
			}
		}
	}
	return line
}

func parseExposePorts(rest string) []string {
	var ports []string
	for _, tok := range dockerfileTokens(rest) {
		p := exposePortToken(tok)
		if p != "" {
			ports = append(ports, p)
		}
	}
	return ports
}

func exposePortToken(tok string) string {
	tok = strings.TrimSpace(tok)
	if tok == "" || strings.Contains(tok, ":") {
		// Skip host:container-shaped tokens so EXPOSE never invents published ports.
		return ""
	}
	if i := strings.IndexByte(tok, '/'); i >= 0 {
		tok = tok[:i]
	}
	if _, err := strconv.Atoi(tok); err != nil {
		return ""
	}
	return tok
}

func parseHealthcheckArgv(rest string) []string {
	rest = trimHealthcheckFlags(rest)
	if rest == "" {
		return nil
	}
	if argv, ok := healthcheckJSONArgv(rest); ok {
		if len(argv) == 0 {
			return nil
		}
		return normalizeHealthcheckTest(append([]string{"CMD"}, argv...))
	}
	return normalizeHealthcheckTest(dockerfileTokens(rest))
}

func healthcheckJSONArgv(rest string) ([]string, bool) {
	s := strings.TrimSpace(rest)
	if strings.HasPrefix(strings.ToUpper(s), "CMD") {
		after := strings.TrimSpace(s[3:])
		if strings.HasPrefix(after, "[") {
			return parseJSONArgv(after), true
		}
		return nil, false
	}
	if strings.HasPrefix(s, "[") {
		return parseJSONArgv(s), true
	}
	return nil, false
}

func trimHealthcheckFlags(rest string) string {
	tokens := skipHealthcheckFlags(dockerfileTokens(rest))
	if len(tokens) == 0 {
		return ""
	}
	// Re-join so JSON exec form stays one substring starting at '['.
	// Flag tokens are already removed; remaining text is directive + argv.
	idx := indexAfterFlags(rest, tokens[0])
	if idx < 0 {
		return strings.Join(tokens, " ")
	}
	return strings.TrimSpace(rest[idx:])
}

func indexAfterFlags(rest, firstKept string) int {
	return strings.Index(rest, firstKept)
}

func skipHealthcheckFlags(tokens []string) []string {
	i := 0
	for i < len(tokens) {
		t := tokens[i]
		if !strings.HasPrefix(t, "--") {
			break
		}
		i++
		if !strings.Contains(t, "=") && i < len(tokens) && !strings.HasPrefix(tokens[i], "--") && !isHealthcheckDirective(tokens[i]) {
			i++
		}
	}
	return tokens[i:]
}

func isHealthcheckDirective(s string) bool {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "CMD", "CMD-SHELL", "NONE":
		return true
	default:
		return false
	}
}

func parseJSONArgv(s string) []string {
	s = strings.TrimSpace(s)
	var argv []string
	if err := json.Unmarshal([]byte(s), &argv); err != nil {
		return nil
	}
	return argv
}

func dockerfileTokens(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	var b strings.Builder
	inQuote := false
	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, b.String())
		b.Reset()
	}
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case unicode.IsSpace(r) && !inQuote:
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return out
}
