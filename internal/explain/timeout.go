package explain

import (
	"strings"
	"time"

	"github.com/1eve1Up/podbay/internal/spec"
)

// ExplainProbeMax caps single-shot explain health probes (proximate-network diagnostic budget).
const ExplainProbeMax = 5 * time.Second

// explainProbeTimeout returns the per-probe deadline for explain health checks.
// Contract timeouts above ExplainProbeMax are capped; shorter values are honored.
// forHTTP applies health.http.timeout override when set.
func explainProbeTimeout(svc spec.Service, forHTTP bool) time.Duration {
	if svc.Health == nil {
		return ExplainProbeMax
	}
	h := svc.Health
	perTry := parseProbeDuration(h.Timeout, 5*time.Second)
	if forHTTP && h.HTTP != nil && strings.TrimSpace(h.HTTP.Timeout) != "" {
		perTry = parseProbeDuration(h.HTTP.Timeout, perTry)
	}
	if perTry > ExplainProbeMax {
		return ExplainProbeMax
	}
	return perTry
}

func parseProbeDuration(s string, def time.Duration) time.Duration {
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
