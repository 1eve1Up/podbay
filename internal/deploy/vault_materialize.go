package deploy

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/1eve1Up/podbay/internal/spec"
	"github.com/1eve1Up/podbay/internal/vault"
	"github.com/1eve1Up/podbay/internal/volumemount"
)

// applyAnsibleVaultMounts replaces bind mount sources listed in svc.AnsibleVaultPaths with
// decrypted temp files via ansible-vault view. cleanup removes all temp files (call after podman run).
func applyAnsibleVaultMounts(contractDir string, svc spec.Service) (_ spec.Service, cleanup func(), err error) {
	noop := func() {}
	if len(svc.AnsibleVaultPaths) == 0 {
		return svc, noop, nil
	}

	want := make(map[string]struct{})
	for _, p := range svc.AnsibleVaultPaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		want[resolvedBindSource(contractDir, p)] = struct{}{}
	}
	if len(want) == 0 {
		return svc, noop, nil
	}

	var cleanups []func()
	deferFn := func() {
		for _, c := range cleanups {
			c()
		}
	}

	out := svc
	out.Volumes = append([]string(nil), svc.Volumes...)
	for i, m := range out.Volumes {
		src, dest, opt := volumemount.SplitVolumeMount(m)
		rs := resolvedBindSource(contractDir, src)
		if rs == "" {
			continue
		}
		if _, ok := want[rs]; !ok {
			continue
		}
		tmp, c, err := vault.DecryptToTemp(rs)
		if err != nil {
			deferFn()
			return svc, noop, fmt.Errorf("ansible vault decrypt for mount source %q: %w", rs, err)
		}
		cleanups = append(cleanups, c)
		out.Volumes[i] = joinVolumeMount(tmp, dest, opt)
	}

	return out, deferFn, nil
}

func resolvedBindSource(contractDir, src string) string {
	src = strings.TrimSpace(src)
	if src == "" {
		return ""
	}
	if filepath.IsAbs(src) {
		return filepath.Clean(src)
	}
	return filepath.Clean(filepath.Join(contractDir, src))
}

func joinVolumeMount(src, dest, opt string) string {
	if strings.TrimSpace(opt) != "" {
		return src + ":" + dest + ":" + opt
	}
	return src + ":" + dest
}
