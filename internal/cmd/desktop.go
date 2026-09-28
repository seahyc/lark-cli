package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/desktop"
)

func applyForwardingRecipients(ruleData map[string]interface{}, recipients []string) (map[string]interface{}, bool, error) {
	action, _ := ruleData["action"].(map[string]interface{})
	items, _ := action["items"].([]interface{})
	
	normalize := func(email string) string {
		return strings.ToLower(strings.TrimSpace(email))
	}
	
	oldNormalized := make(map[string]bool)
	existingItems := make(map[string]map[string]interface{})
	var nonForwardingItems []interface{}
	
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		itemType, _ := itemMap["type"].(float64)
		if int(itemType) == 12 {
			input, _ := itemMap["input"].(string)
			normalized := normalize(input)
			oldNormalized[normalized] = true
			existingItems[normalized] = itemMap
		} else {
			nonForwardingItems = append(nonForwardingItems, item)
		}
	}
	
	newNormalized := make(map[string]bool)
	for _, email := range recipients {
		newNormalized[normalize(email)] = true
	}
	
	changed := len(oldNormalized) != len(newNormalized)
	if !changed {
		for email := range newNormalized {
			if !oldNormalized[email] {
				changed = true
				break
			}
		}
	}
	
	newItems := make([]interface{}, len(nonForwardingItems))
	copy(newItems, nonForwardingItems)
	for _, email := range recipients {
		normalized := normalize(email)
		if existing, ok := existingItems[normalized]; ok {
			reused := make(map[string]interface{})
			for k, v := range existing {
				reused[k] = v
			}
			reused["input"] = email
			newItems = append(newItems, reused)
		} else {
			newItems = append(newItems, map[string]interface{}{
				"type":               float64(12),
				"input":              email,
				"authStatus":         float64(2),
				"enableAutoTransfer": true,
			})
		}
	}
	
	afterRule := make(map[string]interface{})
	for k, v := range ruleData {
		afterRule[k] = v
	}
	afterAction := make(map[string]interface{})
	for k, v := range action {
		afterAction[k] = v
	}
	afterAction["items"] = newItems
	afterRule["action"] = afterAction
	
	return afterRule, changed, nil
}


func desktopJSON(cmd *cobra.Command, v interface{}) error {
	e := json.NewEncoder(cmd.OutOrStdout())
	e.SetIndent("", "  ")
	return e.Encode(v)
}
func desktopCall(cmd *cobra.Command, op string, p interface{}, out interface{}) error {
	s, e := desktop.ReadSession()
	if e != nil {
		return fmt.Errorf("start a desktop session first: %w", e)
	}
	if time.Now().UnixMilli() >= s.ExpiresAt {
		return fmt.Errorf("desktop session expired; restore and start again")
	}
	err := desktop.BridgeCall(cmd.Context(), s.Port, s.Token, op, p, out)
	if err != nil && (op == "updateRule" || op == "updateRuleFields") {
		return fmt.Errorf("%w; write outcome may be uncertain: read current state before retrying", err)
	}
	return err
}

var desktopCmd = &cobra.Command{Use: "desktop", Short: "Experimental native Lark bridge and capability inventory"}

