package mail

import (
	"bytes"
	"fmt"
	netmail "net/mail"
	"strings"

	"github.com/emersion/go-imap/v2"
)

// DraftHeaders captures the recipient/subject headers of an existing draft.
type DraftHeaders struct {
	To      []string
	CC      []string
	BCC     []string
	Subject string

	// Threading headers, preserved so reply drafts stay in their thread.
	InReplyTo  string
	References []string
	Extra      [][2]string // other threading headers, e.g. Thread-Topic, Thread-Index
}

// threadingExtraHeaders lists additional headers (beyond In-Reply-To and
// References) that carry conversation threading state.
var threadingExtraHeaders = []string{"Thread-Topic", "Thread-Index"}

// parseThreadingHeaders extracts threading headers from a raw message or
// header block.
func parseThreadingHeaders(raw []byte, h *DraftHeaders) {
	m, err := netmail.ReadMessage(bytes.NewReader(append(bytes.TrimRight(raw, "\r\n"), '\r', '\n', '\r', '\n')))
	if err != nil {
		return
	}
	h.InReplyTo = strings.TrimSpace(m.Header.Get("In-Reply-To"))
	h.References = strings.Fields(m.Header.Get("References"))
	for _, name := range threadingExtraHeaders {
		if v := strings.TrimSpace(m.Header.Get(name)); v != "" {
			h.Extra = append(h.Extra, [2]string{name, v})
		}
	}
}

// applyDraftHeaders fills unset fields of opts from the existing draft headers
// and returns the SendOptions for the replacement draft.
func applyDraftHeaders(from string, opts *UpdateDraftOptions, existing *DraftHeaders) *SendOptions {
	to, cc, bcc, subject := opts.To, opts.CC, opts.BCC, opts.Subject
	so := &SendOptions{From: from, Body: opts.Body, Attachments: opts.Attachments}
	if existing != nil {
		if to == nil {
			to = existing.To
		}
		if cc == nil {
			cc = existing.CC
		}
		if bcc == nil {
			bcc = existing.BCC
		}
		if subject == "" {
			subject = existing.Subject
		}
		so.InReplyTo = existing.InReplyTo
		so.References = existing.References
		so.ExtraHeaders = existing.Extra
	}
	so.To, so.CC, so.BCC, so.Subject = to, cc, bcc, subject
	return so
}

// FetchDraftHeaders pulls the existing To/Cc/Bcc/Subject from a draft via the
// IMAP ENVELOPE response. Attachments and body are NOT preserved — callers that
// want to keep attachments must re-fetch and reassemble the raw message.
func (c *Client) FetchDraftHeaders(uid imap.UID) (*DraftHeaders, error) {
	uidSet := imap.UIDSetNum(uid)

	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		UID:      true,
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierHeader, Peek: true},
		},
	}

	messages, err := c.imap.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("fetching draft headers: %w", err)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("draft not found: UID %d", uid)
	}

	env := messages[0].Envelope
	if env == nil {
		return nil, fmt.Errorf("draft UID %d has no envelope", uid)
	}

	headers := &DraftHeaders{Subject: env.Subject}
	for _, a := range env.To {
		headers.To = append(headers.To, a.Addr())
	}
	for _, a := range env.Cc {
		headers.CC = append(headers.CC, a.Addr())
	}
	for _, a := range env.Bcc {
		headers.BCC = append(headers.BCC, a.Addr())
	}
	for _, data := range messages[0].BodySection {
		parseThreadingHeaders(data, headers)
		break
	}
	return headers, nil
}

// UpdateDraftOptions configures an in-place draft replacement.
type UpdateDraftOptions struct {
	UID         uint32
	Mailbox     string   // defaults to "Drafts"
	Body        string   // new body (required)
	To          []string // if nil and KeepHeaders, use existing
	CC          []string // if nil and KeepHeaders, use existing
	BCC         []string // if nil and KeepHeaders, use existing
	Subject     string   // if "" and KeepHeaders, use existing
	KeepHeaders bool
	Attachments []string
}

// UpdateDraftResult is the JSON payload for `lark mail update-draft`.
type UpdateDraftResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Mailbox string `json:"mailbox"`
	OldUID  uint32 `json:"old_uid"`
	NewUID  uint32 `json:"new_uid"`
}

// UpdateDraft replaces a draft in-place: fetches the existing headers, appends
// a new draft with the new body, then deletes+expunges the old draft.
//
// Limitations: attachments on the original draft are not preserved unless
// re-supplied via opts.Attachments. The new draft gets a fresh Message-ID.
func UpdateDraft(opts *UpdateDraftOptions) (*UpdateDraftResult, error) {
	if opts.UID == 0 {
		return nil, fmt.Errorf("uid is required")
	}
	if opts.Body == "" {
		return nil, fmt.Errorf("body is required")
	}

	mailbox := opts.Mailbox
	if mailbox == "" {
		mailbox = "Drafts"
	}

	creds, err := LoadCredentials()
	if err != nil {
		return nil, err
	}

	client, err := Connect()
	if err != nil {
		return nil, fmt.Errorf("connecting to IMAP: %w", err)
	}
	defer client.Close()

	if _, err := client.SelectMailbox(mailbox); err != nil {
		// Some servers use singular "Draft"
		if mailbox == "Drafts" {
			if _, e2 := client.SelectMailbox("Draft"); e2 == nil {
				mailbox = "Draft"
			} else {
				return nil, fmt.Errorf("selecting drafts mailbox: %w", err)
			}
		} else {
			return nil, fmt.Errorf("selecting mailbox %s: %w", mailbox, err)
		}
	}

	var existing *DraftHeaders
	if opts.KeepHeaders {
		existing, err = client.FetchDraftHeaders(imap.UID(opts.UID))
		if err != nil {
			return nil, err
		}
	}
	sendOpts := applyDraftHeaders(creds.Username, opts, existing)
	to := sendOpts.To

	if len(to) == 0 {
		return nil, fmt.Errorf("no To recipients (original draft had none and --to not supplied)")
	}

	msg, err := buildMessage(creds.Username, sendOpts)
	if err != nil {
		return nil, fmt.Errorf("building new draft message: %w", err)
	}

	newUID, err := client.AppendDraft(mailbox, msg)
	if err != nil {
		return nil, fmt.Errorf("appending new draft: %w", err)
	}

	// Delete + expunge the old draft.
	if err := client.DeleteMessage(imap.UID(opts.UID)); err != nil {
		return &UpdateDraftResult{
			Success: false,
			Message: fmt.Sprintf("new draft appended (UID %d) but old draft delete failed: %v", newUID, err),
			Mailbox: mailbox,
			OldUID:  opts.UID,
			NewUID:  uint32(newUID),
		}, nil
	}

	return &UpdateDraftResult{
		Success: true,
		Message: "Draft updated",
		Mailbox: mailbox,
		OldUID:  opts.UID,
		NewUID:  uint32(newUID),
	}, nil
}
