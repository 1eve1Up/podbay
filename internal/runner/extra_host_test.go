package runner

import (
	"net"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestParseFirstIPv4FromGetentAhosts(t *testing.T) {
	in := "192.168.127.254 STREAM host.containers.internal\n"
	if got := parseFirstIPv4FromGetentAhosts(in); got != "192.168.127.254" {
		t.Fatalf("got %q", got)
	}
	if got := parseFirstIPv4FromGetentAhosts(""); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultPodmanMachineHostIP(t *testing.T) {
	if net.ParseIP(DefaultPodmanMachineHostIP).To4() == nil {
		t.Fatalf("DefaultPodmanMachineHostIP must be IPv4, got %q", DefaultPodmanMachineHostIP)
	}
}

func TestNormalizeExtraHostPassthrough(t *testing.T) {
	got, err := NormalizeExtraHost("host.docker.internal:192.168.1.1")
	if err != nil || got != "host.docker.internal:192.168.1.1" {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestNormalizeExtraHostHostGatewayDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("Podman Machine rewrite only asserted on darwin/windows")
	}
	_ = os.Unsetenv("PODBAY_HOST_GATEWAY_IP")
	got, err := NormalizeExtraHost("host.docker.internal:host-gateway")
	if err != nil {
		t.Fatal(err)
	}
	host, ip, ok := strings.Cut(got, ":")
	if !ok || !strings.EqualFold(host, "host.docker.internal") {
		t.Fatalf("got %q", got)
	}
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		t.Fatalf("expected IPv4 rewrite, got %q", got)
	}
}

func TestNormalizeExtraHostEnvOverride(t *testing.T) {
	t.Setenv("PODBAY_HOST_GATEWAY_IP", "10.0.0.5")
	got, err := NormalizeExtraHost("host.docker.internal:host-gateway")
	if err != nil {
		t.Fatal(err)
	}
	if got != "host.docker.internal:10.0.0.5" {
		t.Fatalf("got %q", got)
	}
}

func TestComposeDockerDesktopHostAliases(t *testing.T) {
	in := []string{`host.docker.internal:host-gateway`}
	got := ComposeDockerDesktopHostAliases(in)
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if got[1] != `host.containers.internal:host-gateway` {
		t.Fatalf("got %q", got[1])
	}
	// Idempotent when already present
	in2 := []string{`host.docker.internal:host-gateway`, `host.containers.internal:host-gateway`}
	if g2 := ComposeDockerDesktopHostAliases(in2); len(g2) != 2 {
		t.Fatalf("got %v", g2)
	}
}

func TestNormalizeExtraHostLinuxPassthrough(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux passthrough only on linux")
	}
	_ = os.Unsetenv("PODBAY_HOST_GATEWAY_IP")
	got, err := NormalizeExtraHost("host.docker.internal:host-gateway")
	if err != nil {
		t.Fatal(err)
	}
	if got != "host.docker.internal:host-gateway" {
		t.Fatalf("got %q", got)
	}
}
