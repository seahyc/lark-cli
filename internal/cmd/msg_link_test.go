package cmd

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/config"
)

func TestMessageLinkURLValidation(t *testing.T) {
	for _, s := range []string{"https://applink.larksuite.com/client/message/link/open?token=abc%3D", "https://applink.feishu.cn/client/message/link/open?token=abc"} {
		if _, e := larkMessageURL(s); e != nil {
			t.Fatal(e)
		}
	}
	for _, s := range []string{"http://applink.larksuite.com/client/message/link/open?token=abc", "https://applink.larksuite.com.evil.test/client/message/link/open?token=abc", "https://user@applink.larksuite.com/client/message/link/open?token=abc", "https://applink.larksuite.com/client/message/link/open", "https://applink.larksuite.com/client/chat/open?token=abc"} {
		if _, e := larkMessageURL(s); e == nil {
			t.Errorf("accepted %q", s)
		}
	}
}
func TestMessageTargetRequiresExactChatPosition(t *testing.T) {
	got, e := messageTargetFromJump("https://applink.larksuite.com/client/chat/open?chatId=123&position=1128")
	if e != nil || got.ChatID != "123" || got.Position != "1128" {
		t.Fatalf("%+v %v", got, e)
	}
	for _, q := range []string{"chatId=123", "chatId=123&position=-1", "chatId=123&position=nope", "chatId=123&position=3&threadId=456"} {
		if _, e := messageTargetFromJump("https://applink.larksuite.com/client/chat/open?" + q); e == nil {
			t.Errorf("accepted %q", q)
		}
	}
}

type linkPages struct {
	calls  int
	repeat bool
}

func (p *linkPages) ListMessagesAsUser(kind, id string, opts *api.ListMessagesOptions) ([]api.Message, bool, string, error) {
	p.calls++
	if kind != "chat" || id != "123" || opts.SortType != "ByCreateTimeDesc" {
		panic("unexpected query")
	}
	if p.repeat {
		return nil, true, "same", nil
	}
	if p.calls == 1 {
		return []api.Message{{MessageID: "wrong", MessagePosition: "8"}}, true, "page2", nil
	}
	return []api.Message{{MessageID: "exact", MessagePosition: "7", ThreadID: "omt_exact"}}, false, "", nil
}
func TestFindLinkedMessageUsesPositionAcrossPages(t *testing.T) {
	p := &linkPages{}
	m, e := findLinkedMessage(p, messageLinkTarget{"123", "7"})
	if e != nil || m.MessageID != "exact" || m.ThreadID != "omt_exact" || p.calls != 2 {
		t.Fatalf("%+v %v calls=%d", m, e, p.calls)
	}
	p = &linkPages{}
	if _, e := findLinkedMessage(p, messageLinkTarget{"123", "missing"}); e == nil {
		t.Fatal("silently resolved missing message")
	}
	p = &linkPages{repeat: true}
	if _, e := findLinkedMessage(p, messageLinkTarget{"123", "7"}); e == nil || p.calls != 2 {
		t.Fatal("did not reject repeated cursor")
	}
}

func TestResolveMessageURLBackgroundLifecycle(t *testing.T) {
	cfgDir := isolateConfig(t)
	dir := t.TempDir()
	log := dir + "/calls"
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$LARK_TEST_CALLS"
case "$5" in
 tabs) echo '[{"url":"https://example.larksuite.com/next/messenger/"}]';;
 open) echo ABCD1234;;
 eval) echo '{"permission":1,"url":"https://applink.larksuite.com/client/chat/open?chatId=123&position=7"}';;
 close) :;;
 *) exit 1;;
