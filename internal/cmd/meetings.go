package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var meetingsCmd = &cobra.Command{
	Use:   "meetings",
	Short: "Video conferencing: search meetings, fetch notes/transcripts/recordings",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		validateScopeGroup("meetings")
	},
}

// --- meetings search ---

var (
	meetingsSearchAfter       string
	meetingsSearchBefore      string
	meetingsSearchOrganizer   string
	meetingsSearchParticipant string
	meetingsSearchMeetingNo   string
	meetingsSearchLimit       int
)

var meetingsSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search meeting history",
	Long: `Search meetings using time range and optional organizer/participant filters.

Examples:
  lark meetings search --after 2026-04-01 --before 2026-04-15
  lark meetings search --organizer ou_xxx --limit 10
  lark meetings search --meeting-no 1234567890`,
	Run: func(cmd *cobra.Command, args []string) {
		opts := &api.SearchMeetingsOptions{
			MeetingNo:     meetingsSearchMeetingNo,
			OrganizerID:   meetingsSearchOrganizer,
			ParticipantID: meetingsSearchParticipant,
		}
		if meetingsSearchAfter != "" {
			opts.StartTime = parseTimeArg(meetingsSearchAfter)
		}
		if meetingsSearchBefore != "" {
			opts.EndTime = parseTimeArg(meetingsSearchBefore)
		}

		client := api.NewClient()
		var all []api.Meeting
		var pageToken string
		hasMore := true
		remaining := meetingsSearchLimit
		for hasMore {
			pageSize := 20
			if remaining > 0 && remaining < pageSize {
				pageSize = remaining
			}
			opts.PageSize = pageSize
			opts.PageToken = pageToken
			meetings, more, next, err := client.SearchMeetings(opts)
			if err != nil {
				output.Fatal("API_ERROR", err)
			}
			all = append(all, meetings...)
			hasMore = more
			pageToken = next
			if meetingsSearchLimit > 0 {
				remaining = meetingsSearchLimit - len(all)
				if remaining <= 0 {
					break
				}
			}
		}
		if meetingsSearchLimit > 0 && len(all) > meetingsSearchLimit {
			all = all[:meetingsSearchLimit]
		}
		output.JSON(map[string]interface{}{
			"meetings": all,
			"count":    len(all),
		})
	},
}

// --- meetings get ---

var meetingsGetCmd = &cobra.Command{
	Use:   "get <meeting-id>",
	Short: "Get meeting metadata",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		meeting, err := client.GetMeeting(args[0])
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(meeting)
	},
}

// --- meetings notes ---

var meetingsNotesCmd = &cobra.Command{
	Use:   "notes <meeting-id>",
	Short: "Get meeting notes",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		notes, err := client.GetMeetingNotes(args[0])
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(notes)
	},
}

// --- meetings recording ---

var meetingsRecordingCmd = &cobra.Command{
	Use:   "recording <meeting-id>",
	Short: "Get meeting recording info (URL + minute_token)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		rec, err := client.GetMeetingRecording(args[0])
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(rec)
	},
}

// --- meetings transcript ---

var (
	meetingsTranscriptFormat    string
	meetingsTranscriptSpeaker   bool
	meetingsTranscriptTimestamp bool
	meetingsTranscriptOutput    string
)

var meetingsTranscriptCmd = &cobra.Command{
	Use:   "transcript <meeting-id>",
	Short: "Export meeting transcript (resolves minute_token, then exports)",
	Long: `Export the transcript for a meeting.

This resolves the meeting's minute_token via the VC recording API, then delegates
to the existing minutes transcript export. Supports TXT and SRT formats.

Examples:
  lark meetings transcript 707xxxxx
  lark meetings transcript 707xxxxx --format srt --speaker --timestamp
  lark meetings transcript 707xxxxx --output transcript.txt`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		rec, err := client.GetMeetingRecording(args[0])
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		if rec == nil || rec.MinuteToken == "" {
			output.Fatalf("NOT_FOUND", "No minute_token available for meeting %s", args[0])
		}

		opts := api.TranscriptOptions{
			NeedSpeaker:   meetingsTranscriptSpeaker,
			NeedTimestamp: meetingsTranscriptTimestamp,
			FileFormat:    meetingsTranscriptFormat,
		}
		content, err := client.GetMinuteTranscript(rec.MinuteToken, opts)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}

		if meetingsTranscriptOutput != "" {
			if err := os.WriteFile(meetingsTranscriptOutput, content, 0644); err != nil {
				output.Fatal("FILE_ERROR", err)
			}
			output.JSON(map[string]interface{}{
				"meeting_id":   args[0],
				"minute_token": rec.MinuteToken,
				"format":       meetingsTranscriptFormat,
				"file":         meetingsTranscriptOutput,
			})
			return
		}

		output.JSON(map[string]interface{}{
			"meeting_id":   args[0],
			"minute_token": rec.MinuteToken,
			"format":       meetingsTranscriptFormat,
			"content":      string(content),
		})
	},
}

func init() {
	meetingsSearchCmd.Flags().StringVar(&meetingsSearchAfter, "after", "", "Start time (Unix timestamp or ISO 8601)")
	meetingsSearchCmd.Flags().StringVar(&meetingsSearchBefore, "before", "", "End time (Unix timestamp or ISO 8601)")
	meetingsSearchCmd.Flags().StringVar(&meetingsSearchOrganizer, "organizer", "", "Filter by organizer open_id")
	meetingsSearchCmd.Flags().StringVar(&meetingsSearchParticipant, "participant", "", "Filter by participant open_id")
	meetingsSearchCmd.Flags().StringVar(&meetingsSearchMeetingNo, "meeting-no", "", "Filter by meeting number")
	meetingsSearchCmd.Flags().IntVar(&meetingsSearchLimit, "limit", 0, "Maximum number of meetings (0 = no limit)")

	meetingsTranscriptCmd.Flags().StringVar(&meetingsTranscriptFormat, "format", "txt", "Output format (txt or srt)")
	meetingsTranscriptCmd.Flags().BoolVar(&meetingsTranscriptSpeaker, "speaker", false, "Include speaker names")
	meetingsTranscriptCmd.Flags().BoolVar(&meetingsTranscriptTimestamp, "timestamp", false, "Include timestamps")
	meetingsTranscriptCmd.Flags().StringVar(&meetingsTranscriptOutput, "output", "", "Write transcript to file instead of stdout")

	meetingsCmd.AddCommand(meetingsSearchCmd)
	meetingsCmd.AddCommand(meetingsGetCmd)
	meetingsCmd.AddCommand(meetingsNotesCmd)
	meetingsCmd.AddCommand(meetingsRecordingCmd)
	meetingsCmd.AddCommand(meetingsTranscriptCmd)
}
