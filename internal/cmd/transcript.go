package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/yjwong/lark-cli/internal/api"
)

// Transcript rendering: turn raw Lark messages into one clean, fully-decoded
// text block per message that an LLM agent can read directly without writing
// any JSON-walking code. Rich content is decoded deterministically:
//
//	text        -> the text
//	a           -> label (url)
//	at          -> @Name (resolved via the name resolver)
//	img / media -> [image] / [video]  (+ file_key when present)
//	code_block  -> the code, verbatim, newlines intact
//	md          -> the markdown source, verbatim
//	hr          -> ---
//	post        -> title + paragraphs joined with newlines
//	interactive -> [card] <title> + every text/markdown element
//	recalled    -> [recalled]
//
// An unrecognized tag is never dropped silently: it renders as [tag:<name>]
// plus any text it carries, so content loss is visible rather than invisible.
//
// Threads/replies are tagged with a short thread:<id8> marker and reply lines
// are indented so an agent can follow the threading.

// shortID returns a short, stable suffix of an id for compact thread tags.
func shortID(id string) string {
	if id == "" {
		return ""
	}
	// Strip a known prefix (om_, omt_, thread_) then take a tail.
	s := id
	if i := strings.IndexByte(s, '_'); i >= 0 && i < len(s)-1 {
		s = s[i+1:]
	}
	if len(s) > 8 {
		s = s[len(s)-8:]
	}
	return s
}

// decodeMentionNames builds a key -> "@Name" map from a message's mentions,
// resolving each mentioned open_id to a display name. Mentions that already
// carry a name are used as-is (and seed the resolver cache).
func decodeMentionNames(mentions []api.MessageMention, r *nameResolver) map[string]string {
	out := make(map[string]string, len(mentions))
	for _, m := range mentions {
		if m.Key == "" {
			continue
		}
		name := m.Name
		if name != "" && m.ID != "" {
			r.preset(m.ID, name)
		}
		if name == "" && m.ID != "" {
			name = r.resolve(m.ID)
		}
		if name == "" {
			name = m.ID
		}
		out[m.Key] = "@" + name
	}
	return out
}

// decodeMessageContent turns a message's body content into plain text, decoding
// every supported tag type. mentionNames maps @_user_N keys to "@Name".
func decodeMessageContent(m api.Message, mentionNames map[string]string) string {
	if m.Deleted {
		return "[recalled]"
	}
	switch m.MsgType {
	case "image":
		return decodeImageOrMedia(m.Body, "[image]")
	case "media", "file":
		// media is video; file is a generic attachment.
		label := "[video]"
		if m.MsgType == "file" {
			label = "[file]"
		}
		return decodeImageOrMedia(m.Body, label)
	case "audio":
		return "[audio]"
	case "sticker":
		return "[sticker]"
	case "interactive":
		return decodeCard(m.Body)
	case "share_chat":
		return "[shared chat]"
	case "share_user":
		return "[shared user]"
	}

	if m.Body == nil || strings.TrimSpace(m.Body.Content) == "" {
		return ""
	}

	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(m.Body.Content), &obj); err != nil {
		// Unknown shape: return the raw content rather than guess.
		return m.Body.Content
	}

	// Plain text message: {"text":"..."}
	if text, ok := obj["text"].(string); ok {
		return applyMentions(text, mentionNames)
	}

	// Post / rich text: {"title":"","content":[[{tag,...}],...]}
	if decoded, ok := decodePost(obj, mentionNames); ok {
		return decoded
	}

	// Fall back to raw content for shapes we don't model.
	return m.Body.Content
}

