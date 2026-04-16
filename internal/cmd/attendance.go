package cmd

import (
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var (
	attendanceAfter  string
	attendanceBefore string
	attendanceUser   string
	attendanceOT     bool
)

var attendanceCmd = &cobra.Command{
	Use:   "attendance",
	Short: "Attendance (clock-in) records",
	Long:  "Query Lark Attendance user tasks — defaults to the current user and the last 7 days.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		validateScopeGroup("attendance")
	},
}

var attendanceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List attendance user tasks (check-in records)",
	Long: `List attendance check-in/out records for a user over a date range.

By default queries the current user's records for the last 7 days.
Dates accept YYYY-MM-DD or YYYYMMDD; --after defaults to 7 days ago, --before defaults to today.

Examples:
  lark attendance list
  lark attendance list --after 2025-01-01 --before 2025-01-31
  lark attendance list --user ou_xxx --after 2025-03-01`,
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()

		userID := attendanceUser
		if userID == "" {
			me, err := client.GetCurrentUser()
			if err != nil {
				output.Fatal("API_ERROR", err)
			}
			userID = me.OpenID
		}

		now := time.Now()
		from := now.AddDate(0, 0, -7)
		to := now
		if attendanceAfter != "" {
			t, err := parseAttendanceDate(attendanceAfter)
			if err != nil {
				output.Fatalf("VALIDATION_ERROR", "invalid --after: %v", err)
			}
			from = t
		}
		if attendanceBefore != "" {
			t, err := parseAttendanceDate(attendanceBefore)
			if err != nil {
				output.Fatalf("VALIDATION_ERROR", "invalid --before: %v", err)
			}
			to = t
		}

		fromInt := dateToYYYYMMDD(from)
		toInt := dateToYYYYMMDD(to)

		tasks, err := client.ListAttendanceUserTasks([]string{userID}, fromInt, toInt, attendanceOT)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}

		output.JSON(map[string]interface{}{
			"user_id":         userID,
			"check_date_from": fromInt,
			"check_date_to":   toInt,
			"count":           len(tasks),
			"tasks":           tasks,
		})
	},
}

// parseAttendanceDate accepts YYYY-MM-DD or YYYYMMDD and returns a time.Time (UTC, midnight).
func parseAttendanceDate(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse("20060102", s)
}

func dateToYYYYMMDD(t time.Time) int {
	y, m, d := t.Date()
	n, _ := strconv.Atoi(
		// YYYYMMDD
		formatInt4(y) + formatInt2(int(m)) + formatInt2(d),
	)
	return n
}

func formatInt4(n int) string {
	s := strconv.Itoa(n)
	for len(s) < 4 {
		s = "0" + s
	}
	return s
}

func formatInt2(n int) string {
	s := strconv.Itoa(n)
	if len(s) < 2 {
		s = "0" + s
	}
	return s
}

func init() {
	attendanceListCmd.Flags().StringVar(&attendanceAfter, "after", "", "Start date (YYYY-MM-DD); defaults to 7 days ago")
	attendanceListCmd.Flags().StringVar(&attendanceBefore, "before", "", "End date (YYYY-MM-DD); defaults to today")
	attendanceListCmd.Flags().StringVar(&attendanceUser, "user", "", "Target user's open_id; defaults to the current user")
	attendanceListCmd.Flags().BoolVar(&attendanceOT, "overtime", false, "Include overtime result in the response")

	attendanceCmd.AddCommand(attendanceListCmd)
}
