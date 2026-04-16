package cmd

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

// --- task create ---

var (
	taskCreateSummary string
	taskCreateDue     string
	taskCreateAllDay  bool
)

var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a task",
	Long: `Create a new task assigned to yourself.

Examples:
  lark task create --summary "Review PR"
  lark task create --summary "Ship feature" --due 2026-04-20
  lark task create --summary "Daily standup" --due 2026-04-17T09:30:00 --all-day=false`,
	Run: func(cmd *cobra.Command, args []string) {
		if taskCreateSummary == "" {
			output.Fatalf("VALIDATION_ERROR", "--summary is required")
		}
		req := &api.CreateTaskRequest{Summary: taskCreateSummary}
		if taskCreateDue != "" {
			req.Due = &api.TaskDue{
				Timestamp: secondsToMillisString(parseTimeArg(taskCreateDue)),
				IsAllDay:  taskCreateAllDay,
			}
		}
		client := api.NewClient()
		task, err := client.CreateTask(req)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(taskToOutput(*task))
	},
}

// --- task update ---

var (
	taskUpdateSummary     string
	taskUpdateDescription string
	taskUpdateDue         string
)

var taskUpdateCmd = &cobra.Command{
	Use:   "update <task-guid>",
	Short: "Update a task's summary, description, or due date",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fields := map[string]interface{}{}
		var updateFields []string
		if taskUpdateSummary != "" {
			fields["summary"] = taskUpdateSummary
			updateFields = append(updateFields, "summary")
		}
		if taskUpdateDescription != "" {
			fields["description"] = taskUpdateDescription
			updateFields = append(updateFields, "description")
		}
		if taskUpdateDue != "" {
			fields["due"] = map[string]interface{}{
				"timestamp": secondsToMillisString(parseTimeArg(taskUpdateDue)),
			}
			updateFields = append(updateFields, "due")
		}
		if len(updateFields) == 0 {
			output.Fatalf("VALIDATION_ERROR", "at least one of --summary, --description, --due is required")
		}
		client := api.NewClient()
		task, err := client.UpdateTask(args[0], fields, updateFields)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(taskToOutput(*task))
	},
}

// --- task complete / reopen ---

var taskCompleteCmd = &cobra.Command{
	Use:   "complete <task-guid>",
	Short: "Mark a task as completed",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		if err := client.CompleteTask(args[0]); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "task_guid": args[0]})
	},
}

var taskReopenCmd = &cobra.Command{
	Use:   "reopen <task-guid>",
	Short: "Reopen a completed task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		if err := client.ReopenTask(args[0]); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "task_guid": args[0]})
	},
}

// --- task assign ---

var (
	taskAssignAssignees []string
	taskAssignFollowers []string
	taskAssignRemove    bool
)

var taskAssignCmd = &cobra.Command{
	Use:   "assign <task-guid>",
	Short: "Add or remove task assignees / followers",
	Args:  cobra.ExactArgs(1),
	Long: `Add or remove assignees and followers from a task.

Use --remove to remove the listed members instead of adding them.
Members may be open_ids or emails (emails are resolved automatically).

Examples:
  lark task assign <guid> --assignee ou_xxx
  lark task assign <guid> --assignee alice@example.com,bob@example.com
  lark task assign <guid> --follower ou_yyy
  lark task assign <guid> --assignee ou_xxx --remove`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(taskAssignAssignees) == 0 && len(taskAssignFollowers) == 0 {
			output.Fatalf("VALIDATION_ERROR", "--assignee or --follower is required")
		}
		var members []api.TaskMember
		for _, id := range resolveMembers(taskAssignAssignees) {
			members = append(members, api.TaskMember{ID: id, Type: "user", Role: "assignee"})
		}
		for _, id := range resolveMembers(taskAssignFollowers) {
			members = append(members, api.TaskMember{ID: id, Type: "user", Role: "follower"})
		}
		client := api.NewClient()
		var err error
		if taskAssignRemove {
			err = client.RemoveTaskMembers(args[0], members)
		} else {
			err = client.AddTaskMembers(args[0], members)
		}
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":   true,
			"task_guid": args[0],
			"removed":   taskAssignRemove,
			"members":   members,
		})
	},
}

// --- task subtask ---

var taskSubtaskCmd = &cobra.Command{
	Use:   "subtask",
	Short: "Manage subtasks",
}

var (
	taskSubtaskCreateSummary string
	taskSubtaskCreateDue     string
)

