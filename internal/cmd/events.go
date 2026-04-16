package cmd

import (
	"os"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Subscribe to real-time Lark events",
}

// --- events subscribe ---

var (
	eventsSubscribeType    string
	eventsSubscribeMatch   string
	eventsSubscribeOutput  string
	eventsSubscribeCompact bool
)

var eventsSubscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Stream live events (NDJSON)",
	Long: `Subscribe to real-time Lark events and stream them as NDJSON.

By default, events are printed to stdout. Use --output to redirect to a file.
Filter by event type with --type, or by regex on payload with --match.

The command runs until interrupted (Ctrl-C).

Examples:
  lark events subscribe
  lark events subscribe --type im.message.receive_v1
  lark events subscribe --match '"chat_id":"oc_xxx"'
  lark events subscribe --output events.ndjson --compact`,
	Run: func(cmd *cobra.Command, args []string) {
		opts := &api.EventSubscribeOptions{
			EventType: eventsSubscribeType,
			Compact:   eventsSubscribeCompact,
		}
		if eventsSubscribeMatch != "" {
			re, err := regexp.Compile(eventsSubscribeMatch)
			if err != nil {
				output.Fatalf("VALIDATION_ERROR", "--match must be a valid regex: %v", err)
			}
			opts.Match = re
		}

		var writer = os.Stdout
		if eventsSubscribeOutput != "" {
			f, err := os.OpenFile(eventsSubscribeOutput, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				output.Fatal("FILE_ERROR", err)
			}
			defer f.Close()
			writer = f
		}

		client := api.NewClient()
		if err := client.SubscribeEvents(writer, opts); err != nil {
			output.Fatal("API_ERROR", err)
		}
	},
}

func init() {
	eventsSubscribeCmd.Flags().StringVar(&eventsSubscribeType, "type", "", "Filter by event_type (e.g., im.message.receive_v1)")
	eventsSubscribeCmd.Flags().StringVar(&eventsSubscribeMatch, "match", "", "Filter events whose payload matches this regex")
	eventsSubscribeCmd.Flags().StringVar(&eventsSubscribeOutput, "output", "", "Write events to file instead of stdout")
	eventsSubscribeCmd.Flags().BoolVar(&eventsSubscribeCompact, "compact", false, "Emit single-line JSON")

	eventsCmd.AddCommand(eventsSubscribeCmd)
}
