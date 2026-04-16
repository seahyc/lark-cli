package api

// Lark long-connection event subscription WebSocket client.
//
// This is a minimal, NDJSON-emitting reimplementation of the event-subscription
// client from github.com/larksuite/oapi-sdk-go/v3/ws — we keep the exact binary
// frame protocol (see ws_frame.go, vendored from pbbp2.pb.go) but replace the
// typed EventDispatcher with a direct io.Writer sink so every inbound event
// flows through as NDJSON regardless of event type.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/yjwong/lark-cli/internal/config"
)

const (
	wsEndpointURI = "/callback/ws/endpoint"

	wsHeaderType      = "type"
	wsHeaderMessageID = "message_id"
	wsHeaderSum       = "sum"
	wsHeaderSeq       = "seq"

	wsMessageTypePing  = "ping"
	wsMessageTypePong  = "pong"
	wsMessageTypeEvent = "event"

	wsFrameTypeControl int32 = 0
	wsFrameTypeData    int32 = 1

	wsHandshakeStatusHeader = "Handshake-Status"
	wsHandshakeMsgHeader    = "Handshake-Msg"
)

// wsEndpointResp is the /callback/ws/endpoint response.
type wsEndpointResp struct {
	Code int `json:"code"`
	Msg  string `json:"msg"`
	Data *struct {
		URL          string `json:"URL,omitempty"`
		ClientConfig *struct {
			ReconnectCount    int `json:"ReconnectCount,omitempty"`
			ReconnectInterval int `json:"ReconnectInterval,omitempty"`
			ReconnectNonce    int `json:"ReconnectNonce,omitempty"`
			PingInterval      int `json:"PingInterval,omitempty"`
		} `json:"ClientConfig,omitempty"`
	} `json:"data"`
}

// wsClient streams Lark events over the long-connection WebSocket.
type wsClient struct {
	appID     string
	appSecret string
	domain    string // https scheme, e.g. https://open.larksuite.com
	writer    io.Writer
	filter    func(payload []byte) bool

	// Runtime state
	conn      *websocket.Conn
	serviceID string
	connID    string
	mu        sync.Mutex

	reconnectInterval time.Duration
	pingInterval      time.Duration
	partials          map[string][][]byte // for multi-frame messages
	partialsMu        sync.Mutex
}

func newWSClient(writer io.Writer, filter func([]byte) bool) (*wsClient, error) {
	appID := config.GetAppID()
	appSecret := config.GetAppSecret()
	if appID == "" || appSecret == "" {
		return nil, fmt.Errorf("LARK_APP_ID and LARK_APP_SECRET must be set for WebSocket event subscription")
	}
	return &wsClient{
		appID:             appID,
		appSecret:         appSecret,
		domain:            "https://open.larksuite.com",
		writer:            writer,
		filter:            filter,
		reconnectInterval: 2 * time.Second,
		pingInterval:      2 * time.Minute,
		partials:          map[string][][]byte{},
	}, nil
}

// Run blocks, connecting and reconnecting with backoff until ctx is cancelled.
func (c *wsClient) Run(ctx context.Context) error {
	backoff := c.reconnectInterval
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := c.connect(ctx); err != nil {
			// Auth/config errors are terminal
			if isAuthErr(err) {
				return err
			}
			// Otherwise reconnect with backoff
			fmt.Fprintf(c.stderrLike(), "{\"warning\":\"connect failed: %s, retrying in %s\"}\n", err.Error(), backoff)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// On successful connect, reset backoff
		backoff = c.reconnectInterval

		// Run ping + receive loops until disconnect
		runCtx, cancel := context.WithCancel(ctx)
		go c.pingLoop(runCtx)
		err := c.receiveLoop(runCtx)
		cancel()
		c.closeConn()

		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = err // log and continue to reconnect
	}
}

func (c *wsClient) stderrLike() io.Writer {
	// Emit warnings through the writer too, since NDJSON consumers can filter on "warning".
	return c.writer
}

type authError struct{ msg string }

func (e *authError) Error() string { return e.msg }
func isAuthErr(err error) bool {
	_, ok := err.(*authError)
	return ok
}

