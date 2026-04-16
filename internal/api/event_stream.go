package api

import (
	"context"
	"encoding/json"
	"io"
	"regexp"
)

// EventSubscribeOptions holds subscription filter options.
type EventSubscribeOptions struct {
	EventType string         // Filter by event_type in the callback header
	Match     *regexp.Regexp // Filter events whose JSON payload matches this regex
	Compact   bool           // Emit single-line JSON (no-op for WebSocket — frames already arrive as compact JSON)
}

// SubscribeEvents connects to the Lark event WebSocket (wss://open.larksuite.com/...
// the exact URL is issued by POST /callback/ws/endpoint) and streams inbound
// events as NDJSON to writer. The connection reconnects automatically with
// exponential backoff (up to 60s). This method blocks until ctx is cancelled or
// an unrecoverable auth/permission error occurs.
func (c *Client) SubscribeEvents(writer io.Writer, opts *EventSubscribeOptions) error {
	filter := buildEventFilter(opts)

	ws, err := newWSClient(writer, filter)
	if err != nil {
		return err
	}
	return ws.Run(context.Background())
}

// buildEventFilter returns a predicate over raw event payload bytes, combining
// the --type and --match options into a single check.
func buildEventFilter(opts *EventSubscribeOptions) func([]byte) bool {
	if opts == nil || (opts.EventType == "" && opts.Match == nil) {
		return nil
	}
	return func(payload []byte) bool {
		if opts.EventType != "" {
			var env struct {
				Header struct {
					EventType string `json:"event_type"`
				} `json:"header"`
			}
			if err := json.Unmarshal(payload, &env); err == nil {
				if env.Header.EventType != opts.EventType {
					return false
				}
			}
		}
		if opts.Match != nil && !opts.Match.Match(payload) {
			return false
		}
		return true
	}
}
