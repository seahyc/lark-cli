package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	desktopScheduleSelfMinDelay = time.Hour
	desktopScheduleSelfMaxDelay = 366 * 24 * time.Hour
)

type desktopSelfScheduleItem struct {
	MessageID    string `json:"messageId"`
	ChatID       string `json:"chatId"`
	ScheduleTime string `json:"scheduleTime"`
	Status       int    `json:"status"`
	SenderID     string `json:"senderId"`
	Type         int    `json:"type"`
	Text         string `json:"text"`
}

func desktopScheduleSelfAt(value string, now time.Time) (time.Time, error) {
	at, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("--at must be RFC3339 with an explicit offset: %w", err)
	}
	delay := at.Sub(now)
	if delay < desktopScheduleSelfMinDelay || delay > desktopScheduleSelfMaxDelay {
		return time.Time{}, fmt.Errorf("--at must be between 1 hour and 366 days in the future")
	}
	return at, nil
}

func desktopScheduleSelfText(value string) error {
	if strings.TrimSpace(value) == "" || len(value) > 16384 {
		return fmt.Errorf("--text must be non-empty plain text and at most 16 KiB")
	}
	return nil
}

func desktopSelfChat(cmd *cobra.Command) (string, string, error) {
	var identity struct {
		UserID string `json:"userId"`
	}
	if err := desktopCall(cmd, "identity", nil, &identity); err != nil {
		return "", "", err
	}
	if identity.UserID == "" {
		return "", "", fmt.Errorf("native identity is empty")
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
		return "", "", err
	}
	if len(lookup.Data.Chats) != 1 || lookup.Data.Chats[0].UserID != identity.UserID || lookup.Data.Chats[0].ChatID == "" {
		return "", "", fmt.Errorf("no unique native self-chat found; refusing fallback")
	}
	return identity.UserID, lookup.Data.Chats[0].ChatID, nil
}

func desktopReadSelfSchedules(cmd *cobra.Command, chatID string) ([]desktopSelfScheduleItem, json.RawMessage, error) {
	var result struct {
		Data struct {
			Items []desktopSelfScheduleItem `json:"items"`
		} `json:"data"`
	}
	var raw json.RawMessage
	if err := desktopCall(cmd, "triage.scheduled-items", map[string]string{"chatId": chatID}, &raw); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, nil, err
	}
	return result.Data.Items, raw, nil
}

func init() {
	var text, atValue, expectUser string
	var apply bool
	schedule := &cobra.Command{
		Use:   "schedule-self",
		Short: "Preview or schedule plain text in the authenticated native self-chat (experimental)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := desktopScheduleSelfText(text); err != nil {
				return err
			}
			at, err := desktopScheduleSelfAt(atValue, time.Now())
			if err != nil {
				return err
			}
			userID, chatID, err := desktopSelfChat(cmd)
			if err != nil {
				return err
			}
			if apply && expectUser != userID {
				return fmt.Errorf("--expect-user must match the preview identity")
			}
			input := map[string]string{"chatId": chatID, "text": text, "scheduleTime": strconv.FormatInt(at.Unix(), 10)}
			if !apply {
				return desktopJSON(cmd, map[string]interface{}{"apply": false, "userId": userID, "chatId": chatID, "text": text, "scheduledAt": at.Format(time.RFC3339), "next": "Review then repeat with --apply --expect-user matching userId"})
			}
			var out json.RawMessage
			if err := desktopCall(cmd, "triage.schedule-self-text", map[string]interface{}{"expectedUserId": userID, "apply": true, "input": input}, &out); err != nil {
				return fmt.Errorf("%w; schedule outcome may be uncertain; inspect scheduled-self before retrying", err)
			}
			return desktopJSON(cmd, out)
		},
	}
	schedule.Flags().StringVar(&text, "text", "", "Plain text to schedule in your native self-chat")
	schedule.Flags().StringVar(&atValue, "at", "", "RFC3339 time with explicit offset, 1 hour to 366 days ahead")
	schedule.Flags().BoolVar(&apply, "apply", false, "Schedule once after preview; never retries automatically")
	schedule.Flags().StringVar(&expectUser, "expect-user", "", "Native user ID from preview")

	read := &cobra.Command{Use: "scheduled-self", Short: "Read pending schedules in the authenticated native self-chat", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		userID, chatID, err := desktopSelfChat(cmd)
		if err != nil {
			return err
		}
		_, raw, err := desktopReadSelfSchedules(cmd, chatID)
		if err != nil {
			return err
		}
		return desktopJSON(cmd, map[string]interface{}{"userId": userID, "chatId": chatID, "schedules": json.RawMessage(raw)})
	}}

	var cancelExpectUser string
	var cancelApply bool
	cancel := &cobra.Command{Use: "cancel-scheduled-self MESSAGE_ID", Short: "Preview or cancel one current pending self-chat plain-text schedule (experimental)", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		userID, chatID, err := desktopSelfChat(cmd)
		if err != nil {
			return err
		}
		items, _, err := desktopReadSelfSchedules(cmd, chatID)
		if err != nil {
			return err
		}
		var found *desktopSelfScheduleItem
		for index := range items {
			item := &items[index]
			if item.MessageID == args[0] {
				found = item
				break
			}
		}
		if found == nil || found.ChatID != chatID || found.SenderID != userID || found.Type != 4 || found.Status != 1 || found.ScheduleTime == "" {
			return fmt.Errorf("MESSAGE_ID is not a current pending plain-text schedule in the authenticated self-chat")
		}
		input := map[string]string{"messageId": found.MessageID, "chatId": chatID, "scheduleTime": found.ScheduleTime, "expectedText": found.Text}
		if !cancelApply {
			return desktopJSON(cmd, map[string]interface{}{"apply": false, "userId": userID, "before": found, "next": "Review then repeat with --apply --expect-user matching userId"})
		}
		if cancelExpectUser != userID {
			return fmt.Errorf("--expect-user must match the preview identity")
		}
		var out json.RawMessage
		if err := desktopCall(cmd, "triage.schedule-cancel-text", map[string]interface{}{"expectedUserId": userID, "apply": true, "input": input}, &out); err != nil {
			return fmt.Errorf("%w; cancellation outcome may be uncertain; inspect scheduled-self before retrying", err)
		}
		return desktopJSON(cmd, out)
	}}
	cancel.Flags().BoolVar(&cancelApply, "apply", false, "Cancel once after preview; never retries automatically")
	cancel.Flags().StringVar(&cancelExpectUser, "expect-user", "", "Native user ID from preview")

	desktopCmd.AddCommand(schedule, read, cancel)
}
