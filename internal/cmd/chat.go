package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat/group commands",
	Long:  "Search and manage Lark chats and groups",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		validateScopeGroup("messages")
	},
}

// --- chat search ---

var (
	chatSearchLimit int
)

var chatSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for chats/groups",
	Long: `Search for chats and groups visible to the user or bot.

The search supports:
- Group name matching (including internationalized names)
- Group member name matching
- Multiple languages
- Fuzzy search (pinyin, prefix, etc.)

If no query is provided, returns all visible chats.

Examples:
  lark chat search "project"
  lark chat search "团队"
  lark chat search --limit 50`,
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()

		opts := &api.SearchChatsOptions{}
		if len(args) > 0 {
			opts.Query = args[0]
		}

		// Fetch chats with pagination
		var allChats []api.Chat
		var pageToken string
		hasMore := true
		remaining := chatSearchLimit

		for hasMore {
			pageSize := 50
			if remaining > 0 && remaining < pageSize {
				pageSize = remaining
			}
			opts.PageSize = pageSize
			opts.PageToken = pageToken

			chats, more, nextToken, err := client.SearchChats(opts)
			if err != nil {
				output.Fatal("API_ERROR", err)
			}

			allChats = append(allChats, chats...)
			hasMore = more
			pageToken = nextToken

			if chatSearchLimit > 0 {
				remaining = chatSearchLimit - len(allChats)
				if remaining <= 0 {
					break
				}
			}
		}

		// Trim to limit if needed
		if chatSearchLimit > 0 && len(allChats) > chatSearchLimit {
			allChats = allChats[:chatSearchLimit]
		}

		// Convert to output format
		outputChats := make([]api.OutputChat, len(allChats))
		for i, c := range allChats {
			outputChats[i] = api.OutputChat{
				ChatID:      c.ChatID,
				Name:        c.Name,
				Description: c.Description,
				OwnerID:     c.OwnerID,
				External:    c.External,
				ChatStatus:  c.ChatStatus,
			}
		}

		result := api.OutputChatList{
			Chats: outputChats,
			Count: len(outputChats),
		}
		if len(args) > 0 {
			result.Query = args[0]
		}

		output.JSON(result)
	},
}

// --- chat management subcommands ---

var (
	chatCreateName          string
	chatCreateDescription   string
	chatCreateMembers       []string
	chatUpdateName          string
	chatUpdateDescription   string
	chatMemberAddMembers    []string
	chatMemberRemoveMembers []string
	chatPinMessageID        string
	chatAs                  string
)

var chatGetCmd = &cobra.Command{
	Use:   "get <chat-id>",
	Short: "Get chat details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		asUser := chatAs == "user"
		client := api.NewClient()
		chat, err := client.GetChat(args[0], asUser)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(chat)
	},
}

var chatCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a group chat (bot identity required by Lark API)",
	Long: `Create a group chat. Bot identity is required by the Lark API.

Note: Members must have interacted with the bot before they can be added.
If you don't have a pre-existing bot-user relationship, create the chat empty
and have members join via the share link (lark chat link <chat-id>).`,
	Run: func(cmd *cobra.Command, args []string) {
		if chatCreateName == "" {
			output.Fatalf("VALIDATION_ERROR", "--name is required")
		}
		// Create is tenant-only by Lark API design — always use bot identity
		memberIDs := resolveMembers(chatCreateMembers)
		client := api.NewClient()
		req := &api.CreateChatRequest{
			Name:        chatCreateName,
			Description: chatCreateDescription,
			UserIDList:  memberIDs,
		}
		resp, err := client.CreateChat(req, false) // always bot
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success": true,
			"chat_id": resp.Data.ChatID,
		})
	},
}

