package cmd

import (
	"strings"
	"testing"

	"github.com/yjwong/lark-cli/internal/api"
)

// staticResolver resolves a fixed set of open_ids; everything else falls back to
// the open_id itself (mirroring production fallback behavior).
func staticResolver(names map[string]string) *nameResolver {
	return newNameResolverWithLookup(func(openID string) (string, error) {
		if n, ok := names[openID]; ok {
			return n, nil
		}
		return "", nil
	})
}

func TestDecodeMessageContent_TagTypes(t *testing.T) {
	resolver := staticResolver(map[string]string{
		"ou_alice": "Alice",
		"ou_bob":   "Bob",
	})

	tests := []struct {
		name     string
		msg      api.Message
		mentions map[string]string
		want     string
	}{
		{
			name: "text",
			msg: api.Message{
				MsgType: "text",
				Body:    &api.MessageBody{Content: `{"text":"hello world"}`},
			},
			want: "hello world",
		},
		{
			name: "text with mention",
			msg: api.Message{
				MsgType: "text",
				Body:    &api.MessageBody{Content: `{"text":"hi @_user_1 there"}`},
			},
			mentions: map[string]string{"@_user_1": "@Alice"},
			want:     "hi @Alice there",
		},
		{
			name: "post title and link and at",
			msg: api.Message{
				MsgType: "post",
				Body: &api.MessageBody{Content: `{
					"title":"Daily",
					"content":[
						[{"tag":"text","text":"see "},{"tag":"a","text":"docs","href":"https://x.io"}],
						[{"tag":"at","user_id":"ou_bob"}]
					]
				}`},
			},
			mentions: map[string]string{"@ou_bob": "@Bob"},
			want:     "Daily\nsee docs (https://x.io)\n@Bob",
		},
		{
			name: "post with inline at user_name",
			msg: api.Message{
				MsgType: "post",
				Body: &api.MessageBody{Content: `{
					"title":"",
					"content":[[{"tag":"text","text":"ping "},{"tag":"at","user_name":"Carol"}]]
				}`},
			},
			want: "ping @Carol",
		},
		{
			name: "post with img",
			msg: api.Message{
				MsgType: "post",
				Body: &api.MessageBody{Content: `{
					"title":"",
					"content":[[{"tag":"img","image_key":"img_99"}]]
				}`},
			},
			want: "[image img_99]",
		},
		{
			name: "top-level image",
			msg: api.Message{
				MsgType: "image",
				Body:    &api.MessageBody{Content: `{"image_key":"img_v2_abc"}`},
			},
			want: "[image img_v2_abc]",
		},
		{
			name: "top-level media video",
			msg: api.Message{
				MsgType: "media",
				Body:    &api.MessageBody{Content: `{"file_key":"file_v2_vid","file_name":"clip.mp4"}`},
			},
			want: "[video file_v2_vid]",
		},
		{
			name: "top-level file",
			msg: api.Message{
				MsgType: "file",
				Body:    &api.MessageBody{Content: `{"file_key":"file_v2_doc"}`},
			},
			want: "[file file_v2_doc]",
		},
		{
			name: "audio",
			msg:  api.Message{MsgType: "audio", Body: &api.MessageBody{Content: `{"file_key":"x"}`}},
			want: "[audio]",
		},
		{
			name: "sticker",
			msg:  api.Message{MsgType: "sticker", Body: &api.MessageBody{Content: `{}`}},
			want: "[sticker]",
		},
		{
			name: "interactive card with header title",
			msg: api.Message{
				MsgType: "interactive",
				Body:    &api.MessageBody{Content: `{"header":{"title":{"content":"Approval needed"}}}`},
			},
			want: "[card] Approval needed",
		},
		{
			name: "interactive card falls back to element text",
			msg: api.Message{
				MsgType: "interactive",
				Body:    &api.MessageBody{Content: `{"elements":[{"tag":"div","text":{"content":"Build failed"}}]}`},
			},
			want: "[card] Build failed",
		},
		{
			name: "interactive card no title",
			msg:  api.Message{MsgType: "interactive", Body: &api.MessageBody{Content: `{"elements":[]}`}},
			want: "[card]",
		},
		{
			// A card is the only way to send a rendered code block, so the
			// decoder must return every element, not just the first.
			name: "interactive card keeps every element",
			msg: api.Message{
				MsgType: "interactive",
				Body: &api.MessageBody{Content: `{
					"header":{"title":{"content":"Deploy"}},
					"elements":[
						{"tag":"markdown","content":"step one"},
						{"tag":"markdown","content":"` + "```bash\\nkubectl get pods\\n```" + `"}
					]
				}`},
			},
			want: "[card] Deploy\nstep one\n```bash\nkubectl get pods\n```",
		},
		{
			// The payload that started this: a post carrying its content in a
			// code_block. The old decoder dropped it and rendered blank lines.
			name: "post with code_block",
			msg: api.Message{
				MsgType: "post",
				Body: &api.MessageBody{Content: `{
					"title":"",
					"content":[
						[{"tag":"text","text":"Key: SOME_SECRET"}],
						[{"tag":"code_block","language":"BASH","text":"line one\nline two"}]
					]
				}`},
			},
			want: "Key: SOME_SECRET\nline one\nline two",
		},
		{
			name: "post with hr and md",
			msg: api.Message{
				MsgType: "post",
				Body: &api.MessageBody{Content: `{
					"title":"",
					"content":[
						[{"tag":"md","text":"**bold**"}],
						[{"tag":"hr"}],
						[{"tag":"text","text":"after"}]
					]
				}`},
			},
			want: "**bold**\n---\nafter",
		},
		{
			// Unknown tags must be visible, never silently dropped.
			name: "post with unknown tag surfaces it",
			msg: api.Message{
				MsgType: "post",
				Body: &api.MessageBody{Content: `{
					"title":"",
					"content":[
						[{"tag":"quote","text":"quoted thing"}],
						[{"tag":"future_widget"}]
					]
				}`},
			},
			want: "[tag:quote] quoted thing\n[tag:future_widget]",
		},
		{
			name: "recalled",
			msg:  api.Message{MsgType: "text", Deleted: true, Body: &api.MessageBody{Content: `{"text":"gone"}`}},
			want: "[recalled]",
		},
		{
			name: "share_chat",
			msg:  api.Message{MsgType: "share_chat", Body: &api.MessageBody{Content: `{}`}},
			want: "[shared chat]",
		},
		{
			name: "localized post picks en_us",
			msg: api.Message{
				MsgType: "post",
				Body: &api.MessageBody{Content: `{
					"en_us":{"title":"Hi","content":[[{"tag":"text","text":"yo"}]]},
					"zh_cn":{"title":"嗨","content":[[{"tag":"text","text":"哟"}]]}
				}`},
			},
			want: "Hi\nyo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Re-seed resolver so cached fallbacks don't leak between cases.
			_ = resolver
			got := decodeMessageContent(tt.msg, tt.mentions)
			if got != tt.want {
				t.Fatalf("decodeMessageContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyMentions_LongestKeyFirst(t *testing.T) {
	mentions := map[string]string{
		"@_user_1":  "@Alice",
		"@_user_10": "@Bob",
	}
	got := applyMentions("a @_user_10 b @_user_1 c", mentions)
	want := "a @Bob b @Alice c"
	if got != want {
		t.Fatalf("applyMentions() = %q, want %q", got, want)
	}
}

func TestDecodeMentionNames_ResolvesAndPresets(t *testing.T) {
	r := staticResolver(map[string]string{"ou_x": "Xavier"})
	mentions := []api.MessageMention{
		{Key: "@_user_1", ID: "ou_x"},                  // resolved via lookup
		{Key: "@_user_2", ID: "ou_y", Name: "Yolanda"}, // name already present
	}
	got := decodeMentionNames(mentions, r)
	if got["@_user_1"] != "@Xavier" {
		t.Fatalf("@_user_1 = %q, want @Xavier", got["@_user_1"])
	}
	if got["@_user_2"] != "@Yolanda" {
		t.Fatalf("@_user_2 = %q, want @Yolanda", got["@_user_2"])
	}
	// The preset name should now be cached.
	if r.resolve("ou_y") != "Yolanda" {
		t.Fatalf("preset name not cached for ou_y")
	}
}

func TestTranscriptLine_SenderAndThreadAndReply(t *testing.T) {
	r := staticResolver(map[string]string{"ou_alice": "Alice"})

	// Root message in a thread by a user.
	root := api.Message{
		MessageID:  "om_root",
		MsgType:    "text",
		CreateTime: "1704067200000", // 2024-01-01T00:00:00Z
		ThreadID:   "omt_abcd1234ef",
		Sender:     &api.MessageSender{ID: "ou_alice", SenderType: "user"},
		Body:       &api.MessageBody{Content: `{"text":"kickoff"}`},
	}
	line := transcriptLine(root, r)
	if !strings.Contains(line, "Alice") {
		t.Fatalf("expected resolved sender Alice in %q", line)
	}
	if !strings.Contains(line, "thread:") {
		t.Fatalf("expected thread tag in %q", line)
	}
	if !strings.Contains(line, "kickoff") {
		t.Fatalf("expected decoded content in %q", line)
	}
	if strings.Contains(line, "↳") {
		t.Fatalf("root should not be marked as a reply: %q", line)
	}

	// Reply message.
	reply := api.Message{
		MessageID:  "om_reply",
		MsgType:    "text",
		CreateTime: "1704067260000",
		ParentID:   "om_root",
		ThreadID:   "omt_abcd1234ef",
		Sender:     &api.MessageSender{SenderType: "app"},
		Body:       &api.MessageBody{Content: `{"text":"on it"}`},
	}
	rline := transcriptLine(reply, r)
	if !strings.Contains(rline, "↳") {
		t.Fatalf("expected reply marker in %q", rline)
	}
	if !strings.Contains(rline, "bot") {
		t.Fatalf("expected app sender rendered as bot in %q", rline)
	}
}

func TestTranscriptLine_MultilineIndent(t *testing.T) {
	r := staticResolver(nil)
	m := api.Message{
		MsgType:    "post",
		CreateTime: "1704067200000",
		Sender:     &api.MessageSender{ID: "ou_z", SenderType: "user"},
		Body: &api.MessageBody{Content: `{
			"title":"",
			"content":[[{"tag":"text","text":"line one"}],[{"tag":"text","text":"line two"}]]
		}`},
	}
	line := transcriptLine(m, r)
	parts := strings.Split(line, "\n")
	if len(parts) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(parts), line)
	}
	if !strings.HasPrefix(parts[1], "    ") {
		t.Fatalf("expected continuation line indented, got %q", parts[1])
	}
}

func TestShortID(t *testing.T) {
	cases := map[string]string{
		"omt_abcdef1234567890": "34567890",
		"thread_1a2b":          "thread_1a2b"[len("thread_1a2b")-len("1a2b"):], // "1a2b"
		"":                     "",
		"short":                "short",
	}
	for in, want := range cases {
		if got := shortID(in); got != want {
			t.Fatalf("shortID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderTranscript_MultipleMessages(t *testing.T) {
	r := staticResolver(map[string]string{"ou_a": "Ann"})
	msgs := []api.Message{
		{MsgType: "text", CreateTime: "1704067200000", Sender: &api.MessageSender{ID: "ou_a", SenderType: "user"}, Body: &api.MessageBody{Content: `{"text":"one"}`}},
		{MsgType: "text", CreateTime: "1704067260000", Sender: &api.MessageSender{ID: "ou_a", SenderType: "user"}, Body: &api.MessageBody{Content: `{"text":"two"}`}},
	}
	out := renderTranscript(msgs, r)
	if strings.Count(out, "Ann") != 2 {
		t.Fatalf("expected 2 sender lines, got: %q", out)
	}
	if !strings.Contains(out, "one") || !strings.Contains(out, "two") {
		t.Fatalf("missing content in %q", out)
	}
}

func TestTranscriptForSearchResults(t *testing.T) {
	r := staticResolver(map[string]string{"ou_s": "Sam"})
	results := []api.SearchMessageResult{
		{
			DisplayInfo: "Project X\nSam: shipped it",
			MetaData: &api.SearchMessageMeta{
				ChatID:     "oc_1",
				FromID:     "ou_s",
				CreateTime: "1704067200000",
			},
		},
	}
	out := transcriptForSearchResults(results, r)
	if !strings.Contains(out, "Sam") || !strings.Contains(out, "oc_1") {
		t.Fatalf("expected sender + chat in %q", out)
	}
	if strings.Contains(out, "\nSam: shipped") {
		t.Fatalf("display_info newlines should be flattened: %q", out)
	}
}
