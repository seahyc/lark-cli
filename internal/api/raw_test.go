package api

import "testing"

func TestNormalizeAPIPath_OpenApisPrefixStripped(t *testing.T) {
	got := NormalizeAPIPath("https://open.larksuite.com/open-apis", "/open-apis/im/v1/messages")
	want := "https://open.larksuite.com/open-apis/im/v1/messages"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestNormalizeAPIPath_PlainRelativeAppended(t *testing.T) {
	got := NormalizeAPIPath("https://open.larksuite.com/open-apis", "/im/v1/messages")
	want := "https://open.larksuite.com/open-apis/im/v1/messages"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestNormalizeAPIPath_AbsoluteURLPassesThrough(t *testing.T) {
	for _, in := range []string{
		"https://open.larksuite.com/foo",
		"http://localhost:8080/bar",
	} {
		if got := NormalizeAPIPath("https://open.larksuite.com/open-apis", in); got != in {
			t.Fatalf("absolute URL should pass through; want %q, got %q", in, got)
		}
	}
}

func TestNormalizeAPIPath_OpenApisPrefixDoesNotDoubleUp(t *testing.T) {
	// Catches the bug where both base and path carry /open-apis.
	got := NormalizeAPIPath("https://open.larksuite.com/open-apis", "/open-apis/foo")
	if got != "https://open.larksuite.com/open-apis/foo" {
		t.Fatalf("unexpected doubled prefix: %q", got)
	}
}
