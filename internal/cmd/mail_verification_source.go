package cmd

import (
	"errors"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/mail"
	"github.com/yjwong/lark-cli/internal/output"
)

var (
	verificationSourceAccount string
	verificationSourceSender  string
	verificationSourceSince   string
	verificationSourceFolder  string
)

var mailVerificationSourceCmd = &cobra.Command{
	Use:   "verification-source",
	Short: "Read bounded verification messages from INBOX",
	Run: func(cmd *cobra.Command, args []string) {
		if info, err := os.Stdout.Stat(); err != nil || info.Mode()&os.ModeCharDevice != 0 {
			output.Fatal("STDOUT_MUST_BE_PIPE", errors.New("STDOUT_MUST_BE_PIPE"))
		}
		since, err := strconv.ParseInt(verificationSourceSince, 10, 64)
		if err != nil || since <= 0 {
			output.Fatal("INVALID_INPUT", errors.New("INVALID_INPUT"))
		}
		result, err := mail.VerificationSourceInMailbox(verificationSourceAccount, verificationSourceSender, since, verificationSourceFolder)
		if err != nil {
			code := "IMAP_ERROR"
			switch {
			case errors.Is(err, mail.ErrVerificationSourceAccount):
				code = "ACCOUNT_MISMATCH"
			case errors.Is(err, mail.ErrVerificationSourceConfig):
				code = "CONFIG_INVALID"
			case errors.Is(err, mail.ErrVerificationSourceLimit):
				code = "SCAN_LIMIT"
			case errors.Is(err, mail.ErrVerificationSourceMatches):
				code = "MATCH_LIMIT"
			case errors.Is(err, mail.ErrVerificationSourceBody):
				code = "MESSAGE_TOO_LARGE"
			case errors.Is(err, mail.ErrVerificationSourceInput):
				code = "INVALID_INPUT"
			}
			output.Fatal(code, errors.New(code))
		}
		output.JSON(result)
	},
}

func init() {
	mailVerificationSourceCmd.Flags().StringVar(&verificationSourceAccount, "account", "", "Configured mailbox account (required)")
	mailVerificationSourceCmd.Flags().StringVar(&verificationSourceSender, "sender", "", "Exact sender address to match (required)")
	mailVerificationSourceCmd.Flags().StringVar(&verificationSourceSince, "since", "", "Unix timestamp lower bound (required)")
	mailVerificationSourceCmd.Flags().StringVar(&verificationSourceFolder, "folder", "INBOX", "Mailbox to inspect (INBOX or Archive)")
	_ = mailVerificationSourceCmd.MarkFlagRequired("account")
	_ = mailVerificationSourceCmd.MarkFlagRequired("sender")
	_ = mailVerificationSourceCmd.MarkFlagRequired("since")
	mailCmd.AddCommand(mailVerificationSourceCmd)
}