func (c *wsClient) connect(ctx context.Context) error {
	connURL, pingSec, reconnectSec, err := c.getConnURL(ctx)
	if err != nil {
		return err
	}

	u, err := url.Parse(connURL)
	if err != nil {
		return fmt.Errorf("parse endpoint url: %w", err)
	}
	c.connID = u.Query().Get("device_id")
	c.serviceID = u.Query().Get("service_id")

	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.DialContext(ctx, connURL, nil)
	if err != nil {
		if resp != nil {
			status := resp.Header.Get(wsHandshakeStatusHeader)
			msg := resp.Header.Get(wsHandshakeMsgHeader)
			if status != "" {
				// 514 = AuthFailed, 403 = Forbidden — both terminal
				if status == "514" || status == "403" {
					return &authError{msg: fmt.Sprintf("handshake auth failed (status=%s): %s", status, msg)}
				}
				return fmt.Errorf("handshake failed (status=%s): %s", status, msg)
			}
		}
		return fmt.Errorf("dial websocket: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	if pingSec > 0 {
		c.pingInterval = time.Duration(pingSec) * time.Second
	}
	if reconnectSec > 0 {
		c.reconnectInterval = time.Duration(reconnectSec) * time.Second
	}

	return nil
}

func (c *wsClient) getConnURL(ctx context.Context) (string, int, int, error) {
	body, _ := json.Marshal(map[string]string{
		"AppID":     c.appID,
		"AppSecret": c.appSecret,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.domain+wsEndpointURI, bytes.NewBuffer(body))
	if err != nil {
		return "", 0, 0, err
	}
	req.Header.Set("locale", "zh")
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, 0, fmt.Errorf("endpoint request failed: %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, 0, err
	}
	var er wsEndpointResp
	if err := json.Unmarshal(raw, &er); err != nil {
		return "", 0, 0, fmt.Errorf("parse endpoint response: %w", err)
	}
	if er.Code != 0 || er.Data == nil || er.Data.URL == "" {
		return "", 0, 0, &authError{msg: fmt.Sprintf("endpoint error code=%d msg=%s", er.Code, er.Msg)}
	}
	pingSec := 0
	reconnectSec := 0
	if er.Data.ClientConfig != nil {
		pingSec = er.Data.ClientConfig.PingInterval
		reconnectSec = er.Data.ClientConfig.ReconnectInterval
	}
	return er.Data.URL, pingSec, reconnectSec, nil
}

func (c *wsClient) closeConn() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func (c *wsClient) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			svcID, _ := strconv.ParseInt(c.serviceID, 10, 32)
			frame := &Frame{
				Method:  wsFrameTypeControl,
				Service: int32(svcID),
				Headers: []Header{{Key: wsHeaderType, Value: wsMessageTypePing}},
			}
			bs, err := frame.Marshal()
			if err != nil {
				continue
			}
			if err := c.writeBinary(bs); err != nil {
				return
			}
		}
	}
}

func (c *wsClient) writeBinary(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("connection closed")
	}
	return c.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (c *wsClient) receiveLoop(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn == nil {
			return fmt.Errorf("connection closed")
		}

		mt, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if mt != websocket.BinaryMessage {
			continue
		}

		var frame Frame
		if err := frame.Unmarshal(msg); err != nil {
			continue
		}

		switch frame.Method {
		case wsFrameTypeControl:
			c.handleControl(frame)
		case wsFrameTypeData:
			c.handleData(frame)
		}
	}
}

func (c *wsClient) handleControl(frame Frame) {
	t := headersGet(frame.Headers, wsHeaderType)
	if t == wsMessageTypePong {
		// Optional server-pushed ClientConfig in payload — ignored, reconnect timing doesn't need updating mid-session.
		return
	}
}

func (c *wsClient) handleData(frame Frame) {
	t := headersGet(frame.Headers, wsHeaderType)
	if t != wsMessageTypeEvent {
		return // cards/other types: skip
	}

	msgID := headersGet(frame.Headers, wsHeaderMessageID)
	sumStr := headersGet(frame.Headers, wsHeaderSum)
	seqStr := headersGet(frame.Headers, wsHeaderSeq)
	sum, _ := strconv.Atoi(sumStr)
	seq, _ := strconv.Atoi(seqStr)

	payload := frame.Payload
	if sum > 1 {
		payload = c.assembleChunks(msgID, sum, seq, payload)
		if payload == nil {
			return // waiting for more chunks
		}
	}

	// Apply filter
	if c.filter != nil && !c.filter(payload) {
		return
	}

	// Emit as NDJSON line
	_, _ = c.writer.Write(payload)
	_, _ = c.writer.Write([]byte("\n"))

	// Acknowledge the frame (200 OK) so the server knows we consumed it.
	resp := map[string]interface{}{"code": 200}
	rb, _ := json.Marshal(resp)
	out := Frame{
		Method:  frame.Method,
		Service: frame.Service,
		Headers: frame.Headers,
		Payload: rb,
	}
	bs, err := out.Marshal()
	if err == nil {
		_ = c.writeBinary(bs)
	}
}

func (c *wsClient) assembleChunks(msgID string, sum, seq int, bs []byte) []byte {
	c.partialsMu.Lock()
	defer c.partialsMu.Unlock()

	buf, ok := c.partials[msgID]
	if !ok {
		buf = make([][]byte, sum)
	}
	if seq >= 0 && seq < len(buf) {
		buf[seq] = bs
	}

	complete := true
	total := 0
	for _, v := range buf {
		if len(v) == 0 {
			complete = false
			break
		}
		total += len(v)
	}
	if !complete {
		c.partials[msgID] = buf
		return nil
	}
	delete(c.partials, msgID)

	out := make([]byte, 0, total)
	for _, v := range buf {
		out = append(out, v...)
	}
	return out
}

func headersGet(headers []Header, key string) string {
	for _, h := range headers {
		if h.Key == key {
			return h.Value
		}
	}
	return ""
}

// Silence unused-import linter when rand is not otherwise used.
var _ = rand.Int
