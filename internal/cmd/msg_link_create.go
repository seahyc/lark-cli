package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

// messageLinkPlan is the data the in-page script needs to turn a public message
// into Lark's internal ids. Positions and millisecond timestamps come from the
// Open API; the script uses them to find the internal chat and message ids and
// verifies each match by creation time before creating the link.
type messageLinkPlan struct {
	Thread       bool   `json:"thread"`
	Position     int64  `json:"position"`     // chat position of the anchor (the message, or the thread root)
	AnchorMs     string `json:"anchorMs"`     // create time of the anchor in ms
	ThreadPos    int64  `json:"threadPos"`    // position inside the thread (thread replies only)
	MessageMs    string `json:"messageMs"`    // create time of the target message in ms
	InternalChat string `json:"internalChat"` // optional internal chat id override
}

type messageLinkReader interface {
	BatchGetMessages([]string) ([]api.Message, error)
}

func parsePosition(label, value string) (int64, error) {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("message has no usable %s (%q)", label, value)
	}
	return n, nil
}

func fetchOneMessage(client messageLinkReader, id string) (*api.Message, error) {
	messages, err := client.BatchGetMessages([]string{id})
	if err != nil {
		return nil, err
	}
	if len(messages) != 1 {
		return nil, fmt.Errorf("message %s was not returned by the API", id)
	}
	return &messages[0], nil
}

// planMessageLink reads the message (and the thread root for thread replies)
// through the Open API. expectedChat, when set, must match the message's chat.
func planMessageLink(client messageLinkReader, messageID, expectedChat string) (*messageLinkPlan, *api.Message, error) {
	m, err := fetchOneMessage(client, messageID)
	if err != nil {
		return nil, nil, err
	}
	if m.Deleted {
		return nil, nil, fmt.Errorf("message %s was recalled", messageID)
	}
	if expectedChat != "" && m.ChatID != "" && m.ChatID != expectedChat {
		return nil, nil, fmt.Errorf("message belongs to chat %s, not %s", m.ChatID, expectedChat)
	}
	if m.CreateTime == "" {
		return nil, nil, fmt.Errorf("message has no create time")
	}
	plan := &messageLinkPlan{MessageMs: m.CreateTime}
	if m.ThreadID != "" && m.ThreadMessagePosition != "" && m.ThreadMessagePosition != "-1" {
		if plan.ThreadPos, err = parsePosition("thread position", m.ThreadMessagePosition); err != nil || plan.ThreadPos < 0 {
			return nil, nil, fmt.Errorf("message has no usable thread position (%q)", m.ThreadMessagePosition)
		}
		rootID := m.RootID
		if rootID == "" {
			return nil, nil, fmt.Errorf("thread reply has no root message id")
		}
		root, err := fetchOneMessage(client, rootID)
		if err != nil {
			return nil, nil, err
		}
		if plan.Position, err = parsePosition("root position", root.MessagePosition); err != nil || plan.Position <= 0 {
			return nil, nil, fmt.Errorf("thread root has no usable chat position (%q)", root.MessagePosition)
		}
		plan.Thread, plan.AnchorMs = true, root.CreateTime
		return plan, m, nil
	}
	if plan.Position, err = parsePosition("chat position", m.MessagePosition); err != nil || plan.Position <= 0 {
		return nil, nil, fmt.Errorf("message has no usable chat position (%q); topic-style chats are not supported", m.MessagePosition)
	}
	plan.AnchorMs = m.CreateTime
	return plan, m, nil
}

