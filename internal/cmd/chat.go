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
		// Echo chat_id from request since Lark's response omits it
		if chat != nil && chat.ChatID == "" {
			chat.ChatID = args[0]
		}
		output.JSON(chat)
	},
}

var chatCreateTo string

var chatCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a group chat (bot identity required by Lark API)",
	Long: `Create a group chat. Bot identity is required by the Lark API.

Note: Members must have interacted with the bot before they can be added.
If you don't have a pre-existing bot-user relationship, create the chat empty
and share the join link (lark chat link <chat-id>).

Examples:
  lark chat create --name "Launch Room"                              # empty chat
  lark chat create --name "1:1 w/ Francis" --to ou_f87351...          # with one person
  lark chat create --name "1:1 w/ Francis" --to "Francis Goh"         # by name
  lark chat create --name "Sprint" --members ou_a,ou_b,ou_c          # with multiple`,
	Run: func(cmd *cobra.Command, args []string) {
		if chatCreateName == "" {
			output.Fatalf("VALIDATION_ERROR", "--name is required")
		}
		// Merge --to and --members into a single list, resolving names/emails to open_ids
		members := append([]string{}, chatCreateMembers...)
		if chatCreateTo != "" {
			members = append(members, chatCreateTo)
		}
		memberIDs := resolveMembers(members)

		client := api.NewClient()
		req := &api.CreateChatRequest{
			Name:        chatCreateName,
			Description: chatCreateDescription,
			UserIDList:  memberIDs,
		}
		// Create is tenant-only by Lark API design — always use bot identity
		resp, err := client.CreateChat(req, false)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success": true,
			"chat_id": resp.Data.ChatID,
		})
	},
}

var chatDeleteAs string

var chatDeleteCmd = &cobra.Command{
	Use:   "delete <chat-id>",
	Short: "Disband (delete) a group chat",
	Long: `Disband a group chat. Requires owner privileges.

Defaults to --as bot since chats created via 'lark chat create' are bot-owned.
Pass --as user to delete chats you personally own.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Override the persistent chatAs with local --as if provided
		asUser := chatDeleteAs == "user"
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

		// Try to find existing P2P chat_id by searching messages from this user
		chatID := findP2PChatIDForUser(client, openID, name)

		result := map[string]interface{}{
			"open_id": openID,
			"name":    name,
			"email":   email,
		}
		if chatID != "" {
			result["chat_id"] = chatID
			result["read_command"] = "lark msg history --chat-id " + chatID + " --limit 20 --sort desc"
			result["send_command"] = "lark msg send --to " + chatID + ` --text "..."`
		} else {
			result["chat_id"] = ""
			result["send_command"] = "lark msg send --to " + openID + ` --text "..."`
			result["hint"] = "No prior DM found. Send a message to create the P2P chat — the response will include chat_id."
		}
		output.JSON(result)
	},
}

// findP2PChatIDForUser searches recent messages for a P2P chat with the given user
// and returns the chat_id. Returns empty string if no prior DM exists. Best-effort.
//
// Strategy: scan recent message history across multiple search queries, filter to P2P
// chats, and find one where either the sender or recipient matches the target open_id.
// Lark's SenderID filter on the search API isn't strictly enforced, so we filter
// client-side instead.
func findP2PChatIDForUser(client *api.Client, openID, name string) string {
	// Use a few common short queries to surface recent activity
	queries := []string{name, "a", "i", "the", " "}

	seen := make(map[string]bool)
	for _, q := range queries {
		if q == "" {
			continue
		}
		results, _, _, err := client.SearchMessages(q, &api.SearchMessagesOptions{
			PageSize: 50,
		})
		if err != nil {
			continue
		}
		for _, r := range results {
			if r.MetaData == nil || !r.MetaData.IsP2PChat || r.MetaData.ChatID == "" {
				continue
			}
			// Match if the message sender is the target user (most reliable signal)
			if r.MetaData.FromID == openID {
				return r.MetaData.ChatID
			}
			// Otherwise note this P2P chat — counterpart name in display_info may match
			if !seen[r.MetaData.ChatID] {
				seen[r.MetaData.ChatID] = true
				if name != "" && strings.HasPrefix(r.DisplayInfo, name) {
					return r.MetaData.ChatID
				}
			}
		}
	}
	return ""
}

// --- chat list-dms ---

var chatListDMsCmd = &cobra.Command{
	Use:   "list-dms",
	Short: "List recent P2P (direct message) chats with their chat_ids",
	Long: `Enumerate recent P2P (1:1) chats by scanning recent messages.

Lark's /im/v1/chats endpoint only returns group chats — P2P chats are not listed
anywhere directly. This command works around that by searching recent messages
across all chats and filtering to P2P (is_p2p_chat=true), deduping by chat_id.

Examples:
  lark chat list-dms                # last ~50 distinct DMs from recent activity
  lark chat list-dms --limit 100    # scan more messages`,
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()

		// Try several common short queries to maximise coverage of recent messages
		seedQueries := []string{"a", "i", "o", "the", " "}

		dms := make(map[string]map[string]interface{})

		for _, q := range seedQueries {
			results, _, _, err := client.SearchMessages(q, &api.SearchMessagesOptions{
				PageSize: 50,
			})
			if err != nil {
				continue
			}
			for _, r := range results {
				if r.MetaData == nil || !r.MetaData.IsP2PChat || r.MetaData.ChatID == "" {
					continue
				}
				cid := r.MetaData.ChatID
				existing, ok := dms[cid]
				createTime := r.MetaData.CreateTime
				if !ok || createTime > existing["last_message_at"].(string) {
					// Extract counterpart name from display_info ("Chat Name\nSender: ...")
					counterpart := ""
					if idx := strings.Index(r.DisplayInfo, "\n"); idx > 0 {
						counterpart = r.DisplayInfo[:idx]
					}
					dms[cid] = map[string]interface{}{
						"chat_id":         cid,
						"counterpart":     counterpart,
						"last_message_at": createTime,
						"last_sender_id":  r.MetaData.FromID,
					}
				}
			}
			if chatListDMsLimit > 0 && len(dms) >= chatListDMsLimit {
				break
			}
		}

		// Sort by recency
		out := make([]map[string]interface{}, 0, len(dms))
		for _, v := range dms {
			out = append(out, v)
		}
		// Sort descending by last_message_at
		for i := 0; i < len(out); i++ {
			for j := i + 1; j < len(out); j++ {
				if out[j]["last_message_at"].(string) > out[i]["last_message_at"].(string) {
					out[i], out[j] = out[j], out[i]
				}
			}
		}
		if chatListDMsLimit > 0 && len(out) > chatListDMsLimit {
			out = out[:chatListDMsLimit]
		}

		output.JSON(map[string]interface{}{
			"count": len(out),
			"dms":   out,
		})
	},
}