esac
`
	if err := os.WriteFile(dir+"/browserctrl", []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("LARK_TEST_CALLS", log)
	t.Setenv("LARK_MSG_LINK_BROWSER", "chrome")
	target, err := resolveMessageURL(context.Background(), "https://applink.larksuite.com/client/message/link/open?token=test")
	if err != nil || target.ChatID != "123" || target.Position != "7" {
		t.Fatalf("%+v %v", target, err)
	}
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	calls := string(b)
	for _, want := range []string{"--browser chrome", " open https://example.larksuite.com/messenger/ --id", " eval ABCD1234", " close ABCD1234"} {
		if !strings.Contains(calls, want) {
			t.Errorf("missing %q", want)
		}
	}
	if cached, _ := config.GetFileValue("message_links.workspace_url"); cached != "https://example.larksuite.com/messenger/" {
		t.Errorf("discovered workspace not cached in %s: %q", cfgDir, cached)
	}
	if strings.Contains(calls, " focus ") || strings.Contains(calls, "--steal") {
		t.Fatal("resolver must not activate or steal a tab")
	}
}

func TestSystemDefaultBrowserMapping(t *testing.T) {
	for id, want := range map[string]string{"company.thebrowser.dia": "dia", "com.google.Chrome": "chrome", "microsoft-edge.desktop": "edge", "ChromeHTML.ABC": "chrome", "FirefoxURL-123": "firefox", "com.apple.Safari": "safari"} {
		got, e := defaultBrowserKey(id)
		if e != nil || got != want {
			t.Fatalf("%s: %s %v", id, got, e)
		}
	}
	if _, e := defaultBrowserKey("unknown"); e == nil {
		t.Fatal("must not guess an unknown default")
	}
	got, e := macDefaultBrowser([]byte(`{"LSHandlers":[{"LSHandlerURLScheme":"http","LSHandlerRoleAll":"com.google.Chrome"},{"LSHandlerURLScheme":"https","LSHandlerRoleAll":"company.thebrowser.dia"}]}`))
	if e != nil || got != "dia" {
		t.Fatalf("%s %v", got, e)
	}
	got, e = macDefaultBrowser([]byte(`{"LSHandlers":[]}`))
	if e != nil || got != "safari" {
		t.Fatalf("%s %v", got, e)
	}
}

func TestMessageLinkWorkspace(t *testing.T) {
	for _, raw := range []string{"https://team.larksuite.com", "https://team.feishu.cn/", "https://team.larksuite.com/next/messenger/"} {
		if _, e := linkWorkspaceOrigin(raw); e != nil {
			t.Fatal(e)
		}
	}
	for _, raw := range []string{"https://evil.test/", "https://team.larksuite.com.evil.test/", "http://team.larksuite.com/", "https://user@team.larksuite.com/", "https://team.larksuite.com/docx/123"} {
		if _, e := linkWorkspaceOrigin(raw); e == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}

func TestResolveMessageURLWithoutExistingTab(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
case "$5" in
 open) echo ABCD1234;;
 eval) echo '{"permission":1,"url":"https://applink.larksuite.com/client/chat/open?chatId=123&position=7"}';;
 close) :;;
 *) exit 1;;
esac
`
	if e := os.WriteFile(dir+"/browserctrl", []byte(script), 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("LARK_MSG_LINK_BROWSER", "edge")
	t.Setenv("LARK_MSG_LINK_WORKSPACE_URL", "https://team.larksuite.com")
	got, e := resolveMessageURL(context.Background(), "https://applink.larksuite.com/client/message/link/open?token=test")
	if e != nil || got.ChatID != "123" {
		t.Fatalf("%+v %v", got, e)
	}
}

func TestDiscoverWorkspaceOrigin(t *testing.T) {
	cases := []struct {
		name    string
		urls    []string
		want    string
		wantErr bool
	}{
		{"docx tab", []string{"https://glints.sg.larksuite.com/docx/abc"}, "https://glints.sg.larksuite.com/messenger/", false},
		{"sheets tab among others", []string{"https://example.com/", "https://glints.sg.larksuite.com/sheets/x?sheet=1"}, "https://glints.sg.larksuite.com/messenger/", false},
		{"messenger tab", []string{"https://team.larksuite.com/next/messenger/"}, "https://team.larksuite.com/messenger/", false},
		{"feishu", []string{"https://team.feishu.cn/wiki/abc"}, "https://team.feishu.cn/messenger/", false},
		{"same workspace twice", []string{"https://a.larksuite.com/docx/1", "https://a.larksuite.com/sheets/2"}, "https://a.larksuite.com/messenger/", false},
		{"open host ignored", []string{"https://open.larksuite.com/document/x"}, "", false},
		{"applink ignored", []string{"https://applink.larksuite.com/client/message/link/open?token=a"}, "", false},
		{"www accounts passport ignored", []string{"https://www.larksuite.com/", "https://accounts.larksuite.com/x", "https://passport.feishu.cn/"}, "", false},
		{"bare apex ignored", []string{"https://larksuite.com/"}, "", false},
		{"share link ignored", []string{"https://a.larksuite.com/share/base/abc"}, "", false},
		{"http ignored", []string{"http://a.larksuite.com/docx/1"}, "", false},
		{"lookalike ignored", []string{"https://a.larksuite.com.evil.test/docx/1"}, "", false},
		{"port ignored", []string{"https://a.larksuite.com:8443/docx/1"}, "", false},
		{"no tabs", nil, "", false},
		{"multiple workspaces", []string{"https://a.larksuite.com/docx/1", "https://b.larksuite.com/docx/2"}, "", true},
		{"messenger wins over guest doc", []string{"https://home.larksuite.com/next/messenger/", "https://guest.larksuite.com/docx/1"}, "https://home.larksuite.com/messenger/", false},
		{"two messenger workspaces", []string{"https://a.larksuite.com/messenger/", "https://b.larksuite.com/messenger/"}, "", true},
		{"uppercase host lowercased", []string{"https://Team.LarkSuite.com/docx/1"}, "https://team.larksuite.com/messenger/", false},
		{"empty label ignored", []string{"https://a..larksuite.com/docx/1", "https://.larksuite.com/"}, "", false},
		{"vc and meetings ignored", []string{"https://vc.larksuite.com/x", "https://meetings.larksuite.com/y"}, "", false},
		{"ignored host does not cause ambiguity", []string{"https://a.larksuite.com/docx/1", "https://open.larksuite.com/"}, "https://a.larksuite.com/messenger/", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _, err := discoverWorkspaceOrigin(c.urls)
			if (err != nil) != c.wantErr || got != c.want {
				t.Fatalf("got %q, %v; want %q (err=%v)", got, err, c.want, c.wantErr)
			}
		})
	}
}

