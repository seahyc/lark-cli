package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/yjwong/lark-cli/internal/auth"
)

// EventSubscribeOptions holds subscription filter options.
type EventSubscribeOptions struct {
	EventType string         // Filter by event_type in the callback header
	Match     *regexp.Regexp // Filter events whose JSON payload matches this regex
	Compact   bool           // Emit single-line JSON
}

// SubscribeEvents long-polls the Lark outbound event endpoint and streams
// matching events as NDJSON to writer. This is a best-effort fallback while
// the real Lark event WebSocket (wss://open.larksuite.com/open-apis/event/ws)
// is not wired up — use `lark api` for one-off polls if this endpoint
// misbehaves on your tenant.
func (c *Client) SubscribeEvents(writer io.Writer, opts *EventSubscribeOptions) error {
	if err := auth.EnsureValidTenantToken(); err != nil {
		return err
	}

	token := auth.GetTenantTokenStore().GetAccessToken()

	for {
		url := baseURL + "/event/v1/outbound/events"
		req, err := http.NewRequest("POST", url, strings.NewReader("{}"))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if len(body) == 0 || string(body) == "{}" {
			time.Sleep(2 * time.Second)
			continue
		}

		if opts != nil {
			if opts.EventType != "" {
				var event map[string]interface{}
				if err := json.Unmarshal(body, &event); err == nil {
					header, _ := event["header"].(map[string]interface{})
					eventType, _ := header["event_type"].(string)
					if eventType != opts.EventType {
						continue
					}
				}
			}
			if opts.Match != nil && !opts.Match.Match(body) {
				continue
			}
			if opts.Compact {
				var raw json.RawMessage = body
				if compact, err := json.Marshal(raw); err == nil {
					body = compact
				}
			}
		}

		if _, err := fmt.Fprintf(writer, "%s\n", body); err != nil {
			return err
		}
	}
}
