package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestDesktopScheduleSelfAtBounds(t *testing.T) {
	now := time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC)
	for _, value := range []string{
		now.Add(time.Hour).Format(time.RFC3339),
		now.Add(366 * 24 * time.Hour).Format(time.RFC3339),
	} {
		if _, err := desktopScheduleSelfAt(value, now); err != nil {
			t.Fatalf("desktopScheduleSelfAt(%q): %v", value, err)
		}
	}
	for _, value := range []string{
		now.Add(time.Hour - time.Second).Format(time.RFC3339),
		now.Add(366*24*time.Hour + time.Second).Format(time.RFC3339),
		"2026-09-23T10:00:00",
		"not-a-time",
	} {
		if _, err := desktopScheduleSelfAt(value, now); err == nil {
			t.Errorf("desktopScheduleSelfAt(%q) accepted invalid schedule", value)
		}
	}
}

func TestDesktopScheduleSelfText(t *testing.T) {
	for _, value := range []string{"", " \t\n", strings.Repeat("a", 16385)} {
		if err := desktopScheduleSelfText(value); err == nil {
			t.Errorf("desktopScheduleSelfText(%q) accepted invalid text", value)
		}
	}
	if err := desktopScheduleSelfText("😀"); err != nil {
		t.Fatal(err)
	}
}