func TestDiscoverWorkspaceOriginMessengerFlag(t *testing.T) {
	if _, m, _ := discoverWorkspaceOrigin([]string{"https://a.larksuite.com/docx/1"}); m {
		t.Fatal("docx-only discovery must not be marked as messenger")
	}
	if _, m, _ := discoverWorkspaceOrigin([]string{"https://a.larksuite.com/docx/1", "https://a.larksuite.com/messenger/"}); !m {
		t.Fatal("messenger tab must be marked as messenger")
	}
}

func TestLinkWorkspaceOriginValidation(t *testing.T) {
	if got, e := linkWorkspaceOrigin("https://Team.LarkSuite.com"); e != nil || got != "https://team.larksuite.com/messenger/" {
		t.Fatalf("%q %v", got, e)
	}
	for _, raw := range []string{"https://.larksuite.com", "https://a..larksuite.com", "https://open.larksuite.com", "https://vc.larksuite.com", "https://u:p@team.larksuite.com", "https://larksuite.com", "https://team.larksuite.com:444"} {
		if _, e := linkWorkspaceOrigin(raw); e == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}

// fakeBrowserctrl installs a browserctrl stub reporting the given tab URLs.
func fakeBrowserctrl(t *testing.T, tabs string) {
	t.Helper()
	dir := t.TempDir()
	script := `#!/bin/sh
case "$5" in
 tabs) echo '` + tabs + `';;
 open) echo ABCD1234;;
 eval) echo '{"permission":1,"url":"https://applink.larksuite.com/client/chat/open?chatId=123&position=7"}';;
 close) :;;
 *) exit 1;;
esac
`
	if e := os.WriteFile(dir+"/browserctrl", []byte(script), 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("LARK_MSG_LINK_BROWSER", "chrome")
}

const testMsgLink = "https://applink.larksuite.com/client/message/link/open?token=test"

func TestResolveDiscoveryPersistence(t *testing.T) {
	t.Run("non-messenger only is used but not persisted", func(t *testing.T) {
		isolateConfig(t)
		fakeBrowserctrl(t, `[{"url":"https://guest.larksuite.com/docx/abc"}]`)
		if got, e := resolveMessageURL(context.Background(), testMsgLink); e != nil || got.ChatID != "123" {
			t.Fatalf("%+v %v", got, e)
		}
		if v, _ := config.GetFileValue("message_links.workspace_url"); v != "" {
			t.Fatalf("persisted %q", v)
		}
		if _, e := os.Stat(config.ConfigFilePath()); e == nil {
			t.Fatal("config file created")
		}
	})
	t.Run("messenger plus guest doc resolves and persists messenger origin", func(t *testing.T) {
		isolateConfig(t)
		fakeBrowserctrl(t, `[{"url":"https://guest.larksuite.com/docx/abc"},{"url":"https://home.larksuite.com/next/messenger/"}]`)
		if _, e := resolveMessageURL(context.Background(), testMsgLink); e != nil {
			t.Fatal(e)
		}
		if v, _ := config.GetFileValue("message_links.workspace_url"); v != "https://home.larksuite.com/messenger/" {
			t.Fatalf("persisted %q", v)
		}
	})
	for _, name := range []string{"flag", "env", "config"} {
		t.Run(name+" blocks discovery and caching", func(t *testing.T) {
			isolateConfig(t)
			fakeBrowserctrl(t, `[{"url":"https://other.larksuite.com/next/messenger/"}]`)
			old := msgLinkWorkspaceURL
			t.Cleanup(func() { msgLinkWorkspaceURL = old })
			switch name {
			case "flag":
				msgLinkWorkspaceURL = "https://set.larksuite.com"
			case "env":
				t.Setenv("LARK_MSG_LINK_WORKSPACE_URL", "https://set.larksuite.com")
			case "config":
				if e := config.SetFileValue("message_links.workspace_url", "https://set.larksuite.com/messenger/"); e != nil {
					t.Fatal(e)
				}
			}
			if _, e := resolveMessageURL(context.Background(), testMsgLink); e != nil {
				t.Fatal(e)
			}
			want := ""
			if name == "config" {
				want = "https://set.larksuite.com/messenger/"
			}
			if v, _ := config.GetFileValue("message_links.workspace_url"); v != want {
				t.Fatalf("file value %q, want %q", v, want)
			}
		})
	}
}
