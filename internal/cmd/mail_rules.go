package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

// The Auto Filter (mail rule) and folder endpoints are part of the Lark Open
// API mail-v1 domain and use the user access token, unlike the rest of the
// `mail` command which speaks IMAP/SMTP directly. They require the
// `mailrules` scope group (mail:user_mailbox.rule:write,
// mail:user_mailbox.folder:write) — kept separate from the IMAP `mail` group so
// that read-only IMAP usage never breaks if these write scopes aren't granted.

const mailUserMailbox = "me"

// --- shared response helpers ---

type mailAPIResp struct {
	api.BaseResponse
	Data json.RawMessage `json:"data"`
}

type mailFolder struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ParentFolderID string `json:"parent_folder_id"`
	FolderType     int    `json:"folder_type"`
}

type mailFolderList struct {
	Items []mailFolder `json:"items"`
}

func listFolders(client *api.Client) ([]mailFolder, error) {
	path := fmt.Sprintf("/mail/v1/user_mailboxes/%s/folders", mailUserMailbox)
	var resp mailAPIResp
	if err := client.Get(path, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	var fl mailFolderList
	if len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, &fl); err != nil {
			return nil, fmt.Errorf("failed to parse folders: %w", err)
		}
	}
	return fl.Items, nil
}

// resolveFolder returns a folder ID given either a raw folder ID or a folder
// name. Names are matched case-insensitively against the user's folders.
func resolveFolder(client *api.Client, folder string) (string, error) {
	folders, err := listFolders(client)
	if err != nil {
		return "", err
	}
	// Exact ID match first.
	for _, f := range folders {
		if f.ID == folder {
			return f.ID, nil
		}
	}
	// Then case-insensitive name match.
	for _, f := range folders {
		if strings.EqualFold(f.Name, folder) {
			return f.ID, nil
		}
	}
	return "", fmt.Errorf("no folder matching %q (create it first: lark mail folder create --name %q)", folder, folder)
}

// --- mail folder ---

var mailFolderCmd = &cobra.Command{
	Use:   "folder",
	Short: "Manage mailbox folders (Open API)",
	Long:  "List and create Lark Mail folders. Used as the destination for filter rules.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		validateScopeGroup("mailrules")
	},
}

var mailFolderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List mailbox folders",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		folders, err := listFolders(client)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"count":   len(folders),
			"folders": folders,
		})
	},
}

var mailFolderCreateName string
var mailFolderCreateParent string

var mailFolderCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a mailbox folder",
	Run: func(cmd *cobra.Command, args []string) {
		if mailFolderCreateName == "" {
			output.Error("VALIDATION_ERROR", "--name is required")
			return
		}
		client := api.NewClient()
		path := fmt.Sprintf("/mail/v1/user_mailboxes/%s/folders", mailUserMailbox)
		body := map[string]interface{}{
			"name":             mailFolderCreateName,
			"parent_folder_id": mailFolderCreateParent,
		}
		var resp mailAPIResp
		if err := client.Post(path, body, &resp); err != nil {
			output.Fatal("API_ERROR", err)
		}
		if err := resp.Err(); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(json.RawMessage(resp.Data))
	},
}

var mailFolderDeleteCmd = &cobra.Command{
	Use:   "delete <folder-id>",
	Short: "Delete a mailbox folder",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		path := fmt.Sprintf("/mail/v1/user_mailboxes/%s/folders/%s", mailUserMailbox, url.PathEscape(args[0]))
		var resp mailAPIResp
		if err := client.Delete(path, &resp); err != nil {
			output.Fatal("API_ERROR", err)
		}
		if err := resp.Err(); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.Success(fmt.Sprintf("deleted folder %s", args[0]))
	},
}

// --- mail filter (auto filter rules) ---

