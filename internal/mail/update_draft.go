package mail

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
)

// DraftHeaders captures the recipient/subject headers of an existing draft.
type DraftHeaders struct {
	To      []string
	CC      []string
	BCC     []string
	Subject string
}

// FetchDraftHeaders pulls the existing To/Cc/Bcc/Subject from a draft via the
// IMAP ENVELOPE response. Attachments and body are NOT preserved — callers that
// want to keep attachments must re-fetch and reassemble the raw message.
func (c *Client) FetchDraftHeaders(uid imap.UID) (*DraftHeaders, error) {
	uidSet := imap.UIDSetNum(uid)

	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		UID:      true,
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

	// Fetch existing headers from the draft.
	to := opts.To
	cc := opts.CC
	bcc := opts.BCC
	subject := opts.Subject

	if opts.KeepHeaders {
		existing, err := client.FetchDraftHeaders(imap.UID(opts.UID))
		if err != nil {
			return nil, err
		}
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
	}

	if len(to) == 0 {
		return nil, fmt.Errorf("no To recipients (original draft had none and --to not supplied)")
	}

	sendOpts := &SendOptions{
		From:        creds.Username,
		To:          to,
		CC:          cc,
		BCC:         bcc,
		Subject:     subject,
		Body:        opts.Body,
		Attachments: opts.Attachments,
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
