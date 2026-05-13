package runner

import "testing"

func TestPodmanVolumeName(t *testing.T) {
	r := New("flowboard")
	if got := r.PodmanVolumeName("flowboard-data"); got != "podbay_flowboard_flowboard_data" {
		t.Fatalf("got %q", got)
	}
}
