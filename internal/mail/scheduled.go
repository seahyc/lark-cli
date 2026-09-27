package mail

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/yjwong/lark-cli/internal/api"
)

const minimumScheduledSendDelay = 5 * time.Minute

// EncodeNativeDraft converts a complete email message into the raw format the
// Lark Mail draft API accepts. Lark requires LF line endings and RFC 4648
// base64url encoding for the raw field.
func EncodeNativeDraft(raw []byte) string {
	raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	return base64.URLEncoding.EncodeToString(raw)
}

// ScheduledSend is the native Lark Mail request for dispatching a draft at a
// future time. Lark owns delivery after this request succeeds; no local daemon
// or running CLI process is required.
type ScheduledSend struct {
	draftID string
	at      time.Time
}

// NewScheduledSend validates a future delivery time and prepares the request
// Lark expects. The Mail API requires a scheduled time at least five minutes
// in the future.
func NewScheduledSend(draftID string, at, now time.Time) (ScheduledSend, error) {
	draftID = strings.TrimSpace(draftID)
	if draftID == "" {
		return ScheduledSend{}, fmt.Errorf("draft ID is required")
	}
	if at.Before(now.Add(minimumScheduledSendDelay)) {
		return ScheduledSend{}, fmt.Errorf("scheduled time must be at least 5 minutes in the future")
	}
	return ScheduledSend{draftID: draftID, at: at}, nil
}

// Path is the native Lark Mail endpoint for scheduling this draft.
func (s ScheduledSend) Path() string {
	return "/mail/v1/user_mailboxes/me/drafts/" + url.PathEscape(s.draftID) + "/send"
}

// Body is the Unix-seconds timestamp Lark expects in the send_time field.
func (s ScheduledSend) Body() string {
	return fmt.Sprintf("%d", s.at.Unix())
}

// NativeDraftResult identifies the server-side Lark draft used for scheduling.
type NativeDraftResult struct {
	DraftID   string `json:"draft_id"`
	MessageID string `json:"message_id,omitempty"`
	Reference string `json:"reference,omitempty"`
}

type nativeMailResponse struct {
	api.BaseResponse
	Data json.RawMessage `json:"data"`
}

func parseNativeDraftResult(data json.RawMessage) (*NativeDraftResult, error) {
	var result NativeDraftResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing native draft response: %w", err)
	}
	// Some Lark Mail tenants return the new draft's message_id rather than a
	// separate draft_id. That message ID is the identifier accepted by the
	// draft send endpoint on those tenants.
	if result.DraftID == "" {
		result.DraftID = result.MessageID
	}
	if result.DraftID == "" {
		return nil, fmt.Errorf("native draft response did not include a draft_id")
	}
	return &result, nil
}

// CreateNativeDraft creates a Lark server-side draft from the same message
// options used by the IMAP draft command. Unlike an IMAP UID, its DraftID can
// be passed to the native scheduled-send endpoint.
func CreateNativeDraft(opts *SendOptions) (*NativeDraftResult, error) {
	creds, err := LoadCredentials()
	if err != nil {
		return nil, err
	}
	from := opts.From
	if from == "" {
		from = creds.Username
	}
	raw, err := buildMessage(from, opts)
	if err != nil {
		return nil, fmt.Errorf("building native draft: %w", err)
	}

	client := api.NewClient()
	var response nativeMailResponse
	if err := client.Post("/mail/v1/user_mailboxes/me/drafts", map[string]string{
		"raw": EncodeNativeDraft(raw),
	}, &response); err != nil {
		return nil, err
	}
	if err := response.Err(); err != nil {
		return nil, err
	}

	return parseNativeDraftResult(response.Data)
}

// NativeDraftUpdatePath returns the native endpoint for replacing an existing
// unsent draft's complete raw MIME message.
func NativeDraftUpdatePath(draftID string) string {
	return "/mail/v1/user_mailboxes/me/drafts/" + url.PathEscape(strings.TrimSpace(draftID))
}

// UpdateNativeDraft replaces an existing unsent native draft. It is used for
// review revisions, so callers retain the same draft rather than creating
// competing copies in Lark Mail.
func UpdateNativeDraft(draftID string, opts *SendOptions) (*NativeDraftResult, error) {
	draftID = strings.TrimSpace(draftID)
	if draftID == "" {
		return nil, fmt.Errorf("draft ID is required")
	}
	creds, err := LoadCredentials()
	if err != nil {
		return nil, err
	}
	from := opts.From
	if from == "" {
		from = creds.Username
	}
	raw, err := buildMessage(from, opts)
	if err != nil {
		return nil, fmt.Errorf("building native draft: %w", err)
	}
	client := api.NewClient()
	var response nativeMailResponse
	if err := client.Put(NativeDraftUpdatePath(draftID), map[string]string{"raw": EncodeNativeDraft(raw)}, &response); err != nil {
		return nil, err
	}
	if err := response.Err(); err != nil {
		return nil, err
	}
	result, err := parseNativeDraftResult(response.Data)
	if err != nil {
		// Some successful update responses omit identifiers; the existing draft
		// ID remains authoritative in that response shape.
		if strings.Contains(err.Error(), "did not include a draft_id") {
			return &NativeDraftResult{DraftID: draftID}, nil
		}
		return nil, err
	}
	return result, nil
}

// ScheduleNativeDraft asks Lark to deliver an existing native draft at the
// requested time. Delivery is then handled by Lark, not by this CLI process.
func ScheduleNativeDraft(draftID string, at, now time.Time) (ScheduledSend, error) {
	scheduled, err := NewScheduledSend(draftID, at, now)
	if err != nil {
		return ScheduledSend{}, err
	}
	client := api.NewClient()
	var response nativeMailResponse
	if err := client.Post(scheduled.Path(), map[string]string{"send_time": scheduled.Body()}, &response); err != nil {
		return ScheduledSend{}, err
	}
	if err := response.Err(); err != nil {
		return ScheduledSend{}, err
	}
	return scheduled, nil
}

// CancelNativeScheduledDraft restores a not-yet-sent native draft to Drafts.
func CancelNativeScheduledDraft(draftID string) error {
	draftID = strings.TrimSpace(draftID)
	if draftID == "" {
		return fmt.Errorf("draft ID is required")
	}
	client := api.NewClient()
	path := "/mail/v1/user_mailboxes/me/drafts/" + url.PathEscape(draftID) + "/cancel_scheduled_send"
	var response nativeMailResponse
	if err := client.Post(path, nil, &response); err != nil {
		return err
	}
	return response.Err()
}
