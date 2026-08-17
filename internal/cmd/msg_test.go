package cmd

import (
	"strings"
	"testing"

	"github.com/yjwong/lark-cli/internal/config"
)

func TestResolveMessageSendTargetUsesVerifiedDMChat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LARK_CONFIG_DIR", dir)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
	if err := config.RememberVerifiedDMChat("ou_bot", "Access Bot", "oc_bot"); err != nil {
		t.Fatalf("RememberVerifiedDMChat: %v", err)
	}

	receiveIDType, receiveID := resolveMessageSendTarget("open_id", "ou_bot")
	if receiveIDType != "chat_id" || receiveID != "oc_bot" {
		t.Fatalf("resolveMessageSendTarget = (%q, %q), want (chat_id, oc_bot)", receiveIDType, receiveID)
	}
}

func TestResolveMessageSendTargetRequiresChatIDForUnmappedBot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LARK_CONFIG_DIR", dir)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}

	_, _, err := resolveMessageSendTargetE("bot_open_id", "ou_access_bot")
	if err == nil {
		t.Fatal("expected an unresolved bot open_id to fail before sending")
	}
	if !strings.Contains(err.Error(), "lark chat list-dms") || !strings.Contains(err.Error(), "--to oc_") {
		t.Fatalf("error should provide the safe chat_id recovery path, got %q", err)
	}
}

func TestResolveMessageSendTargetUsesVerifiedBotDMChat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LARK_CONFIG_DIR", dir)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
	if err := config.RememberVerifiedDMChat("ou_access_bot", "Access Bot", "oc_access_bot"); err != nil {
		t.Fatalf("RememberVerifiedDMChat: %v", err)
	}

	receiveIDType, receiveID, err := resolveMessageSendTargetE("bot_open_id", "ou_access_bot")
	if err != nil {
		t.Fatalf("resolveMessageSendTargetE returned an error: %v", err)
	}
	if receiveIDType != "chat_id" || receiveID != "oc_access_bot" {
		t.Fatalf("resolveMessageSendTargetE = (%q, %q), want (chat_id, oc_access_bot)", receiveIDType, receiveID)
	}
}

func TestResolveMessageSendTargetRejectsUnverifiedDMCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LARK_CONFIG_DIR", dir)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
	if err := config.RememberDMChat("ou_bot", "Access Bot", "oc_unverified"); err != nil {
		t.Fatalf("RememberDMChat: %v", err)
	}

	receiveIDType, receiveID := resolveMessageSendTarget("open_id", "ou_bot")
	if receiveIDType != "open_id" || receiveID != "ou_bot" {
		t.Fatalf("resolveMessageSendTarget = (%q, %q), want original unverified target", receiveIDType, receiveID)
	}
}

func TestResolveMessageSendTargetLeavesUserOpenIDForBotIdentity(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LARK_CONFIG_DIR", dir)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
	if err := config.RememberVerifiedDMChat("ou_person", "Person", "oc_personal_dm"); err != nil {
		t.Fatalf("RememberVerifiedDMChat: %v", err)
	}

	receiveIDType, receiveID, err := resolveMessageSendTargetForIdentity("open_id", "ou_person", false)
	if err != nil {
		t.Fatalf("resolveMessageSendTargetForIdentity returned an error: %v", err)
	}
	if receiveIDType != "open_id" || receiveID != "ou_person" {
		t.Fatalf("bot identity target = (%q, %q), want original user open_id", receiveIDType, receiveID)
	}
}

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
