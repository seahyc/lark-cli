package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var (
	apiAs     string
	apiParams string
	apiData   string
	apiForms  []string
	apiOutput string
)

var apiCmd = &cobra.Command{
	Use:   "api <METHOD> <path>",
	Short: "Raw API passthrough for any Lark endpoint",
	Long: `Execute raw HTTP requests against any Lark Open API endpoint.

Examples:
  # GET with query params
  lark api GET /open-apis/im/v1/chats --params '{"page_size":20}' --as user

  # POST with JSON body
  lark api POST /open-apis/task/v2/tasks --data '{"summary":"Ship it"}' --as user

  # File upload (multipart)
  lark api POST /open-apis/im/v1/files --form file=@./report.pdf --form file_type=stream

  # File download
  lark api GET /open-apis/im/v1/messages/om_xxx/resources/file_xxx --output ./download.pdf

  # DELETE
  lark api DELETE /open-apis/im/v1/messages/om_xxx --as bot`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Handle "lark api list" subcommand
		if args[0] == "list" {
			runAPIList(args[1:])
			return
		}

		if len(args) < 2 {
			output.Fatalf("VALIDATION_ERROR", "usage: lark api <METHOD> <path>")
		}

		method := strings.ToUpper(args[0])
		path := args[1]

		if apiAs != "" && apiAs != "bot" && apiAs != "user" {
			output.Fatalf("VALIDATION_ERROR", "--as must be 'bot' or 'user'")
		}
		asUser := apiAs == "user"

		// Parse --params
		var params map[string]interface{}
		if apiParams != "" {
			if err := json.Unmarshal([]byte(apiParams), &params); err != nil {
				output.Fatalf("VALIDATION_ERROR", "invalid --params JSON: %v", err)
			}
		}

		// Parse --data
		var data map[string]interface{}
		if apiData != "" {
			if err := json.Unmarshal([]byte(apiData), &data); err != nil {
				output.Fatalf("VALIDATION_ERROR", "invalid --data JSON: %v", err)
			}
		}

		// Parse --form fields
		var formFields []api.FormField
		for _, f := range apiForms {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) != 2 {
				output.Fatalf("VALIDATION_ERROR", "invalid --form format: %s (use key=value or key=@./path)", f)
			}
			field := api.FormField{Key: parts[0]}
			if strings.HasPrefix(parts[1], "@") {
				field.Value = strings.TrimPrefix(parts[1], "@")
				field.IsFile = true
			} else {
				field.Value = parts[1]
			}
			formFields = append(formFields, field)
		}

		client := api.NewClient()
		resp, err := client.DoRawRequest(method, path, params, data, formFields, asUser, apiOutput)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}

		// Print response
		if apiOutput != "" {
			// File download — response body is already the success JSON
			fmt.Println(string(resp.Body))
		} else {
			// Try to pretty-print JSON, fall back to raw
			var prettyJSON map[string]interface{}
			if err := json.Unmarshal(resp.Body, &prettyJSON); err == nil {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(prettyJSON)
			} else {
				fmt.Println(string(resp.Body))
			}
		}
	},
}

func runAPIList(args []string) {
	client := api.NewClient()

	if len(args) == 0 {
		// List all API domains
		body, err := client.FetchAPIIndex()
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		fmt.Println(string(body))
	} else {
		// List endpoints for a specific domain
		body, err := client.FetchAPIModuleDocs(args[0])
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		fmt.Println(string(body))
	}
}

func init() {
	apiCmd.Flags().StringVar(&apiAs, "as", "bot", "Identity: 'bot' (default) or 'user'")
	apiCmd.Flags().StringVar(&apiParams, "params", "", "JSON query parameters")
	apiCmd.Flags().StringVar(&apiData, "data", "", "JSON request body")
	apiCmd.Flags().StringSliceVar(&apiForms, "form", nil, "Multipart form field (repeatable). key=value or key=@./path for files")
	apiCmd.Flags().StringVar(&apiOutput, "output", "", "Save response body to file (for binary downloads)")
}
