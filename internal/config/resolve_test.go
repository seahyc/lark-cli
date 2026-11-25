package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestResolveConfigDir(t *testing.T) {
	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	if err := os.MkdirAll(filepath.Join(real, ".lark"), 0700); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(real, "lark")
	if err := os.WriteFile(exe, nil, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(tmp, "bin", "lark")
	if err := os.MkdirAll(filepath.Dir(link), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(exe, link); err != nil {
		t.Fatal(err)
	}
	exeFn := func(p string) func() (string, error) { return func() (string, error) { return p, nil } }
	wantFallback, _ := filepath.EvalSymlinks(filepath.Join(real, ".lark"))

	t.Run("env wins", func(t *testing.T) {
		d, _, err := ResolveConfigDir(envMap(map[string]string{"LARK_CONFIG_DIR": "/x"}), exeFn(link))
		if err != nil || d != "/x" {
			t.Fatalf("got %q %v", d, err)
		}
	})
	t.Run("legacy env", func(t *testing.T) {
		d, _, err := ResolveConfigDir(envMap(map[string]string{"LARK_CAL_CONFIG_DIR": "/y"}), exeFn(link))
		if err != nil || d != "/y" {
			t.Fatalf("got %q %v", d, err)
		}
	})
	t.Run("fallback via symlink", func(t *testing.T) {
		d, _, err := ResolveConfigDir(envMap(nil), exeFn(link))
		if err != nil {
			t.Fatal(err)
		}
		got, _ := filepath.EvalSymlinks(d)
		if got != wantFallback {
			t.Fatalf("got %q want %q", got, wantFallback)
		}
	})
	t.Run("error names fallback", func(t *testing.T) {
		other := filepath.Join(tmp, "none", "lark")
		_, tried, err := ResolveConfigDir(envMap(nil), exeFn(other))
		if err == nil || !strings.Contains(err.Error(), filepath.Join(tmp, "none", ".lark")) || tried == "" {
			t.Fatalf("err=%v tried=%q", err, tried)
		}
	})
}
