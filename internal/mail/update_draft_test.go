package mail

import (
	"strings"
	"testing"
)

const draftRaw = "From: me@example.com\r\n" +
	"To: you@example.com\r\n" +
	"Subject: Re: Plan\r\n" +
	"In-Reply-To: <orig@larksuite.com>\r\n" +
	"References: <root@larksuite.com>\r\n\t<orig@larksuite.com>\r\n" +
	"Thread-Topic: Plan\r\n" +
	"Thread-Index: AQHabc==\r\n" +
	"\r\n"

func TestParseThreadingHeaders(t *testing.T) {
	h := &DraftHeaders{}
	parseThreadingHeaders([]byte(draftRaw), h)
	if h.InReplyTo != "<orig@larksuite.com>" {
		t.Fatalf("InReplyTo = %q", h.InReplyTo)
	}
	if len(h.References) != 2 || h.References[0] != "<root@larksuite.com>" || h.References[1] != "<orig@larksuite.com>" {
		t.Fatalf("References = %v", h.References)
	}
	if len(h.Extra) != 2 || h.Extra[0] != [2]string{"Thread-Topic", "Plan"} || h.Extra[1] != [2]string{"Thread-Index", "AQHabc=="} {
		t.Fatalf("Extra = %v", h.Extra)
	}
}

func TestUpdateDraftPreservesThreadingHeaders(t *testing.T) {
	existing := &DraftHeaders{To: []string{"you@example.com"}, CC: []string{"cc@example.com"}, Subject: "Re: Plan"}
	parseThreadingHeaders([]byte(draftRaw), existing)

	so := applyDraftHeaders("me@example.com", &UpdateDraftOptions{Body: "new body", KeepHeaders: true}, existing)
	if len(so.To) != 1 || so.Subject != "Re: Plan" || len(so.CC) != 1 {
		t.Fatalf("To/Cc/Subject not kept: %+v", so)
	}
	msg, err := buildMessage("me@example.com", so)
	if err != nil {
		t.Fatal(err)
	}
	s := string(msg)
	for _, want := range []string{
		"\r\nIn-Reply-To: <orig@larksuite.com>\r\n",
		"\r\nReferences: <root@larksuite.com> <orig@larksuite.com>\r\n",
		"\r\nThread-Topic: Plan\r\n",
		"\r\nThread-Index: AQHabc==\r\n",
		"\r\nSubject: Re: Plan\r\n",
		"new body",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestApplyDraftHeadersOverridesAndNoExisting(t *testing.T) {
	existing := &DraftHeaders{To: []string{"old@example.com"}, Subject: "Old", InReplyTo: "<x@y>"}
	so := applyDraftHeaders("me@example.com", &UpdateDraftOptions{Body: "b", To: []string{"new@example.com"}, Subject: "New"}, existing)
	if so.To[0] != "new@example.com" || so.Subject != "New" || so.InReplyTo != "<x@y>" {
		t.Fatalf("unexpected: %+v", so)
	}
	so = applyDraftHeaders("me@example.com", &UpdateDraftOptions{Body: "b", To: []string{"a@b"}}, nil)
	if so.InReplyTo != "" || so.To[0] != "a@b" {
		t.Fatalf("unexpected: %+v", so)
	}
}
