package cmd

import (
	"fmt"
	"testing"

	"github.com/yjwong/lark-cli/internal/api"
)

func searchResult(chatID, fromID string) api.SearchMessageResult {
	return api.SearchMessageResult{
		MetaData: &api.SearchMessageMeta{ChatID: chatID, FromID: fromID},
	}
}

func TestFilterSearchResults_NoFilter(t *testing.T) {
	in := []api.SearchMessageResult{searchResult("oc_1", "ou_a"), searchResult("oc_2", "ou_b")}
	got, applied := filterSearchResults(in, "", "")
	if applied {
		t.Fatalf("expected applied=false when no filters")
	}
	if len(got) != 2 {
		t.Fatalf("expected all results passed through, got %d", len(got))
	}
}

func TestFilterSearchResults_BySender(t *testing.T) {
	in := []api.SearchMessageResult{
		searchResult("oc_1", "ou_a"),
		searchResult("oc_2", "ou_b"),
		searchResult("oc_3", "ou_a"),
	}
	got, applied := filterSearchResults(in, "ou_a", "")
	if !applied {
		t.Fatalf("expected applied=true")
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results from ou_a, got %d", len(got))
	}
	for _, r := range got {
		if r.MetaData.FromID != "ou_a" {
			t.Fatalf("unexpected sender %q", r.MetaData.FromID)
		}
	}
}

func TestFilterSearchResults_ByChat(t *testing.T) {
	in := []api.SearchMessageResult{
		searchResult("oc_1", "ou_a"),
		searchResult("oc_2", "ou_b"),
		searchResult("oc_1", "ou_c"),
	}
	got, _ := filterSearchResults(in, "", "oc_1")
	if len(got) != 2 {
		t.Fatalf("expected 2 results in oc_1, got %d", len(got))
	}
}

func TestFilterSearchResults_BySenderAndChat(t *testing.T) {
	in := []api.SearchMessageResult{
		searchResult("oc_1", "ou_a"),
		searchResult("oc_1", "ou_b"),
		searchResult("oc_2", "ou_a"),
	}
	got, _ := filterSearchResults(in, "ou_a", "oc_1")
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].MetaData.ChatID != "oc_1" || got[0].MetaData.FromID != "ou_a" {
		t.Fatalf("wrong result: %+v", got[0].MetaData)
	}
}

func TestFilterSearchResults_NilMetaSkipped(t *testing.T) {
	in := []api.SearchMessageResult{
		{MetaData: nil},
		searchResult("oc_1", "ou_a"),
	}
	got, _ := filterSearchResults(in, "ou_a", "")
	if len(got) != 1 {
		t.Fatalf("expected nil-meta result skipped, got %d", len(got))
	}
}

func TestRememberSearchDMs_VerifiesMembershipAndDeduplicates(t *testing.T) {
	p2p := func(cid, sender string) api.SearchMessageResult {
		r := searchResult(cid, sender)
		r.MetaData.IsP2PChat = true
		return r
	}
	results := []api.SearchMessageResult{
		{}, searchResult("group", "ou_target"), p2p("wrong", "ou_target"),
		p2p("correct", "ou_self"), p2p("correct", "ou_target"),
	}
	checked := 0
	saved := 0
	rememberSearchDMs(results, func(cid string) (string, string) {
		checked++
		if cid == "correct" {
			return "ou_target", "Target"
		}
		return "", ""
	}, func(id, name, cid string) error {
		saved++
		if id != "ou_target" || name != "Target" || cid != "correct" {
			t.Fatalf("wrong attribution: %s %s %s", id, name, cid)
		}
		return nil
	})
	if checked != 2 || saved != 1 {
		t.Fatalf("checked=%d saved=%d", checked, saved)
	}
}

func TestRememberSearchDMs_BoundsVerification(t *testing.T) {
	var results []api.SearchMessageResult
	for i := 0; i < 20; i++ {
		r := searchResult(fmt.Sprintf("oc_%d", i), "ou_sender")
		r.MetaData.IsP2PChat = true
		results = append(results, r)
	}
	checked := 0
	rememberSearchDMs(results, func(string) (string, string) { checked++; return "", "" }, func(string, string, string) error { t.Fatal("unverified cache entry"); return nil })
	if checked != 12 {
		t.Fatalf("checked=%d", checked)
	}
}
