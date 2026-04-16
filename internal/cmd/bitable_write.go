package cmd

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

// --- bitable create ---

var (
	bitableCreateName   string
	bitableCreateFolder string
)

var bitableCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Bitable app",
	Long: `Create a new Bitable (multi-dimensional table) app.

Examples:
  lark bitable create --name "My Tracker"
  lark bitable create --name "Project DB" --folder fldABC`,
	Run: func(cmd *cobra.Command, args []string) {
		if bitableCreateName == "" {
			output.Fatalf("VALIDATION_ERROR", "--name is required")
		}
		client := api.NewClient()
		app, err := client.CreateBitable(bitableCreateName, bitableCreateFolder)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(app)
	},
}

// --- bitable get ---

var bitableGetCmd = &cobra.Command{
	Use:   "get <app-token-or-url>",
	Short: "Get Bitable metadata",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appToken := extractBitableToken(args[0])
		client := api.NewClient()
		app, err := client.GetBitable(appToken)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(app)
	},
}

// --- bitable table ---

var bitableTableCmd = &cobra.Command{
	Use:   "table",
	Short: "Manage Bitable tables",
}

var (
	bitableTableCreateAppToken string
	bitableTableCreateName     string
)

var bitableTableCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new table in a Bitable",
	Run: func(cmd *cobra.Command, args []string) {
		if bitableTableCreateAppToken == "" || bitableTableCreateName == "" {
			output.Fatalf("VALIDATION_ERROR", "--app-token and --name are required")
		}
		appToken := extractBitableToken(bitableTableCreateAppToken)
		client := api.NewClient()
		tableID, err := client.CreateTable(appToken, bitableTableCreateName)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":   true,
			"app_token": appToken,
			"table_id":  tableID,
			"name":      bitableTableCreateName,
		})
	},
}

var (
	bitableTableDeleteAppToken string
	bitableTableDeleteTableID  string
)

var bitableTableDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a table from a Bitable",
	Run: func(cmd *cobra.Command, args []string) {
		if bitableTableDeleteAppToken == "" || bitableTableDeleteTableID == "" {
			output.Fatalf("VALIDATION_ERROR", "--app-token and --table-id are required")
		}
		appToken := extractBitableToken(bitableTableDeleteAppToken)
		client := api.NewClient()
		if err := client.DeleteTable(appToken, bitableTableDeleteTableID); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":   true,
			"app_token": appToken,
			"table_id":  bitableTableDeleteTableID,
		})
	},
}

// --- bitable field ---

var bitableFieldCmd = &cobra.Command{
	Use:   "field",
	Short: "Manage Bitable table fields",
}

var (
	bitableFieldCreateAppToken string
	bitableFieldCreateTableID  string
	bitableFieldCreateName     string
	bitableFieldCreateType     int
	bitableFieldCreateProperty string
)

var bitableFieldCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a field in a table",
	Long: `Create a new field (column) in a Bitable table.

--type is a numeric Lark field type (1=text, 2=number, 3=single_select, 4=multi_select,
5=date, 7=checkbox, 11=person, 13=phone, 15=url, 17=attachment, etc.).

--property is a JSON object with type-specific options (e.g., '{"options":[{"name":"A"}]}').`,
	Run: func(cmd *cobra.Command, args []string) {
		if bitableFieldCreateAppToken == "" || bitableFieldCreateTableID == "" || bitableFieldCreateName == "" {
			output.Fatalf("VALIDATION_ERROR", "--app-token, --table-id, and --name are required")
		}
		if bitableFieldCreateType == 0 {
			output.Fatalf("VALIDATION_ERROR", "--type is required")
		}
		appToken := extractBitableToken(bitableFieldCreateAppToken)
		req := &api.CreateFieldRequest{
			FieldName: bitableFieldCreateName,
			Type:      bitableFieldCreateType,
		}
		if bitableFieldCreateProperty != "" {
			var prop map[string]interface{}
			if err := json.Unmarshal([]byte(bitableFieldCreateProperty), &prop); err != nil {
				output.Fatalf("VALIDATION_ERROR", "--property must be valid JSON: %v", err)
			}
			req.Property = prop
		}
		client := api.NewClient()
		field, err := client.CreateField(appToken, bitableFieldCreateTableID, req)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(field)
	},
}

var (
	bitableFieldUpdateAppToken string
	bitableFieldUpdateTableID  string
	bitableFieldUpdateFieldID  string
	bitableFieldUpdateName     string
	bitableFieldUpdateType     int
	bitableFieldUpdateProperty string
)

var bitableFieldUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a field in a table",
	Run: func(cmd *cobra.Command, args []string) {
		if bitableFieldUpdateAppToken == "" || bitableFieldUpdateTableID == "" || bitableFieldUpdateFieldID == "" {
			output.Fatalf("VALIDATION_ERROR", "--app-token, --table-id, and --field-id are required")
		}
		appToken := extractBitableToken(bitableFieldUpdateAppToken)
		req := &api.UpdateFieldRequest{
			FieldName: bitableFieldUpdateName,
			Type:      bitableFieldUpdateType,
		}
		if bitableFieldUpdateProperty != "" {
			var prop map[string]interface{}
			if err := json.Unmarshal([]byte(bitableFieldUpdateProperty), &prop); err != nil {
				output.Fatalf("VALIDATION_ERROR", "--property must be valid JSON: %v", err)
			}
			req.Property = prop
		}
		client := api.NewClient()
		if err := client.UpdateField(appToken, bitableFieldUpdateTableID, bitableFieldUpdateFieldID, req); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":   true,
			"app_token": appToken,
			"table_id":  bitableFieldUpdateTableID,
			"field_id":  bitableFieldUpdateFieldID,
		})
	},
}

var (
	bitableFieldDeleteAppToken string
	bitableFieldDeleteTableID  string
	bitableFieldDeleteFieldID  string
)

var bitableFieldDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a field from a table",
	Run: func(cmd *cobra.Command, args []string) {
		if bitableFieldDeleteAppToken == "" || bitableFieldDeleteTableID == "" || bitableFieldDeleteFieldID == "" {
			output.Fatalf("VALIDATION_ERROR", "--app-token, --table-id, and --field-id are required")
		}
		appToken := extractBitableToken(bitableFieldDeleteAppToken)
		client := api.NewClient()
		if err := client.DeleteField(appToken, bitableFieldDeleteTableID, bitableFieldDeleteFieldID); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":   true,
			"app_token": appToken,
			"table_id":  bitableFieldDeleteTableID,
			"field_id":  bitableFieldDeleteFieldID,
		})
	},
}

// --- bitable record ---

var bitableRecordCmd = &cobra.Command{
	Use:   "record",
	Short: "Manage Bitable records",
}

var (
	bitableRecordCreateAppToken string
	bitableRecordCreateTableID  string
	bitableRecordCreateFields   string
)

var bitableRecordCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a record in a table",
	Long: `Create a single record in a Bitable table.

--fields is a JSON object: '{"Field Name":"value","Status":"Active"}'.`,
	Run: func(cmd *cobra.Command, args []string) {
		if bitableRecordCreateAppToken == "" || bitableRecordCreateTableID == "" || bitableRecordCreateFields == "" {
			output.Fatalf("VALIDATION_ERROR", "--app-token, --table-id, and --fields are required")
		}
		var fields map[string]interface{}
		if err := json.Unmarshal([]byte(bitableRecordCreateFields), &fields); err != nil {
			output.Fatalf("VALIDATION_ERROR", "--fields must be valid JSON: %v", err)
		}
		appToken := extractBitableToken(bitableRecordCreateAppToken)
		client := api.NewClient()
		rec, err := client.CreateRecord(appToken, bitableRecordCreateTableID, fields)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(rec)
	},
}

var (
	bitableRecordUpdateAppToken string
	bitableRecordUpdateTableID  string
	bitableRecordUpdateRecordID string
	bitableRecordUpdateFields   string
)

var bitableRecordUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a record in a table",
	Run: func(cmd *cobra.Command, args []string) {
		if bitableRecordUpdateAppToken == "" || bitableRecordUpdateTableID == "" ||
			bitableRecordUpdateRecordID == "" || bitableRecordUpdateFields == "" {
			output.Fatalf("VALIDATION_ERROR", "--app-token, --table-id, --record-id, and --fields are required")
		}
		var fields map[string]interface{}
		if err := json.Unmarshal([]byte(bitableRecordUpdateFields), &fields); err != nil {
			output.Fatalf("VALIDATION_ERROR", "--fields must be valid JSON: %v", err)
		}
		appToken := extractBitableToken(bitableRecordUpdateAppToken)
		client := api.NewClient()
		if err := client.UpdateRecord(appToken, bitableRecordUpdateTableID, bitableRecordUpdateRecordID, fields); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":   true,
			"app_token": appToken,
			"table_id":  bitableRecordUpdateTableID,
			"record_id": bitableRecordUpdateRecordID,
		})
	},
}

var (
	bitableRecordDeleteAppToken string
	bitableRecordDeleteTableID  string
	bitableRecordDeleteRecordID string
)

var bitableRecordDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a record from a table",
	Run: func(cmd *cobra.Command, args []string) {
		if bitableRecordDeleteAppToken == "" || bitableRecordDeleteTableID == "" || bitableRecordDeleteRecordID == "" {
			output.Fatalf("VALIDATION_ERROR", "--app-token, --table-id, and --record-id are required")
		}
		appToken := extractBitableToken(bitableRecordDeleteAppToken)
		client := api.NewClient()
		if err := client.DeleteRecord(appToken, bitableRecordDeleteTableID, bitableRecordDeleteRecordID); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":   true,
			"app_token": appToken,
			"table_id":  bitableRecordDeleteTableID,
			"record_id": bitableRecordDeleteRecordID,
		})
	},
}

var (
	bitableRecordBatchCreateAppToken string
	bitableRecordBatchCreateTableID  string
	bitableRecordBatchCreateRecords  string
)

var bitableRecordBatchCreateCmd = &cobra.Command{
	Use:   "batch-create",
	Short: "Create multiple records in one call",
	Long: `Create multiple records at once.

--records is a JSON array of field maps: '[{"Name":"A"},{"Name":"B"}]'.`,
	Run: func(cmd *cobra.Command, args []string) {
		if bitableRecordBatchCreateAppToken == "" || bitableRecordBatchCreateTableID == "" || bitableRecordBatchCreateRecords == "" {
			output.Fatalf("VALIDATION_ERROR", "--app-token, --table-id, and --records are required")
		}
		var records []map[string]interface{}
		if err := json.Unmarshal([]byte(bitableRecordBatchCreateRecords), &records); err != nil {
			output.Fatalf("VALIDATION_ERROR", "--records must be valid JSON array: %v", err)
		}
		appToken := extractBitableToken(bitableRecordBatchCreateAppToken)
		client := api.NewClient()
		recs, err := client.BatchCreateRecords(appToken, bitableRecordBatchCreateTableID, records)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":   true,
			"app_token": appToken,
			"table_id":  bitableRecordBatchCreateTableID,
			"records":   recs,
			"count":     len(recs),
		})
	},
}

var (
	bitableRecordBatchDeleteAppToken  string
	bitableRecordBatchDeleteTableID   string
	bitableRecordBatchDeleteRecordIDs []string
)

var bitableRecordBatchDeleteCmd = &cobra.Command{
	Use:   "batch-delete",
	Short: "Delete multiple records in one call",
	Run: func(cmd *cobra.Command, args []string) {
		if bitableRecordBatchDeleteAppToken == "" || bitableRecordBatchDeleteTableID == "" || len(bitableRecordBatchDeleteRecordIDs) == 0 {
			output.Fatalf("VALIDATION_ERROR", "--app-token, --table-id, and --record-ids are required")
		}
		appToken := extractBitableToken(bitableRecordBatchDeleteAppToken)
		client := api.NewClient()
		if err := client.BatchDeleteRecords(appToken, bitableRecordBatchDeleteTableID, bitableRecordBatchDeleteRecordIDs); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"success":    true,
			"app_token":  appToken,
			"table_id":   bitableRecordBatchDeleteTableID,
			"record_ids": bitableRecordBatchDeleteRecordIDs,
			"count":      len(bitableRecordBatchDeleteRecordIDs),
		})
	},
}

var (
	bitableRecordSearchAppToken string
	bitableRecordSearchTableID  string
	bitableRecordSearchViewID   string
	bitableRecordSearchFilter   string
	bitableRecordSearchSort     string
	bitableRecordSearchFields   []string
	bitableRecordSearchLimit    int
)

var bitableRecordSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search records using filter/sort expressions",
	Long: `Search records in a table with advanced filter and sort.

--filter is a JSON object (Lark filter object format):
  '{"conjunction":"and","conditions":[{"field_name":"Status","operator":"is","value":["Active"]}]}'

--sort is a JSON array: '[{"field_name":"Created","desc":true}]'.`,
	Run: func(cmd *cobra.Command, args []string) {
		if bitableRecordSearchAppToken == "" || bitableRecordSearchTableID == "" {
			output.Fatalf("VALIDATION_ERROR", "--app-token and --table-id are required")
		}
		appToken := extractBitableToken(bitableRecordSearchAppToken)
		req := &api.SearchRecordsRequest{
			ViewID:     bitableRecordSearchViewID,
			FieldNames: bitableRecordSearchFields,
		}
		if bitableRecordSearchFilter != "" {
			var f map[string]interface{}
			if err := json.Unmarshal([]byte(bitableRecordSearchFilter), &f); err != nil {
				output.Fatalf("VALIDATION_ERROR", "--filter must be valid JSON: %v", err)
			}
			req.Filter = f
		}
		if bitableRecordSearchSort != "" {
			var s []map[string]interface{}
			if err := json.Unmarshal([]byte(bitableRecordSearchSort), &s); err != nil {
				output.Fatalf("VALIDATION_ERROR", "--sort must be valid JSON array: %v", err)
			}
			req.Sort = s
		}

		client := api.NewClient()
		var all []api.BitableRecord
		var pageToken string
		hasMore := true
		remaining := bitableRecordSearchLimit
		for hasMore {
			pageSize := 100
			if remaining > 0 && remaining < pageSize {
				pageSize = remaining
			}
			recs, more, next, err := client.SearchRecords(appToken, bitableRecordSearchTableID, req, pageSize, pageToken)
			if err != nil {
				output.Fatal("API_ERROR", err)
			}
			all = append(all, recs...)
			hasMore = more
			pageToken = next
			if bitableRecordSearchLimit > 0 {
				remaining = bitableRecordSearchLimit - len(all)
				if remaining <= 0 {
					break
				}
			}
		}
		if bitableRecordSearchLimit > 0 && len(all) > bitableRecordSearchLimit {
			all = all[:bitableRecordSearchLimit]
		}
		output.JSON(map[string]interface{}{
			"app_token": appToken,
			"table_id":  bitableRecordSearchTableID,
			"records":   all,
			"count":     len(all),
		})
	},
}

// extractBitableToken returns the app_token from either a raw token or a Lark base URL.
// Lark URLs look like: https://xxx.larksuite.com/base/<app_token>?table=...
var bitableURLRe = regexp.MustCompile(`/base/([^/?#]+)`)

func extractBitableToken(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		if m := bitableURLRe.FindStringSubmatch(s); len(m) > 1 {
			return m[1]
		}
	}
	return s
}

