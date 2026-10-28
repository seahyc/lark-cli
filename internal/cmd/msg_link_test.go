package cmd

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/yjwong/lark-cli/internal/api"
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
