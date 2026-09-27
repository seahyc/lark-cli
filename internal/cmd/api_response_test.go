package cmd

import (
	"github.com/yjwong/lark-cli/internal/api"
	"testing"
)

func TestRawAPIResponseFailed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		failed bool
	}{
		{"unsupported forward", 200, `{"code":1230005,"msg":"unsupported action type"}`, true},
		{"missing scope", 200, `{"code":99991679}`, true},
		{"missing route", 404, "404 page not found", true},
		{"success envelope", 200, `{"code":0,"data":{}}`, false},
		{"empty success", 204, "", false},
		{"plain success", 200, "ok", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := rawAPIResponseFailed(&api.RawAPIResponse{StatusCode: tc.status, Body: []byte(tc.body)})
			if got != tc.failed {
				t.Fatalf("failed=%v, want %v", got, tc.failed)
			}
		})
	}
}
