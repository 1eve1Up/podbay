package composeimport

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/1eve1Up/podbay/internal/composefile"
	"github.com/1eve1Up/podbay/internal/spec"
)

const ansibleVaultPrefix = "$ANSIBLE_VAULT"

func appendConfigSecretVolumes(composeDir string, f *composefile.File, svcName string, cs composefile.ServiceSpec, svc *spec.Service) error {
	composeDir = filepath.Clean(composeDir)
	for _, ref := range cs.Configs {
		srcName := strings.TrimSpace(ref.Source)
		if srcName == "" {
			continue
		}
		def, ok := f.Configs[srcName]
		if !ok {
			return fmt.Errorf("composeimport: service %q: config references undefined config %q", svcName, srcName)
		}
		if def.External {
			return fmt.Errorf("composeimport: config %q: external configs are not supported for import", srcName)
		}
		file := strings.TrimSpace(def.File)
		if file == "" {
			return fmt.Errorf("composeimport: config %q: missing file (define file: under configs)", srcName)
		}
		host, err := absHostPath(composeDir, file)
		if err != nil {
			return fmt.Errorf("composeimport: service %q: config %q: %w", svcName, srcName, err)
		}
		dest := strings.TrimSpace(ref.Target)
		if dest == "" {
			dest = "/" + srcName
		}
		svc.Volumes = append(svc.Volumes, host+":"+dest+":ro")
		if yes, _ := fileIsAnsibleVault(host); yes {
			svc.AnsibleVaultPaths = appendUniqString(svc.AnsibleVaultPaths, host)
		}
	}
	for _, ref := range cs.Secrets {
		srcName := strings.TrimSpace(ref.Source)
		if srcName == "" {
			continue
		}
		def, ok := f.Secrets[srcName]
		if !ok {
			return fmt.Errorf("composeimport: service %q: secret references undefined secret %q", svcName, srcName)
		}
		if def.External {
			return fmt.Errorf("composeimport: secret %q: external secrets are not supported for import", srcName)
		}
		file := strings.TrimSpace(def.File)
		if file == "" {
			return fmt.Errorf("composeimport: secret %q: missing file (define file: under secrets)", srcName)
		}
		host, err := absHostPath(composeDir, file)
		if err != nil {
			return fmt.Errorf("composeimport: service %q: secret %q: %w", svcName, srcName, err)
		}
		dest := strings.TrimSpace(ref.Target)
		if dest == "" {
			dest = "/run/secrets/" + srcName
		}
		svc.Volumes = append(svc.Volumes, host+":"+dest+":ro")
		if yes, _ := fileIsAnsibleVault(host); yes {
			svc.AnsibleVaultPaths = appendUniqString(svc.AnsibleVaultPaths, host)
		}
	}
	return nil
}

func absHostPath(composeDir, p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("empty host path")
	}
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("absolute host paths are not supported (%q); place the file under the compose directory and reference it relatively", p)
	}
	composeDir = filepath.Clean(composeDir)
	joined := filepath.Clean(filepath.Join(composeDir, p))
	rel, err := filepath.Rel(composeDir, joined)
	if err != nil {
		return "", fmt.Errorf("resolve host path %q: %w", p, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("host path %q escapes compose directory", p)
	}
	return joined, nil
}

func fileIsAnsibleVault(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, nil
	}
	defer f.Close()
	buf := make([]byte, 32)
	n, err := f.Read(buf)
	if err != nil || n == 0 {
		return false, nil
	}
	s := string(buf[:n])
	// File may start with BOM or newline; strip leading whitespace for prefix check
	s = strings.TrimLeft(s, " \t\r\n")
	return strings.HasPrefix(s, ansibleVaultPrefix), nil
}

func appendUniqString(slice []string, v string) []string {
	for _, x := range slice {
		if x == v {
			return slice
		}
	}
	return append(slice, v)
}
