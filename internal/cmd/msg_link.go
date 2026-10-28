package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/config"
)

type messageLinkTarget struct{ ChatID, Position string }

func larkMessageURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || (u.Hostname() != "applink.larksuite.com" && u.Hostname() != "applink.feishu.cn") || u.Path != "/client/message/link/open" || u.Query().Get("token") == "" {
		return nil, fmt.Errorf("expected an HTTPS Lark/Feishu message link with a token")
	}
	return u, nil
}

func messageTargetFromJump(raw string) (messageLinkTarget, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || (u.Hostname() != "applink.larksuite.com" && u.Hostname() != "applink.feishu.cn") || u.Path != "/client/chat/open" {
		return messageLinkTarget{}, fmt.Errorf("resolver returned an unsupported message destination")
	}
	q := u.Query()
	// A thread position belongs to a different container; never match it against chat positions.
	if q.Get("threadId") != "" || q.Get("threadPosition") != "" {
		return messageLinkTarget{}, fmt.Errorf("thread-position links are not supported by this resolver yet")
	}
	chat, pos := q.Get("chatId"), q.Get("position")
	if chat == "" {
		chat = q.Get("openChatId")
	}
	n, e := strconv.ParseInt(pos, 10, 64)
	if e != nil || n < 0 || chat == "" {
		return messageLinkTarget{}, fmt.Errorf("resolver returned no exact chat/message position")
	}
	return messageLinkTarget{chat, pos}, nil
}

var msgLinkBrowser, msgLinkWorkspaceURL string

func init() {
	msgCmd.PersistentFlags().StringVar(&msgLinkBrowser, "browser", "", "Browser for message URLs (or LARK_MSG_LINK_BROWSER; defaults to the system default browser)")
	msgCmd.PersistentFlags().StringVar(&msgLinkWorkspaceURL, "workspace-url", "", "Lark workspace URL for background resolution without an existing Messenger tab (or LARK_MSG_LINK_WORKSPACE_URL)")
}

func firstLinkSetting(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func supportedLinkBrowser(browser string) bool {
	switch browser {
	case "chrome", "edge", "brave", "chromium", "arc", "dia":
		return true
	}
	return false
}

func defaultBrowserKey(identifier string) (string, error) {
	id := strings.ToLower(strings.TrimSpace(identifier))
	names := map[string]string{
		"company.thebrowser.dia": "dia", "com.google.chrome": "chrome", "com.microsoft.edgemac": "edge", "com.brave.browser": "brave", "org.chromium.chromium": "chromium", "company.thebrowser.browser": "arc", "com.apple.safari": "safari", "org.mozilla.firefox": "firefox",
		"google-chrome.desktop": "chrome", "google-chrome-stable.desktop": "chrome", "chromium.desktop": "chromium", "chromium-browser.desktop": "chromium", "microsoft-edge.desktop": "edge", "microsoft-edge-stable.desktop": "edge", "brave-browser.desktop": "brave", "firefox.desktop": "firefox",
	}
	if key, ok := names[id]; ok {
		return key, nil
	}
	for prefix, key := range map[string]string{"chromehtml": "chrome", "msedgehtm": "edge", "bravehtml": "brave", "firefoxurl": "firefox"} {
		if strings.HasPrefix(id, prefix) {
			return key, nil
		}
	}
	return "", fmt.Errorf("system default browser %q is not recognized by the resolver; select --browser explicitly", identifier)
}

func macDefaultBrowser(data []byte) (string, error) {
	var prefs struct {
		Handlers []struct {
			Scheme string `json:"LSHandlerURLScheme"`
			App    string `json:"LSHandlerRoleAll"`
		} `json:"LSHandlers"`
	}
	if e := json.Unmarshal(data, &prefs); e != nil {
		return "", fmt.Errorf("cannot read system default browser preferences")
	}
	for _, scheme := range []string{"https", "http"} {
		for _, h := range prefs.Handlers {
			if h.Scheme == scheme && h.App != "" {
				return defaultBrowserKey(h.App)
			}
		}
	}
	// Safari is macOS's built-in HTTPS handler when no user override exists.
	return "safari", nil
}

func systemDefaultLinkBrowser(ctx context.Context) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		plist, e := exec.CommandContext(ctx, "/usr/bin/defaults", "export", "com.apple.LaunchServices/com.apple.launchservices.secure", "-").Output()
		if e != nil {
			return "", fmt.Errorf("cannot read macOS default browser; select --browser explicitly")
		}
		command := exec.CommandContext(ctx, "/usr/bin/plutil", "-convert", "json", "-o", "-", "-")
		command.Stdin = bytes.NewReader(plist)
		data, e := command.Output()
		if e != nil {
			return "", fmt.Errorf("cannot decode macOS default browser preferences")
		}
		return macDefaultBrowser(data)
	case "linux":
		b, e := exec.CommandContext(ctx, "xdg-settings", "get", "default-web-browser").Output()
		if e != nil {
			return "", fmt.Errorf("cannot read Linux default browser; select --browser explicitly")
		}
		return defaultBrowserKey(string(b))
	case "windows":
		b, e := exec.CommandContext(ctx, "reg", "query", `HKCU\Software\Microsoft\Windows\Shell\Associations\UrlAssociations\https\UserChoice`, "/v", "ProgId").Output()
		if e != nil {
			return "", fmt.Errorf("cannot read Windows default browser; select --browser explicitly")
		}
		fields := strings.Fields(string(b))
		for i, f := range fields {
			if f == "REG_SZ" && i+1 < len(fields) {
				return defaultBrowserKey(fields[i+1])
			}
		}
	}
	return "", fmt.Errorf("cannot detect system default browser on this platform; select --browser explicitly")
}

