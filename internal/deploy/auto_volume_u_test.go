package deploy

import "testing"

func TestMaybeAppendNamedVolumeU(t *testing.T) {
	volMap := map[string]string{"data": "podbay_x_data", "prometheus-data": "podbay_x_prometheus_data"}

	if got := maybeAppendNamedVolumeU("data:/var/lib/app", volMap); got != "data:/var/lib/app:U" {
		t.Fatalf("got %q", got)
	}
	if got := maybeAppendNamedVolumeU("data:/var/lib/app:U", volMap); got != "data:/var/lib/app:U" {
		t.Fatalf("explicit U: got %q", got)
	}
	if got := maybeAppendNamedVolumeU("data:/var/lib/app:ro", volMap); got != "data:/var/lib/app:ro" {
		t.Fatalf("ro: got %q", got)
	}
	if got := maybeAppendNamedVolumeU("./host:/mnt", volMap); got != "./host:/mnt" {
		t.Fatalf("bind: got %q", got)
	}
	if got := maybeAppendNamedVolumeU("unknown:/mnt", volMap); got != "unknown:/mnt" {
		t.Fatalf("unknown vol: got %q", got)
	}
}
