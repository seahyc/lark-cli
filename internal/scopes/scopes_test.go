package scopes

import (
	"sort"
	"testing"
)

func TestGetScopesForGroups_EmptyReturnsBaseScopeOnly(t *testing.T) {
	got := GetScopesForGroups(nil)
	if len(got) != 1 || got[0] != BaseScope {
		t.Fatalf("expected [%s], got %v", BaseScope, got)
	}
}

func TestGetScopesForGroups_SingleGroup(t *testing.T) {
	got := GetScopesForGroups([]string{"calendar"})
	if !contains(got, BaseScope) {
		t.Fatalf("expected base scope in result, got %v", got)
	}
	if !contains(got, "calendar:calendar") || !contains(got, "calendar:calendar:readonly") {
		t.Fatalf("expected calendar scopes, got %v", got)
	}
}

func TestGetScopesForGroups_DeduplicatesAcrossGroups(t *testing.T) {
	// documents and slides both include drive:drive.
	got := GetScopesForGroups([]string{"documents", "slides"})
	count := 0
	for _, s := range got {
		if s == "drive:drive" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected drive:drive exactly once, got %d", count)
	}
}

func TestGetScopesForGroups_UnknownGroupIgnored(t *testing.T) {
	got := GetScopesForGroups([]string{"does-not-exist"})
	if len(got) != 1 || got[0] != BaseScope {
		t.Fatalf("unknown group should yield only base scope, got %v", got)
	}
}

func TestParseGroups_Empty(t *testing.T) {
	valid, invalid := ParseGroups("")
	if valid != nil || invalid != nil {
		t.Fatalf("empty input should return nil/nil, got %v / %v", valid, invalid)
	}
}

func TestParseGroups_SplitsValidAndInvalid(t *testing.T) {
	valid, invalid := ParseGroups("calendar,nope,messages")
	if !eq(valid, []string{"calendar", "messages"}) {
		t.Fatalf("unexpected valid groups: %v", valid)
	}
	if !eq(invalid, []string{"nope"}) {
		t.Fatalf("unexpected invalid groups: %v", invalid)
	}
}

func TestParseGroups_TrimsWhitespaceAndSkipsBlanks(t *testing.T) {
	valid, invalid := ParseGroups("  calendar , ,messages  ")
	if !eq(valid, []string{"calendar", "messages"}) {
		t.Fatalf("expected trimmed valid groups, got %v", valid)
	}
	if len(invalid) != 0 {
		t.Fatalf("expected no invalid entries, got %v", invalid)
	}
}

func TestGetGroupForCommand(t *testing.T) {
	g, ok := GetGroupForCommand("cal")
	if !ok || g.Name != "calendar" {
		t.Fatalf("expected cal → calendar group, got ok=%v name=%q", ok, g.Name)
	}
	if _, ok := GetGroupForCommand("unknown-cmd"); ok {
		t.Fatalf("expected unknown command lookup to fail")
	}
}

func TestAllGroupNames_CoversEveryRegisteredGroup(t *testing.T) {
	names := AllGroupNames()
	if len(names) != len(Groups) {
		t.Fatalf("AllGroupNames length %d != Groups length %d", len(names), len(Groups))
	}
	for _, n := range names {
		if _, ok := Groups[n]; !ok {
			t.Fatalf("AllGroupNames references missing group %q", n)
		}
	}
}

// --- helpers ---

func contains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}
