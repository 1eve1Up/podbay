package deploy

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
	"github.com/1eve1Up/podbay/internal/volumemount"
)

func prepareVolumes(ctx *deployContext) (map[string]string, error) {
	return ensureVolumes(ctx.r, ctx.c, ctx.active, ctx.contractDir, ctx.hostSubst, ctx.out, ctx.opt.Quiet)
}

func ensureVolumes(r *runner.Runner, c *spec.Contract, active map[string]spec.Service, contractDir string, host map[string]string, out io.Writer, quiet bool) (map[string]string, error) {
	volMap := map[string]string{}
	volNames := make([]string, 0, len(c.Volumes))
	for k := range c.Volumes {
		volNames = append(volNames, k)
	}
	sort.Strings(volNames)
	for _, name := range volNames {
		vn, created, err := r.EnsureVolume(name)
		if err != nil {
			return nil, fmt.Errorf("volume %q: %w", name, err)
		}
		volMap[name] = vn
		if !quiet {
			st := "Ready"
			if created {
				st = "Created"
			}
			_, _ = fmt.Fprintf(out, " Volume %-45s %s\n", vn, st)
		}
	}
	for svcName, svc := range active {
		svc = expand.ExpandService(svc, host)
		for _, m := range svc.Volumes {
			left, ok := volumeSourceName(m)
			if !ok {
				continue
			}
			left = strings.TrimSpace(left)
			if strings.Contains(left, "/") || strings.HasPrefix(left, ".") {
				p := left
				if !filepath.IsAbs(p) {
					p = filepath.Join(contractDir, p)
				}
				if _, err := filepath.Abs(p); err != nil {
					return nil, err
				}
				continue
			}
			if _, exists := c.Volumes[left]; exists {
				continue
			}
			return nil, fmt.Errorf("service %q: volume mount %q references undefined volume %q (declare it under volumes:)", svcName, m, left)
		}
	}
	return volMap, nil
}

// applyPodmanNamedVolumeU appends :U to mounts whose source is a declared logical volume key and no option segment is set.
func applyPodmanNamedVolumeU(mounts []string, volMap map[string]string) []string {
	if len(mounts) == 0 {
		return mounts
	}
	out := make([]string, len(mounts))
	for i, m := range mounts {
		out[i] = maybeAppendNamedVolumeU(m, volMap)
	}
	return out
}

func maybeAppendNamedVolumeU(m string, volMap map[string]string) string {
	src, dest, opt := volumemount.SplitVolumeMount(m)
	if strings.TrimSpace(opt) != "" {
		return m
	}
	src = strings.TrimSpace(src)
	if src == "" {
		return m
	}
	if _, ok := volMap[src]; !ok {
		return m
	}
	// Bind paths expanded from the contract are not keys in volMap; guard path-like sources anyway.
	if strings.Contains(src, "/") || strings.Contains(src, "\\") || strings.HasPrefix(src, ".") {
		return m
	}
	return src + ":" + dest + ":U"
}

// volumeSourceName returns the mount source (left side), handling optional flags like :ro or :U.
func volumeSourceName(m string) (string, bool) {
	src, _, _ := volumemount.SplitVolumeMount(m)
	if src == "" {
		return "", false
	}
	return src, true
}
