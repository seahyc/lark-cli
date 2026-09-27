package mail

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseNativeDraftResultKeepsDraftIDAndReference(t *testing.T) {
	result, err := parseNativeDraftResult(json.RawMessage(`{"draft_id":"draft_123","reference":"https://mail.example/draft_123"}`))
	if err != nil {
		t.Fatalf("parseNativeDraftResult() error = %v", err)
	}
	if result.DraftID != "draft_123" || result.Reference != "https://mail.example/draft_123" {
		t.Fatalf("parseNativeDraftResult() = %#v", result)
	}
}

func TestParseNativeDraftResultUsesMessageIDWhenDraftIDIsOmitted(t *testing.T) {
	result, err := parseNativeDraftResult(json.RawMessage(`{"message_id":"draft-as-message-id"}`))
	if err != nil {
		t.Fatalf("parseNativeDraftResult() error = %v", err)
	}
	if got, want := result.DraftID, "draft-as-message-id"; got != want {
		t.Fatalf("DraftID = %q, want %q", got, want)
	}
}

func TestParseNativeDraftResultRejectsMissingDraftID(t *testing.T) {
	_, err := parseNativeDraftResult(json.RawMessage(`{"reference":"https://mail.example/draft"}`))
	if err == nil {
		t.Fatal("parseNativeDraftResult() error = nil, want missing draft ID error")
	}
}

func TestEncodeNativeDraftUsesLarkBase64URLAndLF(t *testing.T) {
	encoded := EncodeNativeDraft([]byte("Subject: hello\r\n\r\nhello\r\n"))
	raw, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}
	if got, want := string(raw), "Subject: hello\n\nhello\n"; got != want {
		t.Fatalf("decoded raw = %q, want %q", got, want)
	}
}

func TestNewScheduledSendBuildsNativeRequest(t *testing.T) {
	now := time.Date(2026, time.August, 31, 9, 0, 0, 0, time.UTC)
	at := now.Add(15 * time.Minute)

	scheduled, err := NewScheduledSend("draft_123", at, now)
	if err != nil {
		t.Fatalf("NewScheduledSend() error = %v", err)
	}

	if got, want := scheduled.Path(), "/mail/v1/user_mailboxes/me/drafts/draft_123/send"; got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
	if got, want := scheduled.Body(), "1788167700"; got != want {
		t.Fatalf("Body() = %q, want %q", got, want)
	}
}

func TestNativeDraftUpdatePathEscapesDraftID(t *testing.T) {
	if got, want := NativeDraftUpdatePath("draft/a b"), "/mail/v1/user_mailboxes/me/drafts/draft%2Fa%20b"; got != want {
		t.Fatalf("NativeDraftUpdatePath() = %q, want %q", got, want)
	}
}

func TestNewScheduledSendRejectsTimesLessThanFiveMinutesAway(t *testing.T) {
	now := time.Date(2026, time.August, 31, 9, 0, 0, 0, time.UTC)

	_, err := NewScheduledSend("draft_123", now.Add(4*time.Minute+59*time.Second), now)
	if err == nil {
		t.Fatal("NewScheduledSend() error = nil, want minimum-delay validation error")
	}
	if !strings.Contains(err.Error(), "at least 5 minutes") {
		t.Fatalf("NewScheduledSend() error = %q, want minimum-delay message", err)
	}
}

func TestNewScheduledSendRejectsBlankDraftID(t *testing.T) {
	now := time.Date(2026, time.August, 31, 9, 0, 0, 0, time.UTC)

	_, err := NewScheduledSend("  ", now.Add(10*time.Minute), now)
	if err == nil {
		t.Fatal("NewScheduledSend() error = nil, want draft ID validation error")
	}
}
