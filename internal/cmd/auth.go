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
	loginManual bool
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

		var groups []string
		if loginScopes != "" {
			// Parse and validate scope groups
			parsed, invalid := scopes.ParseGroups(loginScopes)
			if len(invalid) > 0 {
				output.Fatal("VALIDATION_ERROR", fmt.Errorf("invalid scope groups: %s\nValid groups: %s",
					strings.Join(invalid, ", "),
					strings.Join(scopes.AllGroupNames(), ", ")))
			}
			if len(parsed) == 0 {
				output.Fatal("VALIDATION_ERROR", fmt.Errorf("no valid scope groups specified\nValid groups: %s",
					strings.Join(scopes.AllGroupNames(), ", ")))
			}
			groups = parsed
		}

		// --add means incremental authorization: request the union of the
		// currently-granted groups and the new ones, so we never silently drop
		// existing permissions. (Lark's own incremental auth is unreliable — it
		// has been observed dropping groups not named in the request — so we
		// re-request everything explicitly.)
		if loginAdd {
			existing := auth.GetTokenStore().GetGrantedGroupsList()
			groups = unionGroups(existing, groups)
			if len(groups) == 0 {
				output.Fatal("VALIDATION_ERROR", fmt.Errorf(
					"--add has nothing to add: no existing permissions and no --scopes given"))
			}
		}

		// Empty groups (no --scopes, no --add) leaves ScopeGroups nil, which
		// triggers the default (all scopes) in LoginWithOptions.
		opts.ScopeGroups = groups

		// --manual: headless paste-URL flow (no local browser / callback server).
		if loginManual {
			if err := auth.LoginManual(opts); err != nil {
				output.Fatal("AUTH_ERROR", err)
			}
			output.Success("Successfully authenticated with Lark")
			return
		}

		if err := auth.LoginWithOptions(opts); err != nil {
			output.Fatal("AUTH_ERROR", err)
		}
		output.Success("Successfully authenticated with Lark")
	},
}

// unionGroups returns the de-duplicated union of two scope-group name slices,
// preserving order (existing groups first, then any new ones).
func unionGroups(existing, added []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(existing)+len(added))
	for _, g := range append(append([]string{}, existing...), added...) {
		if g == "" || seen[g] {
			continue
		}
		seen[g] = true
		out = append(out, g)
	}
	return out
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

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh the access token using the stored refresh token",
	Long: `Refresh the OAuth access token without any human interaction.

Intended for a keep-alive timer: Lark refresh tokens themselves expire (~30
days), so periodically refreshing keeps the session alive indefinitely and
avoids a full re-login. Exits non-zero if no valid refresh token is available
(i.e. a real re-login is required).`,
	Run: func(cmd *cobra.Command, args []string) {
		store := auth.GetTokenStore()
		if !store.CanRefresh() {
			output.Fatal("AUTH_ERROR", fmt.Errorf(
				"no valid refresh token available; run 'lark auth login --manual'"))
		}
		if err := auth.RefreshAccessToken(); err != nil {
			output.Fatal("AUTH_ERROR", err)
		}
		output.JSON(api.OutputAuthStatus{
			Authenticated: store.IsValid(),
			ExpiresAt:     store.GetExpiresAt(),
			GrantedGroups: store.GetGrantedGroupsList(),
			ScopeGroups:   store.GetGrantedGroups(),
		})
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
	loginCmd.Flags().StringVar(&loginScopes, "scopes", "", "Comma-separated scope groups (calendar,contacts,documents,messages,mail,mailrules,minutes)")
	loginCmd.Flags().BoolVar(&loginAdd, "add", false, "Add to existing permissions (incremental authorization)")
	loginCmd.Flags().BoolVar(&loginManual, "manual", false, "Headless login: print the consent URL and paste the redirect back (no local browser needed)")
	loginCmd.Flags().Lookup("manual").NoOptDefVal = "true"

	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(refreshCmd)
	authCmd.AddCommand(scopesCmd)
}
