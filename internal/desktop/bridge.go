package desktop

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const bridgeLimit = 4 << 20

var allowedBridgeOperations = map[string]struct{}{
	"listRules":        {},
	"verifiedEmails":   {},
	"updateRule":       {},
	"updateRuleFields": {},
	"identity":         {},
}

type bridgeRequest struct {
	ID        string `json:"id"`
	Operation string `json:"operation"`
	Params    any    `json:"params"`
}

type bridgeResponse struct {
	ID    string          `json:"id"`
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}

type bridgeResult struct {
	data json.RawMessage
	err  error
}

// BridgeCall performs one authenticated request against a browser patch's
// loopback websocket. It owns the listener and closes it when the call ends.
func BridgeCall(ctx context.Context, port int, token string, operation string, params any, out any) error {
	if _, ok := allowedBridgeOperations[operation]; !ok && !readOperationExists(operation) && !mutationOperationExists(operation) {
		return fmt.Errorf("unsupported bridge operation")
	}
	if port < 1 || port > 65535 || token == "" {
		return fmt.Errorf("invalid bridge configuration")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()

	var mu sync.Mutex
	var peer *websocket.Conn
	peerTaken := false
	result := make(chan bridgeResult, 1)
	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			o := r.Header.Get("Origin")
			return o == "null" || strings.HasPrefix(o, "file://")
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bridge" {
			http.NotFound(w, r)
			return
		}
		provided := r.URL.Query().Get("token")
		if len(provided) != len(token) || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mu.Lock()
		if peerTaken {
			mu.Unlock()
			http.Error(w, "busy", http.StatusConflict)
			return
		}
		peerTaken = true
		mu.Unlock()
		conn, e := upgrader.Upgrade(w, r, nil)
		if e != nil {
			mu.Lock()
			peerTaken = false
			mu.Unlock()
			return
		}
		mu.Lock()
		peer = conn
		mu.Unlock()
		defer func() {
			conn.Close()
			mu.Lock()
			peer = nil
			mu.Unlock()
		}()

		_ = conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
		req, e := json.Marshal(bridgeRequest{ID: "1", Operation: operation, Params: params})
		if e == nil && len(req) <= bridgeLimit {
			e = conn.WriteMessage(websocket.TextMessage, req)
		} else if e == nil {
			e = errors.New("request too large")
		}
		if e != nil {
			result <- bridgeResult{err: fmt.Errorf("bridge request failed")}
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		conn.SetReadLimit(bridgeLimit)
		_, body, e := conn.ReadMessage()
		if e != nil || len(body) > bridgeLimit {
			result <- bridgeResult{err: fmt.Errorf("bridge response failed")}
			return
		}
		var response bridgeResponse
		if json.Unmarshal(body, &response) != nil || response.ID != "1" {
			result <- bridgeResult{err: fmt.Errorf("invalid bridge response")}
			return
		}
		if !response.OK {
			result <- bridgeResult{err: fmt.Errorf("bridge operation failed: %s", safeBridgeError(response.Error))}
			return
		}
		result <- bridgeResult{data: response.Data}
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	go server.Serve(listener)
	go func() {
		<-callCtx.Done()
		listener.Close()
		mu.Lock()
		if peer != nil {
			peer.Close()
		}
		mu.Unlock()
	}()
	select {
	case response := <-result:
		server.Close()
		if response.err != nil {
			return response.err
		}
		if out != nil && len(response.data) > 0 && json.Unmarshal(response.data, out) != nil {
			return fmt.Errorf("invalid bridge data")
		}
		return nil
	case <-callCtx.Done():
		server.Close()
		if errors.Is(callCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("the patched Email page never connected to the bridge; restart Lark and open the Email tab, then try again")
		}
		return callCtx.Err()
	}
}

func safeBridgeError(s string) string {
	if s == "" {
		return "unknown error"
	}
	if len(s) > 240 {
		s = s[:240]
	}
	for _, r := range s {
		if r < 0x20 || r == '\n' || r == '\r' {
			return "native error"
		}
	}
	return s
}
