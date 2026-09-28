package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestNativeDraftOptionsValidatesAndPreservesThreading(t *testing.T) {
	opts, err := nativeDraftOptions(
		[]string{"recipient@example.com"},
		[]string{"cc@example.com"},
		"Reference letter",
		"Hi there,\n\nAttached is the letter.\n\nYC",
		[]string{"/tmp/letter.docx"},
		"<original@example.com>",
		[]string{"<root@example.com>", "<original@example.com>"},
	)
	if err != nil {
		t.Fatalf("nativeDraftOptions() error = %v", err)
	}
	if got, want := opts.To[0], "recipient@example.com"; got != want {
		t.Fatalf("To = %q, want %q", got, want)
	}
	if got, want := opts.InReplyTo, "<original@example.com>"; got != want {
		t.Fatalf("InReplyTo = %q, want %q", got, want)
	}
	if got, want := len(opts.References), 2; got != want {
		t.Fatalf("References length = %d, want %d", got, want)
	}
}

func TestNativeDraftOptionsRequiresEmailContent(t *testing.T) {
	_, err := nativeDraftOptions([]string{"recipient@example.com"}, nil, "", "", nil, "", nil)
	if err == nil {
		t.Fatal("nativeDraftOptions() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "--subject") {
		t.Fatalf("nativeDraftOptions() error = %q, want subject validation", err)
	}
}

func TestParseScheduledAtUsesRequestedTimezone(t *testing.T) {
	location, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 31, 9, 0, 0, 0, location)

	at, err := parseScheduledAt("2026-09-02 19:00", now, location)
	if err != nil {
		t.Fatalf("parseScheduledAt() error = %v", err)
	}
	if got, want := at.Format(time.RFC3339), "2026-09-02T19:00:00+08:00"; got != want {
		t.Fatalf("parseScheduledAt() = %q, want %q", got, want)
	}
}

func TestParseScheduledAtRejectsTooSoon(t *testing.T) {
	now := time.Date(2026, time.August, 31, 9, 0, 0, 0, time.UTC)
	_, err := parseScheduledAt("2026-08-31T09:04:00Z", now, time.UTC)
	if err == nil {
		t.Fatal("parseScheduledAt() error = nil, want minimum-delay validation")
	}
	if !strings.Contains(err.Error(), "at least 5 minutes") {
		t.Fatalf("parseScheduledAt() error = %q, want minimum-delay message", err)
	}
}
