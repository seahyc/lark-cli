package cmd

import (
	"testing"

	"github.com/yjwong/lark-cli/internal/api"
)

func TestAllBestDMMatches_ExactSingle(t *testing.T) {
	users := []api.SearchUserResult{
		{Name: "Kangrui He", OpenID: "ou_1"},
		{Name: "Kang Rui", OpenID: "ou_2"},
	}
	got := allBestDMMatches("Kangrui He", users)
	if len(got) != 1 || got[0].OpenID != "ou_1" {
		t.Fatalf("expected one exact match ou_1, got %#v", got)
	}
}

func TestAllBestDMMatches_AmbiguousPrefix(t *testing.T) {
	users := []api.SearchUserResult{
		{Name: "Kang Rui", OpenID: "ou_1"},
		{Name: "Kang Rui Tan", OpenID: "ou_2"},
	}
	got := allBestDMMatches("Kang Rui", users)
	if len(got) != 1 {
		t.Fatalf("expected exact match to win and return 1, got %d", len(got))
	}
	if got[0].OpenID != "ou_1" {
		t.Fatalf("expected ou_1, got %s", got[0].OpenID)
	}
}

func TestExtractMessageText_TextMessage(t *testing.T) {
	m := api.Message{
		MsgType: "text",
		Body:    &api.MessageBody{Content: `{"text":"hello world"}`},
	}
	if got := extractMessageText(m); got != "hello world" {
		t.Fatalf("expected plain text, got %q", got)
	}
}

func TestExtractMessageText_PostMessage(t *testing.T) {
	m := api.Message{
		MsgType: "post",
		Body: &api.MessageBody{Content: `{
			"title":"T",
			"content":[
				[
					{"tag":"text","text":"line 1"},
					{"tag":"at","user_name":"Alice"}
				],
				[
					{"tag":"img","image_key":"img_x"}
				]
			]
		}`},
	}
	got := extractMessageText(m)
	want := "T\nline 1@Alice\n[image]"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

