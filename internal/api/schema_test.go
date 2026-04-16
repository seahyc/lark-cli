package api

import (
	"strings"
	"testing"
)

func TestInferModule(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/open-apis/im/v1/messages", "im"},
		{"/im/v1/messages", "im"},
		{"im/v1/messages", "im"},
		{"im.messages.send", "im"},
		{"approval/v4/instances/query", "approval"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := InferModule(tc.in); got != tc.want {
			t.Errorf("InferModule(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSearchNeedles_PathForm(t *testing.T) {
	got := searchNeedles("/open-apis/im/v1/messages")
	if !containsAll(got, "/open-apis/im/v1/messages", "/im/v1/messages") {
		t.Fatalf("missing expected needles, got %v", got)
	}
}

func TestSearchNeedles_DottedForm(t *testing.T) {
	got := searchNeedles("im.messages.send")
	if !containsAll(got, "messages/send", "send") {
		t.Fatalf("missing expected needles, got %v", got)
	}
}

func TestExtractSections_MatchesMiddleSection(t *testing.T) {
	body := strings.Join([]string{
		"## intro",
		"nothing here",
		"## message endpoint",
		"path: /open-apis/im/v1/messages",
		"## unrelated",
		"nothing",
	}, "\n")
	got := extractSections(body, []string{"/open-apis/im/v1/messages"})
	if got == "" || !strings.Contains(got, "message endpoint") || strings.Contains(got, "unrelated") {
		t.Fatalf("expected only middle section, got %q", got)
	}
}

func TestExtractSections_NoSectionsFallbackToFullBody(t *testing.T) {
	body := "no headings here, but contains /im/v1/messages in the middle"
	got := extractSections(body, []string{"/im/v1/messages"})
	if got != body {
		t.Fatalf("expected full body fallback, got %q", got)
	}
}

func TestExtractSections_NoMatchReturnsEmpty(t *testing.T) {
	body := "## a\nfoo\n## b\nbar"
	if got := extractSections(body, []string{"nonexistent"}); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func containsAll(ss []string, targets ...string) bool {
	for _, t := range targets {
		found := false
		for _, s := range ss {
			if s == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
