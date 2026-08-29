package composeimport

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScanDockerfile_missingFile(t *testing.T) {
	_, err := ScanDockerfile(filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("expected read error")
	}
}

func TestScanDockerfile_missingInstructions(t *testing.T) {
	p := writeDockerfile(t, "FROM alpine:3.20\nCMD [\"sleep\", \"infinity\"]\n")
	got, err := ScanDockerfile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Expose) != 0 || got.Health != nil {
		t.Fatalf("expected empty scan, got %+v", got)
	}
}

func TestScanDockerfile_commentsIgnored(t *testing.T) {
	p := writeDockerfile(t, `# EXPOSE 80
FROM alpine
# HEALTHCHECK CMD wget -q -O- http://127.0.0.1/
ENV PORT=8080
USER nobody
WORKDIR /app
EXPOSE 90 # inline comment
`)
	got, err := ScanDockerfile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Expose, []string{"90"}) {
		t.Fatalf("expose=%v", got.Expose)
	}
	if got.Health != nil {
		t.Fatalf("health must stay empty when only comments declare HEALTHCHECK: %+v", got.Health)
	}
}

func TestScanDockerfile_exposeLastWins(t *testing.T) {
	p := writeDockerfile(t, `
FROM alpine
EXPOSE 80 443/tcp
EXPOSE 8080/tcp
`)
	got, err := ScanDockerfile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Expose, []string{"8080"}) {
		t.Fatalf("last-wins expose=%v", got.Expose)
	}
}

func TestScanDockerfile_exposeSkipsHostContainerShape(t *testing.T) {
	p := writeDockerfile(t, "FROM alpine\nEXPOSE 8080:80 90\n")
	got, err := ScanDockerfile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Expose, []string{"90"}) {
		t.Fatalf("must not invent host:container ports: %v", got.Expose)
	}
}

func TestScanDockerfile_healthcheckCMD(t *testing.T) {
	p := writeDockerfile(t, "FROM alpine\nHEALTHCHECK CMD wget -q -O- http://127.0.0.1/\n")
	got, err := ScanDockerfile(p)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"wget", "-q", "-O-", "http://127.0.0.1/"}
	if !reflect.DeepEqual(got.Health, want) {
		t.Fatalf("health=%v want %v", got.Health, want)
	}
}

func TestScanDockerfile_healthcheckCMDShell(t *testing.T) {
	p := writeDockerfile(t, "FROM alpine\nHEALTHCHECK CMD-SHELL curl -f http://127.0.0.1/ || exit 1\n")
	got, err := ScanDockerfile(p)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"sh", "-c", "curl -f http://127.0.0.1/ || exit 1"}
	if !reflect.DeepEqual(got.Health, want) {
		t.Fatalf("health=%v want %v", got.Health, want)
	}
}

func TestScanDockerfile_healthcheckNONE(t *testing.T) {
	p := writeDockerfile(t, `
FROM alpine
HEALTHCHECK CMD wget -q -O- http://127.0.0.1/
HEALTHCHECK NONE
`)
	got, err := ScanDockerfile(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Health != nil {
		t.Fatalf("NONE must clear health: %v", got.Health)
	}
}

func TestScanDockerfile_healthcheckLastWinsAndFlags(t *testing.T) {
	p := writeDockerfile(t, `
FROM alpine
HEALTHCHECK CMD /bin/false
HEALTHCHECK --interval=5s --timeout=3s --retries=2 CMD wget --spider http://127.0.0.1/
`)
	got, err := ScanDockerfile(p)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"wget", "--spider", "http://127.0.0.1/"}
	if !reflect.DeepEqual(got.Health, want) {
		t.Fatalf("health=%v want %v", got.Health, want)
	}
}

func TestScanDockerfile_healthcheckJSON(t *testing.T) {
	p := writeDockerfile(t, "FROM alpine\nHEALTHCHECK CMD [\"curl\", \"-f\", \"http://127.0.0.1/\"]\n")
	got, err := ScanDockerfile(p)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"curl", "-f", "http://127.0.0.1/"}
	if !reflect.DeepEqual(got.Health, want) {
		t.Fatalf("health=%v want %v", got.Health, want)
	}
}

func TestScanDockerfile_lineContinuation(t *testing.T) {
	p := writeDockerfile(t, "FROM alpine\nEXPOSE \\\n  80 \\\n  443\n")
	got, err := ScanDockerfile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Expose, []string{"80", "443"}) {
		t.Fatalf("expose=%v", got.Expose)
	}
}

func writeDockerfile(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "Dockerfile")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}