var chatDeleteCmd = &cobra.Command{
	Use:   "delete <chat-id>",
	Short: "Disband (delete) a group chat",
	Long: `Disband a group chat. Requires owner privileges.

Only the chat owner can disband. Bot-created chats are owned by the bot,
so use --as bot (default) to delete them.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		asUser := chatAs == "user"
		client := api.NewClient()
		if err := client.DeleteChat(args[0], asUser); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "chat_id": args[0]})
	},
}

var chatUpdateCmd = &cobra.Command{
	Use:   "update <chat-id>",
	Short: "Update chat name or description",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if chatUpdateName == "" && chatUpdateDescription == "" {
			output.Fatalf("VALIDATION_ERROR", "--name or --description required")
		}
		asUser := chatAs == "user"
		client := api.NewClient()
		req := &api.UpdateChatRequest{
			Name:        chatUpdateName,
			Description: chatUpdateDescription,
		}
		if err := client.UpdateChat(args[0], req, asUser); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "chat_id": args[0]})
	},
}

var chatMembersCmd = &cobra.Command{
	Use:   "members <chat-id>",
	Short: "List chat members",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		asUser := chatAs == "user"
		client := api.NewClient()
		var allMembers []api.ChatMember
		var pageToken string
		hasMore := true
		for hasMore {
			members, more, nextToken, err := client.ListChatMembers(args[0], 50, pageToken, asUser)
			if err != nil {
				output.Fatal("API_ERROR", err)
			}
			allMembers = append(allMembers, members...)
			hasMore = more
			pageToken = nextToken
		}
		output.JSON(map[string]interface{}{
			"chat_id": args[0],
			"members": allMembers,
			"count":   len(allMembers),
		})
	},
}

var chatMemberCmd = &cobra.Command{
	Use:   "member",
	Short: "Add or remove chat members",
}

var chatMemberAddCmd = &cobra.Command{
	Use:   "add <chat-id>",
	Short: "Add members to a chat",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(chatMemberAddMembers) == 0 {
			output.Fatalf("VALIDATION_ERROR", "--members is required")
		}
		asUser := chatAs == "user"
		memberIDs := resolveMembers(chatMemberAddMembers)
		client := api.NewClient()
		if err := client.AddChatMembers(args[0], memberIDs, asUser); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "chat_id": args[0]})
	},
}

var chatMemberRemoveCmd = &cobra.Command{
	Use:   "remove <chat-id>",
	Short: "Remove members from a chat",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(chatMemberRemoveMembers) == 0 {
			output.Fatalf("VALIDATION_ERROR", "--members is required")
		}
		asUser := chatAs == "user"
		memberIDs := resolveMembers(chatMemberRemoveMembers)
		client := api.NewClient()
		if err := client.RemoveChatMembers(args[0], memberIDs, asUser); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "chat_id": args[0]})
	},
}

var chatPinCmd = &cobra.Command{
	Use:   "pin <chat-id>",
	Short: "Pin a message in a chat",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if chatPinMessageID == "" {
			output.Fatalf("VALIDATION_ERROR", "--message-id is required")
		}
		asUser := chatAs == "user"
		client := api.NewClient()
		pin, err := client.PinMessage(chatPinMessageID, asUser)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "pin": pin})
	},
}

var chatUnpinCmd = &cobra.Command{
	Use:   "unpin <chat-id>",
	Short: "Unpin a message",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if chatPinMessageID == "" {
			output.Fatalf("VALIDATION_ERROR", "--message-id is required")
		}
		asUser := chatAs == "user"
		client := api.NewClient()
		if err := client.UnpinMessage(chatPinMessageID, asUser); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true})
	},
}

var chatPinsCmd = &cobra.Command{
	Use:   "pins <chat-id>",
	Short: "List pinned messages",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		asUser := chatAs == "user"
		client := api.NewClient()
		pins, err := client.ListPinnedMessages(args[0], asUser)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"chat_id": args[0], "pins": pins, "count": len(pins)})
	},
}

var chatLinkCmd = &cobra.Command{
	Use:   "link <chat-id>",
	Short: "Get shareable chat join link",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		asUser := chatAs == "user"
		client := api.NewClient()
		link, err := client.GetChatLink(args[0], asUser)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"chat_id": args[0], "share_link": link})
	},
}

// --- chat dm ---

var chatDMCmd = &cobra.Command{
	Use:   "dm <user>",
	Short: "Look up a user's DM info (email, name, or open_id)",
	Long: `Resolve a user identifier to their open_id and DM info, ready to message.

The argument can be:
  - email (e.g. alice@example.com)
  - open_id (ou_xxx) — passes through
  - user name (e.g. "Francis Goh") — fuzzy match

Returns the user's open_id, name, and a ready-to-use msg send command.
To send a message: lark msg send --to <open_id> --text "Hi"
Lark auto-creates the P2P chat on first message.

Examples:
  lark chat dm alice@example.com
  lark chat dm "Francis Goh"
  lark chat dm ou_f8735159a11237cb442c3d72aee8b073`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		client := api.NewClient()

		var openID, name, email string

		switch {
		case strings.HasPrefix(input, "ou_"):
			openID = input
			user, err := client.GetUser(openID, "open_id")
			if err == nil && user != nil {
				name = user.Name
				email = user.Email
			}
		case strings.Contains(input, "@"):
			email = input
			users, err := client.LookupUsers(api.UserLookupOptions{Emails: []string{input}})
			if err != nil {
				output.Fatal("API_ERROR", err)
			}
			if len(users) == 0 || users[0].UserID == "" {
				output.Fatalf("NOT_FOUND", "no user found for email: %s", input)
			}
			openID = users[0].UserID
			user, err := client.GetUser(openID, "open_id")
			if err == nil && user != nil {
				name = user.Name
			}
		default:
			results, _, _, err := client.SearchUsers(input, 5, "")
			if err != nil {
				output.Fatal("API_ERROR", err)
			}
			if len(results) == 0 {
				output.Fatalf("NOT_FOUND", "no user found matching: %s", input)
			}
			openID = results[0].OpenID
			name = results[0].Name
		}

		output.JSON(map[string]interface{}{
			"open_id":      openID,
			"name":         name,
			"email":        email,
			"send_command": "lark msg send --to " + openID + ` --text "..."`,
			"hint":         "Lark auto-creates the P2P chat on first message. Use msg history --chat-id <oc_id> after sending to read DMs.",
		})
	},
}

// resolveMembers converts emails to open_ids, passes through other IDs unchanged.
func resolveMembers(members []string) []string {
	var resolved []string
	client := api.NewClient()
	for _, m := range members {
		if strings.Contains(m, "@") {
			users, err := client.LookupUsers(api.UserLookupOptions{Emails: []string{m}})
			if err == nil && len(users) > 0 && users[0].UserID != "" {
				resolved = append(resolved, users[0].UserID)
				continue
			}
		}
		resolved = append(resolved, m)
	}
	return resolved
}

func init() {
	chatSearchCmd.Flags().IntVar(&chatSearchLimit, "limit", 0,
		"Maximum number of chats to retrieve (0 = no limit)")

	chatCmd.PersistentFlags().StringVar(&chatAs, "as", "user", "Identity: 'bot' or 'user' (default)")

	chatCreateCmd.Flags().StringVar(&chatCreateName, "name", "", "Chat name (required)")
	chatCreateCmd.Flags().StringVar(&chatCreateDescription, "description", "", "Chat description")
	chatCreateCmd.Flags().StringSliceVar(&chatCreateMembers, "members", nil, "Member IDs or emails (comma-separated)")

	chatUpdateCmd.Flags().StringVar(&chatUpdateName, "name", "", "New chat name")
	chatUpdateCmd.Flags().StringVar(&chatUpdateDescription, "description", "", "New description")

	chatMemberAddCmd.Flags().StringSliceVar(&chatMemberAddMembers, "members", nil, "Member IDs or emails to add")
	chatMemberRemoveCmd.Flags().StringSliceVar(&chatMemberRemoveMembers, "members", nil, "Member IDs or emails to remove")

	chatPinCmd.Flags().StringVar(&chatPinMessageID, "message-id", "", "Message ID to pin")
	chatUnpinCmd.Flags().StringVar(&chatPinMessageID, "message-id", "", "Message ID to unpin")

	chatMemberCmd.AddCommand(chatMemberAddCmd)
	chatMemberCmd.AddCommand(chatMemberRemoveCmd)

	chatCmd.AddCommand(chatSearchCmd)
	chatCmd.AddCommand(chatGetCmd)
	chatCmd.AddCommand(chatCreateCmd)
	chatCmd.AddCommand(chatUpdateCmd)
	chatCmd.AddCommand(chatMembersCmd)
	chatCmd.AddCommand(chatMemberCmd)
	chatCmd.AddCommand(chatPinCmd)
	chatCmd.AddCommand(chatUnpinCmd)
	chatCmd.AddCommand(chatPinsCmd)
	chatCmd.AddCommand(chatLinkCmd)
	chatCmd.AddCommand(chatDMCmd)
	chatCmd.AddCommand(chatDeleteCmd)
}
