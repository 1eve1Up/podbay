package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type contractLoadEntry struct {
	size    int64
	modTime time.Time
	digest  string
	file    string
	c       *Contract
}

type contractLoadMemo struct {
	mu         sync.Mutex
	entries    map[string]contractLoadEntry
	parseCount int
}

func newContractLoadMemo() *contractLoadMemo {
	return &contractLoadMemo{entries: make(map[string]contractLoadEntry)}
}

var contractMemo = newContractLoadMemo()

func resetContractLoadMemoForTest() {
	contractMemo.mu.Lock()
	defer contractMemo.mu.Unlock()
	contractMemo.entries = make(map[string]contractLoadEntry)
	contractMemo.parseCount = 0
}

func contractLoadParseCountForTest() int {
	contractMemo.mu.Lock()
	defer contractMemo.mu.Unlock()
	return contractMemo.parseCount
}

func loadMemoized(file string) (*Contract, string, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return nil, file, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, abs, err
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, abs, err
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])

	contractMemo.mu.Lock()
	defer contractMemo.mu.Unlock()
	if e, ok := contractMemo.entries[abs]; ok && e.digest == digest {
		e.size = st.Size()
		e.modTime = st.ModTime()
		contractMemo.entries[abs] = e
		return e.c, e.file, nil
	}

	c, err := parseContractBytes(data, abs)
	if err != nil {
		return nil, abs, err
	}
	contractMemo.parseCount++
	contractMemo.entries[abs] = contractLoadEntry{
		size:    st.Size(),
		modTime: st.ModTime(),
		digest:  digest,
		file:    abs,
		c:       c,
	}
	return c, abs, nil
}

func parseContractBytes(data []byte, file string) (*Contract, error) {
	var c Contract
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	if c.Version == "" {
		c.Version = "1"
	}
	if c.Services == nil {
		c.Services = map[string]Service{}
	}
	for name, svc := range c.Services {
		for i := range svc.DependsOn {
			svc.DependsOn[i].Condition = NormalizeCondition(svc.DependsOn[i].Condition)
			if svc.DependsOn[i].Condition != ConditionStarted && svc.DependsOn[i].Condition != ConditionHealthy {
				return nil, fmt.Errorf("service %q: depends_on %q: unknown condition %q (use started or healthy)", name, svc.DependsOn[i].Service, svc.DependsOn[i].Condition)
			}
		}
		svc.RedeployPeers = normalizeRedeployPeerNames(svc.RedeployPeers)
		c.Services[name] = svc
	}
	return &c, nil
}
