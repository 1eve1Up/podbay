package deploy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/receipt"
	"github.com/1eve1Up/podbay/internal/runner"
	"github.com/1eve1Up/podbay/internal/spec"
)

// Options for Deploy.
type Options struct {
	Profiles []string
	// DeployServices, when non-empty, selects a partial deploy: each name must be in the profile-active set.
	// The effective set is explicit targets unless DeployDependents is true (see spec.ExpandDependentsTransitive).
	DeployServices []string
	// DeployDependents expands partial deploy to include transitive downstream services (depends_on closure).
	DeployDependents bool
	SkipHealthWait   bool
	HealthTimeout    time.Duration
	// Quiet suppresses progress output (similar to docker compose --quiet).
	Quiet bool
	// Out is the writer for progress when Quiet is false. If nil, os.Stdout is used.
	Out io.Writer
	// ReceiptPath, if non-empty, is the path for a deploy receipt JSON written only after a fully successful deploy.
	ReceiptPath string
}

// bridgeDNSForContract returns podman network create --dns values for the project bridge (first create only).
// On macOS/Windows (Podman Machine), Docker Desktop–style stacks often rely on reliable upstream DNS;
// Podbay defaults to 8.8.8.8 unless podman.disable_default_bridge_dns or podman.network_dns is set.
func bridgeDNSForContract(c *spec.Contract, host map[string]string) []string {
	if c.Podman != nil && len(c.Podman.NetworkDNS) > 0 {
		return nonEmptyDNS(expandStrs(c.Podman.NetworkDNS, host))
	}
	if c.Podman != nil && c.Podman.DisableDefaultBridgeDNS {
		return nil
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return []string{"8.8.8.8"}
	}
	return nil
}

func nonEmptyDNS(in []string) []string {
	var out []string
	for _, d := range in {
		d = strings.TrimSpace(d)
		if d != "" {
			out = append(out, d)
		}
	}
	return out
}

