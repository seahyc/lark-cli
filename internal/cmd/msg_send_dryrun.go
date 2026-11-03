package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yjwong/lark-cli/internal/api"
)

// --- attachment pre-checks ---

// sendAttachment describes one --file or --image before upload.
type sendAttachment struct {
	Kind           string `json:"kind"` // file or image
	Path           string `json:"path"`
	Name           string `json:"name"`
	SizeBytes      int64  `json:"size_bytes"`
	Size           string `json:"size"`
	LimitBytes     int64  `json:"limit_bytes"`
	Limit          string `json:"limit"`
	WithinLimit    bool   `json:"within_limit"`
	UploadFileType string `json:"upload_file_type,omitempty"` // Lark file_type for --file
	MsgType        string `json:"msg_type,omitempty"`         // file, media, audio
	Problem        string `json:"problem,omitempty"`
}

// attachmentError is a pre-upload failure with a stable error code.
type attachmentError struct {
	Code    string
	Message string
}

func (e *attachmentError) Error() string { return e.Message }

// humanBytes renders a byte count with binary units (matching the MiB limits).
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// inspectAttachment stats one path and checks it against the IM upload limit
// for its kind. It never reads file contents.
func inspectAttachment(kind, path string) sendAttachment {
	limit := api.MaxIMFileBytes
	if kind == "image" {
		limit = api.MaxIMImageBytes
	}
	a := sendAttachment{
		Kind: kind, Path: path, Name: filepath.Base(path),
		LimitBytes: limit, Limit: humanBytes(limit),
	}
	if kind == "file" {
		a.UploadFileType = inferFileType(path)
		a.MsgType = messageTypeForFileType(a.UploadFileType)
	}
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		a.Problem = fmt.Sprintf("%s not found: %s", kind, path)
		return a
	case err != nil:
		a.Problem = fmt.Sprintf("cannot stat %s: %v", path, err)
		return a
	case info.IsDir():
		a.Problem = fmt.Sprintf("%s is a directory, not a file", path)
		return a
	}
	a.SizeBytes = info.Size()
	a.Size = humanBytes(a.SizeBytes)
	switch {
	case a.SizeBytes == 0:
		a.Problem = fmt.Sprintf("%s is empty; Lark rejects 0-byte uploads", a.Name)
	case a.SizeBytes > limit:
		a.Problem = oversizeMessage(kind, a.Name, a.SizeBytes, limit)
	default:
		a.WithinLimit = true
	}
	return a
}

func oversizeMessage(kind, name string, size, limit int64) string {
	if kind == "image" {
		return fmt.Sprintf("%s is %s; Lark IM images are limited to %s. Compress it, or send it with --file instead (limit %s)",
			name, humanBytes(size), humanBytes(limit), humanBytes(api.MaxIMFileBytes))
	}
	return fmt.Sprintf("%s is %s; Lark IM file messages are limited to %s. Upload it to Drive instead (`lark doc upload %s`) and send the returned link with --text",
		name, humanBytes(size), humanBytes(limit), name)
}

// inspectSendAttachments checks every --file and --image up front.
func inspectSendAttachments(files, images []string) []sendAttachment {
	out := make([]sendAttachment, 0, len(files)+len(images))
	for _, f := range files {
		out = append(out, inspectAttachment("file", f))
	}
	for _, img := range images {
		out = append(out, inspectAttachment("image", img))
	}
	return out
}

// firstAttachmentProblem returns the first blocking problem as an error, so a
// real send fails before uploading anything.
func firstAttachmentProblem(atts []sendAttachment) error {
	for _, a := range atts {
		if a.Problem == "" {
			continue
		}
		code := "FILE_ERROR"
		if a.SizeBytes > a.LimitBytes {
			code = "FILE_TOO_LARGE"
		}
		return &attachmentError{Code: code, Message: a.Problem}
	}
	return nil
}

// --- dry run ---

// sendIdentity is who the message would be sent as.
type sendIdentity struct {
	As     string `json:"as"` // user or bot
	Name   string `json:"name,omitempty"`
	Email  string `json:"email,omitempty"`
	OpenID string `json:"open_id,omitempty"`
	AppID  string `json:"app_id,omitempty"`
	Note   string `json:"note,omitempty"`
}

// sendChatInfo describes the destination chat when it is known up front.
type sendChatInfo struct {
	ChatID      string           `json:"chat_id,omitempty"`
	ChatMode    string           `json:"chat_mode,omitempty"` // p2p, group, topic
	ChatType    string           `json:"chat_type,omitempty"` // private, public
	Name        string           `json:"name,omitempty"`
	UserCount   int              `json:"user_count,omitempty"`
	BotCount    int              `json:"bot_count,omitempty"`
	External    bool             `json:"external,omitempty"`
	Counterpart *recipientPerson `json:"counterpart,omitempty"`
	Note        string           `json:"note,omitempty"`
}

// sendMessagePlan describes the message(s) that would be sent.
type sendMessagePlan struct {
	MsgType     string           `json:"msg_type"`
	Count       int              `json:"count"`
	TextChars   int              `json:"text_chars,omitempty"`
	Attachments []sendAttachment `json:"attachments,omitempty"`
	ParentID    string           `json:"parent_id,omitempty"`
	RootID      string           `json:"root_id,omitempty"`
}

