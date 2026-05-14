package runner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/1eve1Up/podbay/internal/spec"
)

// Runner executes Podman commands for a project.
type Runner struct {
	Project string
	Network string
}

func New(project string) *Runner {
	return &Runner{
		Project: project,
		Network: "podbay_" + project,
	}
}

func (r *Runner) podman(args ...string) *exec.Cmd {
	return exec.Command("podman", args...)
}

// EnsurePodman checks podman is on PATH.
func EnsurePodman() error {
	if _, err := exec.LookPath("podman"); err != nil {
		return fmt.Errorf("podman not found in PATH: %w", err)
	}
	return nil
}

// EnsureNetwork creates the legacy project bridge (r.Network) if missing.
// If mtu > 0, it is applied as podman network create --opt mtu=<n> (only when creating; existing networks keep their options).
// dns entries become repeated --dns (only when creating).
func (r *Runner) EnsureNetwork(mtu int, dns []string) (created bool, err error) {
	return r.EnsureNamedNetwork(r.Network, mtu, dns)
}

// NetworkPodmanName returns the Podman network name for a logical networks: key (project-scoped).
func (r *Runner) NetworkPodmanName(logical string) string {
	return "podbay_" + r.Project + "_" + sanitize(logical)
}

// ContractNetworkPodmanName returns the Podman network name for a logical networks: entry.
func (r *Runner) ContractNetworkPodmanName(logical string, n spec.Network) string {
	logical = strings.TrimSpace(logical)
	if n.External {
		if s := strings.TrimSpace(n.Name); s != "" {
			return s
		}
		return logical
	}
	return r.NetworkPodmanName(logical)
}

// EnsureExternalNetwork checks that a Podman network already exists (does not create).
func (r *Runner) EnsureExternalNetwork(netName string) error {
	netName = strings.TrimSpace(netName)
	if netName == "" {
		return fmt.Errorf("external network name is empty")
	}
	cmd := r.podman("network", "exists", netName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("external podman network %q does not exist (create it before deploy, e.g. podman network create %s)", netName, netName)
	}
	return nil
}

// EnsureNamedNetwork creates a bridge network by Podman name if missing.
func (r *Runner) EnsureNamedNetwork(netName string, mtu int, dns []string) (created bool, err error) {
	netName = strings.TrimSpace(netName)
	if netName == "" {
		return false, fmt.Errorf("network name is empty")
	}
	cmd := r.podman("network", "exists", netName)
	if cmd.Run() == nil {
		return false, nil
	}
	args := []string{"network", "create"}
	if mtu > 0 {
		args = append(args, "--opt", fmt.Sprintf("mtu=%d", mtu))
	}
	for _, d := range dns {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		args = append(args, "--dns", d)
	}
	args = append(args, netName)
	out, err := r.podman(args...).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("podman network create: %w: %s", err, bytes.TrimSpace(out))
	}
	return true, nil
}

// ContainerName is the deterministic Podman container name for a service.
func (r *Runner) ContainerName(service string) string {
	return fmt.Sprintf("podbay_%s_%s", r.Project, sanitize(service))
}

func sanitize(s string) string {
	b := strings.Builder{}
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			b.WriteRune(c)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "svc"
	}
	return b.String()
}

// RemoveService removes a container if it exists. A "no such container" stderr is treated
// as success; any other failure (busy volume, SELinux denial, ...) is returned so the
// caller does not stumble into "container name already in use" on the subsequent run.
func (r *Runner) RemoveService(service string) error {
	name := r.ContainerName(service)
	cmd := r.podman("rm", "-f", name)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	low := strings.ToLower(string(out))
	if strings.Contains(low, "no such container") || strings.Contains(low, "no container with name") {
		return nil
	}
	trim := bytes.TrimSpace(out)
	return fmt.Errorf("podman rm -f %s: %w: %s", name, err, trim)
}

// ContainerIsRunning reports whether the container exists and podman inspect reports State.Running.
func ContainerIsRunning(container string) (bool, error) {
	out, err := exec.Command("podman", "inspect", "-f", "{{.State.Running}}", container).Output()
	if err != nil {
		trim := strings.TrimSpace(string(out))
		if trim != "" {
			return false, fmt.Errorf("%w: %s", err, trim)
		}
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(string(out)), "true"), nil
}

// PodmanVolumeName is the Podman volume name for a logical key from the contract (no I/O).
func (r *Runner) PodmanVolumeName(logical string) string {
	return "podbay_" + r.Project + "_" + sanitize(logical)
}