func init() {
	d := desktopCmd
	var port int
	start := &cobra.Command{Use: "start", Short: "Back up and temporarily patch Lark's English mail UI for a 30-minute local session", RunE: func(cmd *cobra.Command, args []string) error {
		s, e := desktop.InstallSession(port)
		if e != nil {
			return e
		}
		return desktopJSON(cmd, map[string]interface{}{"installed": true, "expires_at": time.UnixMilli(s.ExpiresAt).Format(time.RFC3339), "next": "Restart Lark, open Email, then use desktop list. Run desktop restore and restart Lark when finished."})
	}}
	start.Flags().IntVar(&port, "port", 9330, "Loopback bridge port")
	restore := &cobra.Command{Use: "restore", Short: "Restore the original mail archive; restart Lark to unload the bridge", RunE: func(cmd *cobra.Command, args []string) error {
		if e := desktop.RestoreSession(); e != nil {
			return e
		}
		return desktopJSON(cmd, map[string]interface{}{"restored": true, "next": "Restart Lark to unload the in-memory bridge."})
	}}
	for _, pair := range [][2]string{{"list", "listRules"}, {"identity", "identity"}, {"verified-emails", "verifiedEmails"}} {
		name, op := pair[0], pair[1]
		d.AddCommand(&cobra.Command{Use: name, Short: "Read native mail " + name, RunE: func(cmd *cobra.Command, args []string) error {
			var out json.RawMessage
			if e := desktopCall(cmd, op, map[string]interface{}{}, &out); e != nil {
				return e
			}
			return desktopJSON(cmd, out)
		}})
	}
	var recipients []string
	var apply bool
	var expectUser string
	update := &cobra.Command{Use: "set-forwarding RULE_ID", Short: "Preview or apply verified forwarding recipients while preserving the rest of a rule", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		var current struct {
			UserID string            `json:"userId"`
			Rules  []json.RawMessage `json:"rules"`
		}
		if e := desktopCall(cmd, "listRules", nil, &current); e != nil {
			return e
		}
		var verified struct {
			Emails []string `json:"emails"`
		}
		if e := desktopCall(cmd, "verifiedEmails", nil, &verified); e != nil {
			return e
		}
		verifiedMap := make(map[string]bool)
		for _, email := range verified.Emails {
			verifiedMap[strings.ToLower(strings.TrimSpace(email))] = true
		}
		var rule json.RawMessage
		for _, r := range current.Rules {
			var id struct {
				ID string `json:"ruleIdString"`
			}
			if e := json.Unmarshal(r, &id); e != nil {
				return e
			}
			if id.ID == args[0] {
				rule = r
				break
			}
		}
		if rule == nil {
			return fmt.Errorf("rule %s not found", args[0])
		}
		if len(recipients) == 0 {
			return fmt.Errorf("provide at least one --to recipient")
		}
		
		normalizedRecipients := make(map[string]string)
		for _, email := range recipients {
			normalized := strings.ToLower(strings.TrimSpace(email))
			if _, exists := normalizedRecipients[normalized]; exists {
				return fmt.Errorf("duplicate recipient: %s (case-insensitive)", email)
			}
			normalizedRecipients[normalized] = email
		}
		
		for normalized := range normalizedRecipients {
			if !verifiedMap[normalized] {
				return fmt.Errorf("recipient %s is not verified; add it via Lark settings first", normalizedRecipients[normalized])
			}
		}
		var ruleData map[string]interface{}
		if e := json.Unmarshal(rule, &ruleData); e != nil {
			return e
		}
		
		afterRule, changed, err := applyForwardingRecipients(ruleData, recipients)
		if err != nil {
			return err
		}
		resultingRule, _ := json.Marshal(afterRule)
		
		p := map[string]interface{}{"expectedUserId": current.UserID, "expectedRule": rule, "recipients": recipients}
		if !apply {
			return desktopJSON(cmd, map[string]interface{}{
				"apply":          false,
				"changed":        changed,
				"userId":         current.UserID,
				"before":         rule,
				"after":          json.RawMessage(resultingRule),
				"forward_to":     recipients,
				"next":           "Review then repeat with --apply --expect-user matching userId",
			})
		}
		if expectUser == "" || expectUser != current.UserID {
			return fmt.Errorf("--expect-user must match the preview userId")
		}
		var out json.RawMessage
		if e := desktopCall(cmd, "updateRule", p, &out); e != nil {
			return e
		}
		return desktopJSON(cmd, out)
	}}
	update.Flags().StringSliceVar(&recipients, "to", nil, "Complete desired forwarding recipient list (must already be verified)")
	update.Flags().BoolVar(&apply, "apply", false, "Apply the reviewed recipient change")
	update.Flags().StringVar(&expectUser, "expect-user", "", "Signed-in userId from preview")
	d.AddCommand(start, restore, update)
	rootCmd.AddCommand(d)
}
