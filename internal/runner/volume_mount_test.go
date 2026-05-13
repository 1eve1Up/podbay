package runner

import "testing"

func TestSplitVolumeTwoParts(t *testing.T) {
	src, dest, opt := splitVolume("./x:/y")
	if src != "./x" || dest != "/y" || opt != "" {
		t.Fatalf("got %q %q %q", src, dest, opt)
	}
}

func TestSplitVolumeNamedRO(t *testing.T) {
	src, dest, opt := splitVolume("myvol:/data:ro")
	if src != "myvol" || dest != "/data" || opt != "ro" {
		t.Fatalf("got %q %q %q", src, dest, opt)
	}
}

func TestSplitVolumeNamedU(t *testing.T) {
	src, dest, opt := splitVolume("podbay_proj_data:/var/lib/flowboard:U")
	if src != "podbay_proj_data" || dest != "/var/lib/flowboard" || opt != "U" {
		t.Fatalf("got %q %q %q", src, dest, opt)
	}
}

func TestSplitVolumeUCommaRO(t *testing.T) {
	src, dest, opt := splitVolume("v:/etc/certs:U,ro")
	if src != "v" || dest != "/etc/certs" || opt != "U,ro" {
		t.Fatalf("got %q %q %q", src, dest, opt)
	}
}

func TestSplitVolumePathNotOpt(t *testing.T) {
	// "/var/lib/foo" is not a mount option — treat as two-part src:dest
	src, dest, opt := splitVolume("volname:/var/lib/foo")
	if src != "volname" || dest != "/var/lib/foo" || opt != "" {
		t.Fatalf("got %q %q %q", src, dest, opt)
	}
}
