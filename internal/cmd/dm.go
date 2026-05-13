package cmd

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var (
	dmLast    int
	dmReply   string
	dmAs      string
	dmCompact bool
	dmDryRun  bool
)

var dmCmd = &cobra.Command{
	Use:   "dm <person>",
	Short: "Read/reply in a direct message with one command",
	Long: `Resolve a person, read recent DM history, and optionally send a reply.

Resolution order is deterministic:
1) open_id (ou_xxx)
2) email
3) exact name match
4) fuzzy name search (single best match or explicit ambiguity error)

Examples:
  lark dm "Kangrui He"
  lark dm kangrui@glints.com --last 30
  lark dm ou_123 --reply "On it"
  lark dm "Kangrui He" --last 20 --compact`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if dmAs != "user" && dmAs != "bot" {
			output.Fatalf("VALIDATION_ERROR", "--as must be 'user' or 'bot'")
		}
		if dmDryRun && strings.TrimSpace(dmReply) == "" {
			output.Fatalf("VALIDATION_ERROR", "--dry-run-reply requires --reply")
		}

		person := args[0]
		client := api.NewClient()

		user, matches, err := resolveDMUser(client, person)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		if user == nil {
			output.Fatalf("NOT_FOUND", "no user found matching: %s", person)
		}

		if len(matches) > 1 {
			choices := make([]map[string]string, 0, len(matches))
			for _, m := range matches {
				choices = append(choices, map[string]string{
					"open_id": m.OpenID,
					"name":    m.Name,
					"use":     "lark dm " + m.OpenID,
				})
			}
			output.JSON(map[string]interface{}{
				"error":   true,
				"code":    "AMBIGUOUS_USER",
				"message": "multiple users matched; rerun with open_id or email",
				"query":   person,
				"matches": choices,
			})
			return
		}

		chatID := findP2PChatIDForUser(client, user.OpenID, user.Name)

		// Optional reply first, so newly created DM chat_id becomes available.
		if strings.TrimSpace(dmReply) != "" {
			target := chatID
			if target == "" {
				target = user.OpenID
			}
			receiveIDType := detectIDType(target)
			if dmDryRun {
				output.JSON(map[string]interface{}{
					"dry_run": true,
					"person": map[string]string{
						"open_id": user.OpenID,
						"name":    user.Name,
					},
					"chat_id":         chatID,
					"send_to":         target,
					"send_to_type":    receiveIDType,
					"send_as":         dmAs,
					"reply_preview":   dmReply,
					"would_use_cmd":   "lark dm " + user.OpenID + ` --reply "` + dmReply + `"`,
					"next_read_hint":  "lark dm " + user.OpenID + " --last " + strconv.Itoa(dmLast),
					"message":         "dry run only; no message sent",
				})
				return
			}

			content, buildErr := buildMarkdownPostContentWithImages(dmReply, nil)
			if buildErr != nil {
				output.Fatal("VALIDATION_ERROR", buildErr)
			}

			var sendResp *api.SendMessageResponse
			if dmAs == "user" {
				sendResp, err = client.SendMessageAsUser(receiveIDType, target, "post", content)
			} else {
				sendResp, err = client.SendMessage(receiveIDType, target, "post", content)
			}
			if err != nil {
				output.Fatal("API_ERROR", err)
			}
			if sendResp != nil && sendResp.Data.ChatID != "" {
				chatID = sendResp.Data.ChatID
			}
		}

		if dmLast <= 0 {
			dmLast = 20
		}

		if chatID == "" {
			output.JSON(map[string]interface{}{
				"person": map[string]string{
					"open_id": user.OpenID,
					"name":    user.Name,
				},
				"chat_id": "",
				"message": "no existing DM history yet; send a first message with --reply to create the chat",
			})
			return
		}

		msgs, _, _, err := listMessagesForDM(client, chatID, dmLast, dmAs == "user")
		if err != nil {
			output.Fatal("API_ERROR", err)
		}

		if dmCompact {
			entries := make([]map[string]interface{}, 0, len(msgs))
			for _, m := range msgs {
				senderName := ""
				if m.Sender != nil {
					if m.Sender.ID == user.OpenID {
						senderName = user.Name
					} else if m.Sender.SenderType == "user" {
						if me, ge := client.GetUser(m.Sender.ID, "open_id"); ge == nil && me != nil {
							senderName = me.Name
						}
					}
				}

				role := "unknown"
				if m.Sender != nil && m.Sender.ID == user.OpenID {
					role = "them"
				} else if m.Sender != nil {
					role = "me"
				}

				entries = append(entries, map[string]interface{}{
					"message_id": m.MessageID,
					"ts":         formatMessageTime(m.CreateTime),
					"role":       role,
					"sender":     senderName,
					"type":       m.MsgType,
					"text":       extractMessageText(m),
				})
			}
			output.JSON(map[string]interface{}{
				"person": map[string]string{
					"open_id": user.OpenID,
					"name":    user.Name,
				},
				"chat_id":   chatID,
				"count":     len(entries),
				"messages":  entries,
				"reply_hint": "lark dm " + user.OpenID + ` --reply "..."`,
			})
			return
		}

		// Default detailed response: reuse existing message conversion shape.
		out := make([]api.OutputMessage, len(msgs))
		for i, m := range msgs {
			out[i] = convertMessage(m)
			if out[i].Sender != nil && m.Sender != nil && m.Sender.ID == user.OpenID {
				out[i].Sender.Name = user.Name
			}
		}

		output.JSON(map[string]interface{}{
			"person": map[string]string{
				"open_id": user.OpenID,
				"name":    user.Name,
			},
			"chat_id":    chatID,
			"count":      len(out),
			"messages":   out,
			"reply_hint": "lark dm " + user.OpenID + ` --reply "..."`,
		})
	},
}

