package cmd

import "testing"

func TestParseTimeArgE_UnixTimestampPassesThrough(t *testing.T) {
	got, err := parseTimeArgE("1704067200")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1704067200" {
		t.Fatalf("want 1704067200, got %q", got)
	}
}

func TestParseTimeArgE_ISO8601WithTimezone(t *testing.T) {
	// 2024-01-01T00:00:00Z == Unix 1704067200
	got, err := parseTimeArgE("2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1704067200" {
		t.Fatalf("want 1704067200, got %q", got)
	}
}

func TestParseTimeArgE_ISO8601WithoutTimezone(t *testing.T) {
	// "2024-01-01T00:00:00" parses as UTC midnight per time.Parse.
	got, err := parseTimeArgE("2024-01-01T00:00:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1704067200" {
		t.Fatalf("want 1704067200, got %q", got)
	}
}

func TestParseTimeArgE_DateOnly(t *testing.T) {
	got, err := parseTimeArgE("2024-01-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1704067200" {
		t.Fatalf("want 1704067200, got %q", got)
	}
}

func TestParseTimeArgE_InvalidReturnsError(t *testing.T) {
	if _, err := parseTimeArgE("not-a-time"); err == nil {
		t.Fatalf("expected error for invalid input")
	}
}
