package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/mail"
	"github.com/yjwong/lark-cli/internal/output"
	timex "github.com/yjwong/lark-cli/internal/time"
)

var (
	mailScheduleTo          []string
	mailScheduleCC          []string
	mailScheduleSubject     string
	mailScheduleBody        string
	mailScheduleBodyFile    string
	mailScheduleAttachments []string
	mailScheduleInReplyTo   string
	mailScheduleReferences  []string
	mailScheduleAt          string
	mailScheduleConfirm     bool
)

// parseScheduledAt accepts the same ISO 8601 formats used elsewhere in the
// CLI and applies the native Lark minimum five-minute scheduling requirement.
func parseScheduledAt(input string, now time.Time, location *time.Location) (time.Time, error) {
	at, err := timex.Parse(input, location)
	if err != nil {
		return time.Time{}, err
	}
	if _, err := mail.NewScheduledSend("validation", at, now); err != nil {
		return time.Time{}, err
	}
	return at, nil
}

// nativeDraftOptions centralizes the validation shared by creating a draft for
// review and by the one-step create-and-schedule command.
func nativeDraftOptions(to, cc []string, subject, body string, attachments []string, inReplyTo string, references []string) (*mail.SendOptions, error) {
	if len(to) == 0 {
		return nil, fmt.Errorf("--to is required")
	}
	if subject == "" {
		return nil, fmt.Errorf("--subject is required")
	}
	if body == "" {
		return nil, fmt.Errorf("--body or --body-file is required")
	}
	return &mail.SendOptions{
		To:          to,
		CC:          cc,
		Subject:     subject,
		Body:        body,
		Attachments: attachments,
		InReplyTo:   inReplyTo,
		References:  references,
	}, nil
}

func nativeDraftOptionsFromFlags() (*mail.SendOptions, error) {
	body := mailScheduleBody
	if mailScheduleBodyFile != "" {
		data, err := os.ReadFile(mailScheduleBodyFile)
		if err != nil {
			return nil, fmt.Errorf("reading body file: %w", err)
		}
		body = string(data)
	}
	return nativeDraftOptions(mailScheduleTo, mailScheduleCC, mailScheduleSubject, body, mailScheduleAttachments, mailScheduleInReplyTo, mailScheduleReferences)
}

var mailScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Schedule an email with Lark's native server-side delivery",
	Long: `Create a Lark Mail draft and schedule its delivery. Lark sends the
email at the requested time; the CLI does not need to remain running.

This command changes outbound state, so it requires --confirm. Use an ISO 8601
time with an explicit offset to avoid ambiguity, for example 2026-09-02T19:00:00+08:00.
Lark requires the scheduled time to be at least five minutes in the future.

Examples:
  lark mail schedule --to rebecca@example.com --subject "PR reference letter" \
    --body "Hi Rebecca, attached is the completed letter." \
    --attach ./letter.docx --at 2026-09-02T19:00:00+08:00 --confirm

  lark mail schedule cancel draft_123`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		validateScopeGroup("mailsend")
	},
	Run: func(cmd *cobra.Command, args []string) {
		if !mailScheduleConfirm {
			output.Fatalf("VALIDATION_ERROR", "--confirm is required before scheduling an email")
		}
		opts, err := nativeDraftOptionsFromFlags()
		if err != nil {
			output.Fatal("VALIDATION_ERROR", err)
		}
		if mailScheduleAt == "" {
			output.Fatalf("VALIDATION_ERROR", "--at is required")
		}
		at, err := parseScheduledAt(mailScheduleAt, time.Now(), time.Local)
		if err != nil {
			output.Fatal("VALIDATION_ERROR", err)
		}

		draft, err := mail.CreateNativeDraft(opts)
		if err != nil {
			output.Fatal("DRAFT_ERROR", err)
		}
		scheduled, err := mail.ScheduleNativeDraft(draft.DraftID, at, time.Now())
		if err != nil {
			output.Fatal("SCHEDULE_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":      true,
			"draft_id":     draft.DraftID,
			"scheduled_at": at.Format(time.RFC3339),
			"send_time":    scheduled.Body(),
			"reference":    draft.Reference,
			"message":      "email scheduled with Lark; cancel with: lark mail schedule cancel " + draft.DraftID,
		})
	},
}

