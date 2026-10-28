package cmd

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/yjwong/lark-cli/internal/api"
)

type fakeMessageReader map[string]api.Message

func (f fakeMessageReader) BatchGetMessages(ids []string) ([]api.Message, error) {
	var out []api.Message
	for _, id := range ids {
		if m, ok := f[id]; ok {
			out = append(out, m)
		}
	}
	return out, nil
}

func TestPlanMessageLinkChatMessage(t *testing.T) {
	r := fakeMessageReader{"om_a": {MessageID: "om_a", ChatID: "oc_1", MessagePosition: "287", CreateTime: "1755669068068"}}
	plan, _, err := planMessageLink(r, "om_a", "oc_1")
	if err != nil || plan.Thread || plan.Position != 287 || plan.AnchorMs != "1755669068068" || plan.MessageMs != "1755669068068" {
		t.Fatalf("%+v %v", plan, err)
	}
	if _, _, err = planMessageLink(r, "om_a", "oc_other"); err == nil {
		t.Fatal("accepted a message from another chat")
	}
	r["om_a"] = api.Message{MessageID: "om_a", ChatID: "oc_1", MessagePosition: "-3", CreateTime: "1"}
	if _, _, err = planMessageLink(r, "om_a", ""); err == nil {
		t.Fatal("accepted a negative chat position")
	}
}

func TestPlanMessageLinkThreadReply(t *testing.T) {
	r := fakeMessageReader{
		"om_root":  {MessageID: "om_root", ChatID: "oc_1", MessagePosition: "287", CreateTime: "1000", ThreadID: "omt_1", ThreadMessagePosition: "-1"},
		"om_reply": {MessageID: "om_reply", ChatID: "oc_1", MessagePosition: "-3", CreateTime: "2000", ThreadID: "omt_1", ThreadMessagePosition: "1", RootID: "om_root"},
	}
	plan, _, err := planMessageLink(r, "om_reply", "")
	if err != nil || !plan.Thread || plan.Position != 287 || plan.AnchorMs != "1000" || plan.ThreadPos != 1 || plan.MessageMs != "2000" {
		t.Fatalf("%+v %v", plan, err)
	}
	// The thread root itself is an ordinary chat message.
	plan, _, err = planMessageLink(r, "om_root", "")
	if err != nil || plan.Thread || plan.Position != 287 {
		t.Fatalf("%+v %v", plan, err)
	}
	reply := r["om_reply"]
	reply.RootID = ""
	r["om_reply"] = reply
	if _, _, err = planMessageLink(r, "om_reply", ""); err == nil {
		t.Fatal("accepted a thread reply without a root")
	}
}

func TestCreateMessageLinkScriptShape(t *testing.T) {
	s := createMessageLinkScript("https://team.larksuite.com/messenger/", messageLinkPlan{Thread: true, Position: 287, AnchorMs: "1000", ThreadPos: 1, MessageMs: "2000"})
	for _, want := range []string{`"1150|messages.PutMessageLinkRequest|messages.PutMessageLinkResponse"`, `copiedId:[copied]`, `from=3;fromId=anchor.id`, `"anchorMs":"1000"`, `"threadPos":1`, `https://team.larksuite.com`} {
		if !strings.Contains(s, want) {
			t.Errorf("script missing %q", want)
		}
	}
	for _, bad := range []string{"sendMessage", "focus", "SendMessage"} {
		if strings.Contains(s, bad) {
			t.Errorf("script contains %q", bad)
		}
	}
}

func TestValidMessageLinkURL(t *testing.T) {
	got, err := validMessageLinkURL("https://applink.larksuite.com/client/message/link/open?token=AmZq%3D")
	if err != nil || got != "https://applink.larksuite.com/client/message/link/open?token=AmZq%3D" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err = validMessageLinkURL("https://evil.test/client/message/link/open?token=x"); err == nil {
		t.Fatal("accepted a foreign host")
	}
}

func TestCreateMessageLinkBackgroundLifecycle(t *testing.T) {
	dir := t.TempDir()
	log := dir + "/calls"
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$LARK_TEST_CALLS"
case "$5" in
 open) echo ABCD1234;;
 eval) echo '{"url":"https://applink.larksuite.com/client/message/link/open?token=AmZq%3D"}';;
 close) :;;
 *) exit 1;;
esac
`
	if err := os.WriteFile(dir+"/browserctrl", []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("LARK_TEST_CALLS", log)
	t.Setenv("LARK_MSG_LINK_BROWSER", "dia")
	t.Setenv("LARK_MSG_LINK_WORKSPACE_URL", "https://team.larksuite.com")
	got, err := createMessageLink(context.Background(), messageLinkPlan{Position: 7, AnchorMs: "1", MessageMs: "1"})
	if err != nil || !strings.HasSuffix(got, "token=AmZq%3D") {
		t.Fatalf("%q %v", got, err)
	}
	b, _ := os.ReadFile(log)
	calls := string(b)
	for _, want := range []string{"--browser dia", " open https://team.larksuite.com/messenger/ --id", " eval ABCD1234", " close ABCD1234"} {
		if !strings.Contains(calls, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(calls, " focus ") || strings.Contains(calls, "--steal") {
		t.Fatal("must not activate or steal a tab")
	}
}