// ListProjectContainers returns container names that carry label podbay.project=<Runner.Project>.
func (r *Runner) ListProjectContainers() ([]string, error) {
	cmd := r.podman("ps", "-a", "--filter", "label=podbay.project="+r.Project, "--format", "{{.Names}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("podman ps: %w", err)
	}
	var names []string
	for _, line := range strings.Split(string(bytes.TrimSpace(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		for _, name := range strings.Split(line, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return names, nil
}

// RemoveContainersForce runs podman rm -f for the given names (missing containers are ignored).
func (r *Runner) RemoveContainersForce(names []string) {
	var args []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			args = append(args, n)
		}
	}
	if len(args) == 0 {
		return
	}
	_ = r.podman(append([]string{"rm", "-f"}, args...)...).Run()
}

// RemoveNetwork removes the project bridge network. Returns an error if Podman fails for a reason other than "not found".
func (r *Runner) RemoveNetwork() error {
	return r.RemoveNamedNetwork(r.Network)
}

// RemoveNamedNetwork removes a network by Podman name (no error if missing).
func (r *Runner) RemoveNamedNetwork(netName string) error {
	netName = strings.TrimSpace(netName)
	if netName == "" {
		return nil
	}
	out, err := r.podman("network", "rm", netName).CombinedOutput()
	msg := strings.ToLower(string(out))
	if err != nil {
		if strings.Contains(msg, "no such network") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "unable to find") {
			return nil
		}
		return fmt.Errorf("podman network rm: %w: %s", err, bytes.TrimSpace(out))
	}
	return nil
}

// RemoveNamedVolume removes a Podman volume by full name. No error if the volume does not exist.
func (r *Runner) RemoveNamedVolume(volName string) error {
	out, err := r.podman("volume", "rm", volName).CombinedOutput()
	msg := strings.ToLower(string(out))
	if err != nil {
		if strings.Contains(msg, "no such volume") || strings.Contains(msg, "does not exist") {
			return nil
		}
		return fmt.Errorf("podman volume rm %q: %w: %s", volName, err, bytes.TrimSpace(out))
	}
	return nil
}

// BuildImage runs podman build when Build is set. context and dockerfile paths are relative to contractDir.
// If log is non-nil, build output is streamed there (like docker compose --build). If log is nil, output is buffered and included only on failure.
func (r *Runner) BuildImage(contractDir string, b *spec.Build, image string, log io.Writer) error {
	if b == nil {
		return nil
	}
	ctx := filepath.Join(contractDir, filepath.FromSlash(b.Context))
	df := b.Dockerfile
	if df == "" {
		df = "Dockerfile"
	}
	dfPath := filepath.Join(ctx, filepath.FromSlash(df))
	args := []string{"build", "-t", image, "-f", dfPath, ctx}
	cmd := r.podman(args...)
	if log != nil {
		cmd.Stdout = log
		cmd.Stderr = log
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("podman build: %w", err)
		}
		return nil
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("podman build: %w: %s", err, bytes.TrimSpace(out))
	}
	return nil
}

// StartService runs a container for the given service definition.
// networkPodmans are Podman network names (one = legacy bridge; multiple = joined networks).
func (r *Runner) StartService(serviceName string, svc spec.Service, networkPodmans []string, volMap map[string]string, env map[string]string) error {
	if len(networkPodmans) == 0 {
		return fmt.Errorf("service %q: no podman networks to attach", serviceName)
	}
	name := r.ContainerName(serviceName)
	args := []string{
		"run", "-d",
		"--name", name,
	}
	for _, n := range networkPodmans {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		args = append(args, "--network", n)
	}
	// Compose default network gives each service a DNS name (e.g. api, web). Podman only
	// auto-resolves container names on user-defined networks, so alias the contract service key.
	args = append(args, "--network-alias", serviceName)
	args = append(args,
		"--label", "podbay.project="+r.Project,
		"--label", "podbay.service="+serviceName,
	)
	if svc.Restart != "" {
		args = append(args, "--restart", svc.Restart)
	}
	for _, h := range ComposeDockerDesktopHostAliases([]string(svc.ExtraHosts)) {
		nh, err := NormalizeExtraHost(h)
		if err != nil {
			return fmt.Errorf("service %q extra_hosts: %w", serviceName, err)
		}
		args = append(args, "--add-host", nh)
	}
	for k, v := range svc.Labels {
		args = append(args, "--label", k+"="+v)
	}
	for k, v := range env {
		args = append(args, "-e", k+"="+v)
	}
	if u := strings.TrimSpace(svc.User); u != "" {
		args = append(args, "--user", u)
	}
	for _, d := range svc.DNS {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		args = append(args, "--dns", d)
	}
	for _, p := range svc.Ports {
		args = append(args, "-p", p)
	}
	for _, v := range svc.Volumes {
		exp := expandVolume(v, volMap)
		src, dest, opt := splitVolume(exp)
		arg := src + ":" + dest
		if opt != "" {
			arg += ":" + opt
		}
		args = append(args, "-v", arg)
	}
	// Insert "--" so a service Command starting with a hyphenated token
	// (e.g. "--config /etc/x") is parsed as container argv, not as a podman run flag.
	args = append(args, svc.Image, "--")
	args = append(args, svc.Command...)
	cmd := r.podman(args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		trim := bytes.TrimSpace(out)
		return fmt.Errorf("podman run %s: %w: %s%s%s", serviceName, err, trim, podmanRunKeyringQuotaHint(trim), podmanRunHostGatewayHint(trim))
	}
	return nil
}

