package volumemount

import "strings"

// SplitVolumeMount parses a `podman run -v` style string: src:dest[:opts].
func SplitVolumeMount(s string) (src, dest, opt string) {
	return splitVolume(s)
}

func splitVolume(s string) (src, dest, opt string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", ""
	}
	last := strings.LastIndex(s, ":")
	if last < 0 {
		return s, "", ""
	}
	tail := strings.TrimSpace(s[last+1:])
	if tail != "" && mountOptsValid(tail) {
		mid := strings.TrimSpace(s[:last])
		idx := strings.LastIndex(mid, ":")
		if idx <= 0 {
			return mid, "", tail
		}
		return strings.TrimSpace(mid[:idx]), strings.TrimSpace(mid[idx+1:]), tail
	}
	return strings.TrimSpace(s[:last]), tail, ""
}

func mountOptsValid(tail string) bool {
	if tail == "" {
		return false
	}
	for _, p := range strings.Split(tail, ",") {
		p = strings.TrimSpace(p)
		if !mountOptionTokenOK(p) {
			return false
		}
	}
	return true
}

func mountOptionTokenOK(t string) bool {
	switch t {
	case "ro", "rw",
		"Z", "z", "U", "O",
		"noexec", "nodev", "nosuid",
		"copy", "nocopy", "bind",
		"shared", "slave", "private",
		"rshared", "rslave", "rprivate",
		"delegate":
		return true
	default:
		return false
	}
}