func listMessagesForDM(client *api.Client, chatID string, limit int, asUser bool) ([]api.Message, bool, string, error) {
	var all []api.Message
	var pageToken string
	hasMore := true
	remaining := limit
	for hasMore {
		pageSize := 50
		if remaining > 0 && remaining < pageSize {
			pageSize = remaining
		}
		opts := &api.ListMessagesOptions{
			PageSize:  pageSize,
			PageToken: pageToken,
			SortType:  "ByCreateTimeDesc",
		}
		var (
			items []api.Message
			more  bool
			next  string
			err   error
		)
		if asUser {
			items, more, next, err = client.ListMessagesAsUser("chat", chatID, opts)
		} else {
			items, more, next, err = client.ListMessages("chat", chatID, opts)
		}
		if err != nil {
			return nil, false, "", err
		}
		all = append(all, items...)
		hasMore = more
		pageToken = next
		if limit > 0 {
			remaining = limit - len(all)
			if remaining <= 0 {
				break
			}
		}
	}
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, hasMore, pageToken, nil
}

func resolveDMUser(client *api.Client, input string) (*api.SearchUserResult, []api.SearchUserResult, error) {
	in := strings.TrimSpace(input)
	switch {
	case strings.HasPrefix(in, "ou_"):
		u, err := client.GetUser(in, "open_id")
		if err != nil {
			return nil, nil, err
		}
		if u == nil {
			return nil, nil, nil
		}
		r := api.SearchUserResult{OpenID: in, Name: u.Name}
		return &r, []api.SearchUserResult{r}, nil
	case strings.Contains(in, "@"):
		users, err := client.LookupUsers(api.UserLookupOptions{Emails: []string{in}})
		if err != nil {
			return nil, nil, err
		}
		if len(users) == 0 || users[0].UserID == "" {
			return nil, nil, nil
		}
		u, err := client.GetUser(users[0].UserID, "open_id")
		if err != nil {
			return nil, nil, err
		}
		name := ""
		if u != nil {
			name = u.Name
		}
		r := api.SearchUserResult{OpenID: users[0].UserID, Name: name}
		return &r, []api.SearchUserResult{r}, nil
	default:
		results, _, _, err := client.SearchUsers(in, 50, "")
		if err != nil {
			return nil, nil, err
		}
		return pickBestDMUserMatch(in, results), allBestDMMatches(in, results), nil
	}
}

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.Join(strings.Fields(s), " ")
}

func pickBestDMUserMatch(query string, users []api.SearchUserResult) *api.SearchUserResult {
	best := allBestDMMatches(query, users)
	if len(best) == 1 {
		return &best[0]
	}
	return nil
}

func allBestDMMatches(query string, users []api.SearchUserResult) []api.SearchUserResult {
	nq := normalizeName(query)
	if nq == "" || len(users) == 0 {
		return nil
	}
	exact := make([]api.SearchUserResult, 0)
	prefix := make([]api.SearchUserResult, 0)
	for _, u := range users {
		nu := normalizeName(u.Name)
		if nu == nq {
			exact = append(exact, u)
			continue
		}
		if strings.HasPrefix(nu, nq) {
			prefix = append(prefix, u)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	if len(prefix) > 0 {
		return prefix
	}
	if len(users) == 1 {
		return users
	}
	return nil
}

func extractMessageText(m api.Message) string {
	if m.Body == nil || strings.TrimSpace(m.Body.Content) == "" {
		return ""
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(m.Body.Content), &obj); err != nil {
		return m.Body.Content
	}
	if text, ok := obj["text"].(string); ok {
		return text
	}

	// post message format: {"title":"","content":[[{...}], ...]}
	lines := make([]string, 0)
	if title, ok := obj["title"].(string); ok && strings.TrimSpace(title) != "" {
		lines = append(lines, title)
	}
	if blocks, ok := obj["content"].([]interface{}); ok {
		for _, b := range blocks {
			row, ok := b.([]interface{})
			if !ok {
				continue
			}
			var rowText strings.Builder
			for _, it := range row {
				cell, ok := it.(map[string]interface{})
				if !ok {
					continue
				}
				tag, _ := cell["tag"].(string)
				switch tag {
				case "text":
					if t, ok := cell["text"].(string); ok {
						rowText.WriteString(t)
					}
				case "a":
					if t, ok := cell["text"].(string); ok {
						rowText.WriteString(t)
					}
				case "at":
					if uname, ok := cell["user_name"].(string); ok && uname != "" {
						rowText.WriteString("@")
						rowText.WriteString(uname)
					}
				case "img":
					rowText.WriteString("[image]")
				}
			}
			if strings.TrimSpace(rowText.String()) != "" {
				lines = append(lines, rowText.String())
			}
		}
	}
	if len(lines) == 0 {
		return m.Body.Content
	}
	return strings.Join(lines, "\n")
}

func init() {
	dmCmd.Flags().IntVar(&dmLast, "last", 20, "Number of recent messages to return")
	dmCmd.Flags().StringVar(&dmReply, "reply", "", "Reply text to send before fetching history")
	dmCmd.Flags().BoolVar(&dmDryRun, "dry-run-reply", false, "Resolve + preview reply target without sending any message")
	dmCmd.Flags().StringVar(&dmAs, "as", "user", "Identity for read/send: 'user' (default) or 'bot'")
	dmCmd.Flags().BoolVar(&dmCompact, "compact", false, "Return compact parsed transcript text (agent-friendly)")
}
