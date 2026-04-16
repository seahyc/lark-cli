package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var approvalCmd = &cobra.Command{
	Use:   "approval",
	Short: "Approval workflows",
	Long:  "List, view, approve, reject, transfer, cancel, and cc Lark approval tasks",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		validateScopeGroup("approval")
	},
}

// --- approval list ---

var (
	approvalListUserID    string
	approvalListPageSize  int
	approvalListPageToken string
	approvalListLocale    string
)

var approvalListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending approval tasks for the current user",
	Long: `Query pending approval tasks the user needs to act on.

Returns a paginated list of approval tasks. If --user is omitted, Lark infers
the current user from the tenant token identity.

Examples:
  lark approval list
  lark approval list --user ou_abc123
  lark approval list --page-size 50`,
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		tasks, hasMore, pageToken, err := client.ListApprovalTasks(
			approvalListUserID, approvalListPageSize, approvalListPageToken, approvalListLocale,
		)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"tasks":      tasks,
			"count":      len(tasks),
			"has_more":   hasMore,
			"page_token": pageToken,
		})
	},
}

// --- approval get ---

var approvalGetLocale string

var approvalGetCmd = &cobra.Command{
	Use:   "get <instance-code>",
	Short: "Get approval instance details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		inst, err := client.GetApprovalInstance(args[0], approvalGetLocale)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(inst)
	},
}

// --- approval approve / reject / transfer ---

var (
	approvalActApprovalCode string
	approvalActInstanceCode string
	approvalActUserID       string
	approvalActComment      string
	approvalActTransferTo   string
)

var approvalApproveCmd = &cobra.Command{
	Use:   "approve <task-id>",
	Short: "Approve a pending approval task",
	Long: `Approve an approval task. All identifiers are required by the Lark API:
--approval-code, --instance-code, and --user (the approver).`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if approvalActApprovalCode == "" || approvalActInstanceCode == "" || approvalActUserID == "" {
			output.Fatalf("VALIDATION_ERROR", "--approval-code, --instance-code and --user are required")
		}
		client := api.NewClient()
		if err := client.ApproveApprovalTask(
			approvalActApprovalCode, approvalActInstanceCode, approvalActUserID, args[0], approvalActComment,
		); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "task_id": args[0]})
	},
}

var approvalRejectCmd = &cobra.Command{
	Use:   "reject <task-id>",
	Short: "Reject a pending approval task",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if approvalActApprovalCode == "" || approvalActInstanceCode == "" || approvalActUserID == "" {
			output.Fatalf("VALIDATION_ERROR", "--approval-code, --instance-code and --user are required")
		}
		client := api.NewClient()
		if err := client.RejectApprovalTask(
			approvalActApprovalCode, approvalActInstanceCode, approvalActUserID, args[0], approvalActComment,
		); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "task_id": args[0]})
	},
}

var approvalTransferCmd = &cobra.Command{
	Use:   "transfer <task-id>",
	Short: "Transfer a pending approval task to another user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if approvalActApprovalCode == "" || approvalActInstanceCode == "" || approvalActUserID == "" || approvalActTransferTo == "" {
			output.Fatalf("VALIDATION_ERROR", "--approval-code, --instance-code, --user and --to are required")
		}
		client := api.NewClient()
		if err := client.TransferApprovalTask(
			approvalActApprovalCode, approvalActInstanceCode, approvalActUserID, args[0], approvalActTransferTo, approvalActComment,
		); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "task_id": args[0], "transferred_to": approvalActTransferTo})
	},
}

// --- approval cancel ---

var approvalCancelCmd = &cobra.Command{
	Use:   "cancel <instance-code>",
	Short: "Cancel an approval instance started by the user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if approvalActApprovalCode == "" || approvalActUserID == "" {
			output.Fatalf("VALIDATION_ERROR", "--approval-code and --user are required")
		}
		client := api.NewClient()
		if err := client.CancelApprovalInstance(approvalActApprovalCode, args[0], approvalActUserID); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "instance_code": args[0]})
	},
}

// --- approval cc ---

var approvalCCUsers []string

var approvalCCCmd = &cobra.Command{
	Use:   "cc <instance-code>",
	Short: "CC an approval instance to additional users",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if approvalActApprovalCode == "" || approvalActUserID == "" || len(approvalCCUsers) == 0 {
			output.Fatalf("VALIDATION_ERROR", "--approval-code, --user, and --to are required")
		}
		client := api.NewClient()
		if err := client.CCApprovalInstance(approvalActApprovalCode, args[0], approvalActUserID, approvalCCUsers, approvalActComment); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "instance_code": args[0], "cc_user_ids": approvalCCUsers})
	},
}

func init() {
	approvalListCmd.Flags().StringVar(&approvalListUserID, "user", "", "User open_id to query for (default: current user)")
	approvalListCmd.Flags().IntVar(&approvalListPageSize, "page-size", 20, "Tasks per page")
	approvalListCmd.Flags().StringVar(&approvalListPageToken, "page-token", "", "Pagination token")
	approvalListCmd.Flags().StringVar(&approvalListLocale, "locale", "en-US", "Locale for titles (en-US, zh-CN, ja-JP)")

	approvalGetCmd.Flags().StringVar(&approvalGetLocale, "locale", "en-US", "Locale for titles")

	// Shared flags for action commands
	for _, c := range []*cobra.Command{approvalApproveCmd, approvalRejectCmd, approvalTransferCmd, approvalCancelCmd, approvalCCCmd} {
		c.Flags().StringVar(&approvalActApprovalCode, "approval-code", "", "Approval code (required)")
		c.Flags().StringVar(&approvalActUserID, "user", "", "Acting user open_id (required)")
	}
	for _, c := range []*cobra.Command{approvalApproveCmd, approvalRejectCmd, approvalTransferCmd} {
		c.Flags().StringVar(&approvalActInstanceCode, "instance-code", "", "Approval instance code (required)")
		c.Flags().StringVar(&approvalActComment, "comment", "", "Comment for the action")
	}
	approvalTransferCmd.Flags().StringVar(&approvalActTransferTo, "to", "", "Transfer target user open_id (required)")
	approvalCCCmd.Flags().StringSliceVar(&approvalCCUsers, "to", nil, "User open_ids to cc (comma-separated, required)")
	approvalCCCmd.Flags().StringVar(&approvalActComment, "comment", "", "CC comment")

	approvalCmd.AddCommand(approvalListCmd)
	approvalCmd.AddCommand(approvalGetCmd)
	approvalCmd.AddCommand(approvalApproveCmd)
	approvalCmd.AddCommand(approvalRejectCmd)
	approvalCmd.AddCommand(approvalTransferCmd)
	approvalCmd.AddCommand(approvalCancelCmd)
	approvalCmd.AddCommand(approvalCCCmd)
}
