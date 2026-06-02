package cmd

import (
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
