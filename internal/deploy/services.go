package deploy

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
)

func deployServicesInOrder(ctx *deployContext, volMap map[string]string) ([]string, error) {
	c := ctx.c
	opt := ctx.opt
	r := ctx.r
	logf := ctx.logf

	var order []string
	var topoErr error
	if ctx.partial {
		order, topoErr = spec.TopologicalOrderSubset(ctx.active)
	} else {
		order, topoErr = spec.TopologicalOrder(ctx.active)
	}
	if topoErr != nil {
		return nil, topoErr
	}

	var buildLog io.Writer
	if !opt.Quiet {
		buildLog = ctx.out
	}

	for _, name := range order {
		svc := ctx.active[name]
		svc = expand.ExpandService(svc, ctx.hostSubst)
		if ctx.autoU {
			svc.Volumes = applyPodmanNamedVolumeU(svc.Volumes, volMap)
		}
		svc, cleanupVault, err := applyAnsibleVaultMounts(ctx.contractDir, svc)
		if err != nil {
			return nil, fmt.Errorf("service %q: %w", name, err)
		}
		defer func(c func()) { c() }(cleanupVault)

		if !opt.Quiet {
			logf("")
			logf(" Service %q", name)
		}

		if svc.Build != nil && strings.TrimSpace(svc.Image) == "" {
			return nil, fmt.Errorf("service %q: build requires image tag for podman run", name)
		}
		if svc.Build != nil {
			if !opt.Quiet {
				logf("  Building image %s ...", svc.Image)
			}
			if err := r.BuildImage(ctx.contractDir, svc.Build, svc.Image, buildLog); err != nil {
				return nil, fmt.Errorf("service %q: %w", name, err)
			}
			if !opt.Quiet {
				logf("  Built %s", svc.Image)
			}
		} else if !opt.Quiet && strings.TrimSpace(svc.Image) != "" {
			logf("  Image %s (pull on run if missing)", svc.Image)
		}

		env, err := containerEnv(ctx.contractDir, svc, ctx.hostSubst)
		if err != nil {
			return nil, fmt.Errorf("service %q: %w", name, err)
		}

		if err := waitExternalDependsOn(c, r, ctx.active, name, svc, ctx.hostSubst, ctx.partial, ctx.out, opt.Quiet, opt.SkipHealthWait, opt.HealthTimeout); err != nil {
			return order, err
		}

		cname := r.ContainerName(name)
		if !opt.Quiet {
			logf("  Creating container %s ...", cname)
		}
		if err := r.RemoveService(name); err != nil {
			return order, err
		}
		logicalNets, err := spec.EffectiveServiceNetworks(c, name, svc)
		if err != nil {
			return order, fmt.Errorf("service %q: %w", name, err)
		}
		var podNets []string
		if logicalNets == nil {
			podNets = []string{r.Network}
		} else {
			for _, L := range logicalNets {
				podNets = append(podNets, r.ContractNetworkPodmanName(L, c.Networks[L]))
			}
		}
		if err := r.StartService(name, svc, podNets, volMap, env); err != nil {
			return order, fmt.Errorf("service %q: %w", name, err)
		}
		if !opt.Quiet {
			logf("  Started %s", cname)
		}

		if opt.SkipHealthWait || !svc.Health.HasProbe() {
			continue
		}
		if !needsHealthWait(ctx.active, name) {
			continue
		}
		if err := waitServiceHealth(ctx.out, opt.Quiet, name, cname, svc, opt.HealthTimeout); err != nil {
			return order, err
		}
	}
	if !opt.Quiet {
		logf("")
		logf("Done.")
	}
	return order, nil
}

