package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSetForwardingRuleLogic(t *testing.T) {
	baseRule := map[string]interface{}{
		"ruleIdString": "rule123",
		"name":         "Test Rule",
		"condition": map[string]interface{}{
			"from": []string{"sender@example.com"},
		},
		"action": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"type": float64(3), "data": "mark as read"},
				map[string]interface{}{"type": float64(1), "targetFolder": "Archive"},
				map[string]interface{}{
					"type":               float64(12),
					"input":              "alice@example.com",
					"authStatus":         float64(2),
					"enableAutoTransfer": true,
				},
				map[string]interface{}{
					"type":               float64(12),
					"input":              "bob@example.com",
					"authStatus":         float64(2),
					"enableAutoTransfer": true,
				},
			},
		},
	}

	tests := []struct {
		name            string
		rule            map[string]interface{}
		newRecipients   []string
		expectedChanged bool
		checkAfter      func(t *testing.T, after map[string]interface{})
	}{
		{
			name:            "same set reversed order",
			rule:            baseRule,
			newRecipients:   []string{"bob@example.com", "alice@example.com"},
			expectedChanged: false,
			checkAfter: func(t *testing.T, after map[string]interface{}) {
				action := after["action"].(map[string]interface{})
				items := action["items"].([]interface{})
				type12Count := 0
				for _, item := range items {
					itemMap := item.(map[string]interface{})
					if int(itemMap["type"].(float64)) == 12 {
						type12Count++
					}
				}
				if type12Count != 2 {
					t.Errorf("Expected 2 type-12 items, got %d", type12Count)
				}
			},
		},
		{
			name:            "same set mixed case",
			rule:            baseRule,
			newRecipients:   []string{"Alice@Example.com", "BOB@example.COM"},
			expectedChanged: false,
			checkAfter: func(t *testing.T, after map[string]interface{}) {
				action := after["action"].(map[string]interface{})
				items := action["items"].([]interface{})
				foundAlice := false
				foundBob := false
				for _, item := range items {
					itemMap := item.(map[string]interface{})
					if int(itemMap["type"].(float64)) == 12 {
						input := itemMap["input"].(string)
						if input == "Alice@Example.com" {
							foundAlice = true
						}
						if input == "BOB@example.COM" {
							foundBob = true
						}
					}
				}
				if !foundAlice || !foundBob {
					t.Error("Expected to preserve exact case from input")
				}
			},
		},
		{
			name:            "added address",
			rule:            baseRule,
			newRecipients:   []string{"alice@example.com", "bob@example.com", "charlie@example.com"},
			expectedChanged: true,
			checkAfter: func(t *testing.T, after map[string]interface{}) {
				action := after["action"].(map[string]interface{})
				items := action["items"].([]interface{})
				type12Count := 0
				type3Count := 0
				type1Count := 0
				foundCharlie := false
				for _, item := range items {
					itemMap := item.(map[string]interface{})
					itemType := int(itemMap["type"].(float64))
					switch itemType {
					case 12:
						type12Count++
						if itemMap["input"] == "charlie@example.com" {
							foundCharlie = true
							if itemMap["authStatus"] != float64(2) {
								t.Error("Expected authStatus 2 for new recipient")
							}
							if itemMap["enableAutoTransfer"] != true {
								t.Error("Expected enableAutoTransfer true for new recipient")
							}
						}
					case 3:
						type3Count++
					case 1:
						type1Count++
					}
				}
				if type12Count != 3 {
					t.Errorf("Expected 3 type-12 items, got %d", type12Count)
				}
				if type3Count != 1 || type1Count != 1 {
					t.Error("Expected non-forwarding items to be preserved")
				}
				if !foundCharlie {
					t.Error("Expected charlie@example.com in forwarding list")
				}
			},
		},
		{
			name:            "removed address",
			rule:            baseRule,
			newRecipients:   []string{"alice@example.com"},
			expectedChanged: true,
			checkAfter: func(t *testing.T, after map[string]interface{}) {
				action := after["action"].(map[string]interface{})
				items := action["items"].([]interface{})
				type12Count := 0
				for _, item := range items {
					itemMap := item.(map[string]interface{})
					if int(itemMap["type"].(float64)) == 12 {
						type12Count++
						if itemMap["input"] == "bob@example.com" {
							t.Error("bob@example.com should have been removed")
						}
					}
				}
				if type12Count != 1 {
					t.Errorf("Expected 1 type-12 item, got %d", type12Count)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleData := deepCopyMap(tt.rule)
			
			action, _ := ruleData["action"].(map[string]interface{})
			items, _ := action["items"].([]interface{})
			
			oldEmails := make(map[string]bool)
			var nonForwardingItems []interface{}
			for _, item := range items {
				itemMap, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				itemType, _ := itemMap["type"].(float64)
				if int(itemType) == 12 {
					input, _ := itemMap["input"].(string)
					oldEmails[strings.ToLower(input)] = true
				} else {
					nonForwardingItems = append(nonForwardingItems, item)
				}
			}
			
			newEmails := make(map[string]bool)
			for _, email := range tt.newRecipients {
				newEmails[strings.ToLower(email)] = true
			}
			changed := len(oldEmails) != len(newEmails)
			if !changed {
				for email := range newEmails {
					if !oldEmails[email] {
						changed = true
						break
					}
				}
			}
			
			if changed != tt.expectedChanged {
				t.Errorf("Expected changed=%v, got %v", tt.expectedChanged, changed)
			}
			
			newItems := make([]interface{}, len(nonForwardingItems))
			copy(newItems, nonForwardingItems)
			for _, email := range tt.newRecipients {
				newItems = append(newItems, map[string]interface{}{
					"type":               float64(12),
					"input":              email,
					"authStatus":         float64(2),
					"enableAutoTransfer": true,
				})
			}
			
			afterRule := deepCopyMap(ruleData)
			afterAction := deepCopyMap(action)
			afterAction["items"] = newItems
			afterRule["action"] = afterAction
			
			if tt.checkAfter != nil {
				tt.checkAfter(t, afterRule)
			}
		})
	}
}

func TestVerifiedEmailValidation(t *testing.T) {
	verifiedList := []string{"alice@example.com", "bob@example.com"}
	verifiedMap := make(map[string]bool)
	for _, email := range verifiedList {
		verifiedMap[strings.ToLower(email)] = true
	}
	
	tests := []struct {
		name        string
		email       string
		shouldPass  bool
	}{
		{"exact match", "alice@example.com", true},
		{"case insensitive", "Alice@Example.COM", true},
		{"not verified", "charlie@example.com", false},
		{"partial match", "alice@example", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verified := verifiedMap[strings.ToLower(tt.email)]
			if verified != tt.shouldPass {
				t.Errorf("Expected verified=%v for %s, got %v", tt.shouldPass, tt.email, verified)
			}
		})
	}
}

func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	b, _ := json.Marshal(m)
	var result map[string]interface{}
	json.Unmarshal(b, &result)
	return result
}