// createMessageLinkScript returns the in-page program. Internal request shape,
// taken from the web client's Copy link action:
//
//	1150 PutMessageLinkRequest {fromId, from, copiedId: [messageId]}
//	from 1 = CHAT (fromId = chat id), 3 = MESSAGE_THREAD (fromId = thread id)
//
// Internal ids are numeric strings. The chat id is found among the feed chats by
// pulling the message at its known position and matching its create time.
func createMessageLinkScript(origin string, plan messageLinkPlan) string {
	input, _ := json.Marshal(plan)
	return `(async()=>{` + messengerGuardJS(origin) + `const P=` + string(input) + `;
const call=(c,p)=>LarkAPI.transport.callServerApi(c,p);
const feed=()=>[...new Set([...document.querySelectorAll("[data-feed-id]")].map(e=>e.getAttribute("data-feed-id")))];
let chats=P.internalChat?[P.internalChat]:feed();
for(let i=0;i<100&&!chats.length;i++){await new Promise(r=>setTimeout(r,100));chats=feed();}
let chatId,anchor;
for(const c of chats){try{const r=await call("43|messages.PullMessagesByPositionsRequest|messages.PullMessagesByPositionsResponse",{chatId:c,positions:[P.position]});const m=r.data?.messages?.[String(P.position)];if(m&&m.createTimeMs===P.anchorMs){chatId=c;anchor=m;break;}}catch(e){}}
if(!chatId)throw Error("chat not found among Messenger feed chats");
let from=1,fromId=chatId,copied=anchor.id;
if(P.thread){const r=await call("8109|thread.PullThreadMessagesByPositionsRequest|thread.PullThreadMessagesByPositionsResponse",{threadId:anchor.id,positions:[P.threadPos]});const m=r.data?.messages?.[String(P.threadPos)];if(!m||m.createTimeMs!==P.messageMs)throw Error("thread message not found");from=3;fromId=anchor.id;copied=m.id;}
else if(anchor.createTimeMs!==P.messageMs)throw Error("message mismatch");
const r=await call("1150|messages.PutMessageLinkRequest|messages.PutMessageLinkResponse",{fromId:fromId,from:from,copiedId:[copied]});
return JSON.stringify({url:r.data?.tokenUrl});})()`
}

func validMessageLinkURL(raw string) (string, error) {
	u, err := larkMessageURL(raw)
	if err != nil {
		return "", fmt.Errorf("Lark returned an unexpected link")
	}
	return u.Scheme + "://" + u.Host + u.Path + "?token=" + url.QueryEscape(u.Query().Get("token")), nil
}

func createMessageLink(ctx context.Context, plan messageLinkPlan) (string, error) {
	b, err := evalInMessengerTab(ctx, func(origin string) string { return createMessageLinkScript(origin, plan) })
	if err != nil {
		return "", err
	}
	var response struct {
		URL string `json:"url"`
	}
	if json.Unmarshal(b, &response) != nil || response.URL == "" {
		return "", fmt.Errorf("Lark did not return a message link")
	}
	return validMessageLinkURL(response.URL)
}

var msgLinkChatID, msgLinkInternalChatID string

var msgLinkCmd = &cobra.Command{
	Use:   "link <message-id>",
	Short: "Create a shareable link for a message (same as Copy link in the web client)",
	Long: `Create the applink URL the web client's "Copy link" produces for a message.
Works for chat messages and thread replies. A link to a thread reply opens the
thread (root plus replies inline), as Lark's own Copy link does; "lark msg get"
on such a link returns the thread root. Lark has no public API for this, so
the link is created from a background tab using the signed-in Lark web session.
The message's chat must appear in the Messenger chat list (recent chats).

Examples:
  lark msg link om_xxx
  lark msg link om_xxx --chat-id oc_yyy --browser dia --workspace-url https://team.larksuite.com
  lark msg get "$(lark msg link om_xxx)" --format text`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := strings.TrimSpace(args[0])
		if !strings.HasPrefix(id, "om_") {
			output.Fatalf("VALIDATION_ERROR", "message ID must start with om_")
		}
		if msgLinkChatID != "" && !strings.HasPrefix(msgLinkChatID, "oc_") {
			output.Fatalf("VALIDATION_ERROR", "--chat-id must start with oc_")
		}
		plan, _, err := planMessageLink(api.NewClient(), id, msgLinkChatID)
		if err != nil {
			output.Fatal("API_ERROR", err)
			return
		}
		plan.InternalChat = msgLinkInternalChatID
		link, err := createMessageLink(cmd.Context(), *plan)
		if err != nil {
			output.Fatal("LINK_CREATION_ERROR", err)
			return
		}
		fmt.Println(link)
	},
}

func init() {
	msgLinkCmd.Flags().StringVar(&msgLinkChatID, "chat-id", "", "Expected chat (oc_...); fails if the message belongs to another chat")
	msgLinkCmd.Flags().StringVar(&msgLinkInternalChatID, "internal-chat-id", "", "Numeric Lark chat id to use when the chat is not in the Messenger feed")
	msgCmd.AddCommand(msgLinkCmd)
}
