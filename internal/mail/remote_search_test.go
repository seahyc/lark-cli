package mail

import (
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
)

func TestParseRemoteSearchOptionsValidatesConstraintsAndDates(t *testing.T) {
	if _, err := ParseRemoteSearchOptions("", "", "", "", "", 50); err == nil {
		t.Fatal("expected missing constraint error")
	}
	if _, err := ParseRemoteSearchOptions("a", "", "", "2026-02-30", "", 50); err == nil {
		t.Fatal("expected invalid date error")
	}
	if _, err := ParseRemoteSearchOptions("a", "", "", "2026-02-02", "2026-02-02", 50); err == nil {
		t.Fatal("expected invalid date range error")
	}
	if _, err := ParseRemoteSearchOptions("a", "", "", "", "", 101); err == nil {
		t.Fatal("expected limit error")
	}
	opts, err := ParseRemoteSearchOptions(" sender@example.test ", "subject", "", "2026-01-01", "2026-02-01", 0)
	if err != nil {
		t.Fatal(err)
	}
	if opts.From != "sender@example.test" || opts.Limit != remoteSearchDefaultLimit || opts.Since == nil || opts.Before == nil {
		t.Fatalf("unexpected options: %#v", opts)
	}
}

func TestRemoteSearchCriteriaUsesServerSideFields(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	criteria, err := remoteSearchCriteria(&RemoteSearchOptions{From: "a@example.test", Subject: "report", Body: "needle", Since: &since, Before: &before, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(criteria.Header) != 2 || criteria.Header[0] != (imap.SearchCriteriaHeaderField{Key: "From", Value: "a@example.test"}) || criteria.Header[1] != (imap.SearchCriteriaHeaderField{Key: "Subject", Value: "report"}) {
		t.Fatalf("unexpected headers: %#v", criteria.Header)
	}
	if len(criteria.Body) != 1 || criteria.Body[0] != "needle" || !criteria.SentSince.Equal(since) || !criteria.SentBefore.Equal(before) {
		t.Fatalf("unexpected criteria: %#v", criteria)
	}
}

func TestNewestUIDsCapsBeforeEnvelopeFetch(t *testing.T) {
	got := newestUIDs([]imap.UID{4, 12, 7, 20, 19}, 3)
	want := []imap.UID{20, 19, 12}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	}
}