func waitExternalDependsOn(c *spec.Contract, r *runner.Runner, active map[string]spec.Service, svcName string, svc spec.Service, hostSubst map[string]string, partial bool, out io.Writer, quiet bool, skipHealth bool, healthTimeout time.Duration) error {
	if !partial {
		return nil
	}
	for _, d := range svc.DependsOn {
		if _, ok := active[d.Service]; ok {
			continue
		}
		depDef, ok := c.Services[d.Service]
		if !ok {
			return fmt.Errorf("service %q: depends on unknown service %q", svcName, d.Service)
		}
		depDef = expand.ExpandService(depDef, hostSubst)
		cname := r.ContainerName(d.Service)
		runOK, err := runner.ContainerIsRunning(cname)
		if err != nil || !runOK {
			var detail error = err
			if err == nil {
				detail = fmt.Errorf("container not running")
			}
			return fmt.Errorf("partial deploy service %q: dependency %q is not running: %w", svcName, d.Service, detail)
		}
		if d.Condition != spec.ConditionHealthy {
			continue
		}
		if skipHealth {
			if !quiet {
				_, _ = fmt.Fprintf(out, "  Skipping health wait for external dependency %q (--skip-health-wait)\n", d.Service)
			}
			continue
		}
		if !depDef.Health.HasProbe() {
			return fmt.Errorf("partial deploy service %q: dependency %q is healthy but has no health probe in the contract", svcName, d.Service)
		}
		if err := waitServiceHealth(out, quiet, d.Service, cname, depDef, healthTimeout); err != nil {
			if hg, ok := err.(*HealthGateFailure); ok {
				out := *hg
				out.ExternalDep = true
				out.RequestedBy = svcName
				return &out
			}
			return newExternalDepHealthFailure(svcName, d.Service, ProbeKind(probeKind(depDef)), err)
		}
	}
	return nil
}

func needsHealthWait(active map[string]spec.Service, svcName string) bool {
	for _, svc := range active {
		for _, d := range svc.DependsOn {
			if d.Service == svcName && d.Condition == spec.ConditionHealthy {
				return true
			}
		}
	}
	return false
}

// healthTiming returns start_period, poll interval, and probe window (time allowed for checks after start_period).
func healthTiming(svc spec.Service, cliMax time.Duration) (startPeriod, interval, probeBudget time.Duration) {
	h := svc.Health
	if h == nil {
		return 0, 500 * time.Millisecond, cliMax
	}
	startPeriod = parseDur(h.StartPeriod, 0)
	interval = parseDur(h.Interval, 500*time.Millisecond)
	if h.HTTP != nil && strings.TrimSpace(h.HTTP.Interval) != "" {
		interval = parseDur(h.HTTP.Interval, interval)
	}
	perTry := parseDur(h.Timeout, 5*time.Second)
	if h.HTTP != nil && strings.TrimSpace(h.HTTP.Timeout) != "" {
		perTry = parseDur(h.HTTP.Timeout, perTry)
	}
	retries := h.Retries
	if retries <= 0 {
		retries = 6
	}
	probeBudget = time.Duration(retries+2) * interval
	if probeBudget < 2*perTry {
		probeBudget = 2 * perTry
	}
	if probeBudget < 5*time.Second {
		probeBudget = 5 * time.Second
	}
	if cliMax > 0 {
		capProbe := cliMax - startPeriod
		if capProbe < 5*time.Second {
			capProbe = 5 * time.Second
		}
		if probeBudget > capProbe {
			probeBudget = capProbe
		}
	}
	return startPeriod, interval, probeBudget
}

func parseDur(s string, def time.Duration) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

func containerEnv(contractDir string, svc spec.Service, host map[string]string) (map[string]string, error) {
	refs := make([]expand.ServiceEnvFile, len(svc.EnvFile))
	for i, e := range svc.EnvFile {
		refs[i] = expand.ServiceEnvFile{Path: e.Path, Required: e.Required}
	}
	fileEnv, err := expand.MergeServiceEnvFiles(contractDir, refs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for k, v := range fileEnv {
		out[k] = v
	}
	for k, v := range svc.Environment {
		out[k] = expand.String(v, host)
	}
	return out, nil
}