// applyMentions replaces @_user_N placeholder keys with their resolved @Name.
func applyMentions(text string, mentionNames map[string]string) string {
	if len(mentionNames) == 0 {
		return text
	}
	// Replace longer keys first to avoid @_user_1 clobbering @_user_10.
	keys := make([]string, 0, len(mentionNames))
	for k := range mentionNames {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, k := range keys {
		text = strings.ReplaceAll(text, k, mentionNames[k])
	}
	return text
}

// decodePost decodes a post/rich-text object into plain text. Returns ok=false
// if the object isn't a post.
func decodePost(obj map[string]interface{}, mentionNames map[string]string) (string, bool) {
	// Some posts are localized: {"zh_cn":{...},"en_us":{...}}. Pick the first
	// localized block that has content.
	if _, hasContent := obj["content"]; !hasContent {
		for _, k := range []string{"en_us", "zh_cn", "ja_jp"} {
			if loc, ok := obj[k].(map[string]interface{}); ok {
				if _, ok := loc["content"]; ok {
					return decodePost(loc, mentionNames)
				}
			}
		}
		// Pick any localized block deterministically.
		locKeys := make([]string, 0)
		for k, v := range obj {
			if _, ok := v.(map[string]interface{}); ok {
				locKeys = append(locKeys, k)
			}
		}
		sort.Strings(locKeys)
		for _, k := range locKeys {
			loc := obj[k].(map[string]interface{})
			if _, ok := loc["content"]; ok {
				return decodePost(loc, mentionNames)
			}
		}
		return "", false
	}

	lines := make([]string, 0)
	if title, ok := obj["title"].(string); ok && strings.TrimSpace(title) != "" {
		lines = append(lines, title)
	}
	blocks, ok := obj["content"].([]interface{})
	if !ok {
		return "", false
	}
	for _, b := range blocks {
		row, ok := b.([]interface{})
		if !ok {
			continue
		}
		var sb strings.Builder
		for _, it := range row {
			cell, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			sb.WriteString(decodePostCell(cell, mentionNames))
		}
		lines = append(lines, sb.String())
	}
	return strings.Join(lines, "\n"), true
}

// decodePostCell decodes one element inside a post content row.
func decodePostCell(cell map[string]interface{}, mentionNames map[string]string) string {
	tag, _ := cell["tag"].(string)
	switch tag {
	case "text":
		t, _ := cell["text"].(string)
		return t
	case "a":
		label, _ := cell["text"].(string)
		href, _ := cell["href"].(string)
		if href == "" {
			return label
		}
		return fmt.Sprintf("%s (%s)", label, href)
	case "at":
		// Prefer an inline user_name, then a mention-key lookup, then user_id.
		if name, ok := cell["user_name"].(string); ok && name != "" {
			return "@" + name
		}
		if uid, ok := cell["user_id"].(string); ok && uid != "" {
			if mapped, ok := mentionNames["@"+uid]; ok {
				return mapped
			}
			return "@" + uid
		}
		return "@unknown"
	case "img":
		key, _ := cell["image_key"].(string)
		if key != "" {
			return fmt.Sprintf("[image %s]", key)
		}
		return "[image]"
	case "media":
		key, _ := cell["file_key"].(string)
		if key != "" {
			return fmt.Sprintf("[video %s]", key)
		}
		return "[video]"
	case "emotion":
		if k, ok := cell["emoji_type"].(string); ok && k != "" {
			return ":" + k + ":"
		}
		return ""
	case "code_block":
		// Lark wraps fenced code in its own tag. The text carries real
		// newlines, so return it verbatim; the caller indents continuation
		// lines. Dropping this loses whole payloads (keys, configs, logs).
		code, _ := cell["text"].(string)
		return code
	case "md":
		// Markdown source, already human-readable. Return as-is.
		md, _ := cell["text"].(string)
		return md
	case "hr":
		return "---"
	case "file":
		key, _ := cell["file_key"].(string)
		name, _ := cell["file_name"].(string)
		switch {
		case name != "":
			return fmt.Sprintf("[file %s]", name)
		case key != "":
			return fmt.Sprintf("[file %s]", key)
		}
		return "[file]"
	default:
		// Never drop an unknown tag silently. Surface the tag name and any
		// text it carries so missing content is visible in the transcript.
		if t, ok := cell["text"].(string); ok && t != "" {
			return fmt.Sprintf("[tag:%s] %s", tag, t)
		}
		if tag == "" {
			return ""
		}
		return fmt.Sprintf("[tag:%s]", tag)
	}
}

// decodeImageOrMedia decodes a top-level image/media/file message body. label is
// the bare form (e.g. "[image]"); when a key/name is present it is folded inside
// the brackets (e.g. "[image img_v2_abc]") to match the in-post img/media form.
func decodeImageOrMedia(body *api.MessageBody, label string) string {
	if body == nil || strings.TrimSpace(body.Content) == "" {
		return label
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(body.Content), &obj); err != nil {
		return label
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(label, "["), "]")
	for _, k := range []string{"image_key", "file_key"} {
		if key, ok := obj[k].(string); ok && key != "" {
			return fmt.Sprintf("[%s %s]", inner, key)
		}
	}
	if name, ok := obj["file_name"].(string); ok && name != "" {
		return fmt.Sprintf("[%s %s]", inner, name)
	}
	return label
}

// decodeCard decodes an interactive (card) message. The header title leads, then
// every text/markdown element in document order, one per line. Cards are the
// only way to send a rendered code block (see the messages skill), so summarizing
// one by its first element alone would hide the payload an agent came to read.
func decodeCard(body *api.MessageBody) string {
	if body == nil || strings.TrimSpace(body.Content) == "" {
		return "[card]"
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(body.Content), &obj); err != nil {
		return "[card]"
	}

	parts := make([]string, 0, 4)
	// Card header title: {"header":{"title":{"content":"..."}}}
	if header, ok := obj["header"].(map[string]interface{}); ok {
		if title, ok := header["title"].(map[string]interface{}); ok {
			if c, ok := title["content"].(string); ok && strings.TrimSpace(c) != "" {
				parts = append(parts, c)
			}
		}
	}
	collectCardText(obj["elements"], &parts)

	if len(parts) == 0 {
		return "[card]"
	}
	return "[card] " + strings.Join(parts, "\n")
}

// collectCardText walks card elements in document order, appending every
// non-empty text/markdown content it finds. Duplicates are suppressed so a
// wrapper element and its child don't both contribute the same string.
func collectCardText(v interface{}, out *[]string) {
	add := func(s string) {
		if strings.TrimSpace(s) == "" {
			return
		}
		if len(*out) > 0 && (*out)[len(*out)-1] == s {
			return
		}
		*out = append(*out, s)
	}
	switch x := v.(type) {
	case []interface{}:
		for _, e := range x {
			collectCardText(e, out)
		}
	case map[string]interface{}:
		// {"text":{"content":"..."}} (div/note) and {"content":"..."} (markdown).
		if text, ok := x["text"].(map[string]interface{}); ok {
			if c, ok := text["content"].(string); ok {
				add(c)
			}
		}
		if c, ok := x["content"].(string); ok {
			add(c)
		}
		// Nested containers: elements, fields, actions, columns.
		for _, key := range []string{"elements", "fields", "actions", "columns"} {
			if nested, ok := x[key]; ok {
				collectCardText(nested, out)
			}
		}
	}
}

// transcriptLine renders a single message as a transcript block:
//
//	[2026-05-29T10:11:12Z] Alice (thread:1a2b3c4d): line one
//	    line two
//
// Reply messages are prefixed with a "↳ " marker on the first line and the
// continuation lines are indented to match.
func transcriptLine(m api.Message, r *nameResolver) string {
	ts := formatMessageTime(m.CreateTime)

	sender := "unknown"
	if m.Sender != nil {
		switch m.Sender.SenderType {
		case "user":
			sender = r.resolve(m.Sender.ID)
		case "app":
			sender = "bot"
		case "anonymous":
			sender = "anonymous"
		default:
			if m.Sender.ID != "" {
				sender = m.Sender.ID
			}
		}
	}

	mentionNames := decodeMentionNames(m.Mentions, r)
	content := decodeMessageContent(m, mentionNames)

	var tags []string
	isReply := m.RootID != "" || m.ParentID != ""
	if m.ThreadID != "" {
		tags = append(tags, "thread:"+shortID(m.ThreadID))
	}
	tag := ""
	if len(tags) > 0 {
		tag = " (" + strings.Join(tags, " ") + ")"
	}

	prefix := ""
	// Continuation lines are always indented so multi-line content can't be
	// mistaken for a separate message; replies get a deeper indent + ↳ marker.
	indent := "    "
	if isReply {
		prefix = "↳ "
		indent = "      "
	}

	header := fmt.Sprintf("[%s] %s%s%s: ", ts, prefix, sender, tag)

	contentLines := strings.Split(content, "\n")
	if len(contentLines) <= 1 {
		return header + content
	}
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(contentLines[0])
	for _, l := range contentLines[1:] {
		sb.WriteByte('\n')
		sb.WriteString(indent)
		sb.WriteString(l)
	}
	return sb.String()
}

// renderTranscript renders a slice of messages (in display order) as a single
// transcript string, one block per message separated by newlines.
func renderTranscript(msgs []api.Message, r *nameResolver) string {
	lines := make([]string, 0, len(msgs))
	for _, m := range msgs {
		lines = append(lines, transcriptLine(m, r))
	}
	return strings.Join(lines, "\n")
}

// printTranscript writes a transcript to stdout (the transcript format bypasses
// the JSON formatter by design — it is plain text).
func printTranscript(msgs []api.Message, r *nameResolver) {
	fmt.Println(renderTranscript(msgs, r))
}

// transcriptForSearchResults renders search hits as a compact transcript. Search
// results don't carry full content, so each line shows ts, chat, sender, and the
// display_info snippet from the API.
func transcriptForSearchResults(results []api.SearchMessageResult, r *nameResolver) string {
	lines := make([]string, 0, len(results))
	for _, res := range results {
		ts := ""
		sender := "unknown"
		chat := ""
		if res.MetaData != nil {
			ts = formatMessageTime(res.MetaData.CreateTime)
			if res.MetaData.FromID != "" {
				sender = r.resolve(res.MetaData.FromID)
			}
			chat = res.MetaData.ChatID
		}
		snippet := strings.ReplaceAll(strings.TrimSpace(res.DisplayInfo), "\n", " · ")
		lines = append(lines, fmt.Sprintf("[%s] %s @ %s: %s", ts, sender, chat, snippet))
	}
	return strings.Join(lines, "\n")
}