// Deploy applies the contract: network, volumes, builds, containers in order, then targeted health gates.
func Deploy(c *spec.Contract, contractFile string, project string, opt Options) error {
	if err := runner.EnsurePodman(); err != nil {
		return err
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

	contractDir := filepath.Dir(contractFile)
	hostSubst, err := expand.LoadHostSubst(contractDir, c.HostEnvFiles)
	if err != nil {
		return err
	}

	profileActive := c.ServicesForProfiles(opt.Profiles)
	active := profileActive
	partial := len(opt.DeployServices) > 0
	if partial {
		sub, err := spec.ServicesForDeployTargets(profileActive, opt.DeployServices)
		if err != nil {
			return err
		}
		active = sub
		if opt.DeployDependents {
			active = spec.ExpandDependentsTransitive(profileActive, sub)
		}
	}
	if len(active) == 0 {
		return fmt.Errorf("no services selected (check --profile and profiles: in the contract)")
	}

	r := runner.New(project)

	autoU := true
	if c.Podman != nil && c.Podman.DisableAutoVolumeU {
		autoU = false
	}

	if !opt.Quiet {
		if partial {
			names := spec.ServiceNamesSorted(active)
			if opt.DeployDependents {
				logf("Partial deploy: project %q — explicit target(s) %s (%d service(s) after dependents expansion): %s",
					project, strings.Join(opt.DeployServices, ", "), len(active), strings.Join(names, ", "))
			} else {
				logf("Partial deploy: project %q — explicit target(s) %s (%d service(s); explicit targets only): %s",
					project, strings.Join(opt.DeployServices, ", "), len(active), strings.Join(names, ", "))
			}
		} else {
			logf("Deploying project %q (full profile-active set, %d service(s))", project, len(active))
		}
		if len(opt.Profiles) > 0 {
			logf(" Profiles: %s", strings.Join(opt.Profiles, ", "))
		}
		if opt.SkipHealthWait {
			logf(" Health wait disabled (--skip-health-wait)")
		}
		if autoU {
			logf(" Podman: named volume mounts without options get :U (set podman.disable_auto_volume_u to turn off)")
		}
	}

	bridgeMTU := 0
	if c.Network != nil && c.Network.MTU > 0 {
		bridgeMTU = c.Network.MTU
	}
	bridgeDNS := bridgeDNSForContract(c, hostSubst)
	multiNet := len(c.Networks) > 0
	if !multiNet {
		netCreated, err := r.EnsureNetwork(bridgeMTU, bridgeDNS)
		if err != nil {
			return err
		}
		if !opt.Quiet {
			netStatus := "Ready"
			if netCreated {
				netStatus = "Created"
			}
			dnsNote := ""
			if len(bridgeDNS) > 0 {
				dnsNote = fmt.Sprintf(" dns=%v", bridgeDNS)
			}
			if bridgeMTU > 0 {
				logf(" Network %-45s %s (mtu=%d)%s", r.Network, netStatus, bridgeMTU, dnsNote)
			} else {
				logf(" Network %-45s %s%s", r.Network, netStatus, dnsNote)
			}
			if bridgeMTU > 0 && !netCreated {
				logf(" warning: network.mtu applies only at create time; remove %q and redeploy to apply MTU: podman network rm %s", r.Network, r.Network)
			}
			if len(bridgeDNS) > 0 && !netCreated {
				logf(" warning: bridge --dns applies only at create time; to apply %v: podman network rm %s", bridgeDNS, r.Network)
			}
		}
	} else {
		for _, key := range spec.NetworkKeysSorted(c.Networks) {
			n := c.Networks[key]
			pn := r.ContractNetworkPodmanName(key, n)
			if n.External {
				if err := r.EnsureExternalNetwork(pn); err != nil {
					return err
				}
				if !opt.Quiet {
					logf(" Network %-45s %s (external)", pn, "Ready")
				}
				continue
			}
			netCreated, err := r.EnsureNamedNetwork(pn, bridgeMTU, bridgeDNS)
			if err != nil {
				return err
			}
			if !opt.Quiet {
				netStatus := "Ready"
				if netCreated {
					netStatus = "Created"
				}
				dnsNote := ""
				if len(bridgeDNS) > 0 {
					dnsNote = fmt.Sprintf(" dns=%v", bridgeDNS)
				}
				if bridgeMTU > 0 {
					logf(" Network %-45s %s (mtu=%d)%s", pn, netStatus, bridgeMTU, dnsNote)
				} else {
					logf(" Network %-45s %s%s", pn, netStatus, dnsNote)
				}
				if bridgeMTU > 0 && !netCreated {
					logf(" warning: network.mtu applies only at create time; remove %q and redeploy to apply MTU: podman network rm %s", pn, pn)
				}
				if len(bridgeDNS) > 0 && !netCreated {
					logf(" warning: bridge --dns applies only at create time; to apply %v: podman network rm %s", bridgeDNS, pn)
				}
			}
		}
	}

	volMap, err := ensureVolumes(r, c, active, contractDir, hostSubst, out, opt.Quiet)
	if err != nil {
		return err
	}

	var order []string
	var topoErr error
	if partial {
		order, topoErr = spec.TopologicalOrderSubset(active)
	} else {
		order, topoErr = spec.TopologicalOrder(active)
	}
	if topoErr != nil {
		return topoErr
	}

	var buildLog io.Writer
	if !opt.Quiet {
		buildLog = out
	}

	for _, name := range order {
		svc := active[name]
		svc = expandService(svc, hostSubst)
		if autoU {
			svc.Volumes = applyPodmanNamedVolumeU(svc.Volumes, volMap)
		}
		svc, cleanupVault, err := applyAnsibleVaultMounts(contractDir, svc)
		if err != nil {
			return fmt.Errorf("service %q: %w", name, err)
		}
		defer func(c func()) { c() }(cleanupVault)

		if !opt.Quiet {
			logf("")
			logf(" Service %q", name)
		}

		if svc.Build != nil && strings.TrimSpace(svc.Image) == "" {
			return fmt.Errorf("service %q: build requires image tag for podman run", name)
		}
		if svc.Build != nil {
			if !opt.Quiet {
				logf("  Building image %s ...", svc.Image)
			}
			if err := r.BuildImage(contractDir, svc.Build, svc.Image, buildLog); err != nil {
				return fmt.Errorf("service %q: %w", name, err)
			}
			if !opt.Quiet {
				logf("  Built %s", svc.Image)
			}
		} else if !opt.Quiet && strings.TrimSpace(svc.Image) != "" {
			logf("  Image %s (pull on run if missing)", svc.Image)
		}

		env, err := containerEnv(contractDir, svc, hostSubst)
		if err != nil {
			return fmt.Errorf("service %q: %w", name, err)
		}

		if err := waitExternalDependsOn(c, r, active, name, svc, hostSubst, partial, out, opt.Quiet, opt.SkipHealthWait, opt.HealthTimeout); err != nil {
			return err
		}

		cname := r.ContainerName(name)
		if !opt.Quiet {
			logf("  Creating container %s ...", cname)
		}
		if err := r.RemoveService(name); err != nil {
			return err
		}
		logicalNets, err := spec.EffectiveServiceNetworks(c, name, svc)
		if err != nil {
			return fmt.Errorf("service %q: %w", name, err)
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
			return fmt.Errorf("service %q: %w", name, err)
		}
		if !opt.Quiet {
			logf("  Started %s", cname)
		}

		if opt.SkipHealthWait || !svc.Health.HasProbe() {
			continue
		}
		if !needsHealthWait(active, name) {
			continue
		}
		if err := waitServiceHealth(out, opt.Quiet, name, cname, svc, opt.HealthTimeout); err != nil {
			return fmt.Errorf("service %q: %w", name, err)
		}
	}
	if !opt.Quiet {
		logf("")
		logf("Done.")
	}

	if rp := strings.TrimSpace(opt.ReceiptPath); rp != "" {
		absReceipt, err := filepath.Abs(rp)
		if err != nil {
			return fmt.Errorf("receipt: resolve path: %w", err)
		}
		absContract, err := filepath.Abs(contractFile)
		if err != nil {
			return fmt.Errorf("receipt: resolve contract path: %w", err)
		}
		records, err := buildReceiptServiceRecords(r, order, active, hostSubst)
		if err != nil {
			return fmt.Errorf("receipt: build records: %w", err)
		}
		rec := receipt.New(absContract, project, opt.Profiles, records)
		if err := receipt.WriteAtomic(absReceipt, rec); err != nil {
			return fmt.Errorf("receipt: %w", err)
		}
		if !opt.Quiet {
			logf(" Wrote receipt %s", absReceipt)
		}
	}
	return nil
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
		depDef = expandService(depDef, hostSubst)
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
			return fmt.Errorf("partial deploy service %q: waiting for dependency %q: %w", svcName, d.Service, err)
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

func waitServiceHealth(out io.Writer, quiet bool, service, container string, svc spec.Service, cliMax time.Duration) error {
	h := svc.Health
	start, interval, probe := healthTiming(svc, cliMax)
	// Total wall clock for probes: start_period + retry window (Compose runs checks immediately;
	// start_period only ignores failures toward marking unhealthy — we approximate with one deadline).
	total := start + probe
	if cliMax > 0 && total > cliMax {
		total = cliMax
	}
	if total < 5*time.Second {
		total = 5 * time.Second
	}
	if !quiet {
		kind := probeKind(svc)
		if kind != "" {
			_, _ = fmt.Fprintf(out, "  Waiting for service %q health (%s), up to %v ...\n", service, kind, total)
		}
	}
	var err error
	if h.HTTP != nil && h.HTTP.URL != "" {
		err = runner.WaitHTTPHealth(h.HTTP.URL, total, h.HTTP.Insecure, interval)
	} else if h.Exec != nil && len(h.Exec.Command) > 0 {
		err = runner.WaitExecHealth(container, h.Exec.Command, interval, total)
	}
	if err != nil {
		return err
	}
	if !quiet {
		_, _ = fmt.Fprintf(out, "  Service %q is healthy\n", service)
	}
	return nil
}

func probeKind(svc spec.Service) string {
	if svc.Health == nil {
		return ""
	}
	if svc.Health.HTTP != nil && strings.TrimSpace(svc.Health.HTTP.URL) != "" {
		return "http"
	}
	if svc.Health.Exec != nil && len(svc.Health.Exec.Command) > 0 {
		return "exec"
	}
	return ""
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

func expandService(svc spec.Service, host map[string]string) spec.Service {
	svc.Ports = expandStrs(svc.Ports, host)
	svc.Volumes = expandStrs(svc.Volumes, host)
	svc.AnsibleVaultPaths = expandStrs(svc.AnsibleVaultPaths, host)
	svc.Environment = expandMap(svc.Environment, host)
	svc.User = expand.String(svc.User, host)
	svc.DNS = expandStrs(svc.DNS, host)
	svc.ExtraHosts = spec.ExtraHostList(expandStrs([]string(svc.ExtraHosts), host))
	if svc.Health != nil && svc.Health.HTTP != nil {
		svc.Health.HTTP.URL = expand.String(svc.Health.HTTP.URL, host)
	}
	return svc
}

func expandStrs(in []string, m map[string]string) []string {
	if len(in) == 0 {
		return in
	}
	o := make([]string, len(in))
	for i, s := range in {
		o[i] = expand.String(s, m)
	}
	return o
}

func expandMap(in map[string]string, m map[string]string) map[string]string {
	if len(in) == 0 {
		return in
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = expand.String(v, m)
	}
	return out
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
		svc = expandService(svc, host)
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
	src, dest, opt := runner.SplitVolumeMount(m)
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
	src, _, _ := runner.SplitVolumeMount(m)
	if src == "" {
		return "", false
	}
	return src, true
}
