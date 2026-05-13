package runner

import "strings"

// SplitVolumeMount parses a `podman run -v` style string: src:dest[:opts].
// See splitVolume for details.
func SplitVolumeMount(s string) (src, dest, opt string) {
	return splitVolume(s)
}

// splitVolume parses a -v argument into source, destination, and optional mount flags
// (e.g. ro, rw, U). Supports Podman's :U (chown volume to the container user), which
// Docker Desktop often makes unnecessary for the same image.
//
// Forms: src:dest, src:dest:ro, src:dest:U, src:dest:ro,U, etc.
// If the segment after the last ":" is not a known option list, the whole string is
// split once as src:dest (bind paths like ./foo:/bar).
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
