package runtimestate

import "testing"

func TestParseContainerIdentityJSON(t *testing.T) {
	raw := `[{"Id":"sha256:abc123","Image":"docker.io/library/nginx:latest"}]`
	id, img, err := ParseContainerIdentityJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if id != "sha256:abc123" || img != "docker.io/library/nginx:latest" {
		t.Fatalf("got id=%q img=%q", id, img)
	}
}
