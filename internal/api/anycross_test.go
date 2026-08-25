package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestAnyCrossGetWorkflowRedactsCredentials(t *testing.T) {
	definition := map[string]interface{}{
		"steps": []interface{}{map[string]interface{}{
			"name": "Query",
			"auth": map[string]interface{}{
				"key":         "mysql-prod",
				"credentials": map[string]interface{}{"password": "do-not-leak"},
			},
		}},
	}
	encodedDefinition, _ := json.Marshal(definition)
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "session=test" {
			t.Fatalf("missing session cookie")
		}
		var payload bytes.Buffer
		_ = json.NewEncoder(&payload).Encode(map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"id": "flow-1", "name": "Test", "draft": map[string]string{"definition": string(encodedDefinition)},
			},
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(payload.Bytes())),
			Request:    r,
		}, nil
	})}

	client := newAnyCrossClientForTest("https://anycross.test", "session=test", "csrf", httpClient)
	workflow, err := client.GetWorkflow("flow-1")
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(workflow)
	if string(payload) == "" || contains(string(payload), "do-not-leak") {
		t.Fatalf("secret leaked in workflow output: %s", payload)
	}
	if !contains(string(payload), "[REDACTED]") {
		t.Fatalf("expected redaction marker: %s", payload)
	}
}

func TestAnyCrossPostRequiresCSRFToken(t *testing.T) {
	client := newAnyCrossClientForTest("https://example.invalid", "session=test", "", http.DefaultClient)
	err := client.do(http.MethodPost, "/webapi/read", map[string]string{"read": "only"}, nil)
	if err == nil || !contains(err.Error(), "ANYCROSS_CSRF_TOKEN") {
		t.Fatalf("expected CSRF configuration error, got %v", err)
	}
}

func TestFlattenWorkflowsTracksFolders(t *testing.T) {
	nodes := []AnyCrossResourceNode{{
		Name: "Reports", Type: 3,
		ChildResourceNodes: []AnyCrossResourceNode{{
			Name: "Daily", Type: 2, ResourceID: "flow-1",
		}},
	}}
	var got []AnyCrossWorkflowSummary
	flattenWorkflows(nodes, "", &got)
	if len(got) != 1 || got[0].ID != "flow-1" || got[0].Folder != "Reports" {
		t.Fatalf("unexpected workflows: %#v", got)
	}
}

func TestApplyAnyCrossJSONPatchReplacesStepField(t *testing.T) {
	document := map[string]interface{}{
		"steps":     []interface{}{map[string]interface{}{"name": "Before"}},
		"structure": []interface{}{},
	}
	patched, err := applyAnyCrossJSONPatch(document, []AnyCrossJSONPatchOperation{{
		Op: "replace", Path: "/steps/0/name", Value: "After",
	}})
	if err != nil {
		t.Fatal(err)
	}
	steps := patched.(map[string]interface{})["steps"].([]interface{})
	if got := steps[0].(map[string]interface{})["name"]; got != "After" {
		t.Fatalf("expected patched name, got %#v", got)
	}
	if got := document["steps"].([]interface{})[0].(map[string]interface{})["name"]; got != "Before" {
		t.Fatalf("input document was mutated: %#v", got)
	}
}

func TestApplyAnyCrossJSONPatchRejectsSensitivePath(t *testing.T) {
	document := map[string]interface{}{
		"steps": []interface{}{map[string]interface{}{
			"auth": map[string]interface{}{"credentials": map[string]interface{}{"password": "existing"}},
		}},
		"structure": []interface{}{},
	}
	_, err := applyAnyCrossJSONPatch(document, []AnyCrossJSONPatchOperation{{
		Op: "replace", Path: "/steps/0/auth/credentials/password", Value: "new-secret",
	}})
	if err == nil || !contains(err.Error(), "sensitive path") {
		t.Fatalf("expected sensitive-path rejection, got %v", err)
	}
}

func TestPatchWorkflowDraftDryRunRedactsSecretsAndDoesNotWrite(t *testing.T) {
	definition := map[string]interface{}{
		"steps": []interface{}{map[string]interface{}{
			"name": "Before", "auth": map[string]interface{}{"credentials": map[string]interface{}{"password": "do-not-leak"}},
		}},
		"structure": []interface{}{},
	}
	encodedDefinition, _ := json.Marshal(definition)
	requests := 0
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.Method != http.MethodGet {
			t.Fatalf("dry run made a write request: %s", r.Method)
		}
		var payload bytes.Buffer
		_ = json.NewEncoder(&payload).Encode(map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"id": "flow-1", "updatedAt": int64(1234),
				"draft": map[string]string{"definition": string(encodedDefinition)},
			},
		})
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(payload.Bytes())), Request: r}, nil
	})}
	client := newAnyCrossClientForTest("https://anycross.test", "session=test", "csrf", httpClient)
	plan, err := client.PatchWorkflowDraft("flow-1", 1234, []AnyCrossJSONPatchOperation{{
		Op: "replace", Path: "/steps/0/name", Value: "After",
	}}, false)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(plan)
	if requests != 1 || plan.Applied || contains(string(payload), "do-not-leak") || !contains(string(payload), "[REDACTED]") {
		t.Fatalf("unsafe dry-run result (requests=%d): %s", requests, payload)
	}
}

func contains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