var mailFilterCmd = &cobra.Command{
	Use:   "filter",
	Short: "Manage inbox filter rules (Open API auto-filter)",
	Long: `Create, list, and delete Lark Mail filter rules ("auto filters").

Rules match incoming mail by sender and move it to a folder. Note: the Lark
Open API does not support an auto-forward action (only move-to-folder, flag,
mark-read, archive, spam). To fan a folder out to a chat, run a separate
forwarder that watches the folder.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		validateScopeGroup("mailrules")
	},
}

var mailFilterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List filter rules",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		path := fmt.Sprintf("/mail/v1/user_mailboxes/%s/rules", mailUserMailbox)
		var resp mailAPIResp
		if err := client.Get(path, &resp); err != nil {
			output.Fatal("API_ERROR", err)
		}
		if err := resp.Err(); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(json.RawMessage(resp.Data))
	},
}

var (
	mailFilterName    string
	mailFilterFrom    []string
	mailFilterFolder  string
	mailFilterFwdEmail []string
	mailFilterFwdChat []string
	mailFilterStop    bool
	mailFilterDisable bool
)

var mailFilterCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a filter rule matched by sender (move to folder and/or forward)",
	Long: `Create a filter rule that acts on mail from the given sender(s).

Each --from entry chooses its own match operator for minimal false positives:
  - an entry starting with "@" (e.g. @email.apple.com) matches by domain (contains)
  - a full address (e.g. googleplay-noreply@google.com) matches exactly (is)

Multiple --from entries are OR'd together (any sender matches).

Actions (at least one required, and they combine):
  --folder <id|name>     move matched mail to a folder (action type 11)
  --forward-email <addr> forward matched mail to an email address (type 12)
  --forward-chat <id>    forward matched mail to a Lark chat (type 13); the id
                         is the mail-rule chat id (as seen in existing rules /
                         the mail web UI), not necessarily an oc_ id.

Note: the published Open API docs claim forwarding is unsupported, but the
rule engine accepts and honours forward actions (types 12/13) in practice.

Examples:
  # Move to a folder
  lark mail filter create --name "3rd-party alerts" --folder "3rd-party-alerts" \
    --from googleplay-noreply@google.com --from @email.apple.com

  # Forward a sender straight to a common Lark chat
  lark mail filter create --name "WhatsApp alerts" --forward-chat 7341233491117391903 \
    --from @business.whatsapp.com`,
	Run: func(cmd *cobra.Command, args []string) {
		if mailFilterName == "" {
			output.Error("VALIDATION_ERROR", "--name is required")
			return
		}
		if len(mailFilterFrom) == 0 {
			output.Error("VALIDATION_ERROR", "at least one --from sender is required")
			return
		}
		if mailFilterFolder == "" && len(mailFilterFwdEmail) == 0 && len(mailFilterFwdChat) == 0 {
			output.Error("VALIDATION_ERROR", "at least one action is required: --folder, --forward-email, or --forward-chat")
			return
		}

		client := api.NewClient()

		// Build action items in a stable order: move-to-folder, then email
		// forwards, then chat forwards.
		actions := make([]map[string]interface{}, 0, 1+len(mailFilterFwdEmail)+len(mailFilterFwdChat))
		if mailFilterFolder != "" {
			folderID, err := resolveFolder(client, mailFilterFolder)
			if err != nil {
				output.Fatal("VALIDATION_ERROR", err)
			}
			actions = append(actions, map[string]interface{}{"type": 11, "input": folderID})
		}
		for _, addr := range mailFilterFwdEmail {
			addr = strings.TrimSpace(addr)
			if addr != "" {
				actions = append(actions, map[string]interface{}{"type": 12, "input": addr})
			}
		}
		for _, chat := range mailFilterFwdChat {
			chat = strings.TrimSpace(chat)
			if chat != "" {
				actions = append(actions, map[string]interface{}{"type": 13, "input": chat})
			}
		}

		// Build condition items: exact-address -> operator 5 (is);
		// "@domain" -> operator 1 (contains). type 1 = sender.
		items := make([]map[string]interface{}, 0, len(mailFilterFrom))
		for _, sender := range mailFilterFrom {
			sender = strings.TrimSpace(sender)
			if sender == "" {
				continue
			}
			operator := 5 // is (exact address)
			if strings.HasPrefix(sender, "@") {
				operator = 1 // contains (domain match)
			}
			items = append(items, map[string]interface{}{
				"type":     1,
				"operator": operator,
				"input":    sender,
			})
		}

		// match_type: 1 = all, 2 = any. Multiple senders => any.
		matchType := 2
		if len(items) == 1 {
			matchType = 1
		}

		body := map[string]interface{}{
			"condition": map[string]interface{}{
				"match_type": matchType,
				"items":      items,
			},
			"action": map[string]interface{}{
				"items": actions,
			},
			"ignore_the_rest_of_rules": mailFilterStop,
			"name":                     mailFilterName,
			"is_enable":                !mailFilterDisable,
		}

		path := fmt.Sprintf("/mail/v1/user_mailboxes/%s/rules", mailUserMailbox)
		var resp mailAPIResp
		if err := client.Post(path, body, &resp); err != nil {
			output.Fatal("API_ERROR", err)
		}
		if err := resp.Err(); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(json.RawMessage(resp.Data))
	},
}

var mailFilterDeleteCmd = &cobra.Command{
	Use:   "delete <rule-id>",
	Short: "Delete a filter rule",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		path := fmt.Sprintf("/mail/v1/user_mailboxes/%s/rules/%s", mailUserMailbox, url.PathEscape(args[0]))
		var resp mailAPIResp
		if err := client.Delete(path, &resp); err != nil {
			output.Fatal("API_ERROR", err)
		}
		if err := resp.Err(); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.Success(fmt.Sprintf("deleted rule %s", args[0]))
	},
}

func init() {
	mailFolderCreateCmd.Flags().StringVar(&mailFolderCreateName, "name", "", "Folder name (required)")
	mailFolderCreateCmd.Flags().StringVar(&mailFolderCreateParent, "parent", "0", "Parent folder ID (0 = root)")
	mailFolderCmd.AddCommand(mailFolderListCmd)
	mailFolderCmd.AddCommand(mailFolderCreateCmd)
	mailFolderCmd.AddCommand(mailFolderDeleteCmd)

	mailFilterCreateCmd.Flags().StringVar(&mailFilterName, "name", "", "Rule name (required)")
	mailFilterCreateCmd.Flags().StringSliceVar(&mailFilterFrom, "from", nil, "Sender to match: full address (exact) or @domain (contains). Repeatable.")
	mailFilterCreateCmd.Flags().StringVar(&mailFilterFolder, "folder", "", "Destination folder ID or name (move-to-folder action)")
	mailFilterCreateCmd.Flags().StringSliceVar(&mailFilterFwdEmail, "forward-email", nil, "Forward matched mail to an email address. Repeatable.")
	mailFilterCreateCmd.Flags().StringSliceVar(&mailFilterFwdChat, "forward-chat", nil, "Forward matched mail to a Lark chat id. Repeatable.")
	mailFilterCreateCmd.Flags().BoolVar(&mailFilterStop, "stop", false, "Stop processing further rules after this one matches")
	mailFilterCreateCmd.Flags().BoolVar(&mailFilterDisable, "disable", false, "Create the rule disabled")
	mailFilterCmd.AddCommand(mailFilterListCmd)
	mailFilterCmd.AddCommand(mailFilterCreateCmd)
	mailFilterCmd.AddCommand(mailFilterDeleteCmd)

	mailCmd.AddCommand(mailFolderCmd)
	mailCmd.AddCommand(mailFilterCmd)
}
