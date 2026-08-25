package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var anycrossCmd = &cobra.Command{
	Use:   "anycross",
	Short: "Inspect AnyCross and safely manage workflow drafts",
	Long: `Inspect projects, workflows, and runtime logs through the AnyCross browser-console API.

Set ANYCROSS_SESSION_COOKIE to an authenticated AnyCross console Cookie header.
Commands that use POST also require ANYCROSS_CSRF_TOKEN. Workflow definitions
and log payloads are recursively redacted before output. Draft creation and
patching are dry-run by default, require exact target confirmation to apply,
and never publish, enable, or run a workflow.`,
}

var anycrossTenantCmd = &cobra.Command{
	Use:   "tenant",
	Short: "Show the current AnyCross tenant and verify authentication",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustAnyCrossClient()
		tenant, err := client.CurrentTenant()
		if err != nil {
			output.Fatal("ANYCROSS_API_ERROR", err)
		}
		output.JSON(tenant)
	},
}

var (
	anycrossProjectQuery    string
	anycrossProjectPage     int
	anycrossProjectPageSize int
)

var anycrossProjectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List AnyCross integration projects",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustAnyCrossClient()
		projects, err := client.ListProjects(anycrossProjectQuery, anycrossProjectPage, anycrossProjectPageSize)
		if err != nil {
			output.Fatal("ANYCROSS_API_ERROR", err)
		}
		output.JSON(projects)
	},
}

var anycrossWorkflowsCmd = &cobra.Command{
	Use:   "workflows <project-id>",
	Short: "List workflows in an AnyCross project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustAnyCrossClient()
		workflows, err := client.ListWorkflows(args[0])
		if err != nil {
			output.Fatal("ANYCROSS_API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"workflows": workflows, "count": len(workflows)})
	},
}

var anycrossWorkflowCmd = &cobra.Command{
	Use:   "workflow <flow-id>",
	Short: "Read a workflow definition with credentials redacted",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustAnyCrossClient()
		workflow, err := client.GetWorkflow(args[0])
		if err != nil {
			output.Fatal("ANYCROSS_API_ERROR", err)
		}
		output.JSON(workflow)
	},
}

var (
	anycrossCreateProjectID        string
	anycrossCreateName             string
	anycrossCreateDescription      string
	anycrossCreateTriggerVersionID string
	anycrossCreateFolderID         string
	anycrossCreateApply            bool
	anycrossCreateConfirmation     string
)

var anycrossCreateWorkflowCmd = &cobra.Command{
	Use:   "create-workflow",
	Short: "Create an unpublished AnyCross workflow draft",
	Long: `Create an unpublished workflow with a trigger connector.

The command is a dry run unless --apply is supplied. Applying also requires
--confirmation to exactly match the target project ID. It never publishes,
enables, or runs the new workflow.`,
	Run: func(cmd *cobra.Command, args []string) {
		if strings.TrimSpace(anycrossCreateProjectID) == "" || strings.TrimSpace(anycrossCreateName) == "" || strings.TrimSpace(anycrossCreateTriggerVersionID) == "" {
			output.Fatalf("VALIDATION_ERROR", "--project-id, --name, and --trigger-version-id are required")
		}
		if !anycrossCreateApply {
			output.JSON(map[string]interface{}{
				"applied": false, "requires_confirmation": true,
				"project_id": anycrossCreateProjectID, "name": anycrossCreateName,
				"description":        anycrossCreateDescription,
				"trigger_version_id": anycrossCreateTriggerVersionID,
				"folder_id":          anycrossCreateFolderID,
			})
			return
		}
		if anycrossCreateConfirmation != anycrossCreateProjectID {
			output.Fatalf("CONFIRMATION_REQUIRED", "--confirmation must exactly match project ID %s", anycrossCreateProjectID)
		}
		client := mustAnyCrossClient()
		flowID, err := client.CreateWorkflowDraft(api.AnyCrossCreateWorkflow{
			ProjectID: anycrossCreateProjectID, Name: anycrossCreateName,
			Description:      anycrossCreateDescription,
			TriggerVersionID: anycrossCreateTriggerVersionID, FolderID: anycrossCreateFolderID,
		})
		if err != nil {
			output.Fatal("ANYCROSS_API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"applied": true, "flow_id": flowID, "published": false, "enabled": false,
		})
	},
}

