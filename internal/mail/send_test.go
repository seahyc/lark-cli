package mail

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildMessageOmitsEmptyTo(t *testing.T) {
	msg, err := buildMessage("me@example.com", &SendOptions{Subject: "Hi", Body: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(msg), "\r\nTo:") {
		t.Fatalf("expected no To header, got:\n%s", msg)
	}
}

func TestBuildMessageHTMLAlternative(t *testing.T) {
	html := "<p><b>Hello</b></p>"
	msg, err := buildMessage("me@example.com", &SendOptions{
		To: []string{"you@example.com"}, Subject: "Hi", Body: "Hello", HTMLBody: html,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(msg)
	for _, want := range []string{
		"Content-Type: multipart/alternative;",
		"Content-Type: text/plain; charset=\"utf-8\"",
		"Content-Type: text/html; charset=\"utf-8\"",
		base64.StdEncoding.EncodeToString([]byte(html)),
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in:\n%s", want, s)
		}
	}
	if strings.Index(s, "text/plain") > strings.Index(s, "text/html") {
		t.Fatal("plain text part must come before the HTML part")
	}
}