func linkWorkspaceOrigin(raw string) (string, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || !(strings.HasSuffix(u.Hostname(), ".larksuite.com") || strings.HasSuffix(u.Hostname(), ".feishu.cn")) || (u.Path != "" && u.Path != "/" && u.Path != "/messenger/" && u.Path != "/next/messenger/") || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("workspace URL must be an HTTPS Lark/Feishu workspace origin or Messenger URL")
	}
	return u.Scheme + "://" + u.Host + "/messenger/", nil
}

// evalInMessengerTab opens a dedicated background Messenger tab in the selected
// browser, evaluates the script built for the workspace origin, and always closes
// the tab. It never focuses or raises the tab. The script must return a string.
func evalInMessengerTab(ctx context.Context, buildScript func(origin string) string) ([]byte, error) {
	var err error
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	agent := fmt.Sprintf("lark-msg-link-%d", os.Getpid())
	browser := firstLinkSetting(msgLinkBrowser, os.Getenv("LARK_MSG_LINK_BROWSER"), config.Get().MessageLinks.Browser, os.Getenv("BROWSERCTRL_BROWSER"))
	if browser == "" {
		browser, err = systemDefaultLinkBrowser(ctx)
	}
	if err != nil {
		return nil, err
	}
	if !supportedLinkBrowser(browser) {
		return nil, fmt.Errorf("default/selected browser %q is not supported for background resolution; use --browser with a browserctrl-enabled Chromium browser", browser)
	}
	run := func(args ...string) ([]byte, error) {
		args = append([]string{"--browser", browser, "--agent-id", agent}, args...)
		c := exec.CommandContext(ctx, "browserctrl", args...)
		b, e := c.Output()
		if e != nil {
			return nil, fmt.Errorf("background browser resolver failed; ensure browserctrl can access the selected browser and its signed-in Lark Messenger session")
		}
		return b, nil
	}
	origin := ""
	workspace := firstLinkSetting(msgLinkWorkspaceURL, os.Getenv("LARK_MSG_LINK_WORKSPACE_URL"), config.Get().MessageLinks.WorkspaceURL)
	if workspace != "" {
		origin, err = linkWorkspaceOrigin(workspace)
		if err != nil {
			return nil, err
		}
	} else {
		tabs, e := run("tabs", "--json")
		if e != nil {
			return nil, e
		}
		var inventory []struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(tabs, &inventory) != nil {
			return nil, fmt.Errorf("cannot read background browser inventory")
		}
		origins := map[string]bool{}
		for _, tab := range inventory {
			t, e := url.Parse(tab.URL)
			if e != nil || (t.Path != "/messenger/" && t.Path != "/next/messenger/") {
				continue
			}
			candidate, e := linkWorkspaceOrigin(tab.URL)
			if e == nil {
				origins[candidate] = true
			}
		}
		if len(origins) > 1 {
			return nil, fmt.Errorf("multiple Lark workspaces are open; select --workspace-url")
		}
		for candidate := range origins {
			origin = candidate
		}
	}
	if origin == "" {
		return nil, fmt.Errorf("set --workspace-url or LARK_MSG_LINK_WORKSPACE_URL, or open a signed-in web Messenger tab in the selected browser")
	}
	b, err := run("open", origin, "--id")
	if err != nil {
		return nil, err
	}
	tab := strings.TrimSpace(string(b))
	if len(tab) != 8 || strings.Trim(tab, "0123456789ABCDEF") != "" {
		return nil, fmt.Errorf("cannot identify resolver background tab")
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_ = exec.CommandContext(cleanup, "browserctrl", "--browser", browser, "--agent-id", agent, "close", tab).Run()
	}()
	return run("eval", tab, buildScript(origin))
}

