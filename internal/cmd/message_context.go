package cmd

import (
	"sort"

	"github.com/yjwong/lark-cli/internal/api"
)

const messageThreadPreviewLimit = 3

type messageReadContextClient interface {
	ListMessageReactions(string, *api.ListMessageReactionsOptions) ([]api.MessageReaction, bool, string, error)
	ListMessageReactionsAsUser(string, *api.ListMessageReactionsOptions) ([]api.MessageReaction, bool, string, error)
	ListMessages(string, string, *api.ListMessagesOptions) ([]api.Message, bool, string, error)
	ListMessagesAsUser(string, string, *api.ListMessagesOptions) ([]api.Message, bool, string, error)
}

// enrichMessageReadContexts fetches the small pieces of surrounding activity a
// person sees in Lark below a message. Context is intentionally best-effort:
// the original message remains readable if a reaction or thread endpoint is
// unavailable in a particular chat.
func enrichMessageReadContexts(client messageReadContextClient, messages []api.Message, asUser bool, resolver *nameResolver) map[string]*api.MessageReadContext {
	contexts := make(map[string]*api.MessageReadContext, len(messages))
	for _, message := range messages {
		if message.MessageID == "" {
			continue
		}
		reactions := listAllMessageReactions(client, message.MessageID, asUser)
		var replies []api.Message
		if message.ThreadID != "" && message.RootID == "" && message.ParentID == "" {
			replies = listAllThreadReplies(client, message.ThreadID, asUser)
		}
		if context := buildMessageReadContext(reactions, replies, resolver); context != nil {
			contexts[message.MessageID] = context
		}
	}
	return contexts
}

func listAllMessageReactions(client messageReadContextClient, messageID string, asUser bool) []api.MessageReaction {
	var all []api.MessageReaction
	pageToken := ""
	for {
		opts := &api.ListMessageReactionsOptions{PageSize: 50, PageToken: pageToken}
		var (
			items   []api.MessageReaction
			hasMore bool
			next    string
			err     error
		)
		if asUser {
			items, hasMore, next, err = client.ListMessageReactionsAsUser(messageID, opts)
		} else {
			items, hasMore, next, err = client.ListMessageReactions(messageID, opts)
		}
		if err != nil {
			return all
		}
		all = append(all, items...)
		if !hasMore || next == "" {
			return all
		}
		pageToken = next
	}
}

func listAllThreadReplies(client messageReadContextClient, threadID string, asUser bool) []api.Message {
	var all []api.Message
	pageToken := ""
	for {
		opts := &api.ListMessagesOptions{PageSize: 50, PageToken: pageToken, SortType: "ByCreateTimeAsc"}
		var (
			items   []api.Message
			hasMore bool
			next    string
			err     error
		)
		if asUser {
			items, hasMore, next, err = client.ListMessagesAsUser("thread", threadID, opts)
		} else {
			items, hasMore, next, err = client.ListMessages("thread", threadID, opts)
		}
		if err != nil {
			return all
		}
		all = append(all, items...)
		if !hasMore || next == "" {
			return all
		}
		pageToken = next
	}
}

// buildMessageReadContext turns the surrounding Lark activity into the same
// compact context a person sees below a message: grouped reactions and the
// latest thread replies.
func buildMessageReadContext(reactions []api.MessageReaction, replies []api.Message, resolver *nameResolver) *api.MessageReadContext {
	context := &api.MessageReadContext{}
	if len(reactions) > 0 {
		byEmoji := make(map[string]*api.MessageReactionSummary)
		order := make([]string, 0)
		for _, reaction := range reactions {
			emoji := ""
			if reaction.ReactionType != nil {
				emoji = reaction.ReactionType.EmojiType
			}
			if emoji == "" {
				continue
			}
			summary, ok := byEmoji[emoji]
			if !ok {
				summary = &api.MessageReactionSummary{EmojiType: emoji}
				byEmoji[emoji] = summary
				order = append(order, emoji)
			}
			summary.Count++
			if reaction.Operator != nil && reaction.Operator.OperatorID != "" {
				summary.Operators = append(summary.Operators, resolver.resolve(reaction.Operator.OperatorID))
			}
		}
		sort.Strings(order)
		context.Reactions = make([]api.MessageReactionSummary, 0, len(order))
		for _, emoji := range order {
			context.Reactions = append(context.Reactions, *byEmoji[emoji])
		}
	}

	if replies != nil {
		previewStart := len(replies) - messageThreadPreviewLimit
		if previewStart < 0 {
			previewStart = 0
		}
		preview := make([]api.MessageThreadPreviewItem, 0, len(replies)-previewStart)
		for _, reply := range replies[previewStart:] {
			sender := "unknown"
			if reply.Sender != nil {
				switch reply.Sender.SenderType {
				case "user":
					sender = resolver.resolve(reply.Sender.ID)
				case "app":
					sender = "bot"
				default:
					sender = reply.Sender.ID
				}
			}
			preview = append(preview, api.MessageThreadPreviewItem{
				Sender:  sender,
				Content: decodeMessageContent(reply, decodeMentionNames(reply.Mentions, resolver)),
			})
		}
		context.Thread = &api.MessageThreadPreview{
			ReplyCount: len(replies),
			Preview:    preview,
		}
	}

	if len(context.Reactions) == 0 && context.Thread == nil {
		return nil
	}
	return context
}
