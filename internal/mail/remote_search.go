package mail

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
)

const (
	remoteSearchMaxTextLength = 256
	remoteSearchDefaultLimit  = 50
	remoteSearchMaxLimit      = 100
)

// RemoteSearchOptions is a deliberately small subset of standard IMAP SEARCH.
// Body matching happens on the IMAP server; matching bodies are never fetched
// or returned by this package.
type RemoteSearchOptions struct {
	From    string
	Subject string
	Body    string
	Since   *time.Time
	Before  *time.Time
	Limit   int
}

// RemoteSearchResult contains only envelope metadata for the selected mailbox.
// MatchedUIDs records the server's full UID SEARCH result size. IMAP SEARCH has
// no portable server-side result limit, so this can exceed the returned count.
type RemoteSearchResult struct {
	Mailbox          string           `json:"mailbox"`
	MatchedUIDs      int              `json:"matched_uids"`
	Results          []CachedEnvelope `json:"results"`
	Count            int              `json:"count"`
	ResultsTruncated bool             `json:"results_truncated"`
}

// ParseRemoteSearchOptions validates bounded CLI-style IMAP SEARCH inputs.
func ParseRemoteSearchOptions(from, subject, body, since, before string, limit int) (*RemoteSearchOptions, error) {
	opts := &RemoteSearchOptions{
		From:    strings.TrimSpace(from),
		Subject: strings.TrimSpace(subject),
		Body:    strings.TrimSpace(body),
		Limit:   limit,
	}
	for _, field := range []struct {
		name  string
		value string
	}{{"from", opts.From}, {"subject", opts.Subject}, {"body", opts.Body}} {
		if len(field.value) > remoteSearchMaxTextLength {
			return nil, fmt.Errorf("--%s exceeds %d characters", field.name, remoteSearchMaxTextLength)
		}
	}
	if since != "" {
		parsed, parseErr := time.Parse("2006-01-02", since)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid --since date %q: expected YYYY-MM-DD", since)
		}
		opts.Since = &parsed
	}
	if before != "" {
		parsed, parseErr := time.Parse("2006-01-02", before)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid --before date %q: expected YYYY-MM-DD", before)
		}
		opts.Before = &parsed
	}
	if opts.Since != nil && opts.Before != nil && !opts.Before.After(*opts.Since) {
		return nil, fmt.Errorf("--before must be after --since")
	}
	if opts.From == "" && opts.Subject == "" && opts.Body == "" && opts.Since == nil && opts.Before == nil {
		return nil, fmt.Errorf("provide at least one of --from, --subject, --body, --since, or --before")
	}
	if opts.Limit == 0 {
		opts.Limit = remoteSearchDefaultLimit
	}
	if opts.Limit < 1 || opts.Limit > remoteSearchMaxLimit {
		return nil, fmt.Errorf("--limit must be between 1 and %d", remoteSearchMaxLimit)
	}
	return opts, nil
}

func remoteSearchCriteria(opts *RemoteSearchOptions) (*imap.SearchCriteria, error) {
	if opts == nil {
		return nil, fmt.Errorf("remote search options are required")
	}
	if opts.From == "" && opts.Subject == "" && opts.Body == "" && opts.Since == nil && opts.Before == nil {
		return nil, fmt.Errorf("remote search requires a text or date constraint")
	}
	criteria := &imap.SearchCriteria{}
	if opts.From != "" {
		criteria.Header = append(criteria.Header, imap.SearchCriteriaHeaderField{Key: "From", Value: opts.From})
	}
	if opts.Subject != "" {
		criteria.Header = append(criteria.Header, imap.SearchCriteriaHeaderField{Key: "Subject", Value: opts.Subject})
	}
	if opts.Body != "" {
		criteria.Body = append(criteria.Body, opts.Body)
	}
	if opts.Since != nil {
		criteria.SentSince = *opts.Since
	}
	if opts.Before != nil {
		criteria.SentBefore = *opts.Before
	}
	return criteria, nil
}

func newestUIDs(uids []imap.UID, limit int) []imap.UID {
	if len(uids) == 0 || limit <= 0 {
		return nil
	}
	selected := append([]imap.UID(nil), uids...)
	sort.Slice(selected, func(i, j int) bool { return selected[i] > selected[j] })
	if len(selected) > limit {
		selected = selected[:limit]
	}
	return selected
}

// RemoteSearch searches one selected IMAP mailbox, then fetches envelopes only
// for the capped most-recent UID set. It uses the existing configured IMAP
// client and its existing connection behavior; it does not write credentials,
// mailbox state, or local cache.
func RemoteSearch(mailbox string, opts *RemoteSearchOptions) (*RemoteSearchResult, error) {
	if mailbox == "" {
		return nil, fmt.Errorf("mailbox is required")
	}
	if opts == nil || opts.Limit < 1 || opts.Limit > remoteSearchMaxLimit {
		return nil, fmt.Errorf("remote search options must use a limit from 1 to %d", remoteSearchMaxLimit)
	}
	criteria, err := remoteSearchCriteria(opts)
	if err != nil {
		return nil, err
	}

	client, err := Connect()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	if _, err := client.SelectMailbox(mailbox); err != nil {
		return nil, err
	}

	data, err := client.imap.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("searching IMAP mailbox: %w", err)
	}
	allUIDs := data.AllUIDs()
	selected := newestUIDs(allUIDs, opts.Limit)
	envelopes, err := client.FetchEnvelopesByUID(selected)
	if err != nil {
		return nil, err
	}
	sort.Slice(envelopes, func(i, j int) bool { return envelopes[i].Date > envelopes[j].Date })
	results := make([]CachedEnvelope, 0, len(envelopes))
	for _, envelope := range envelopes {
		results = append(results, CachedEnvelope{
			UID:       uint32(envelope.UID),
			MessageID: envelope.MessageID,
			Date:      time.Unix(envelope.Date, 0),
			FromAddr:  envelope.FromAddr,
			FromName:  envelope.FromName,
			Subject:   envelope.Subject,
		})
	}
	return &RemoteSearchResult{
		Mailbox:          mailbox,
		MatchedUIDs:      len(allUIDs),
		Results:          results,
		Count:            len(results),
		ResultsTruncated: len(allUIDs) > len(selected),
	}, nil
}
