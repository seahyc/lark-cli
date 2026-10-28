package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/yjwong/lark-cli/internal/config"
)

// isolateConfig points the config at a fresh temp dir and restores a clean state afterwards.
func isolateConfig(t *testing.T) string {
	t.Helper()
	// Registered before Setenv so it runs after the env is restored (LIFO).
	t.Cleanup(func() { viper.Reset() })
	dir := filepath.Join(t.TempDir(), ".lark")
	t.Setenv("LARK_CONFIG_DIR", dir)
	viper.Reset() // viper accumulates config paths across Init calls
	if err := config.Init(); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestConfigSetGet(t *testing.T) {
	cases := []struct {
		name, key, value, want string
		wantErr                bool
	}{
		{"workspace origin", "message_links.workspace_url", "https://team.larksuite.com", "https://team.larksuite.com/messenger/", false},
		{"workspace messenger url", "message_links.workspace_url", "https://team.larksuite.com/messenger/", "https://team.larksuite.com/messenger/", false},
		{"workspace bad host", "message_links.workspace_url", "https://example.com", "", true},
		{"workspace service host", "message_links.workspace_url", "https://open.larksuite.com", "", true},
		{"workspace empty label", "message_links.workspace_url", "https://a..larksuite.com", "", true},
		{"workspace userinfo", "message_links.workspace_url", "https://u@team.larksuite.com", "", true},
		{"workspace uppercase", "message_links.workspace_url", "https://Team.LarkSuite.com", "https://team.larksuite.com/messenger/", false},
		{"workspace http", "message_links.workspace_url", "http://team.larksuite.com", "", true},
		{"browser ok", "message_links.browser", " Dia ", "dia", false},
		{"browser unsupported", "message_links.browser", "safari", "", true},
		{"unknown key", "app_secret", "x", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := isolateConfig(t)
			got, err := setConfigValue(c.key, c.value)
			if (err != nil) != c.wantErr || got != c.want {
				t.Fatalf("got %q, %v", got, err)
			}
			if c.wantErr {
				if _, e := os.Stat(filepath.Join(dir, "config.yaml")); e == nil {
					t.Fatal("config written despite error")
				}
				return
			}
			if v, _ := getConfigValue(c.key); v != c.want {
				t.Fatalf("get = %q", v)
			}
			viper.Reset()
			if err := config.Init(); err != nil { // persisted across re-init
				t.Fatal(err)
			}
			if v, _ := getConfigValue(c.key); v != c.want {
				t.Fatalf("after reinit get = %q", v)
			}
		})
	}
}

func TestSetFileValueIfUnsetDoesNotOverwrite(t *testing.T) {
	isolateConfig(t)
	if ok, err := config.SetFileValueIfUnset("message_links.workspace_url", "https://a.larksuite.com/messenger/"); !ok || err != nil {
		t.Fatalf("first: %v %v", ok, err)
	}
	if ok, err := config.SetFileValueIfUnset("message_links.workspace_url", "https://b.larksuite.com/messenger/"); ok || err != nil {
		t.Fatalf("second: %v %v", ok, err)
	}
	if v, _ := config.GetFileValue("message_links.workspace_url"); v != "https://a.larksuite.com/messenger/" {
		t.Fatalf("overwritten: %q", v)
	}
	if _, err := setConfigValue("message_links.browser", "dia"); err != nil {
		t.Fatal(err)
	}
	if v, _ := config.GetFileValue("message_links.workspace_url"); v == "" {
		t.Fatal("other keys lost")
	}
}

func TestSetFileValuePreservesContent(t *testing.T) {
	dir := isolateConfig(t)
	path := filepath.Join(dir, "config.yaml")
	orig := "# my config\napp_id: cli_abc\napp_secret: s3cret # keep\ncustom_emojis:\n  MixedCase: \"x\"\n  other: y\n"
	if err := os.WriteFile(path, []byte(orig), 0644); err != nil {
		t.Fatal(err)
	}
	if err := config.SetFileValue("message_links.workspace_url", "https://a.larksuite.com/messenger/"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	got := string(b)
	for _, want := range []string{"# my config", "app_id: cli_abc", "app_secret: s3cret # keep", "MixedCase:", "other: y", "message_links:", "workspace_url: https://a.larksuite.com/messenger/"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0600 {
		t.Fatalf("mode after rewrite %v", st.Mode().Perm())
	}
	if v, _ := config.GetFileValue("message_links.workspace_url"); v == "" {
		t.Fatal("value not readable")
	}
	if e := config.SetFileValue("message_links.browser", "dia"); e != nil {
		t.Fatal(e)
	}
	if v, _ := config.GetFileValue("message_links.workspace_url"); v == "" {
		t.Fatal("sibling lost")
	}
	if e := config.SetFileValue("message_links.workspace_url", "https://b.larksuite.com/messenger/"); e != nil {
		t.Fatal(e)
	}
	b, _ = os.ReadFile(path)
	if strings.Count(string(b), "workspace_url") != 1 {
		t.Fatalf("duplicate key:\n%s", b)
	}
}

func TestSetFileValueCreateMode(t *testing.T) {
	dir := isolateConfig(t)
	if err := config.SetFileValue("message_links.browser", "dia"); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(filepath.Join(dir, "config.yaml"))
	if err != nil || st.Mode().Perm() != 0600 {
		t.Fatalf("%v %v", st, err)
	}
	if ents, _ := os.ReadDir(dir); len(ents) > 2 {
		for _, e := range ents {
			if strings.HasSuffix(e.Name(), ".tmp") {
				t.Fatalf("leftover temp file %s", e.Name())
			}
		}
	}
}

func TestSetFileValueMalformedAborts(t *testing.T) {
	dir := isolateConfig(t)
	path := filepath.Join(dir, "config.yaml")
	orig := []byte("app_id: [unterminated\n  bad: : :\n")
	if err := os.WriteFile(path, orig, 0600); err != nil {
		t.Fatal(err)
	}
	if err := config.SetFileValue("message_links.browser", "dia"); err == nil {
		t.Fatal("expected error")
	}
	if b, _ := os.ReadFile(path); !bytes.Equal(b, orig) {
		t.Fatalf("file changed: %q", b)
	}
}

func TestSetFileValueRefusesShadowedConfig(t *testing.T) {
	for _, name := range []string{"config.yml", "config.json", "config.toml"} {
		t.Run(name, func(t *testing.T) {
			dir := isolateConfig(t)
			if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0600); err != nil {
				t.Fatal(err)
			}
			if err := config.SetFileValue("message_links.browser", "dia"); err == nil {
				t.Fatal("expected error")
			}
			if _, e := os.Stat(filepath.Join(dir, "config.yaml")); e == nil {
				t.Fatal("config.yaml created")
			}
		})
	}
}

func TestSetFileValueConcurrent(t *testing.T) {
	isolateConfig(t)
	var wg sync.WaitGroup
	for k, v := range map[string]string{"message_links.browser": "dia", "message_links.workspace_url": "https://a.larksuite.com/messenger/"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := config.SetFileValue(k, v); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	for k, want := range map[string]string{"message_links.browser": "dia", "message_links.workspace_url": "https://a.larksuite.com/messenger/"} {
		if got, err := config.GetFileValue(k); err != nil || got != want {
			t.Fatalf("%s = %q, %v", k, got, err)
		}
	}
}