var mailScheduleDraftCmd = &cobra.Command{
	Use:   "draft",
	Short: "Create a native Lark draft for review before scheduling",
	Long: `Create a server-side Lark Mail draft without sending or scheduling it.
Use this review-first workflow before a scheduled outbound email, then run
"lark mail schedule send <draft-id> --at <time> --confirm" after approval.`,
	Run: func(cmd *cobra.Command, args []string) {
		opts, err := nativeDraftOptionsFromFlags()
		if err != nil {
			output.Fatal("VALIDATION_ERROR", err)
		}
		draft, err := mail.CreateNativeDraft(opts)
		if err != nil {
			output.Fatal("DRAFT_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":   true,
			"draft_id":  draft.DraftID,
			"reference": draft.Reference,
			"message":   "native Lark draft created; review it in Mail Drafts, then schedule with: lark mail schedule send " + draft.DraftID + " --at <ISO-8601 time> --confirm",
		})
	},
}

var mailScheduleSendCmd = &cobra.Command{
	Use:   "send <draft-id>",
	Short: "Schedule an existing native Lark draft for delivery",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !mailScheduleConfirm {
			output.Fatalf("VALIDATION_ERROR", "--confirm is required before scheduling an email")
		}
		if mailScheduleAt == "" {
			output.Fatalf("VALIDATION_ERROR", "--at is required")
		}
		at, err := parseScheduledAt(mailScheduleAt, time.Now(), time.Local)
		if err != nil {
			output.Fatal("VALIDATION_ERROR", err)
		}
		scheduled, err := mail.ScheduleNativeDraft(args[0], at, time.Now())
		if err != nil {
			output.Fatal("SCHEDULE_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":      true,
			"draft_id":     args[0],
			"scheduled_at": at.Format(time.RFC3339),
			"send_time":    scheduled.Body(),
			"message":      "email scheduled with Lark; cancel with: lark mail schedule cancel " + args[0],
		})
	},
}

var mailScheduleUpdateCmd = &cobra.Command{
	Use:   "update <draft-id>",
	Short: "Replace an unsent native Lark draft after review",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		opts, err := nativeDraftOptionsFromFlags()
		if err != nil {
			output.Fatal("VALIDATION_ERROR", err)
		}
		draft, err := mail.UpdateNativeDraft(args[0], opts)
		if err != nil {
			output.Fatal("DRAFT_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":   true,
			"draft_id":  draft.DraftID,
			"reference": draft.Reference,
			"message":   "native Lark draft updated; it remains unsent and unscheduled",
		})
	},
}

var mailScheduleCancelCmd = &cobra.Command{
	Use:   "cancel <draft-id>",
	Short: "Cancel a native Lark scheduled email and restore it to Drafts",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := mail.CancelNativeScheduledDraft(args[0]); err != nil {
			output.Fatal("SCHEDULE_ERROR", err)
		}
		output.Success("scheduled email cancelled and restored to Drafts")
	},
}

func init() {
	mailScheduleCmd.PersistentFlags().StringSliceVar(&mailScheduleTo, "to", nil, "Recipient email address(es) (required for draft)")
	mailScheduleCmd.PersistentFlags().StringSliceVar(&mailScheduleCC, "cc", nil, "CC email address(es)")
	mailScheduleCmd.PersistentFlags().StringVar(&mailScheduleSubject, "subject", "", "Email subject (required for draft)")
	mailScheduleCmd.PersistentFlags().StringVar(&mailScheduleBody, "body", "", "Email body text")
	mailScheduleCmd.PersistentFlags().StringVar(&mailScheduleBodyFile, "body-file", "", "Read email body from file")
	mailScheduleCmd.PersistentFlags().StringSliceVar(&mailScheduleAttachments, "attach", nil, "File path(s) to attach")
	mailScheduleCmd.PersistentFlags().StringVar(&mailScheduleInReplyTo, "in-reply-to", "", "Message-ID to reply to (preserves email threading)")
	mailScheduleCmd.PersistentFlags().StringSliceVar(&mailScheduleReferences, "references", nil, "Message-ID chain for threading")
	mailScheduleCmd.PersistentFlags().StringVar(&mailScheduleAt, "at", "", "Delivery time in ISO 8601 format (required for send)")
	mailScheduleCmd.PersistentFlags().BoolVar(&mailScheduleConfirm, "confirm", false, "Confirm scheduling of this outbound email")
	mailScheduleCmd.AddCommand(mailScheduleDraftCmd)
	mailScheduleCmd.AddCommand(mailScheduleUpdateCmd)
	mailScheduleCmd.AddCommand(mailScheduleSendCmd)
	mailScheduleCmd.AddCommand(mailScheduleCancelCmd)
	mailCmd.AddCommand(mailScheduleCmd)
}
