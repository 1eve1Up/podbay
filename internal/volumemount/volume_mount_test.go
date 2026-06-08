package volumemount

import "testing"

func TestSplitVolumeTwoParts(t *testing.T) {
	src, dest, opt := SplitVolumeMount("./x:/y")
	if src != "./x" || dest != "/y" || opt != "" {
		t.Fatalf("got %q %q %q", src, dest, opt)
	}
}

func TestSplitVolumeNamedRO(t *testing.T) {
	src, dest, opt := SplitVolumeMount("myvol:/data:ro")
	if src != "myvol" || dest != "/data" || opt != "ro" {
		t.Fatalf("got %q %q %q", src, dest, opt)
	}
}

func TestSplitVolumeNamedU(t *testing.T) {
	src, dest, opt := SplitVolumeMount("podbay_proj_data:/var/lib/flowboard:U")
	if src != "podbay_proj_data" || dest != "/var/lib/flowboard" || opt != "U" {
		t.Fatalf("got %q %q %q", src, dest, opt)
	}
}

func TestSplitVolumeUCommaRO(t *testing.T) {
	src, dest, opt := SplitVolumeMount("v:/etc/certs:U,ro")
	if src != "v" || dest != "/etc/certs" || opt != "U,ro" {
		t.Fatalf("got %q %q %q", src, dest, opt)
	}
}

func TestSplitVolumePathNotOpt(t *testing.T) {
	src, dest, opt := SplitVolumeMount("volname:/var/lib/foo")
	if src != "volname" || dest != "/var/lib/foo" || opt != "" {
		t.Fatalf("got %q %q %q", src, dest, opt)
	}
}
