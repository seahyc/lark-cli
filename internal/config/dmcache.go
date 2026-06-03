package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DM cache: a local person -> P2P chat_id store. Lark exposes no read-only way
// to get a 1:1 chat_id from an open_id (GET /im/v1/chats lists groups only, and
// there is no P2P-create endpoint), so once a chat_id becomes known — from a send
// response, a history read, or list-dms — we persist it here. Subsequent
// `lark dm`/`chat dm` calls consult this cache first and can read history without
// a fresh send (which would notify the counterpart).

// DMCacheEntry is one cached person -> chat_id mapping.
type DMCacheEntry struct {
	OpenID string `json:"open_id"`
	Name   string `json:"name,omitempty"`
	ChatID string `json:"chat_id"`
}

// DMCache is the on-disk structure.
type DMCache struct {
	// Entries keyed by open_id.
	Entries map[string]DMCacheEntry `json:"entries"`
}

var dmCacheMu sync.Mutex

// DMCacheFilePath returns the path to the DM cache file in the config dir.
func DMCacheFilePath() string {
	return filepath.Join(cfgDir, "dm_cache.json")
}

// normalizeDMName lowercases and collapses whitespace for name-based lookups.
func normalizeDMName(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

// LoadDMCache reads the DM cache from disk. A missing or unreadable file yields
// an empty cache (never an error) so callers can treat it as best-effort.
func LoadDMCache() *DMCache {
	c := &DMCache{Entries: make(map[string]DMCacheEntry)}
	if cfgDir == "" {
		return c
	}
	data, err := os.ReadFile(DMCacheFilePath())
	if err != nil {
		return c
	}
	var loaded DMCache
	if err := json.Unmarshal(data, &loaded); err != nil {
		return c
	}
	if loaded.Entries == nil {
		loaded.Entries = make(map[string]DMCacheEntry)
	}
	return &loaded
}

// save writes the cache to disk atomically (temp file + rename).
func (c *DMCache) save() error {
	if cfgDir == "" {
		return nil
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	path := DMCacheFilePath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// GetByOpenID returns the cached chat_id for an open_id, or "" if not cached.
func (c *DMCache) GetByOpenID(openID string) string {
	if openID == "" {
		return ""
	}
	if e, ok := c.Entries[openID]; ok {
		return e.ChatID
	}
	return ""
}

// GetByName returns the cached chat_id and open_id for a normalized name match,
// or ("", "") if not found. Matching is case/whitespace-insensitive.
func (c *DMCache) GetByName(name string) (chatID, openID string) {
	want := normalizeDMName(name)
	if want == "" {
		return "", ""
	}
	for _, e := range c.Entries {
		if normalizeDMName(e.Name) == want {
			return e.ChatID, e.OpenID
		}
	}
	return "", ""
}

// Set records a person -> chat_id mapping in memory (call Save to persist). A
// non-empty name overwrites a previously blank one; a blank name never clears an
// existing one.
func (c *DMCache) Set(openID, name, chatID string) {
	if openID == "" || chatID == "" {
		return
	}
	if c.Entries == nil {
		c.Entries = make(map[string]DMCacheEntry)
	}
	existing := c.Entries[openID]
	if name == "" {
		name = existing.Name
	}
	c.Entries[openID] = DMCacheEntry{OpenID: openID, Name: name, ChatID: chatID}
}

// Save persists the cache to disk (best-effort; returns any write error).
func (c *DMCache) Save() error {
	return c.save()
}

// RememberDMChat is a convenience helper: load the cache, set one mapping, and
// persist. Safe to call concurrently and cheap enough for one-off updates. Errors
// are returned but callers typically ignore them (the cache is an optimization).
func RememberDMChat(openID, name, chatID string) error {
	if openID == "" || chatID == "" {
		return nil
	}
	dmCacheMu.Lock()
	defer dmCacheMu.Unlock()
	c := LoadDMCache()
	c.Set(openID, name, chatID)
	return c.Save()
}
