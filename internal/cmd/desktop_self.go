package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	var text, expectUser string
	var apply bool
	command := &cobra.Command{Use: "send-self", Short: "Preview or send plain text to the authenticated native self-chat", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if text == "" || len(text) > 16384 {
			return fmt.Errorf("--text must be non-empty and at most 16 KiB")
		}
		var identity struct {
			UserID string `json:"userId"`
		}
		if err := desktopCall(cmd, "identity", nil, &identity); err != nil {
			return err
		}
		if apply && expectUser != identity.UserID {
			return fmt.Errorf("--expect-user must match the preview identity")
		}
		var lookup struct {
			Data struct {
				Chats []struct {
					UserID string `json:"userId"`
					ChatID string `json:"chatId"`
				} `json:"chats"`
			} `json:"data"`
		}
		if err := desktopCall(cmd, "messaging.p2p-chat", map[string]string{"userId": identity.UserID}, &lookup); err != nil {
			return err
		}
		if len(lookup.Data.Chats) != 1 || lookup.Data.Chats[0].UserID != identity.UserID || lookup.Data.Chats[0].ChatID == "" {
			return fmt.Errorf("no unique native self-chat found; refusing fallback")
		}
		var result json.RawMessage
		err := desktopCall(cmd, "messaging.send-self-text", map[string]interface{}{"expectedUserId": identity.UserID, "apply": apply, "input": map[string]string{"chatId": lookup.Data.Chats[0].ChatID, "text": text}}, &result)
		if err != nil && apply {
			return fmt.Errorf("%w; send outcome may be uncertain; inspect self-chat before retrying", err)
		}
		if err != nil {
			return err
		}
		return desktopJSON(cmd, result)
	}}
	command.Flags().StringVar(&text, "text", "", "Plain text to send to your native self-chat")
	command.Flags().BoolVar(&apply, "apply", false, "Send once after preview; never retries automatically")
	command.Flags().StringVar(&expectUser, "expect-user", "", "Native user ID from preview")
	desktopCmd.AddCommand(command)
}
