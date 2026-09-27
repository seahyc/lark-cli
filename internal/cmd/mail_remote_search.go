package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/mail"
	"github.com/yjwong/lark-cli/internal/output"
)

func init() {
	var from, subject, body, since, before, folder string
	var limit int
	remoteSearch := &cobra.Command{
		Use:   "server-search",
		Short: "Search one IMAP mailbox on the server without updating the local cache",
		Long: `Run a bounded simple IMAP SEARCH against one mailbox.

The server receives the predicate and returns all matching UIDs; standard IMAP
has no portable server-side result cap. This command fetches and prints at most
100 most-recent matching envelopes and never prints message bodies. It inherits
the existing IMAP client's connection timeout policy; this command adds no
per-request socket deadline.`,
		Run: func(cmd *cobra.Command, args []string) {
			opts, err := mail.ParseRemoteSearchOptions(from, subject, body, since, before, limit)
			if err != nil {
				output.Fatal("VALIDATION_ERROR", err)
			}
			if folder == "" {
				output.Fatal("VALIDATION_ERROR", fmt.Errorf("--folder is required"))
			}
			result, err := mail.RemoteSearch(folder, opts)
			if err != nil {
				output.Fatal("IMAP_SEARCH_ERROR", err)
			}
			output.JSON(result)
		},
	}
	remoteSearch.Flags().StringVar(&from, "from", "", "Server-side From-header substring, at most 256 characters")
	remoteSearch.Flags().StringVar(&subject, "subject", "", "Server-side Subject-header substring, at most 256 characters")
	remoteSearch.Flags().StringVar(&body, "body", "", "Server-side body substring, at most 256 characters")
	remoteSearch.Flags().StringVar(&since, "since", "", "Sent on or after date (YYYY-MM-DD)")
	remoteSearch.Flags().StringVar(&before, "before", "", "Sent before date (YYYY-MM-DD)")
	remoteSearch.Flags().StringVar(&folder, "folder", "INBOX", "Single IMAP mailbox to search")
	remoteSearch.Flags().IntVar(&limit, "limit", 50, "Maximum envelopes to fetch and return (1-100)")
	mailCmd.AddCommand(remoteSearch)
}