// podmanRunKeyringQuotaHint explains crun's misleading "Disk quota exceeded" on keyring create (Podman Machine).
func podmanRunKeyringQuotaHint(stderr []byte) string {
	s := strings.ToLower(string(stderr))
	if !strings.Contains(s, "create keyring") || !strings.Contains(s, "disk quota exceeded") {
		return ""
	}
	return "\n\nhint: this is usually the kernel keyring limit inside Podman Machine (not your disk filling up)." +
		"\n  Until the VM reboots: podman machine ssh -- sudo sysctl -w kernel.keys.maxkeys=20000" +
		"\n  Check:    podman machine ssh -- sh -c 'wc -l < /proc/keys; sysctl kernel.keys.maxkeys'" +
		"\n  Also try: podman machine stop && podman machine start  (and upgrade Podman for gvproxy fixes)" +
		"\n  See: https://github.com/containers/podman/issues/23784"
}

// podmanRunHostGatewayHint explains Podman Machine rejecting literal host-gateway in --add-host.
func podmanRunHostGatewayHint(stderr []byte) string {
	s := strings.ToLower(string(stderr))
	if !strings.Contains(s, "host-gateway") || !strings.Contains(s, "host containers internal") {
		return ""
	}
	return "\n\nhint: Podman could not apply host-gateway in --add-host. Use a current podbay build (resolves host IP on macOS/Windows), or set PODBAY_HOST_GATEWAY_IP before deploy. See README troubleshooting: host.docker.internal."
}

// expandVolume maps named volumes to podman volume names.
func expandVolume(v string, volMap map[string]string) string {
	parts := strings.SplitN(v, ":", 2)
	if len(parts) != 2 {
		return v
	}
	left, right := parts[0], parts[1]
	// handle named:dest:ro — expand only first segment
	rest := right
	if pv, ok := volMap[left]; ok {
		return pv + ":" + rest
	}
	return v
}

// WaitHTTPHealth polls url until 2xx or totalTimeout elapses (first attempt is immediate).
// pollInterval is the delay between attempts; zero defaults to 500ms.
func WaitHTTPHealth(url string, totalTimeout time.Duration, insecure bool, pollInterval time.Duration) error {
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	deadline := time.Now().Add(totalTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		code, err := httpGetCode(url, insecure)
		if err == nil && code >= 200 && code < 300 {
			return nil
		}
		lastErr = err
		if err == nil {
			lastErr = fmt.Errorf("HTTP %d", code)
		}
		time.Sleep(pollInterval)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("timeout")
	}
	return fmt.Errorf("health check failed: %w", lastErr)
}

func httpGetCode(rawURL string, insecure bool) (int, error) {
	args := []string{"-s", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "5"}
	if insecure {
		args = append(args, "-k")
	}
	args = append(args, rawURL)
	cmd := exec.Command("curl", args...)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	code, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, err
	}
	return code, nil
}

