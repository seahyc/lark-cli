package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/yjwong/lark-cli/internal/auth"
	"github.com/yjwong/lark-cli/internal/config"
)

func TestSearchTasksWireAndErrors(t *testing.T) {
	t.Setenv("LARK_CONFIG_DIR", t.TempDir())
	if err := config.Init(); err != nil {
		t.Fatal(err)
	}
	store := auth.GetTokenStore()
	store.AccessToken = "synthetic-test-token"
	store.ExpiresAt = time.Now().Add(time.Hour)
	completed := false
	for _, scenario := range []struct {
		name, body string
		wantErr    bool
	}{
		{"success", `{"code":0,"data":{"items":[{"id":"task-guid","display_info":"limit","meta_data":{"app_link":"task-link"}}],"has_more":true,"page_token":"next"}}`, false},
		{"permission denied", `{"code":99991663,"msg":"permission denied"}`, true},
		{"missing data", `{"code":0}`, true},
		{"malformed", `not json`, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			client := &Client{httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != "POST" || req.URL.Path != "/open-apis/task/v2/tasks/search" || req.URL.Query().Get("page_token") != "cursor" || req.Header.Get("Authorization") != "Bearer synthetic-test-token" {
					t.Fatalf("unexpected search request: %s %s", req.Method, req.URL)
				}
				var body map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				filter := body["filter"].(map[string]interface{})
				if body["query"] != "limit" || filter["is_completed"] != false {
					t.Fatalf("body=%v", body)
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(scenario.body))}, nil
			})}}
			result, err := client.SearchTasks(TaskSearchOptions{Query: " limit ", PageToken: "cursor", Filter: &TaskSearchFilter{IsCompleted: &completed}})
			if (err != nil) != scenario.wantErr {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if !scenario.wantErr && (len(result.Items) != 1 || result.Items[0].ID != "task-guid" || !result.HasMore || result.PageToken != "next") {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

func TestSearchTasksRejectsEmptyQuery(t *testing.T) {
	if _, err := (&Client{}).SearchTasks(TaskSearchOptions{}); err == nil {
		t.Fatal("empty search accepted")
	}
}
