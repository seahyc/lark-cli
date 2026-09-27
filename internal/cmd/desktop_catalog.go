package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/desktop"
	"strings"
)

func init() {
	var domain, search string
	var details bool
	c := &cobra.Command{Use: "catalog", Short: "Inspect shipped native command declarations (not executable or verified capabilities)", RunE: func(cmd *cobra.Command, args []string) error {
		ops, err := desktop.NativeCatalog(domain, search)
		if err != nil {
			return err
		}
		counts := map[string]int{}
		names := []string{}
		for _, op := range ops {
			counts[strings.Split(op.Request, ".")[0]]++
			names = append(names, op.Name)
		}
		result := map[string]interface{}{"count": len(ops), "domains": counts, "status": "declaration-only", "note": "Presence does not prove request schema, permissions, CLI gap, or successful execution. This catalog is not an arbitrary native API executor."}
		if details {
			result["operations"] = ops
		} else if domain != "" || search != "" {
			result["operations"] = names
		}
		return desktopJSON(cmd, result)
	}}
	c.Flags().StringVar(&domain, "domain", "", "Native namespace, e.g. im, email, feed, calendar")
	c.Flags().StringVar(&search, "search", "", "Filter declaration names")
	c.Flags().BoolVar(&details, "details", false, "Include wire command and static source locations")
	desktopCmd.AddCommand(c)
}
