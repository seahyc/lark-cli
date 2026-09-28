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
					"extraField":         "preserved",
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
			name:            "reuses existing item with extraField",
			rule:            baseRule,
			newRecipients:   []string{"ALICE@example.com"},
			expectedChanged: true,
			checkAfter: func(t *testing.T, after map[string]interface{}) {
				action := after["action"].(map[string]interface{})
				items := action["items"].([]interface{})
				var aliceItem map[string]interface{}
				for _, item := range items {
					itemMap := item.(map[string]interface{})
					if int(itemMap["type"].(float64)) == 12 && itemMap["input"] == "ALICE@example.com" {
						aliceItem = itemMap
						break
					}
				}
				if aliceItem == nil {
					t.Fatal("Alice item not found")
				}
				if aliceItem["extraField"] != "preserved" {
					t.Error("Expected extraField to be preserved from existing item")
				}
				if aliceItem["authStatus"] != float64(2) {
					t.Error("Expected authStatus to be preserved")
				}
			},
		},
		{
			name:            "added address creates new item",
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
				var charlieItem map[string]interface{}
				for _, item := range items {
					itemMap := item.(map[string]interface{})
					itemType := int(itemMap["type"].(float64))
					switch itemType {
					case 12:
						type12Count++
						if itemMap["input"] == "charlie@example.com" {
							foundCharlie = true
							charlieItem = itemMap
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
				if charlieItem != nil {
					if charlieItem["extraField"] != nil {
						t.Error("New item should not have extraField")
					}
					if charlieItem["authStatus"] != float64(2) {
						t.Error("Expected authStatus 2 for new recipient")
					}
					if charlieItem["enableAutoTransfer"] != true {
						t.Error("Expected enableAutoTransfer true for new recipient")
					}
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
			
			afterRule, changed, err := applyForwardingRecipients(ruleData, tt.newRecipients)
			if err != nil {
				t.Fatalf("applyForwardingRecipients failed: %v", err)
			}
			
			if changed != tt.expectedChanged {
				t.Errorf("Expected changed=%v, got %v", tt.expectedChanged, changed)
			}
			
			if tt.checkAfter != nil {
				tt.checkAfter(t, afterRule)
			}
		})
	}
}

func TestVerifiedEmailValidation(t *testing.T) {
	verifiedList := []string{"alice@example.com", "bob@example.com", " charlie@example.com "}
	verifiedMap := make(map[string]bool)
	for _, email := range verifiedList {
		verifiedMap[strings.ToLower(strings.TrimSpace(email))] = true
	}
	
	tests := []struct {
		name        string
		email       string
		shouldPass  bool
	}{
		{"exact match", "alice@example.com", true},
		{"case insensitive", "Alice@Example.COM", true},
		{"trimmed match", "charlie@example.com", true},
		{"not verified", "david@example.com", false},
		{"partial match", "alice@example", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verified := verifiedMap[strings.ToLower(strings.TrimSpace(tt.email))]
			if verified != tt.shouldPass {
				t.Errorf("Expected verified=%v for %s, got %v", tt.shouldPass, tt.email, verified)
			}
		})
	}
}

func TestDuplicateRecipients(t *testing.T) {
	tests := []struct {
		name       string
		recipients []string
		isDup      bool
	}{
		{"no duplicates", []string{"alice@example.com", "bob@example.com"}, false},
		{"exact duplicate", []string{"alice@example.com", "alice@example.com"}, true},
		{"case-insensitive duplicate", []string{"alice@example.com", "Alice@Example.COM"}, true},
		{"trimmed duplicate", []string{"alice@example.com", " alice@example.com "}, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizedRecipients := make(map[string]string)
			isDuplicate := false
			for _, email := range tt.recipients {
				normalized := strings.ToLower(strings.TrimSpace(email))
				if _, exists := normalizedRecipients[normalized]; exists {
					isDuplicate = true
					break
				}
				normalizedRecipients[normalized] = email
			}
			if isDuplicate != tt.isDup {
				t.Errorf("Expected duplicate=%v, got %v", tt.isDup, isDuplicate)
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
