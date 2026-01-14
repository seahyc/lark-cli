package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var minutesCmd = &cobra.Command{
	Use:   "minutes",
	Short: "Minutes commands",
	Long:  "Access Lark Minutes recordings - get metadata, export transcripts, and download media",
}

// --- minutes get ---

var minutesGetCmd = &cobra.Command{
	Use:   "get <minute_token>",
	Short: "Get minutes metadata",
	Long: `Get metadata for a Lark Minutes recording.

The minute_token can be obtained from the minutes URL.
It is typically the last 24 characters of the URL.

Examples:
  lark minutes get obcnq3b9jl72l83w4f14xxxx`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		minuteToken := args[0]

		client := api.NewClient()

		minute, err := client.GetMinute(minuteToken)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}

		if minute == nil {
			output.Fatalf("NOT_FOUND", "Minutes not found: %s", minuteToken)
		}

		// Parse timestamps and duration
		createTimeISO := ""
		if minute.CreateTime != "" {
			if ms, err := strconv.ParseInt(minute.CreateTime, 10, 64); err == nil {
				t := time.UnixMilli(ms)
				createTimeISO = t.Format(time.RFC3339)
			}
		}

		durationSeconds := 0
		durationDisplay := ""
		if minute.Duration != "" {
			if ms, err := strconv.ParseInt(minute.Duration, 10, 64); err == nil {
				durationSeconds = int(ms / 1000)
				durationDisplay = formatDuration(durationSeconds)
			}
		}

		result := api.OutputMinute{
			Token:           minute.Token,
			Title:           minute.Title,
			OwnerID:         minute.OwnerID,
			CreateTime:      createTimeISO,
			DurationSeconds: durationSeconds,
			DurationDisplay: durationDisplay,
			URL:             minute.URL,
		}

		output.JSON(result)
	},
}

// --- minutes transcript ---

var (
	transcriptFormat    string
	transcriptSpeaker   bool
	transcriptTimestamp bool
	transcriptOutput    string
)

var minutesTranscriptCmd = &cobra.Command{
	Use:   "transcript <minute_token>",
	Short: "Export minutes transcript",
	Long: `Export the transcript of a Lark Minutes recording.

Supports TXT and SRT formats. Can optionally include speaker names and timestamps.

Examples:
  lark minutes transcript obcnq3b9jl72l83w4f14xxxx
  lark minutes transcript obcnq3b9jl72l83w4f14xxxx --format srt
  lark minutes transcript obcnq3b9jl72l83w4f14xxxx --speaker --timestamp
  lark minutes transcript obcnq3b9jl72l83w4f14xxxx --output transcript.txt`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		minuteToken := args[0]

		client := api.NewClient()

		opts := api.TranscriptOptions{
			NeedSpeaker:   transcriptSpeaker,
			NeedTimestamp: transcriptTimestamp,
			FileFormat:    transcriptFormat,
		}

		content, err := client.GetMinuteTranscript(minuteToken, opts)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}

		// Write to file if output path specified
		if transcriptOutput != "" {
			if err := os.WriteFile(transcriptOutput, content, 0644); err != nil {
				output.Fatal("FILE_ERROR", err)
			}

			result := api.OutputMinuteTranscript{
				Token:  minuteToken,
				Format: transcriptFormat,
				File:   transcriptOutput,
			}
			output.JSON(result)
			return
		}

		// Output transcript content
		result := api.OutputMinuteTranscript{
			Token:   minuteToken,
			Format:  transcriptFormat,
			Content: string(content),
		}
		output.JSON(result)
	},
}

// --- minutes media ---

var minutesMediaCmd = &cobra.Command{
	Use:   "media <minute_token>",
	Short: "Get media download URL",
	Long: `Get the download URL for a minutes audio/video file.

The returned URL is valid for 24 hours.

Examples:
  lark minutes media obcnq3b9jl72l83w4f14xxxx`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		minuteToken := args[0]

		client := api.NewClient()

		downloadURL, err := client.GetMinuteMediaURL(minuteToken)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}

		if downloadURL == "" {
			output.Fatalf("NOT_FOUND", "No media URL available for: %s", minuteToken)
		}

		result := api.OutputMinuteMedia{
			Token:       minuteToken,
			DownloadURL: downloadURL,
		}

		output.JSON(result)
	},
}

// formatDuration converts seconds to a human-readable duration string
func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func init() {
	// minutes transcript flags
	minutesTranscriptCmd.Flags().StringVar(&transcriptFormat, "format", "txt", "Output format (txt or srt)")
	minutesTranscriptCmd.Flags().BoolVar(&transcriptSpeaker, "speaker", false, "Include speaker names")
	minutesTranscriptCmd.Flags().BoolVar(&transcriptTimestamp, "timestamp", false, "Include timestamps")
	minutesTranscriptCmd.Flags().StringVar(&transcriptOutput, "output", "", "Write transcript to file instead of stdout")

	// Register subcommands
	minutesCmd.AddCommand(minutesGetCmd)
	minutesCmd.AddCommand(minutesTranscriptCmd)
	minutesCmd.AddCommand(minutesMediaCmd)
}