func init() {
	bitableCreateCmd.Flags().StringVar(&bitableCreateName, "name", "", "Bitable name (required)")
	bitableCreateCmd.Flags().StringVar(&bitableCreateFolder, "folder", "", "Parent folder token")

	bitableTableCreateCmd.Flags().StringVar(&bitableTableCreateAppToken, "app-token", "", "Bitable app token or URL (required)")
	bitableTableCreateCmd.Flags().StringVar(&bitableTableCreateName, "name", "", "Table name (required)")
	bitableTableDeleteCmd.Flags().StringVar(&bitableTableDeleteAppToken, "app-token", "", "Bitable app token or URL (required)")
	bitableTableDeleteCmd.Flags().StringVar(&bitableTableDeleteTableID, "table-id", "", "Table ID (required)")

	bitableFieldCreateCmd.Flags().StringVar(&bitableFieldCreateAppToken, "app-token", "", "Bitable app token or URL (required)")
	bitableFieldCreateCmd.Flags().StringVar(&bitableFieldCreateTableID, "table-id", "", "Table ID (required)")
	bitableFieldCreateCmd.Flags().StringVar(&bitableFieldCreateName, "name", "", "Field name (required)")
	bitableFieldCreateCmd.Flags().IntVar(&bitableFieldCreateType, "type", 0, "Lark field type number (required)")
	bitableFieldCreateCmd.Flags().StringVar(&bitableFieldCreateProperty, "property", "", "Field property as JSON object")
	bitableFieldUpdateCmd.Flags().StringVar(&bitableFieldUpdateAppToken, "app-token", "", "Bitable app token or URL (required)")
	bitableFieldUpdateCmd.Flags().StringVar(&bitableFieldUpdateTableID, "table-id", "", "Table ID (required)")
	bitableFieldUpdateCmd.Flags().StringVar(&bitableFieldUpdateFieldID, "field-id", "", "Field ID (required)")
	bitableFieldUpdateCmd.Flags().StringVar(&bitableFieldUpdateName, "name", "", "New field name")
	bitableFieldUpdateCmd.Flags().IntVar(&bitableFieldUpdateType, "type", 0, "New field type number")
	bitableFieldUpdateCmd.Flags().StringVar(&bitableFieldUpdateProperty, "property", "", "New field property as JSON object")
	bitableFieldDeleteCmd.Flags().StringVar(&bitableFieldDeleteAppToken, "app-token", "", "Bitable app token or URL (required)")
	bitableFieldDeleteCmd.Flags().StringVar(&bitableFieldDeleteTableID, "table-id", "", "Table ID (required)")
	bitableFieldDeleteCmd.Flags().StringVar(&bitableFieldDeleteFieldID, "field-id", "", "Field ID (required)")

	bitableRecordCreateCmd.Flags().StringVar(&bitableRecordCreateAppToken, "app-token", "", "Bitable app token or URL (required)")
	bitableRecordCreateCmd.Flags().StringVar(&bitableRecordCreateTableID, "table-id", "", "Table ID (required)")
	bitableRecordCreateCmd.Flags().StringVar(&bitableRecordCreateFields, "fields", "", "Fields as JSON object (required)")
	bitableRecordUpdateCmd.Flags().StringVar(&bitableRecordUpdateAppToken, "app-token", "", "Bitable app token or URL (required)")
	bitableRecordUpdateCmd.Flags().StringVar(&bitableRecordUpdateTableID, "table-id", "", "Table ID (required)")
	bitableRecordUpdateCmd.Flags().StringVar(&bitableRecordUpdateRecordID, "record-id", "", "Record ID (required)")
	bitableRecordUpdateCmd.Flags().StringVar(&bitableRecordUpdateFields, "fields", "", "Fields as JSON object (required)")
	bitableRecordDeleteCmd.Flags().StringVar(&bitableRecordDeleteAppToken, "app-token", "", "Bitable app token or URL (required)")
	bitableRecordDeleteCmd.Flags().StringVar(&bitableRecordDeleteTableID, "table-id", "", "Table ID (required)")
	bitableRecordDeleteCmd.Flags().StringVar(&bitableRecordDeleteRecordID, "record-id", "", "Record ID (required)")
	bitableRecordBatchCreateCmd.Flags().StringVar(&bitableRecordBatchCreateAppToken, "app-token", "", "Bitable app token or URL (required)")
	bitableRecordBatchCreateCmd.Flags().StringVar(&bitableRecordBatchCreateTableID, "table-id", "", "Table ID (required)")
	bitableRecordBatchCreateCmd.Flags().StringVar(&bitableRecordBatchCreateRecords, "records", "", "Records as JSON array of field maps (required)")
	bitableRecordBatchDeleteCmd.Flags().StringVar(&bitableRecordBatchDeleteAppToken, "app-token", "", "Bitable app token or URL (required)")
	bitableRecordBatchDeleteCmd.Flags().StringVar(&bitableRecordBatchDeleteTableID, "table-id", "", "Table ID (required)")
	bitableRecordBatchDeleteCmd.Flags().StringSliceVar(&bitableRecordBatchDeleteRecordIDs, "record-ids", nil, "Record IDs (comma-separated, required)")
	bitableRecordSearchCmd.Flags().StringVar(&bitableRecordSearchAppToken, "app-token", "", "Bitable app token or URL (required)")
	bitableRecordSearchCmd.Flags().StringVar(&bitableRecordSearchTableID, "table-id", "", "Table ID (required)")
	bitableRecordSearchCmd.Flags().StringVar(&bitableRecordSearchViewID, "view", "", "View ID to filter records")
	bitableRecordSearchCmd.Flags().StringVar(&bitableRecordSearchFilter, "filter", "", "Filter as JSON object")
	bitableRecordSearchCmd.Flags().StringVar(&bitableRecordSearchSort, "sort", "", "Sort as JSON array")
	bitableRecordSearchCmd.Flags().StringSliceVar(&bitableRecordSearchFields, "field", nil, "Field names to return (repeatable)")
	bitableRecordSearchCmd.Flags().IntVar(&bitableRecordSearchLimit, "limit", 0, "Maximum number of records (0 = no limit)")

	bitableTableCmd.AddCommand(bitableTableCreateCmd)
	bitableTableCmd.AddCommand(bitableTableDeleteCmd)

	bitableFieldCmd.AddCommand(bitableFieldCreateCmd)
	bitableFieldCmd.AddCommand(bitableFieldUpdateCmd)
	bitableFieldCmd.AddCommand(bitableFieldDeleteCmd)

	bitableRecordCmd.AddCommand(bitableRecordCreateCmd)
	bitableRecordCmd.AddCommand(bitableRecordUpdateCmd)
	bitableRecordCmd.AddCommand(bitableRecordDeleteCmd)
	bitableRecordCmd.AddCommand(bitableRecordBatchCreateCmd)
	bitableRecordCmd.AddCommand(bitableRecordBatchDeleteCmd)
	bitableRecordCmd.AddCommand(bitableRecordSearchCmd)

	bitableCmd.AddCommand(bitableCreateCmd)
	bitableCmd.AddCommand(bitableGetCmd)
	bitableCmd.AddCommand(bitableTableCmd)
	bitableCmd.AddCommand(bitableFieldCmd)
	bitableCmd.AddCommand(bitableRecordCmd)
}