// sendDryRun is the JSON report printed by `lark msg send --dry-run`.
type sendDryRun struct {
	DryRun        bool            `json:"dry_run"`
	OK            bool            `json:"ok"`
	Identity      sendIdentity    `json:"identity"`
	Recipient     *sendRecipient  `json:"recipient"`
	ReceiveIDType string          `json:"receive_id_type"`
	ReceiveID     string          `json:"receive_id"`
	Routing       string          `json:"routing,omitempty"`
	Chat          *sendChatInfo   `json:"chat,omitempty"`
	Message       sendMessagePlan `json:"message"`
	Problems      []string        `json:"problems,omitempty"`
	Warnings      []string        `json:"warnings,omitempty"`
}

// sendDryRunInput is everything buildSendDryRun needs, gathered from flags.
type sendDryRunInput struct {
	AsUser        bool
	Recipient     *sendRecipient
	ReceiveIDType string // after DM-cache routing
	ReceiveID     string
	MsgType       string // text or post (ignored for files)
	Text          string
	Attachments   []sendAttachment
	FileCount     int
	ParentID      string
	RootID        string
}

// buildSendDryRun resolves identity and destination details through read-only
// lookups and assembles the report. It never uploads or sends anything.
func buildSendDryRun(dir recipientDirectory, in sendDryRunInput) *sendDryRun {
	r := &sendDryRun{
		DryRun:        true,
		Recipient:     in.Recipient,
		ReceiveIDType: in.ReceiveIDType,
		ReceiveID:     in.ReceiveID,
	}

	// Identity
	if in.AsUser {
		r.Identity = sendIdentity{As: "user"}
		if u, err := dir.CurrentUser(); err == nil && u != nil {
			r.Identity.Name, r.Identity.Email, r.Identity.OpenID = u.Name, u.Email, u.OpenID
		} else {
			r.Problems = append(r.Problems, fmt.Sprintf("cannot read the signed-in user (run `lark auth login`): %v", err))
		}
	} else {
		r.Identity = sendIdentity{As: "bot", AppID: dir.AppID(), Note: "sent by the app bot; the recipient must be able to receive messages from this bot"}
	}

	// Recipient details for raw IDs
	if in.Recipient != nil {
		if err := enrichRecipient(dir, in.Recipient); err != nil {
			r.Problems = append(r.Problems, err.Error())
		}
	}

	// Routing / chat
	if in.Recipient != nil && in.Recipient.ReceiveIDType != "chat_id" && in.ReceiveIDType == "chat_id" {
		r.Routing = "verified_dm_cache"
	} else {
		r.Routing = "direct"
	}
	if in.ReceiveIDType == "chat_id" {
		r.Chat = describeSendChat(dir, in.ReceiveID, in.AsUser, &r.Warnings)
	} else {
		r.Chat = &sendChatInfo{ChatMode: "p2p", Note: "no verified 1:1 chat_id cached; Lark delivers to the 1:1 chat between the sender and this person (created if needed)"}
	}
	if r.Chat != nil && r.Chat.ChatMode == "p2p" && r.Chat.Counterpart != nil && in.Recipient != nil &&
		in.Recipient.Person != nil && in.Recipient.Person.OpenID != "" &&
		r.Chat.Counterpart.OpenID != "" && r.Chat.Counterpart.OpenID != in.Recipient.Person.OpenID {
		r.Problems = append(r.Problems, fmt.Sprintf("routed chat %s belongs to %s, not %s", in.ReceiveID, r.Chat.Counterpart.OpenID, in.Recipient.Person.OpenID))
	}

	// Message plan
	if in.FileCount > 0 {
		r.Message = sendMessagePlan{MsgType: "file", Count: in.FileCount}
		if in.FileCount == 1 && len(in.Attachments) > 0 && in.Attachments[0].MsgType != "" {
			r.Message.MsgType = in.Attachments[0].MsgType
		}
	} else {
		r.Message = sendMessagePlan{MsgType: in.MsgType, Count: 1, TextChars: len([]rune(in.Text))}
	}
	r.Message.Attachments = in.Attachments
	r.Message.ParentID, r.Message.RootID = in.ParentID, in.RootID
	if in.ParentID != "" {
		r.Warnings = append(r.Warnings, "--parent-id is set: the message is delivered as a reply in the parent message's chat, not to --to")
	}
	for _, a := range in.Attachments {
		if a.Problem != "" {
			r.Problems = append(r.Problems, a.Problem)
		}
	}

	r.OK = len(r.Problems) == 0
	return r
}

// describeSendChat fetches chat metadata. For a 1:1 chat Lark returns no
// useful name, so the counterpart is read from the member list instead.
func describeSendChat(dir recipientDirectory, chatID string, asUser bool, warnings *[]string) *sendChatInfo {
	info := &sendChatInfo{ChatID: chatID}
	chat, err := dir.Chat(chatID, asUser)
	if err != nil || chat == nil {
		*warnings = append(*warnings, fmt.Sprintf("could not read chat %s as %s: %v", chatID, identityName(asUser), err))
		return info
	}
	info.ChatMode, info.ChatType, info.Name = chat.ChatMode, chat.ChatType, chat.Name
	info.UserCount, info.BotCount, info.External = int(chat.UserCount), int(chat.BotCount), chat.External
	if chat.ChatMode == "p2p" {
		if openID, name := dir.P2PCounterpart(chatID); openID != "" {
			info.Counterpart = describePerson(dir, openID, name, nil)
		} else {
			*warnings = append(*warnings, "p2p chat counterpart could not be determined (bot DM or note-to-self)")
		}
	}
	return info
}

func identityName(asUser bool) string {
	if asUser {
		return "user"
	}
	return "bot"
}
