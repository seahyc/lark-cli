package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

type taskSearcher interface {
	SearchTasks(api.TaskSearchOptions) (*api.TaskSearchPage, error)
}

func collectTaskSearch(client taskSearcher, opts api.TaskSearchOptions, pageLimit int) (*api.TaskSearchPage, error) {
	result := &api.TaskSearchPage{Items: []api.TaskSearchResult{}}
	seenTokens := map[string]bool{opts.PageToken: true}
	for page := 0; page < pageLimit; page++ {
		next, err := client.SearchTasks(opts)
		if err != nil {
			return nil, err
		}
		result.Items = append(result.Items, next.Items...)
		result.HasMore, result.PageToken = next.HasMore, next.PageToken
		if next.Notice != "" {
			result.Notice = next.Notice
		}
		if !next.HasMore {
			return result, nil
		}
		if next.PageToken == "" || seenTokens[next.PageToken] {
			return nil, fmt.Errorf("task search returned a missing or repeated pagination token")
		}
		seenTokens[next.PageToken] = true
		opts.PageToken = next.PageToken
	}
	return result, nil
}

func splitTaskSearchIDs(value string) []string {
	var ids []string
	for _, id := range strings.Split(value, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func newTaskSearchCommand() *cobra.Command {
	var query, token, creator, assignee, follower string
	var pageAll, completed bool
	var pageLimit int
	cmd := &cobra.Command{Use: "search [query]", Aliases: []string{"+search"}, Short: "Search tasks visible to your Lark user", Args: cobra.MaximumNArgs(1), Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			if query != "" {
				output.Fatalf("VALIDATION_ERROR", "use either a positional query or --query")
			}
			query = args[0]
		}
		if pageLimit < 1 || pageLimit > 40 {
			output.Fatalf("VALIDATION_ERROR", "--page-limit must be between 1 and 40")
		}
		if pageAll && !cmd.Flags().Changed("page-limit") {
			pageLimit = 40
		}
		filter := &api.TaskSearchFilter{CreatorIDs: splitTaskSearchIDs(creator), AssigneeIDs: splitTaskSearchIDs(assignee), FollowerIDs: splitTaskSearchIDs(follower)}
		if cmd.Flags().Changed("completed") {
			filter.IsCompleted = &completed
		}
		if strings.TrimSpace(query) == "" && len(filter.CreatorIDs) == 0 && len(filter.AssigneeIDs) == 0 && len(filter.FollowerIDs) == 0 && filter.IsCompleted == nil {
			output.Fatalf("VALIDATION_ERROR", "task search requires a query or filter")
		}
		result, err := collectTaskSearch(api.NewClient(), api.TaskSearchOptions{Query: query, Filter: filter, PageToken: token}, pageLimit)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(result)
	}}
	cmd.Flags().StringVar(&query, "query", "", "Search text, task number, or source URL")
	cmd.Flags().StringVar(&token, "page-token", "", "Resume from a returned pagination token")
	cmd.Flags().BoolVar(&pageAll, "page-all", false, "Retrieve up to 40 pages")
	cmd.Flags().IntVar(&pageLimit, "page-limit", 1, "Maximum pages to retrieve (1-40)")
	cmd.Flags().StringVar(&creator, "creator", "", "Creator open_ids, comma-separated")
	cmd.Flags().StringVar(&assignee, "assignee", "", "Assignee open_ids, comma-separated")
	cmd.Flags().StringVar(&follower, "follower", "", "Follower open_ids, comma-separated")
	cmd.Flags().BoolVar(&completed, "completed", false, "Filter completed (true) or incomplete (false) tasks; omitted matches either")
	return cmd
}

func init() { taskCmd.AddCommand(newTaskSearchCommand()) }
