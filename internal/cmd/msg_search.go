package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

// --- msg search ---

var (
	msgSearchChatID    string
	msgSearchSender    string
	msgSearchAfter     string
	msgSearchBefore    string
	msgSearchType      string
	msgSearchLimit     int
)

var msgSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search messages across chats (user token)",
	Long: `Search messages across all chats visible to the authenticated user.

This command always uses the user token (im:message:search scope).

Examples:
  lark msg search "incident"
  lark msg search "review" --chat-id oc_xxx --limit 20
  lark msg search "report" --after 2026-04-01 --before 2026-04-15
  lark msg search "design" --sender ou_xxx --type text`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		opts := &api.SearchMessagesOptions{
			ChatID:   msgSearchChatID,
			SenderID: msgSearchSender,
			MsgType:  msgSearchType,
		}
		if msgSearchAfter != "" {
			opts.StartTime = parseTimeArg(msgSearchAfter)
		}
		if msgSearchBefore != "" {
			opts.EndTime = parseTimeArg(msgSearchBefore)
		}

		client := api.NewClient()
		var allMessages []api.SearchMessageResult
		var pageToken string
		hasMore := true
		remaining := msgSearchLimit
		for hasMore {
			pageSize := 20
			if remaining > 0 && remaining < pageSize {
				pageSize = remaining
			}
			opts.PageSize = pageSize
			opts.PageToken = pageToken
			messages, more, nextToken, err := client.SearchMessages(query, opts)
			if err != nil {
				output.Fatal("API_ERROR", err)
			}
			allMessages = append(allMessages, messages...)
			hasMore = more
			pageToken = nextToken
			if msgSearchLimit > 0 {
				remaining = msgSearchLimit - len(allMessages)
				if remaining <= 0 {
					break
				}
			}
		}
		if msgSearchLimit > 0 && len(allMessages) > msgSearchLimit {
			allMessages = allMessages[:msgSearchLimit]
		}

		output.JSON(map[string]interface{}{
			"query":    query,
			"messages": allMessages,
			"count":    len(allMessages),
		})
	},
}

// --- msg get ---

var msgGetCmd = &cobra.Command{
	Use:   "get <message-ids>",
	Short: "Batch fetch messages by ID (comma-separated)",
	Long: `Fetch one or more messages by ID.

Pass multiple IDs as a comma-separated list.

Examples:
  lark msg get om_xxx
  lark msg get om_aaa,om_bbb,om_ccc`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		raw := args[0]
		parts := strings.Split(raw, ",")
		ids := make([]string, 0, len(parts))
		for _, p := range parts {
			id := strings.TrimSpace(p)
			if id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			output.Fatalf("VALIDATION_ERROR", "at least one message ID is required")
		}

		client := api.NewClient()
		messages, err := client.BatchGetMessages(ids)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}

		outputMessages := make([]api.OutputMessage, len(messages))
		for i, m := range messages {
			outputMessages[i] = convertMessage(m)
		}
		output.JSON(map[string]interface{}{
			"messages": outputMessages,
			"count":    len(outputMessages),
		})
	},
}

// --- msg forward ---

var (
	msgForwardMessageID string
	msgForwardTo        string
	msgForwardToType    string
	msgForwardAs        string
)

var msgForwardCmd = &cobra.Command{
	Use:   "forward",
	Short: "Forward a message to another chat or user",
	Long: `Forward an existing message to another recipient.

Examples:
  lark msg forward --message-id om_xxx --to oc_yyy
  lark msg forward --message-id om_xxx --to ou_yyy --as user`,
	Run: func(cmd *cobra.Command, args []string) {
		if msgForwardMessageID == "" {
			output.Fatalf("VALIDATION_ERROR", "--message-id is required")
		}
		if msgForwardTo == "" {
			output.Fatalf("VALIDATION_ERROR", "--to is required")
		}
		if msgForwardAs != "" && msgForwardAs != "bot" && msgForwardAs != "user" {
			output.Fatalf("VALIDATION_ERROR", "--as must be 'bot' or 'user'")
		}
		asUser := msgForwardAs == "user"

		receiveIDType := msgForwardToType
		if receiveIDType == "" {
			receiveIDType = detectIDType(msgForwardTo)
		}

		client := api.NewClient()
		resp, err := client.ForwardMessage(msgForwardMessageID, receiveIDType, msgForwardTo, asUser)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(api.OutputSendMessage{
			Success:    true,
			MessageID:  resp.Data.MessageID,
			ChatID:     resp.Data.ChatID,
			CreateTime: formatMessageTime(resp.Data.CreateTime),
		})
	},
}

