package explain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/1eve1Up/podbay/internal/expand"
	"github.com/1eve1Up/podbay/internal/spec"
)

func TestLoadHostSubst_oncePerHelperCall(t *testing.T) {
	n := 0
	orig := loadHostSubstFn
	loadHostSubstFn = func(dir string, files []string) (map[string]string, error) {
		n++
		return orig(dir, files)
	}
	t.Cleanup(func() { loadHostSubstFn = orig })

	dir := t.TempDir()
	body := "project: demo\nservices:\n  web:\n    image: nginx\n"
	if err := os.WriteFile(filepath.Join(dir, "podbay.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	c, path, err := spec.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	m, err := loadHostSubst(c, path)
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("expected subst map")
	}
	if n != 1 {
		t.Fatalf("loadHostSubst calls = %d want 1", n)
	}
	if _, err := expand.LoadHostSubst(dir, c.HostEnvFiles); err != nil {
		t.Fatal(err)
	}
}