var (
	anycrossPatchJSON         string
	anycrossPatchFile         string
	anycrossPatchExpectedAt   int64
	anycrossPatchApply        bool
	anycrossPatchConfirmation string
)

var anycrossPatchWorkflowCmd = &cobra.Command{
	Use:   "patch-workflow <flow-id>",
	Short: "Preview or apply a guarded patch to a workflow draft",
	Long: `Patch an unpublished workflow draft using add/replace/remove JSON Patch operations.

Only /steps and /structure paths are accepted; credential and secret paths are
rejected. Existing credentials remain in memory and are preserved. The command
is a dry run unless --apply is supplied. Applying requires --confirmation to
exactly match the flow ID and --expected-updated-at for optimistic locking.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		flowID := args[0]
		if anycrossPatchExpectedAt <= 0 {
			output.Fatalf("VALIDATION_ERROR", "--expected-updated-at is required")
		}
		if (anycrossPatchJSON == "") == (anycrossPatchFile == "") {
			output.Fatalf("VALIDATION_ERROR", "provide exactly one of --patch or --patch-file")
		}
		patchBytes := []byte(anycrossPatchJSON)
		if anycrossPatchFile != "" {
			var err error
			patchBytes, err = os.ReadFile(anycrossPatchFile)
			if err != nil {
				output.Fatal("FILE_ERROR", err)
			}
		}
		var operations []api.AnyCrossJSONPatchOperation
		if err := json.Unmarshal(patchBytes, &operations); err != nil {
			output.Fatalf("VALIDATION_ERROR", "invalid JSON patch: %v", err)
		}
		if anycrossPatchApply && anycrossPatchConfirmation != flowID {
			output.Fatalf("CONFIRMATION_REQUIRED", "--confirmation must exactly match flow ID %s", flowID)
		}
		client := mustAnyCrossClient()
		plan, err := client.PatchWorkflowDraft(flowID, anycrossPatchExpectedAt, operations, anycrossPatchApply)
		if err != nil {
			output.Fatal("ANYCROSS_API_ERROR", err)
		}
		output.JSON(plan)
	},
}

var (
	anycrossLogsAfter    string
	anycrossLogsBefore   string
	anycrossLogsProjects string
	anycrossLogsFlows    string
	anycrossLogsStatuses string
	anycrossLogsPage     int
	anycrossLogsPageSize int
	anycrossLogsDebug    bool
)

var anycrossLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Search AnyCross runtime logs",
	Run: func(cmd *cobra.Command, args []string) {
		client := mustAnyCrossClient()
		opts := api.AnyCrossLogSearch{
			ProjectIDs: splitCSV(anycrossLogsProjects), FlowIDs: splitCSV(anycrossLogsFlows),
			Statuses: splitCSV(anycrossLogsStatuses), Page: anycrossLogsPage,
			PageSize: anycrossLogsPageSize, Debug: anycrossLogsDebug,
		}
		if anycrossLogsAfter != "" {
			opts.AfterMS = parseAnyCrossTime(anycrossLogsAfter)
		}
		if anycrossLogsBefore != "" {
			opts.BeforeMS = parseAnyCrossTime(anycrossLogsBefore)
		}
		logs, err := client.SearchLogs(opts)
		if err != nil {
			output.Fatal("ANYCROSS_API_ERROR", err)
		}
		output.JSON(logs)
	},
}

var anycrossLogDebug bool

var anycrossLogCmd = &cobra.Command{
	Use:   "log <flow-id> <flow-instance-id>",
	Short: "Read a runtime log and its node tree",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := mustAnyCrossClient()
		log, err := client.GetLog(args[0], args[1], anycrossLogDebug)
		if err != nil {
			output.Fatal("ANYCROSS_API_ERROR", err)
		}
		output.JSON(log)
	},
}

func mustAnyCrossClient() *api.AnyCrossClient {
	client, err := api.NewAnyCrossClient()
	if err != nil {
		output.Fatal("ANYCROSS_CONFIG_ERROR", err)
	}
	return client
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseAnyCrossTime(value string) int64 {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UnixMilli()
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.UnixMilli()
	}
	output.Fatalf("VALIDATION_ERROR", "invalid time %q; use RFC3339 or YYYY-MM-DD", value)
	return 0
}

func init() {
	anycrossProjectsCmd.Flags().StringVar(&anycrossProjectQuery, "query", "", "Search project names")
	anycrossProjectsCmd.Flags().IntVar(&anycrossProjectPage, "page", 1, "Page number")
	anycrossProjectsCmd.Flags().IntVar(&anycrossProjectPageSize, "page-size", 40, "Projects per page (max 100)")

	anycrossLogsCmd.Flags().StringVar(&anycrossLogsAfter, "after", "", "Start time (RFC3339 or YYYY-MM-DD; default 24h ago)")
	anycrossLogsCmd.Flags().StringVar(&anycrossLogsBefore, "before", "", "End time (RFC3339 or YYYY-MM-DD; default now)")
	anycrossLogsCmd.Flags().StringVar(&anycrossLogsProjects, "projects", "", "Comma-separated project IDs")
	anycrossLogsCmd.Flags().StringVar(&anycrossLogsFlows, "flows", "", "Comma-separated workflow IDs")
	anycrossLogsCmd.Flags().StringVar(&anycrossLogsStatuses, "statuses", "", "Comma-separated statuses")
	anycrossLogsCmd.Flags().IntVar(&anycrossLogsPage, "page", 1, "Page number")
	anycrossLogsCmd.Flags().IntVar(&anycrossLogsPageSize, "page-size", 20, "Logs per page (max 100)")
	anycrossLogsCmd.Flags().BoolVar(&anycrossLogsDebug, "debug", false, "Search debug runs")
	anycrossLogCmd.Flags().BoolVar(&anycrossLogDebug, "debug", false, "Read a debug run")

	anycrossCreateWorkflowCmd.Flags().StringVar(&anycrossCreateProjectID, "project-id", "", "Target project ID")
	anycrossCreateWorkflowCmd.Flags().StringVar(&anycrossCreateName, "name", "", "Workflow name")
	anycrossCreateWorkflowCmd.Flags().StringVar(&anycrossCreateDescription, "description", "", "Workflow description")
	anycrossCreateWorkflowCmd.Flags().StringVar(&anycrossCreateTriggerVersionID, "trigger-version-id", "", "Trigger connector version ID")
	anycrossCreateWorkflowCmd.Flags().StringVar(&anycrossCreateFolderID, "folder-id", "", "Optional target folder ID")
	anycrossCreateWorkflowCmd.Flags().BoolVar(&anycrossCreateApply, "apply", false, "Create the draft (default is a dry run)")
	anycrossCreateWorkflowCmd.Flags().StringVar(&anycrossCreateConfirmation, "confirmation", "", "Must exactly match project ID when --apply is used")

	anycrossPatchWorkflowCmd.Flags().StringVar(&anycrossPatchJSON, "patch", "", "JSON array of patch operations")
	anycrossPatchWorkflowCmd.Flags().StringVar(&anycrossPatchFile, "patch-file", "", "Read patch operations from a JSON file")
	anycrossPatchWorkflowCmd.Flags().Int64Var(&anycrossPatchExpectedAt, "expected-updated-at", 0, "Expected workflow updatedAt timestamp")
	anycrossPatchWorkflowCmd.Flags().BoolVar(&anycrossPatchApply, "apply", false, "Save the draft (default is preview only)")
	anycrossPatchWorkflowCmd.Flags().StringVar(&anycrossPatchConfirmation, "confirmation", "", "Must exactly match flow ID when --apply is used")

	anycrossCmd.AddCommand(anycrossTenantCmd)
	anycrossCmd.AddCommand(anycrossProjectsCmd)
	anycrossCmd.AddCommand(anycrossWorkflowsCmd)
	anycrossCmd.AddCommand(anycrossWorkflowCmd)
	anycrossCmd.AddCommand(anycrossCreateWorkflowCmd)
	anycrossCmd.AddCommand(anycrossPatchWorkflowCmd)
	anycrossCmd.AddCommand(anycrossLogsCmd)
	anycrossCmd.AddCommand(anycrossLogCmd)
}