var taskSubtaskCreateCmd = &cobra.Command{
	Use:   "create <parent-guid>",
	Short: "Create a subtask under a parent task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if taskSubtaskCreateSummary == "" {
			output.Fatalf("VALIDATION_ERROR", "--summary is required")
		}
		req := &api.CreateSubtaskRequest{Summary: taskSubtaskCreateSummary}
		if taskSubtaskCreateDue != "" {
			req.Due = &api.TaskDue{
				Timestamp: secondsToMillisString(parseTimeArg(taskSubtaskCreateDue)),
			}
		}
		client := api.NewClient()
		task, err := client.CreateSubtask(args[0], req)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(taskToOutput(*task))
	},
}

var (
	taskSubtaskListLimit int
)

var taskSubtaskListCmd = &cobra.Command{
	Use:   "list <parent-guid>",
	Short: "List subtasks of a parent task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		var allTasks []api.Task
		var pageToken string
		hasMore := true
		remaining := taskSubtaskListLimit
		for hasMore {
			pageSize := 50
			if remaining > 0 && remaining < pageSize {
				pageSize = remaining
			}
			tasks, more, nextToken, err := client.ListSubtasks(args[0], pageSize, pageToken)
			if err != nil {
				output.Fatal("API_ERROR", err)
			}
			allTasks = append(allTasks, tasks...)
			hasMore = more
			pageToken = nextToken
			if taskSubtaskListLimit > 0 {
				remaining = taskSubtaskListLimit - len(allTasks)
				if remaining <= 0 {
					break
				}
			}
		}
		if taskSubtaskListLimit > 0 && len(allTasks) > taskSubtaskListLimit {
			allTasks = allTasks[:taskSubtaskListLimit]
		}
		outputTasks := make([]api.OutputTask, len(allTasks))
		for i, t := range allTasks {
			outputTasks[i] = taskToOutput(t)
		}
		output.JSON(map[string]interface{}{
			"parent_guid": args[0],
			"subtasks":    outputTasks,
			"count":       len(outputTasks),
		})
	},
}

// --- task comment ---

var taskCommentContent string

var taskCommentCmd = &cobra.Command{
	Use:   "comment <task-guid>",
	Short: "Add a comment to a task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if taskCommentContent == "" {
			output.Fatalf("VALIDATION_ERROR", "--content is required")
		}
		client := api.NewClient()
		if err := client.AddTaskComment(args[0], taskCommentContent); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "task_guid": args[0]})
	},
}

// --- task remind ---

var taskRemindMinutes int

var taskRemindCmd = &cobra.Command{
	Use:   "remind <task-guid>",
	Short: "Set a reminder on a task",
	Long: `Set a reminder relative to the task's due time.

--minutes specifies how many minutes before the due time the reminder fires.
Use 0 for at the due time, negative numbers for after.

Examples:
  lark task remind <guid> --minutes 15
  lark task remind <guid> --minutes 0`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		if err := client.AddTaskReminder(args[0], taskRemindMinutes); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":          true,
			"task_guid":        args[0],
			"relative_minutes": taskRemindMinutes,
		})
	},
}

// --- task tasklist ---

var taskTasklistCmd = &cobra.Command{
	Use:   "tasklist",
	Short: "Manage tasklists",
}

var taskTasklistListLimit int

var taskTasklistListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasklists",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		var allLists []api.Tasklist
		var pageToken string
		hasMore := true
		remaining := taskTasklistListLimit
		for hasMore {
			pageSize := 50
			if remaining > 0 && remaining < pageSize {
				pageSize = remaining
			}
			lists, more, nextToken, err := client.ListTasklists(pageSize, pageToken)
			if err != nil {
				output.Fatal("API_ERROR", err)
			}
			allLists = append(allLists, lists...)
			hasMore = more
			pageToken = nextToken
			if taskTasklistListLimit > 0 {
				remaining = taskTasklistListLimit - len(allLists)
				if remaining <= 0 {
					break
				}
			}
		}
		if taskTasklistListLimit > 0 && len(allLists) > taskTasklistListLimit {
			allLists = allLists[:taskTasklistListLimit]
		}
		output.JSON(map[string]interface{}{
			"tasklists": allLists,
			"count":     len(allLists),
		})
	},
}

var taskTasklistCreateName string

var taskTasklistCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new tasklist",
	Run: func(cmd *cobra.Command, args []string) {
		if taskTasklistCreateName == "" {
			output.Fatalf("VALIDATION_ERROR", "--name is required")
		}
		client := api.NewClient()
		tl, err := client.CreateTasklist(taskTasklistCreateName)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(tl)
	},
}

