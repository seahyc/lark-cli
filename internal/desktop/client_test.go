package desktop

import (
	"runtime"
	"testing"
)

func TestDesktopTargetIsolation(t *testing.T) {
	tests := []struct {
		url  string
		want bool
		goos string
	}{
		{"file:///Applications/LarkSuite.app/Contents/Frameworks/Lark%20Framework.framework/Versions/147/Resources/webcontent/mail/mail/en-US.html", true, "darwin"},
		{"file:///Applications/LarkSuite.app/Contents/Resources/webcontent/mail/AutoFilterDialog/en-US.html", true, "darwin"},
		{"file:///opt/bytedance/lark/webcontent/mail/mail/en-US.html", true, "linux"},
		{"file:///opt/bytedance/lark/webcontent/mail/AutoFilterDialog/en-US.html", true, "linux"},
		{"https://example.com/webcontent/mail/mail/en-US.html", false, ""},
		{"file:///tmp/webcontent/mail/mail/en-US.html", false, ""},
		{"file:///Applications/LarkSuite.app/Contents/Resources/webcontent/messenger/en-US.html", false, "darwin"},
		{"file:///opt/bytedance/lark/webcontent/messenger/en-US.html", false, "linux"},
	}
	
	for _, tc := range tests {
		if tc.goos != "" && tc.goos != runtime.GOOS {
			continue
		}
		if got := IsMailTarget(tc.url); got != tc.want {
			t.Errorf("target %s (goos=%s): got %v, want %v", tc.url, runtime.GOOS, got, tc.want)
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
