package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var (
	findTypes []string
	findLimit int
)

var supportedFindTypes = map[string]bool{
	"messages": true,
	"docs":     true,
	"people":   true,
	"chats":    true,
}

func normalizeFindTypes(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{"messages", "docs", "people", "chats"}, nil
	}
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			kind := strings.ToLower(strings.TrimSpace(part))
			if !supportedFindTypes[kind] {
				return nil, fmt.Errorf("unsupported type %q; want messages, docs, people, or chats", kind)
			}
			if !seen[kind] {
				seen[kind] = true
				result = append(result, kind)
			}
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("at least one search type is required")
	}
	return result, nil
}

func capFindResults[T any](items []T, limit int) []T {
	if limit > 0 && len(items) > limit {
		return items[:limit]
	}
	return items
}

var findCmd = &cobra.Command{
	Use:   "find <query>",
	Short: "Search messages, documents, people, and chats",
	Long: `Search the main Lark information surfaces with the public APIs.

This command does not require Lark Desktop or the experimental desktop bridge.
Each result section reports its own error so a missing scope for one surface does
not hide successful results from the others.

Examples:
  lark find "quarterly planning"
  lark find "Oswald" --type people --type messages
  lark find "incident" --type messages,docs --limit 10`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.TrimSpace(args[0])
		if query == "" {
			output.Fatalf("VALIDATION_ERROR", "query must not be empty")
		}
		if findLimit < 1 || findLimit > 50 {
			output.Fatalf("VALIDATION_ERROR", "--limit must be between 1 and 50")
		}
		types, err := normalizeFindTypes(findTypes)
		if err != nil {
			output.Fatal("VALIDATION_ERROR", err)
		}

		client := api.NewClient()
		sections := map[string]interface{}{}
		errorsByType := map[string]string{}
		for _, kind := range types {
			switch kind {
			case "messages":
				items, more, next, searchErr := client.SearchMessages(query, &api.SearchMessagesOptions{PageSize: findLimit})
				if searchErr != nil {
					errorsByType[kind] = searchErr.Error()
					continue
				}
				sections[kind] = map[string]interface{}{"items": capFindResults(items, findLimit), "count": len(capFindResults(items, findLimit)), "has_more": more, "page_token": next}
			case "docs":
				items, total, searchErr := client.SearchDocuments(query, nil, nil, nil)
				if searchErr != nil {
					errorsByType[kind] = searchErr.Error()
					continue
				}
				items = capFindResults(items, findLimit)
				sections[kind] = map[string]interface{}{"items": items, "count": len(items), "total": total, "truncated": total > len(items)}
			case "people":
				items, more, next, searchErr := client.SearchUsers(query, findLimit, "")
				if searchErr != nil {
					errorsByType[kind] = searchErr.Error()
					continue
				}
				sections[kind] = map[string]interface{}{"items": capFindResults(items, findLimit), "count": len(capFindResults(items, findLimit)), "has_more": more, "page_token": next}
			case "chats":
				items, more, next, searchErr := client.SearchChats(&api.SearchChatsOptions{Query: query, PageSize: findLimit})
				if searchErr != nil {
					errorsByType[kind] = searchErr.Error()
					continue
				}
				sections[kind] = map[string]interface{}{"items": capFindResults(items, findLimit), "count": len(capFindResults(items, findLimit)), "has_more": more, "page_token": next}
			}
		}

		sort.Strings(types)
		output.JSON(map[string]interface{}{
			"query":            query,
			"types":            types,
			"results":          sections,
			"errors":           errorsByType,
			"successful_types": len(sections),
			"desktop_required": false,
		})
	},
}

func init() {
	findCmd.Flags().StringSliceVar(&findTypes, "type", nil, "Search type: messages, docs, people, or chats (repeatable or comma-separated)")
	findCmd.Flags().IntVar(&findLimit, "limit", 20, "Maximum results per type (1-50)")
}