// WaitExecHealth runs podman exec until success or totalTimeout elapses (first attempt is immediate).
// This matches Docker Compose: health checks run from container start; start_period only affects
// when failures count toward "unhealthy", not when probes begin.
// pollInterval is the delay between attempts; zero defaults to 500ms.
// Fast-fails when the container reaches a terminal state (exited, dead) so a crash-looping
// service is reported within a poll interval instead of after the full totalTimeout.
func WaitExecHealth(container string, argv []string, pollInterval, totalTimeout time.Duration) error {
	if len(argv) == 0 {
		return fmt.Errorf("exec health: empty command")
	}
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	deadline := time.Now().Add(totalTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		out, err := podmanExecCombined(container, argv)
		if err == nil {
			return nil
		}
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			lastErr = fmt.Errorf("%w: %s", err, msg)
		} else {
			lastErr = err
		}
		if isTerminalContainerState(container) {
			break
		}
		time.Sleep(pollInterval)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("timeout")
	}
	st := inspectContainerBrief(container)
	suffix := execHealthFailureSuffix(container, st)
	return fmt.Errorf("exec health failed: %w\ncontainer %q: %s%s", lastErr, container, st, suffix)
}

// isTerminalContainerState reports whether the container is in a state that will not
// recover without orchestrator intervention (exited or dead).
func isTerminalContainerState(name string) bool {
	out, err := exec.Command("podman", "inspect", "-f", "{{.State.Status}}", name).Output()
	if err != nil {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(string(out)))
	return status == "exited" || status == "dead"
}

func podmanExecCombined(container string, argv []string) ([]byte, error) {
	args := []string{"exec", container}
	// Only insert "--" when the in-container command looks like a Podman flag; otherwise
	// Podman 5.8+ may pass "--" through to crun as the executable (ENOENT on "--").
	if len(argv) > 0 && strings.HasPrefix(argv[0], "-") {
		args = append(args, "--")
	}
	args = append(args, argv...)
	cmd := exec.Command("podman", args...)
	return cmd.CombinedOutput()
}

func inspectContainerBrief(name string) string {
	out, err := exec.Command("podman", "inspect", "-f", "{{.State.Status}} exit={{.State.ExitCode}}", name).Output()
	if err != nil {
		return fmt.Sprintf("inspect: %v", err)
	}
	return strings.TrimSpace(string(out))
}

const execHealthLogTailLines = 40
const execHealthLogMaxBytes = 32000

// execHealthFailureSuffix adds recent container logs when the container has exited (common case for exit 125 on exec).
func execHealthFailureSuffix(container, inspectStatus string) string {
	low := strings.ToLower(inspectStatus)
	if !strings.Contains(low, "exited") {
		return "\nhint: podman logs --tail=50 " + container
	}
	out, err := exec.Command("podman", "logs", fmt.Sprintf("--tail=%d", execHealthLogTailLines), container).CombinedOutput()
	if err != nil || len(bytes.TrimSpace(out)) == 0 {
		return "\nhint: podman logs --tail=50 " + container
	}
	s := string(out)
	if len(s) > execHealthLogMaxBytes {
		s = s[len(s)-execHealthLogMaxBytes:]
	}
	return "\n\n--- podman logs (last " + strconv.Itoa(execHealthLogTailLines) + " lines) ---\n" + strings.TrimRight(s, "\n")
}

// ExecOnce runs a single non-interactive exec and returns combined output.
func ExecOnce(container string, argv []string) ([]byte, error) {
	return podmanExecCombined(container, argv)
}

// EnsureVolume creates a named podman volume; returns the Podman volume name and whether it was newly created.
func (r *Runner) EnsureVolume(logicalName string) (volName string, created bool, err error) {
	volName = "podbay_" + r.Project + "_" + sanitize(logicalName)
	cmd := r.podman("volume", "inspect", volName)
	if cmd.Run() == nil {
		return volName, false, nil
	}
	out, err := r.podman("volume", "create", volName).CombinedOutput()
	if err != nil {
		return "", false, fmt.Errorf("volume create: %w: %s", err, bytes.TrimSpace(out))
	}
	_ = out
	return volName, true, nil
}

// Logs runs `podman logs` for a container, writing interleaved stdout/stderr to w.
func Logs(w io.Writer, container string, follow bool, tail int, since string) error {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	if tail > 0 {
		args = append(args, "--tail", strconv.Itoa(tail))
	}
	since = strings.TrimSpace(since)
	if since != "" {
		args = append(args, "--since", since)
	}
	args = append(args, container)
	cmd := exec.Command("podman", args...)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

// WorkingDir returns current directory for default project name.
func WorkingDir() (string, error) {
	return os.Getwd()
}
