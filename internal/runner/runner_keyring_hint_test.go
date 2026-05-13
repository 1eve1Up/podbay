package runner

import "testing"

func TestPodmanRunKeyringQuotaHint(t *testing.T) {
	msg := []byte("Error: crun: create keyring `abc`: Disk quota exceeded: OCI runtime error")
	h := podmanRunKeyringQuotaHint(msg)
	if h == "" {
		t.Fatal("expected hint for keyring + disk quota message")
	}
	if got := podmanRunKeyringQuotaHint([]byte("some other error")); got != "" {
		t.Fatalf("unexpected hint: %q", got)
	}
}
