package expand

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type envFileEntry struct {
	size    int64
	modTime time.Time
	digest  string
	vars    map[string]string
}

type envFileMemo struct {
	mu         sync.Mutex
	entries    map[string]envFileEntry
	parseCount int
}

func newEnvFileMemo() *envFileMemo {
	return &envFileMemo{entries: make(map[string]envFileEntry)}
}

var hostEnvFileMemo = newEnvFileMemo()

func resetHostEnvFileMemoForTest() {
	hostEnvFileMemo.mu.Lock()
	defer hostEnvFileMemo.mu.Unlock()
	hostEnvFileMemo.entries = make(map[string]envFileEntry)
	hostEnvFileMemo.parseCount = 0
}

func hostEnvFileParseCountForTest() int {
	hostEnvFileMemo.mu.Lock()
	defer hostEnvFileMemo.mu.Unlock()
	return hostEnvFileMemo.parseCount
}

func copyEnvMap(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func parseEnvFileBytes(data []byte) map[string]string {
	dst := map[string]string{}
	for _, raw := range bytes.Split(data, []byte("\n")) {
		line := strings.TrimSpace(string(raw))
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
		dst[k] = v
	}
	return dst
}

// loadHostEnvFile returns a copy of the parsed dotenv map for path.
// Missing files yield an empty map. Hits reuse a content-digest memo and
// skip re-parsing when file bytes are unchanged.
func loadHostEnvFile(path string) (map[string]string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	if st.IsDir() {
		return map[string]string{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])

	hostEnvFileMemo.mu.Lock()
	defer hostEnvFileMemo.mu.Unlock()
	if e, ok := hostEnvFileMemo.entries[path]; ok && e.digest == digest {
		e.size = st.Size()
		e.modTime = st.ModTime()
		hostEnvFileMemo.entries[path] = e
		return copyEnvMap(e.vars), nil
	}

	vars := parseEnvFileBytes(data)
	hostEnvFileMemo.parseCount++
	hostEnvFileMemo.entries[path] = envFileEntry{
		size:    st.Size(),
		modTime: st.ModTime(),
		digest:  digest,
		vars:    vars,
	}
	return copyEnvMap(vars), nil
}
