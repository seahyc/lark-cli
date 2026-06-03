package cmd

import (
	"errors"
	"testing"
)

func TestNameResolver_ResolvesAndCaches(t *testing.T) {
	calls := 0
	r := newNameResolverWithLookup(func(openID string) (string, error) {
		calls++
		if openID == "ou_1" {
			return "Alice", nil
		}
		return "", nil
	})

	if got := r.resolve("ou_1"); got != "Alice" {
		t.Fatalf("resolve(ou_1) = %q, want Alice", got)
	}
	// Second call should hit the cache, not the lookup.
	if got := r.resolve("ou_1"); got != "Alice" {
		t.Fatalf("cached resolve = %q, want Alice", got)
	}
	if calls != 1 {
		t.Fatalf("expected 1 lookup call, got %d", calls)
	}
}

func TestNameResolver_FallbackOnError(t *testing.T) {
	r := newNameResolverWithLookup(func(openID string) (string, error) {
		return "", errors.New("boom")
	})
	if got := r.resolve("ou_err"); got != "ou_err" {
		t.Fatalf("expected fallback to open_id, got %q", got)
	}
	// Cached as the open_id so a second call doesn't retry the failing lookup.
	if got := r.resolve("ou_err"); got != "ou_err" {
		t.Fatalf("expected cached fallback, got %q", got)
	}
}

func TestNameResolver_FallbackOnEmpty(t *testing.T) {
	r := newNameResolverWithLookup(func(openID string) (string, error) {
		return "", nil // found nothing
	})
	if got := r.resolve("ou_empty"); got != "ou_empty" {
		t.Fatalf("expected fallback to open_id on empty name, got %q", got)
	}
}

func TestNameResolver_PresetWins(t *testing.T) {
	lookups := 0
	r := newNameResolverWithLookup(func(openID string) (string, error) {
		lookups++
		return "FromAPI", nil
	})
	r.preset("ou_p", "Preset Name")
	if got := r.resolve("ou_p"); got != "Preset Name" {
		t.Fatalf("preset = %q, want Preset Name", got)
	}
	if lookups != 0 {
		t.Fatalf("preset should avoid lookup, got %d lookups", lookups)
	}
}

func TestNameResolver_EmptyOpenIDReturnsEmpty(t *testing.T) {
	r := newNameResolverWithLookup(func(string) (string, error) { return "X", nil })
	if got := r.resolve(""); got != "" {
		t.Fatalf("resolve(\"\") = %q, want empty", got)
	}
}
