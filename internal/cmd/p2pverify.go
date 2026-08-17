package cmd

import (
	"strings"
	"sync"

	"github.com/yjwong/lark-cli/internal/api"
)

// P2P chat_id attribution has to be verified, never inferred.
//
// Lark exposes no read-only way to map an open_id to a 1:1 chat_id, so the CLI
// discovers them by scanning recent messages. The signals that scan produces
// (message sender, chat display name) are suggestive but not proof: a message
// *we* sent inside someone else's DM has our own open_id as its sender, so
// "sender == the person I asked about" happily returns an unrelated chat when the
// person asked about is ourselves. Acting on that means reading — or replying
// into — the wrong person's DM.
//
// So every candidate chat_id is checked against the chat's actual membership
// before it is returned or cached.

var (
	selfOpenIDOnce  sync.Once
	selfOpenIDValue string
)

// selfOpenID returns the authenticated user's own open_id, fetched once per
// process. Returns "" if it can't be determined, in which case verification
// fails closed (see p2pMembersMatch).
func selfOpenID(client *api.Client) string {
	selfOpenIDOnce.Do(func() {
		user, err := client.GetCurrentUser()
		if err == nil && user != nil {
			selfOpenIDValue = user.OpenID
		}
	})
	return selfOpenIDValue
}

// p2pMembersMatch classifies a chat's member list against the 1:1 chat we are
// looking for.
//
// ok is true only when membership alone proves the match. soloChat is true for
// the one shape membership cannot settle: a chat whose only member is us. Lark
// omits bots from member lists, so a DM with a bot and the note-to-self chat both
// come back as exactly [self] — telling them apart needs a second signal (see
// verifyP2PChat).
//
// Rules:
//   - self must be known; an unknown self can't be excluded from the member list,
//     so we refuse rather than guess
//   - a third party means this isn't the 1:1 we asked for (also rejects group
//     chats a search result mislabelled as P2P)
//   - otherwise the target must appear in the member list
func p2pMembersMatch(memberIDs []string, self, target string) (ok bool, soloChat bool) {
	if self == "" || target == "" || len(memberIDs) == 0 {
		return false, false
	}
	targetSeen := false
	othersSeen := false
	for _, id := range memberIDs {
		switch id {
		case self:
			// our own membership is expected in any of our chats
		case target:
			targetSeen = true
			othersSeen = true
		default:
			return false, false // third party -> not the 1:1 we asked for
		}
	}
	if !othersSeen {
		// Only us in the member list: note-to-self, or a DM with a bot.
		return false, true
	}
	return targetSeen, false
}

// verifyP2PChat confirms chatID is the 1:1 chat between us and openID. Any
// error, or any doubt, returns false.
func verifyP2PChat(client *api.Client, chatID, openID string) bool {
	if chatID == "" || openID == "" {
		return false
	}
	self := selfOpenID(client)
	if self == "" {
		return false
	}
	// A 1:1 chat has at most 2 members; asking for a few more is enough to spot
	// a group chat without paginating.
	members, _, _, err := client.ListChatMembers(chatID, 10, "", true)
	if err != nil {
		return false
	}
	ids := make([]string, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.MemberID)
	}
	ok, soloChat := p2pMembersMatch(ids, self, openID)
	if !soloChat {
		return ok
	}

	// Solo member list. owner_id separates the two cases: the note-to-self chat
	// is owned by us, a bot DM has no owner.
	chat, err := client.GetChat(chatID, true)
	if err != nil || chat == nil {
		return false
	}
	if openID == self {
		return chat.OwnerID == self
	}
	if chat.OwnerID != "" {
		return false
	}
	// Bot DM: the bot isn't a member, so confirm it is the party actually talking
	// in this chat.
	return chatHasMessageFrom(client, chatID, openID)
}

// chatHasMessageFrom reports whether any recent message in the chat was sent by
// the given open_id. Used to attribute a bot DM, since bots don't appear in
// member lists.
func chatHasMessageFrom(client *api.Client, chatID, openID string) bool {
	msgs, _, _, err := listMessagesForDM(client, chatID, 50, true)
	if err != nil {
		return false
	}
	for _, m := range msgs {
		if m.Sender != nil && m.Sender.ID == openID {
			return true
		}
	}
	return false
}

// counterpartOfP2PChat returns the open_id and name of the other party in a 1:1
// chat. Returns ("", "") when the chat isn't a clean 1:1 (or on any API error),
// so callers can skip it rather than attribute it to the wrong person.
func counterpartOfP2PChat(client *api.Client, chatID string) (openID, name string) {
	if chatID == "" {
		return "", ""
	}
	self := selfOpenID(client)
	if self == "" {
		return "", ""
	}
	members, _, _, err := client.ListChatMembers(chatID, 10, "", true)
	if err != nil {
		return "", ""
	}
	var others []api.ChatMember
	for _, m := range members {
		if m.MemberID != self && strings.HasPrefix(m.MemberID, "ou_") {
			others = append(others, m)
		}
	}
	switch len(others) {
	case 0:
		// Only us in the member list: either the note-to-self chat, or a DM with a
		// bot (bots are omitted from member lists). Claim it only when the chat is
		// owned by us, which is what makes it the note-to-self chat; attributing a
		// bot DM to ourselves would file a stranger's chat_id under our open_id.
		if len(members) == 0 {
			return "", ""
		}
		if chat, err := client.GetChat(chatID, true); err == nil && chat != nil && chat.OwnerID == self {
			return self, ""
		}
		return "", ""
	case 1:
		return others[0].MemberID, others[0].Name
	default:
		return "", "" // group chat, or more than one other party
	}
}