// --- msg merge-forward ---

var (
	msgMergeForwardMessageIDs []string
	msgMergeForwardTo         string
	msgMergeForwardToType     string
	msgMergeForwardAs         string
)

var msgMergeForwardCmd = &cobra.Command{
	Use:   "merge-forward",
	Short: "Merge-forward multiple messages to another recipient",
	Long: `Merge-forward a set of messages as a single combined message.

Examples:
  lark msg merge-forward --message-ids om_a,om_b,om_c --to oc_yyy
  lark msg merge-forward --message-ids om_a,om_b --to ou_yyy --as user`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(msgMergeForwardMessageIDs) == 0 {
			output.Fatalf("VALIDATION_ERROR", "--message-ids is required")
		}
		if msgMergeForwardTo == "" {
			output.Fatalf("VALIDATION_ERROR", "--to is required")
		}
		if msgMergeForwardAs != "" && msgMergeForwardAs != "bot" && msgMergeForwardAs != "user" {
			output.Fatalf("VALIDATION_ERROR", "--as must be 'bot' or 'user'")
		}
		asUser := msgMergeForwardAs == "user"

		receiveIDType := msgMergeForwardToType
		if receiveIDType == "" {
			receiveIDType = detectIDType(msgMergeForwardTo)
		}

		client := api.NewClient()
		resp, err := client.MergeForwardMessages(msgMergeForwardMessageIDs, receiveIDType, msgMergeForwardTo, asUser)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(api.OutputSendMessage{
			Success:    true,
			MessageID:  resp.Data.MessageID,
			ChatID:     resp.Data.ChatID,
			CreateTime: formatMessageTime(resp.Data.CreateTime),
		})
	},
}

func init() {
	msgSearchCmd.Flags().StringVar(&msgSearchChatID, "chat-id", "", "Restrict search to a specific chat")
	msgSearchCmd.Flags().StringVar(&msgSearchSender, "sender", "", "Restrict search to messages from a sender (open_id)")
	msgSearchCmd.Flags().StringVar(&msgSearchAfter, "after", "", "Start time (Unix timestamp or ISO 8601)")
	msgSearchCmd.Flags().StringVar(&msgSearchBefore, "before", "", "End time (Unix timestamp or ISO 8601)")
	msgSearchCmd.Flags().StringVar(&msgSearchType, "type", "", "Filter by message type (text, image, file, etc.)")
	msgSearchCmd.Flags().IntVar(&msgSearchLimit, "limit", 0, "Maximum number of results (0 = no limit)")

	msgForwardCmd.Flags().StringVar(&msgForwardMessageID, "message-id", "", "Message ID to forward (required)")
	msgForwardCmd.Flags().StringVar(&msgForwardTo, "to", "", "Recipient ID (required)")
	msgForwardCmd.Flags().StringVar(&msgForwardToType, "to-type", "", "Recipient ID type (auto-detected if not specified)")
	msgForwardCmd.Flags().StringVar(&msgForwardAs, "as", "bot", "Send as 'bot' (default, only option supported by forward API) or 'user'")

	msgMergeForwardCmd.Flags().StringSliceVar(&msgMergeForwardMessageIDs, "message-ids", nil, "Message IDs to merge-forward (comma-separated, required)")
	msgMergeForwardCmd.Flags().StringVar(&msgMergeForwardTo, "to", "", "Recipient ID (required)")
	msgMergeForwardCmd.Flags().StringVar(&msgMergeForwardToType, "to-type", "", "Recipient ID type (auto-detected if not specified)")
	msgMergeForwardCmd.Flags().StringVar(&msgMergeForwardAs, "as", "bot", "Send as 'bot' (default, only option supported by forward API) or 'user'")

	msgCmd.AddCommand(msgSearchCmd)
	msgCmd.AddCommand(msgGetCmd)
	msgCmd.AddCommand(msgForwardCmd)
	msgCmd.AddCommand(msgMergeForwardCmd)
}
