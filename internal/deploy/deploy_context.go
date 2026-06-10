package deploy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
)

// deployContext holds shared state for Deploy() phase functions.
type deployContext struct {
	c            *spec.Contract
	contractFile string
	project      string
	opt          Options
	contractDir  string
	hostSubst    map[string]string
	active       map[string]spec.Service
	partial      bool
	r            *runner.Runner
	out          io.Writer
	logf         func(format string, args ...any)
	autoU        bool
}

func newDeployContext(c *spec.Contract, contractFile, project string, opt Options) (*deployContext, error) {
	if err := runner.EnsurePodman(); err != nil {
		return nil, err
	}
	out := io.Writer(io.Discard)
	if !opt.Quiet {
		if opt.Out != nil {
			out = opt.Out
		} else {
			out = os.Stdout
		}
	}
	ctx := &deployContext{
		c:            c,
		contractFile: contractFile,
		project:      project,
		opt:          opt,
		out:          out,
		logf: func(format string, args ...any) {
			_, _ = fmt.Fprintf(out, format+"\n", args...)
		},
	}
	ctx.contractDir = filepath.Dir(contractFile)
	hostSubst, err := expand.LoadHostSubst(ctx.contractDir, c.HostEnvFiles)
	if err != nil {
		return nil, err
	}
	ctx.hostSubst = hostSubst

	profileActive := c.ServicesForProfiles(opt.Profiles)
	active, err := spec.ObservabilityActiveServices(profileActive, opt.DeployServices, opt.DeployDependents)
	if err != nil {
		return nil, err
	}
	ctx.active = active
	ctx.partial = len(opt.DeployServices) > 0
	if len(active) == 0 {
		return nil, fmt.Errorf("no services selected (check --profile and profiles: in the contract)")
	}

	ctx.r = runner.New(project)
	ctx.autoU = true
	if c.Podman != nil && c.Podman.DisableAutoVolumeU {
		ctx.autoU = false
	}

	if !opt.Quiet {
		if ctx.partial {
			names := spec.ServiceNamesSorted(active)
			if opt.DeployDependents {
				ctx.logf("Partial deploy: project %q — explicit target(s) %s (%d service(s) after dependents expansion): %s",
					project, strings.Join(opt.DeployServices, ", "), len(active), strings.Join(names, ", "))
			} else {
				ctx.logf("Partial deploy: project %q — explicit target(s) %s (%d service(s); explicit targets only): %s",
					project, strings.Join(opt.DeployServices, ", "), len(active), strings.Join(names, ", "))
			}
		} else {
			ctx.logf("Deploying project %q (full profile-active set, %d service(s))", project, len(active))
		}
		if len(opt.Profiles) > 0 {
			ctx.logf(" Profiles: %s", strings.Join(opt.Profiles, ", "))
		}
		if opt.SkipHealthWait {
			ctx.logf(" Health wait disabled (--skip-health-wait)")
		}
		if ctx.autoU {
			ctx.logf(" Podman: named volume mounts without options get :U (set podman.disable_auto_volume_u to turn off)")
		}
	}
	return ctx, nil
}