// messengerGuardJS verifies the tab is a signed-in Messenger page of the expected
// workspace and waits for the in-page transport. It evaluates to a statement list.
func messengerGuardJS(origin string) string {
	expectedOrigin, _ := json.Marshal(strings.TrimSuffix(origin, "/messenger/"))
	return `if(location.origin!==` + string(expectedOrigin) + `||!/^\/(next\/)?messenger\/?$/.test(location.pathname))throw Error("Lark Messenger session unavailable");for(let i=0;i<100&&!window.LarkAPI?.transport?.callServerApi;i++)await new Promise(r=>setTimeout(r,100));if(!window.LarkAPI?.transport?.callServerApi)throw Error("Lark session unavailable");`
}

// resolveMessageURL uses only a read operation in a dedicated background tab.
// The session stays in the browser; only the canonical jump URL is returned.
func resolveMessageURL(ctx context.Context, raw string) (messageLinkTarget, error) {
	u, err := larkMessageURL(raw)
	if err != nil {
		return messageLinkTarget{}, err
	}
	token, _ := json.Marshal(u.Query().Get("token"))
	b, err := evalInMessengerTab(ctx, func(origin string) string {
		return `(async()=>{` + messengerGuardJS(origin) + `const r=await LarkAPI.transport.callServerApi("1151|messages.GetMessageLinkPermissionRequest|messages.GetMessageLinkPermissionResponse",{token:` + string(token) + `});return JSON.stringify({permission:r.data?.permission,url:r.data?.applinkUrl});})()`
	})
	if err != nil {
		return messageLinkTarget{}, err
	}
	var response struct {
		Permission int    `json:"permission"`
		URL        string `json:"url"`
	}
	if json.Unmarshal(b, &response) != nil || response.Permission != 1 || response.URL == "" {
		return messageLinkTarget{}, fmt.Errorf("Lark did not grant access to the linked message")
	}
	return messageTargetFromJump(response.URL)
}

type linkedMessageReader interface {
	ListMessagesAsUser(string, string, *api.ListMessagesOptions) ([]api.Message, bool, string, error)
}

func findLinkedMessage(client linkedMessageReader, target messageLinkTarget) (*api.Message, error) {
	page := ""
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		messages, more, next, err := client.ListMessagesAsUser("chat", target.ChatID, &api.ListMessagesOptions{PageSize: 50, PageToken: page, SortType: "ByCreateTimeDesc"})
		if err != nil {
			return nil, err
		}
		for _, m := range messages {
			if m.MessagePosition == target.Position {
				result := m
				return &result, nil
			}
		}
		if !more {
			return nil, fmt.Errorf("linked message is unavailable in accessible chat history")
		}
		if next == "" || seen[next] {
			return nil, fmt.Errorf("message resolution stopped: history pagination did not advance")
		}
		seen[next] = true
		page = next
	}
	return nil, fmt.Errorf("linked message was not found within the 10,000-message lookup limit")
}

func messageFromURL(ctx context.Context, client linkedMessageReader, raw string) (*api.Message, error) {
	target, err := resolveMessageURL(ctx, raw)
	if err != nil {
		return nil, err
	}
	return findLinkedMessage(client, target)
}
