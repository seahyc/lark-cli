// Package desktop calls the signed-in Lark app through its local DevTools socket.
// It never reads, exports, or stores the desktop session's credentials.
package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type Target struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	WebSocketURL string `json:"webSocketDebuggerUrl"`
}

type Client struct {
	conn   *websocket.Conn
	nextID int
	Target Target
}

func localSocket(raw string, port int) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "ws" && u.Hostname() == "127.0.0.1" && u.Port() == strconv.Itoa(port) && u.User == nil
}

func Connect(ctx context.Context, port int) (*Client, error) {
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("desktop port must be between 1 and 65535")
	}
	hc := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://127.0.0.1:%d/json/list", port), nil)
	resp, err := hc.Do(req)
	if err != nil {
		platform, _ := DetectPlatform()
		hint := "start Lark with a temporary --remote-debugging-port=%d and open Email"
		if platform != nil && len(platform.LauncherPaths) > 0 {
			hint = fmt.Sprintf("start Lark with %s --remote-debugging-port=%d and open Email", platform.LauncherPaths[0], port)
		}
		return nil, fmt.Errorf("Lark desktop debugging is unavailable on localhost:%d; %s: %w", port, hint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("desktop target discovery returned HTTP %d", resp.StatusCode)
	}
	var targets []Target
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&targets); err != nil {
		return nil, fmt.Errorf("decode desktop targets: %w", err)
	}
	for _, t := range targets {
		if !IsMailTarget(t.URL) {
			continue
		}
		if !localSocket(t.WebSocketURL, port) {
			return nil, fmt.Errorf("refusing non-local desktop WebSocket")
		}
		dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second, Proxy: nil}
		conn, _, err := dialer.DialContext(ctx, t.WebSocketURL, nil)
		if err != nil {
			return nil, fmt.Errorf("attach to Lark mail: %w", err)
		}
		conn.SetReadLimit(16 << 20)
		return &Client{conn: conn, Target: t}, nil
	}
	return nil, fmt.Errorf("no Lark desktop Email view found on port %d; open Email in the signed-in Lark app", port)
}

func (c *Client) Close() error { return c.conn.Close() }

// Eval executes once. A timeout after dispatch has an uncertain result; callers
// must read current state before deciding whether another write is necessary.
func (c *Client) Eval(ctx context.Context, expression string, out interface{}) error {
	c.nextID++
	deadline := time.Now().Add(20 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	c.conn.SetWriteDeadline(deadline)
	c.conn.SetReadDeadline(deadline)
	err := c.conn.WriteJSON(map[string]interface{}{"id": c.nextID, "method": "Runtime.evaluate", "params": map[string]interface{}{"expression": expression, "awaitPromise": true, "returnByValue": true, "timeout": 18000}})
	if err != nil {
		return fmt.Errorf("desktop dispatch failed (do not automatically retry writes): %w", err)
	}
	for {
		var reply struct {
			ID     int             `json:"id"`
			Error  json.RawMessage `json:"error"`
			Result struct {
				Result struct {
					Value json.RawMessage `json:"value"`
				} `json:"result"`
				Exception json.RawMessage `json:"exceptionDetails"`
			} `json:"result"`
		}
		if err := c.conn.ReadJSON(&reply); err != nil {
			return fmt.Errorf("desktop response unavailable; operation result is uncertain, read back before retrying: %w", err)
		}
		if reply.ID != c.nextID {
			continue
		}
		if len(reply.Error) > 0 || len(reply.Result.Exception) > 0 {
			return fmt.Errorf("desktop bridge rejected the operation; no automatic retry")
		}
		if len(reply.Result.Result.Value) == 0 {
			return fmt.Errorf("desktop bridge returned no value")
		}
		return json.Unmarshal(reply.Result.Result.Value, out)
	}
}
