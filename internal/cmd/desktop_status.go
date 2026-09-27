package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/desktop"
)

func init() {
	var probe bool
	command := &cobra.Command{Use: "status", Short: "Inspect the temporary desktop session without exposing its token", RunE: func(cmd *cobra.Command, args []string) error {
		reads, err := desktop.ReadOperations()
		if err != nil {
			return err
		}
		writes, err := desktop.MutationOperations()
		if err != nil {
			return err
		}
		result := map[string]interface{}{"read_operations": len(reads), "mutation_operations": len(writes), "session_present": false}
		session, err := desktop.ReadSession()
		if os.IsNotExist(err) {
			if probe {
				return fmt.Errorf("no desktop session; use desktop start")
			}
			return desktopJSON(cmd, result)
		}
		if err != nil {
			return err
		}
		result["session_present"] = true
		result["expired"] = time.Now().UnixMilli() >= session.ExpiresAt
		result["expires_at"] = time.UnixMilli(session.ExpiresAt).Format(time.RFC3339)
		result["loopback_port"] = session.Port
		if probe {
			var identity struct {
				UserID string `json:"userId"`
			}
			if err = desktopCall(cmd, "identity", nil, &identity); err != nil {
				return err
			}
			result["bridge_reachable"] = true
			result["user_id"] = identity.UserID
		}
		return desktopJSON(cmd, result)
	}}
	command.Flags().BoolVar(&probe, "probe", false, "Check identity through the native bridge (up to 30 seconds)")
	desktopCmd.AddCommand(command)
}
