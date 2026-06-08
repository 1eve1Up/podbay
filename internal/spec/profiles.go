package spec

// IsIncluded implements Compose profiles: services without profiles always run;
// services with profiles run only when a matching --profile is passed.
func (s *Service) IsIncluded(profiles []string) bool {
	if len(s.Profiles) == 0 {
		return true
	}
	if len(profiles) == 0 {
		return false
	}
	want := map[string]struct{}{}
	for _, p := range profiles {
		want[p] = struct{}{}
	}
	for _, p := range s.Profiles {
		if _, ok := want[p]; ok {
			return true
		}
	}
	return false
}

// ServicesForProfiles returns the subset of services that should run.
func (c *Contract) ServicesForProfiles(profiles []string) map[string]Service {
	out := make(map[string]Service)
	for n, svc := range c.Services {
		if svc.IsIncluded(profiles) {
			out[n] = svc
		}
	}
	return out
}
