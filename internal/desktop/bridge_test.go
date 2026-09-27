package desktop

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func freeBridgePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return p
}

func peer(t *testing.T, port int, token, origin string, response any) {
	t.Helper()
	go func() {
		for i := 0; i < 100; i++ {
			h := http.Header{}
			h.Set("Origin", origin)
			c, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:"+itoa(port)+"/bridge?token="+token, h)
			if err == nil {
				defer c.Close()
				_, _, _ = c.ReadMessage()
				_ = c.WriteJSON(response)
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Errorf("peer could not connect")
	}()
}

func itoa(v int) string { return fmt.Sprintf("%d", v) }

func TestBridgeCallSuccessAndAllowlist(t *testing.T) {
	port, token := freeBridgePort(t), "test-token"
	peer(t, port, token, "null", map[string]any{"id": "1", "ok": true, "data": map[string]any{"ok": true}})
	var out map[string]bool
	if err := BridgeCall(context.Background(), port, token, "identity", nil, &out); err != nil {
		t.Fatal(err)
	}
	if !out["ok"] {
		t.Fatalf("unexpected output: %#v", out)
	}
	if err := BridgeCall(context.Background(), freeBridgePort(t), token, "deleteRule", nil, nil); err == nil {
		t.Fatal("allowlist accepted deleteRule")
	}
}

func TestBridgeCallCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	port, token := freeBridgePort(t), "cancel-token"
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	if err := BridgeCall(ctx, port, token, "listRules", nil, nil); err != context.Canceled {
		t.Fatalf("got %v", err)
	}
}

func TestBridgeRejectsBadOrigin(t *testing.T) {
	port, token := freeBridgePort(t), "origin-token"
	go func() {
		for i := 0; i < 100; i++ {
			h := http.Header{"Origin": []string{"https://evil.example"}}
			_, resp, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:"+itoa(port)+"/bridge?token="+token, h)
			if err == nil {
				t.Errorf("bad origin connected")
				return
			}
			if resp != nil {
				_ = resp.Body.Close()
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := BridgeCall(ctx, port, token, "identity", nil, nil); err == nil {
		t.Fatal("expected timeout")
	}
}

func TestBridgeCallTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := BridgeCall(ctx, freeBridgePort(t), "timeout-token", "listRules", nil, nil); err == nil {
		t.Fatal("expected timeout")
	}
}
