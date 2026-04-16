package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var schemaCmd = &cobra.Command{
	Use:   "schema <api-path-or-name>",
	Short: "Fetch and print the Lark llms-docs section for an API endpoint",
	Long: `Fetch the Lark open-platform llms-docs reference for a given API path
or dotted name, and print the matching section(s).

Examples:
  lark schema /open-apis/im/v1/messages
  lark schema im/v1/messages
  lark schema im.messages.send
  lark schema approval/v4/instances/query

Use "lark schema modules" to list the supported llms-docs modules.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pathOrName := args[0]
		module, url, section, err := api.FetchSchemaSection(pathOrName)
		if err != nil {
			output.Fatal("SCHEMA_ERROR", err)
		}
		output.JSON(map[string]interface{}{
			"module":  module,
			"source":  url,
			"query":   pathOrName,
			"section": section,
		})
	},
}

var schemaModulesCmd = &cobra.Command{
	Use:   "modules",
	Short: "List the supported llms-docs modules",
	Run: func(cmd *cobra.Command, args []string) {
		mods := api.ListSchemaModules()
		rows := make([]map[string]interface{}, 0, len(mods))
		for k, v := range mods {
			rows = append(rows, map[string]interface{}{
				"module": k,
				"url":    v,
			})
		}
		output.JSON(rows)
	},
}

func init() {
	schemaCmd.AddCommand(schemaModulesCmd)
	// Ensure `lark schema` with no args prints help instead of the error below.
	schemaCmd.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires an API path or dotted name argument (or use 'schema modules')")
		}
		return nil
	}
}
