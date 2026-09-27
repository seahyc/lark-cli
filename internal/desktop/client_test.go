package desktop

import "testing"

func TestDesktopTargetIsolation(t *testing.T) {
	for _, tc := range []struct {
		url  string
		want bool
	}{
		{"file:///Applications/LarkSuite.app/Contents/Frameworks/Lark%20Framework.framework/Versions/147/Resources/webcontent/mail/mail/en-US.html", true},
		{"file:///Applications/LarkSuite.app/Contents/Resources/webcontent/mail/AutoFilterDialog/en-US.html", true},
		{"https://example.com/webcontent/mail/mail/en-US.html", false},
		{"file:///tmp/webcontent/mail/mail/en-US.html", false},
		{"file:///Applications/LarkSuite.app/Contents/Resources/webcontent/messenger/en-US.html", false},
	} {
		if got := isMailTarget(Target{URL: tc.url}); got != tc.want {
			t.Errorf("target %s: %v", tc.url, got)
		}
	}
	for _, raw := range []string{"ws://example.com:9330/x", "ws://127.0.0.1:9331/x", "ws://user@127.0.0.1:9330/x", "wss://127.0.0.1:9330/x"} {
		if localSocket(raw, 9330) {
			t.Errorf("accepted %s", raw)
		}
	}
	if !localSocket("ws://127.0.0.1:9330/devtools/page/123", 9330) {
		t.Fatal("rejected local socket")
	}
}
