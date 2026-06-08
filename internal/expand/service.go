package expand

import "github.com/1eve1Up/podbay/internal/spec"

// ExpandStrings applies host env substitution to each entry in in.
func ExpandStrings(in []string, host map[string]string) []string {
	return expandStrs(in, host)
}

// ExpandService applies host env substitution to runtime-facing Service fields
// before validate, deploy, or explain use expanded values.
func ExpandService(svc spec.Service, host map[string]string) spec.Service {
	svc.Ports = expandStrs(svc.Ports, host)
	svc.Volumes = expandStrs(svc.Volumes, host)
	svc.AnsibleVaultPaths = expandStrs(svc.AnsibleVaultPaths, host)
	svc.Environment = expandMap(svc.Environment, host)
	svc.User = String(svc.User, host)
	svc.DNS = expandStrs(svc.DNS, host)
	svc.ExtraHosts = spec.ExtraHostList(expandStrs([]string(svc.ExtraHosts), host))
	if svc.Health != nil && svc.Health.HTTP != nil {
		svc.Health.HTTP.URL = String(svc.Health.HTTP.URL, host)
	}
	return svc
}

func expandStrs(in []string, m map[string]string) []string {
	if len(in) == 0 {
		return in
	}
	o := make([]string, len(in))
	for i, s := range in {
		o[i] = String(s, m)
	}
	return o
}

func expandMap(in map[string]string, m map[string]string) map[string]string {
	if len(in) == 0 {
		return in
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = String(v, m)
	}
	return out
}
