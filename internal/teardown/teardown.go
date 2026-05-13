package teardown

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
)

// Options control teardown behavior.
type Options struct {
	Quiet bool
	// Out receives progress when Quiet is false. If nil, os.Stdout is used.
	Out io.Writer
	// Volumes removes named volumes declared under top-level volumes: (like docker compose down -v).
	Volumes bool
	// KeepNetwork skips podman network rm (default is to remove the project network).
	KeepNetwork bool
	// PartialServices lists service names whose containers should be removed. When nil or empty
	// (after trimming), all project-labeled containers are removed (full teardown). Names must
	// match contract service keys; container names are derived via runner.ContainerName.
	PartialServices []string
}

// Run removes Podman resources for the contract: all containers labeled with this project, then optionally the network and named volumes.
func Run(c *spec.Contract, project string, opt Options) error {
	_, err := Execute(c, project, opt)
	return err
}

func normalizePartialServices(in []string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// containersForPartialRemoval returns the subset of allNames that correspond to partialServices
// for this project (deterministic podbay_<project>_<service> names).
func containersForPartialRemoval(project string, allNames []string, partialServices []string) []string {
	r := runner.New(project)
	want := make(map[string]struct{}, len(partialServices))
	for _, s := range partialServices {
		want[r.ContainerName(s)] = struct{}{}
	}
	var out []string
	for _, n := range allNames {
		n = strings.TrimSpace(n)
		if _, ok := want[n]; ok {
			out = append(out, n)
		}
	}
	return out
}

// skipNetworkAfterPartialTeardown reports whether we should not run podman network rm.
func skipNetworkAfterPartialTeardown(keepNetwork, partialMode bool, remainingLabelled int) bool {
	return keepNetwork || (partialMode && remainingLabelled > 0)
}

// Execute runs teardown and returns structured facts. When opt.Quiet is false, human progress is written to opt.Out (or stdout).
func Execute(c *spec.Contract, project string, opt Options) (TeardownResult, error) {
	var res TeardownResult
	res.Project = project

	if err := runner.EnsurePodman(); err != nil {
		return res, NewFatalError(CodePodmanError, err)
	}

	out := io.Writer(io.Discard)
	if !opt.Quiet {
		if opt.Out != nil {
			out = opt.Out
		} else {
			out = os.Stdout
		}
	}
	logf := func(format string, args ...any) {
		_, _ = fmt.Fprintf(out, format+"\n", args...)
	}

	if !opt.Quiet {
		logf("Tearing down project %q", project)
	}

	r := runner.New(project)
	res.Network = r.Network

	names, err := r.ListProjectContainers()
	if err != nil {
		return res, NewFatalError(CodeListError, err)
	}

	partial := normalizePartialServices(opt.PartialServices)
	partialMode := len(partial) > 0
	if partialMode && opt.Volumes {
		return res, NewFatalError(CodeVolumeError, fmt.Errorf("cannot use --volumes with partial service selection: omit service names for a full teardown, or drop --volumes"))
	}

	toRemove := append([]string(nil), names...)
	if partialMode {
		toRemove = containersForPartialRemoval(project, names, partial)
	}
	res.ContainerNames = append([]string(nil), toRemove...)

	if len(toRemove) == 0 {
		if !opt.Quiet {
			if partialMode {
				logf(" No matching containers to remove for selected service(s)")
			} else {
				logf(" No containers labeled podbay.project=%s", project)
			}
		}
	} else {
		if !opt.Quiet {
			for _, n := range toRemove {
				logf(" Removing container %s", n)
			}
		}
		r.RemoveContainersForce(toRemove)
	}

	remainingAfter, errRem := r.ListProjectContainers()
	if errRem != nil {
		return res, NewFatalError(CodeListError, errRem)
	}
	skipNetworkPartial := skipNetworkAfterPartialTeardown(opt.KeepNetwork, partialMode, len(remainingAfter))

	if !opt.KeepNetwork && !skipNetworkPartial {
		if len(c.Networks) > 0 {
			for _, key := range spec.NetworkKeysSorted(c.Networks) {
				n := c.Networks[key]
				if n.External {
					pn := r.ContractNetworkPodmanName(key, n)
					if !opt.Quiet {
						logf(" Skipping external network %s (not removed)", pn)
					}
					continue
				}
				pn := r.NetworkPodmanName(key)
				if !opt.Quiet {
					logf(" Removing network %s", pn)
				}
				if err := r.RemoveNamedNetwork(pn); err != nil {
					res.NetworkWarning = err.Error()
					if !opt.Quiet {
						_, _ = fmt.Fprintf(out, " warning: %v\n", err)
					}
				} else {
					res.NetworkRemoved = true
				}
			}
		} else {
			if !opt.Quiet {
				logf(" Removing network %s", r.Network)
			}
			if err := r.RemoveNetwork(); err != nil {
				res.NetworkWarning = err.Error()
				if !opt.Quiet {
					_, _ = fmt.Fprintf(out, " warning: %v\n", err)
				}
			} else {
				res.NetworkRemoved = true
			}
		}
	} else {
		res.NetworkKept = true
		if !opt.Quiet {
			if opt.KeepNetwork {
				if len(c.Networks) > 0 {
					logf(" Keeping project networks (--keep-network)")
				} else {
					logf(" Keeping network %s (--keep-network)", r.Network)
				}
			} else {
				logf(" Skipping project network removal: %d container(s) still labeled for this project (partial teardown)", len(remainingAfter))
			}
		}
	}

	if opt.Volumes && !partialMode {
		vkeys := make([]string, 0, len(c.Volumes))
		for k := range c.Volumes {
			vkeys = append(vkeys, k)
		}
		sort.Strings(vkeys)
		for _, k := range vkeys {
			vn := r.PodmanVolumeName(k)
			res.VolumeNames = append(res.VolumeNames, vn)
			if !opt.Quiet {
				logf(" Removing volume %s", vn)
			}
			if err := r.RemoveNamedVolume(vn); err != nil {
				return res, NewFatalError(CodeVolumeError, err)
			}
		}
	}

	if !opt.Quiet {
		logf("Done.")
	}
	return res, nil
}
