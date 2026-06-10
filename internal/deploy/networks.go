package deploy

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/spec"
)

// bridgeDNSForContract returns podman network create --dns values for the project bridge (first create only).
// On macOS/Windows (Podman Machine), Docker Desktop–style stacks often rely on reliable upstream DNS;
// Podbay defaults to 8.8.8.8 unless podman.disable_default_bridge_dns or podman.network_dns is set.
func bridgeDNSForContract(c *spec.Contract, host map[string]string) []string {
	if c.Podman != nil && len(c.Podman.NetworkDNS) > 0 {
		return nonEmptyDNS(expand.ExpandStrings(c.Podman.NetworkDNS, host))
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

func prepareNetworks(ctx *deployContext) error {
	c := ctx.c
	r := ctx.r
	logf := ctx.logf
	opt := ctx.opt

	bridgeMTU := 0
	if c.Network != nil && c.Network.MTU > 0 {
		bridgeMTU = c.Network.MTU
	}
	bridgeDNS := bridgeDNSForContract(c, ctx.hostSubst)
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
		return nil
	}
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
	return nil
}
