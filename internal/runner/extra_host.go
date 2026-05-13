package runner

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// DefaultPodmanMachineHostIP is the gvproxy-side address Podman Machine uses for the host
// when we cannot resolve it from inside the VM (machine stopped, ssh fails).
// Upstream documents 192.168.127.254 for gvproxy; override with PODBAY_HOST_GATEWAY_IP if needed.
const DefaultPodmanMachineHostIP = "192.168.127.254"

// NormalizeExtraHost maps Docker-Compose-style host-gateway for podman run --add-host.
//
// On Linux, host-gateway is passed through so Podman resolves it (bridge / rootless / pasta).
//
// On macOS/Windows (Podman Machine), Podman often rejects the literal host-gateway token
// ("host containers internal IP address is empty"). Podbay resolves a concrete IPv4 by:
//  1. PODBAY_HOST_GATEWAY_IP
//  2. podman machine ssh … getent ahosts host.containers.internal (authoritative: same view as containers)
//  3. DefaultPodmanMachineHostIP
//
// We do not use net.LookupIP on the Mac/Windows host for this: the host resolver often returns
// 192.168.127.1 (wrong for reaching the physical host from a container); see containers/podman#21681.
func NormalizeExtraHost(entry string) (string, error) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return "", fmt.Errorf("extra_hosts: empty entry")
	}
	// Use last ":" so "host.docker.internal:host-gateway" splits correctly (same as first colon here).
	last := strings.LastIndex(entry, ":")
	if last < 0 {
		return "", fmt.Errorf("extra_hosts %q: expected hostname:ip (e.g. host.docker.internal:host-gateway)", entry)
	}
	host := strings.TrimSpace(entry[:last])
	val := strings.TrimSpace(entry[last+1:])
	if host == "" || val == "" {
		return "", fmt.Errorf("extra_hosts %q: empty hostname or ip", entry)
	}
	if !strings.EqualFold(val, "host-gateway") {
		return entry, nil
	}
	if ip := strings.TrimSpace(os.Getenv("PODBAY_HOST_GATEWAY_IP")); ip != "" {
		return host + ":" + ip, nil
	}
	switch runtime.GOOS {
	case "darwin", "windows":
		return host + ":" + machineHostForExtraHost(), nil
	default:
		return entry, nil
	}
}

func machineHostForExtraHost() string {
	if ip := machineHostFromPodmanMachine(); ip != "" {
		return ip
	}
	return DefaultPodmanMachineHostIP
}

// machineHostFromPodmanMachine asks the Podman VM for host.containers.internal (gvproxy DNS).
// Cached for the process: one ssh per deploy batch, not one per --add-host line.
var machineHostFromPodmanMachineCached = sync.OnceValue(func() string {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "podman", "machine", "ssh", "--",
		"getent", "ahosts", "host.containers.internal")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return parseFirstIPv4FromGetentAhosts(string(out))
})

func machineHostFromPodmanMachine() string {
	return machineHostFromPodmanMachineCached()
}

func parseFirstIPv4FromGetentAhosts(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		ip := net.ParseIP(fields[0])
		if ip != nil && ip.To4() != nil {
			return ip.String()
		}
	}
	return ""
}

// ComposeDockerDesktopHostAliases appends host.containers.internal with the same target as
// host.docker.internal when the contract lists the latter but not the former — matching
// Docker Desktop’s default /etc/hosts entries for reaching the host from a container.
func ComposeDockerDesktopHostAliases(hosts []string) []string {
	if len(hosts) == 0 {
		return hosts
	}
	var rhs string
	foundDocker := false
	hasContainers := false
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		low := strings.ToLower(h)
		if strings.HasPrefix(low, "host.containers.internal:") {
			hasContainers = true
		}
		if strings.HasPrefix(low, "host.docker.internal:") {
			foundDocker = true
			last := strings.LastIndex(h, ":")
			if last >= 0 && last+1 < len(h) {
				rhs = strings.TrimSpace(h[last+1:])
			}
		}
	}
	if !foundDocker || hasContainers || rhs == "" {
		return hosts
	}
	out := make([]string, 0, len(hosts)+1)
	out = append(out, hosts...)
	out = append(out, "host.containers.internal:"+rhs)
	return out
}
