package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/auth"
	"github.com/yjwong/lark-cli/internal/output"
	"github.com/yjwong/lark-cli/internal/scopes"
)

var (
	loginScopes string
	loginAdd    bool
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  "Manage Lark OAuth authentication",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Lark",
	Long: `Authenticate with Lark using OAuth browser flow.

By default, all permissions are requested. Use --scopes to request only specific
scope groups for a minimal permission setup.

Scope groups: calendar, contacts, documents, messages, mail, minutes

Examples:
  lark auth login                           # All permissions (default)
  lark auth login --scopes calendar         # Only calendar permissions
  lark auth login --scopes calendar,contacts # Calendar and contacts
  lark auth login --add --scopes messages   # Add messaging to existing permissions`,
	Run: func(cmd *cobra.Command, args []string) {
		opts := auth.LoginOptions{}

		if loginScopes != "" {
			// Parse and validate scope groups
			groups, invalid := scopes.ParseGroups(loginScopes)
			if len(invalid) > 0 {
				output.Fatal("VALIDATION_ERROR", fmt.Errorf("invalid scope groups: %s\nValid groups: %s",
					strings.Join(invalid, ", "),
					strings.Join(scopes.AllGroupNames(), ", ")))
			}
			if len(groups) == 0 {
				output.Fatal("VALIDATION_ERROR", fmt.Errorf("no valid scope groups specified\nValid groups: %s",
					strings.Join(scopes.AllGroupNames(), ", ")))
			}
			opts.ScopeGroups = groups
		}
		// If loginScopes is empty, opts.ScopeGroups remains nil, triggering default (all scopes)

		if err := auth.LoginWithOptions(opts); err != nil {
			output.Fatal("AUTH_ERROR", err)
		}
		output.Success("Successfully authenticated with Lark")
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from Lark",
	Long:  "Clear stored authentication credentials",
	Run: func(cmd *cobra.Command, args []string) {
		if err := auth.Logout(); err != nil {
			output.Fatal("AUTH_ERROR", err)
		}
		output.Success("Successfully logged out")
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  "Display current authentication status, token expiry, and granted permissions",
	Run: func(cmd *cobra.Command, args []string) {
		store := auth.GetTokenStore()

		status := api.OutputAuthStatus{
			Authenticated: store.IsValid(),
			ExpiresAt:     store.GetExpiresAt(),
		}

		if !status.Authenticated && store.CanRefresh() {
			// Token expired but we can refresh
			if err := auth.RefreshAccessToken(); err == nil {
				status.Authenticated = true
				status.ExpiresAt = store.GetExpiresAt()
			}
		}

		// Add scope information
		if status.Authenticated {
			status.GrantedGroups = store.GetGrantedGroupsList()
			status.ScopeGroups = store.GetGrantedGroups()
		}

		output.JSON(status)
	},
}

var scopesCmd = &cobra.Command{
	Use:   "scopes",
	Short: "List available scope groups",
	Long:  "Display all available scope groups and their permissions",
	Run: func(cmd *cobra.Command, args []string) {
		type scopeGroupOutput struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Commands    []string `json:"commands"`
			Scopes      []string `json:"scopes"`
		}

		groups := make([]scopeGroupOutput, 0, len(scopes.AllGroupNames()))
		for _, name := range scopes.AllGroupNames() {
			group := scopes.Groups[name]
			groups = append(groups, scopeGroupOutput{
				Name:        group.Name,
				Description: group.Description,
				Commands:    group.Commands,
				Scopes:      group.Scopes,
			})
		}

		output.JSON(map[string]interface{}{
			"groups": groups,
			"usage":  "lark auth login --scopes <group1,group2,...>",
		})
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginScopes, "scopes", "", "Comma-separated scope groups (calendar,contacts,documents,messages,mail,minutes)")
	loginCmd.Flags().BoolVar(&loginAdd, "add", false, "Add to existing permissions (incremental authorization)")

	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(scopesCmd)
}