var chatListDMsLimit int

// classifyMemberIdent returns one of "open_id", "email", or "name" based on the
// shape of the argument. open_id wins on the "ou_" prefix; any string containing
// "@" is treated as an email; everything else is a name.
func classifyMemberIdent(m string) string {
	switch {
	case strings.HasPrefix(m, "ou_"):
		return "open_id"
	case strings.Contains(m, "@"):
		return "email"
	default:
		return "name"
	}
}

// resolveMembers converts emails or names to open_ids, passes through ou_* IDs unchanged.
func resolveMembers(members []string) []string {
	var resolved []string
	client := api.NewClient()
	for _, m := range members {
		switch classifyMemberIdent(m) {
		case "open_id":
			resolved = append(resolved, m)
		case "email":
			users, err := client.LookupUsers(api.UserLookupOptions{Emails: []string{m}})
			if err == nil && len(users) > 0 && users[0].UserID != "" {
				resolved = append(resolved, users[0].UserID)
				continue
			}
			resolved = append(resolved, m)
		default:
			results, _, _, err := client.SearchUsers(m, 1, "")
			if err == nil && len(results) > 0 && results[0].OpenID != "" {
				resolved = append(resolved, results[0].OpenID)
				continue
			}
			resolved = append(resolved, m)
		}
	}
	return resolved
}

func init() {
	chatSearchCmd.Flags().IntVar(&chatSearchLimit, "limit", 0,
		"Maximum number of chats to retrieve (0 = no limit)")

	chatCmd.PersistentFlags().StringVar(&chatAs, "as", "user", "Identity: 'bot' or 'user' (default)")

	chatCreateCmd.Flags().StringVar(&chatCreateName, "name", "", "Chat name (required)")
	chatCreateCmd.Flags().StringVar(&chatCreateDescription, "description", "", "Chat description")
	chatCreateCmd.Flags().StringSliceVar(&chatCreateMembers, "members", nil, "Member IDs, emails, or names (comma-separated)")
	chatCreateCmd.Flags().StringVar(&chatCreateTo, "to", "", "Shortcut to add a single person (open_id, email, or name)")
	chatDeleteCmd.Flags().StringVar(&chatDeleteAs, "as", "bot", "Delete as 'bot' (default, for bot-created chats) or 'user'")

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

	chatListDMsCmd.Flags().IntVar(&chatListDMsLimit, "limit", 50, "Maximum number of DMs to return")
	chatCmd.AddCommand(chatListDMsCmd)
}