var taskTasklistAddTaskGUID string

var taskTasklistAddCmd = &cobra.Command{
	Use:   "add <tasklist-guid>",
	Short: "Add a task to a tasklist",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if taskTasklistAddTaskGUID == "" {
			output.Fatalf("VALIDATION_ERROR", "--task-guid is required")
		}
		client := api.NewClient()
		if err := client.AddTaskToTasklist(args[0], taskTasklistAddTaskGUID); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":       true,
			"tasklist_guid": args[0],
			"task_guid":     taskTasklistAddTaskGUID,
		})
	},
}

// secondsToMillisString converts a Unix-seconds string to a Unix-milliseconds string.
// Tasks v2 API returns/accepts timestamps as Unix milliseconds (string).
func secondsToMillisString(s string) string {
	if s == "" {
		return ""
	}
	if strings.Contains(s, "-") || strings.Contains(s, "T") {
		// Already non-numeric (parseTimeArg should have converted, but be safe)
		return s
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return s
	}
	// If already in milliseconds (13+ digits), pass through
	if n > 1_000_000_000_000 {
		return s
	}
	return strconv.FormatInt(n*1000, 10)
}

func init() {
	taskCreateCmd.Flags().StringVar(&taskCreateSummary, "summary", "", "Task summary (required)")
	taskCreateCmd.Flags().StringVar(&taskCreateDue, "due", "", "Due date (Unix timestamp or ISO 8601)")
	taskCreateCmd.Flags().BoolVar(&taskCreateAllDay, "all-day", true, "Treat due date as all-day (default true)")

	taskUpdateCmd.Flags().StringVar(&taskUpdateSummary, "summary", "", "New summary")
	taskUpdateCmd.Flags().StringVar(&taskUpdateDescription, "description", "", "New description")
	taskUpdateCmd.Flags().StringVar(&taskUpdateDue, "due", "", "New due date (Unix timestamp or ISO 8601)")

	taskAssignCmd.Flags().StringSliceVar(&taskAssignAssignees, "assignee", nil, "Assignee IDs or emails (repeatable, comma-separated)")
	taskAssignCmd.Flags().StringSliceVar(&taskAssignFollowers, "follower", nil, "Follower IDs or emails (repeatable, comma-separated)")
	taskAssignCmd.Flags().BoolVar(&taskAssignRemove, "remove", false, "Remove the listed members instead of adding")

	taskSubtaskCreateCmd.Flags().StringVar(&taskSubtaskCreateSummary, "summary", "", "Subtask summary (required)")
	taskSubtaskCreateCmd.Flags().StringVar(&taskSubtaskCreateDue, "due", "", "Due date (Unix timestamp or ISO 8601)")
	taskSubtaskListCmd.Flags().IntVar(&taskSubtaskListLimit, "limit", 0, "Maximum number of subtasks (0 = no limit)")

	taskCommentCmd.Flags().StringVar(&taskCommentContent, "content", "", "Comment content (required)")

	taskRemindCmd.Flags().IntVar(&taskRemindMinutes, "minutes", 0, "Minutes before due time when reminder fires (0 = at due time)")

	taskTasklistListCmd.Flags().IntVar(&taskTasklistListLimit, "limit", 0, "Maximum number of tasklists (0 = no limit)")
	taskTasklistCreateCmd.Flags().StringVar(&taskTasklistCreateName, "name", "", "Tasklist name (required)")
	taskTasklistAddCmd.Flags().StringVar(&taskTasklistAddTaskGUID, "task-guid", "", "Task GUID to add (required)")

	taskSubtaskCmd.AddCommand(taskSubtaskCreateCmd)
	taskSubtaskCmd.AddCommand(taskSubtaskListCmd)

	taskTasklistCmd.AddCommand(taskTasklistListCmd)
	taskTasklistCmd.AddCommand(taskTasklistCreateCmd)
	taskTasklistCmd.AddCommand(taskTasklistAddCmd)

	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskCompleteCmd)
	taskCmd.AddCommand(taskReopenCmd)
	taskCmd.AddCommand(taskAssignCmd)
	taskCmd.AddCommand(taskSubtaskCmd)
	taskCmd.AddCommand(taskCommentCmd)
	taskCmd.AddCommand(taskRemindCmd)
	taskCmd.AddCommand(taskTasklistCmd)
}
