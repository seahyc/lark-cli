package config

import (
	"os"
	"testing"
)

// setupCacheDir points the config dir at a temp dir for cache tests.
func setupCacheDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("LARK_CONFIG_DIR", dir)
	if err := Init(); err != nil {
		t.Fatalf("config.Init failed: %v", err)
	}
}

func TestDMCache_SetGetByOpenID(t *testing.T) {
	setupCacheDir(t)
	c := LoadDMCache()
	c.Set("ou_1", "Alice", "oc_1")
	if got := c.GetByOpenID("ou_1"); got != "oc_1" {
		t.Fatalf("GetByOpenID = %q, want oc_1", got)
	}
	if got := c.GetByOpenID("ou_missing"); got != "" {
		t.Fatalf("expected empty for missing, got %q", got)
	}
}

func TestDMCache_GetByName(t *testing.T) {
	setupCacheDir(t)
	c := LoadDMCache()
	c.Set("ou_1", "Alice Tan", "oc_1")
	// Case + whitespace insensitive.
	chatID, openID := c.GetByName("  alice   tan ")
	if chatID != "oc_1" || openID != "ou_1" {
		t.Fatalf("GetByName = (%q,%q), want (oc_1,ou_1)", chatID, openID)
	}
	if cid, _ := c.GetByName("nobody"); cid != "" {
		t.Fatalf("expected empty for unknown name, got %q", cid)
	}
}

func TestDMCache_Persistence(t *testing.T) {
	setupCacheDir(t)
	c := LoadDMCache()
	c.Set("ou_persist", "Bob", "oc_persist")
	if err := c.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	// Reload from disk.
	reloaded := LoadDMCache()
	if got := reloaded.GetByOpenID("ou_persist"); got != "oc_persist" {
		t.Fatalf("after reload GetByOpenID = %q, want oc_persist", got)
	}
	// The file should exist.
	if _, err := os.Stat(DMCacheFilePath()); err != nil {
		t.Fatalf("cache file not written: %v", err)
	}
}

func TestDMCache_RememberDMChat(t *testing.T) {
	setupCacheDir(t)
	if err := RememberDMChat("ou_r", "Carol", "oc_r"); err != nil {
		t.Fatalf("RememberDMChat failed: %v", err)
	}
	if got := LoadDMCache().GetByOpenID("ou_r"); got != "oc_r" {
		t.Fatalf("GetByOpenID = %q, want oc_r", got)
	}
}

func TestDMCache_SetBlankNameKeepsExisting(t *testing.T) {
	setupCacheDir(t)
	c := LoadDMCache()
	c.Set("ou_1", "Dave", "oc_1")
	// Updating with a blank name should not wipe the existing name.
	c.Set("ou_1", "", "oc_2")
	if got := c.Entries["ou_1"].Name; got != "Dave" {
		t.Fatalf("name = %q, want Dave preserved", got)
	}
	if got := c.GetByOpenID("ou_1"); got != "oc_2" {
		t.Fatalf("chat_id should update to oc_2, got %q", got)
	}
}

func TestDMCache_SetIgnoresEmptyArgs(t *testing.T) {
	setupCacheDir(t)
	c := LoadDMCache()
	c.Set("", "X", "oc_1")
	c.Set("ou_1", "X", "")
	if len(c.Entries) != 0 {
		t.Fatalf("expected no entries for empty open_id/chat_id, got %d", len(c.Entries))
	}
}

func TestLoadDMCache_MissingFileIsEmpty(t *testing.T) {
	setupCacheDir(t)
	c := LoadDMCache()
	if c == nil || len(c.Entries) != 0 {
		t.Fatalf("expected empty cache for missing file")
	}
}
