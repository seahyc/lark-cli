package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	var name string
	var enabled bool
	var stop bool
	var apply bool
	var expectUser string

	patch := &cobra.Command{
		Use:   "rule-patch RULE_ID",
		Short: "Preview or apply selected native mail-rule fields while preserving the full current rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var current struct {
				UserID string            `json:"userId"`
				Rules  []json.RawMessage `json:"rules"`
			}
			if err := desktopCall(cmd, "listRules", nil, &current); err != nil {
				return err
			}

			var expectedRule json.RawMessage
			for _, rule := range current.Rules {
				var identity struct {
					ID string `json:"ruleIdString"`
				}
				if err := json.Unmarshal(rule, &identity); err != nil {
					return err
				}
				if identity.ID == args[0] {
					expectedRule = rule
					break
				}
			}
			if expectedRule == nil {
				return fmt.Errorf("rule %s not found", args[0])
			}

			changes := map[string]interface{}{}
			if cmd.Flags().Changed("name") {
				trimmed := strings.TrimSpace(name)
				if trimmed == "" || len(trimmed) > 256 {
					return fmt.Errorf("--name must be non-empty after trimming and at most 256 characters")
				}
				changes["name"] = trimmed
			}
			if cmd.Flags().Changed("enabled") {
				changes["isEnable"] = enabled
			}
			if cmd.Flags().Changed("stop") {
				changes["ignoreTheRestOfRules"] = stop
			}
			if len(changes) == 0 {
				return fmt.Errorf("provide at least one of --name, --enabled, or --stop")
			}

			payload := map[string]interface{}{
				"expectedUserId": current.UserID,
				"expectedRule":   expectedRule,
				"patch":          changes,
			}
			if !apply {
				return desktopJSON(cmd, map[string]interface{}{
					"apply":   false,
					"account": current.UserID,
					"before":  expectedRule,
					"changes": changes,
					"next":    "Review then repeat with --apply --expect-user matching account",
				})
			}
			if expectUser == "" || expectUser != current.UserID {
				return fmt.Errorf("--expect-user must match the preview account")
			}

			var out json.RawMessage
			if err := desktopCall(cmd, "updateRuleFields", payload, &out); err != nil {
				return err
			}
			return desktopJSON(cmd, out)
		},
	}
	patch.Flags().StringVar(&name, "name", "", "New rule name (trimmed, at most 256 characters)")
	patch.Flags().BoolVar(&enabled, "enabled", false, "Set whether the rule is enabled")
	patch.Flags().BoolVar(&stop, "stop", false, "Set whether matching stops later rules")
	patch.Flags().BoolVar(&apply, "apply", false, "Apply the reviewed field changes")
	patch.Flags().StringVar(&expectUser, "expect-user", "", "Signed-in user ID from preview")
	desktopCmd.AddCommand(patch)
}
