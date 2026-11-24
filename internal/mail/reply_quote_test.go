package mail

import (
	"encoding/base64"
	"strings"
	"testing"
)

func b64Wrapped(s string) string {
	e := base64.StdEncoding.EncodeToString([]byte(s))
	var out []string
	for i := 0; i < len(e); i += 76 {
		end := i + 76
		if end > len(e) {
			end = len(e)
		}
		out = append(out, e[i:end])
	}
	return strings.Join(out, "\r\n")
}

func TestExtractAndQuoteBodyBase64MultipartRelated(t *testing.T) {
	plain := "Hi team,\r\n\r\nPlease review the plan \u2013 caf\u00e9 menu.\r\nThanks"
	html := "<html><body><p>Hi team</p></body></html>"
	raw := "From: a@example.com\r\n" +
		"Subject: Plan\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/related; boundary=\"REL\"\r\n\r\n" +
		"--REL\r\n" +
		"Content-Type: multipart/alternative; boundary=\"ALT\"\r\n\r\n" +
		"--ALT\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"Content-Transfer-Encoding: base64\r\n\r\n" +
		b64Wrapped(plain) + "\r\n" +
		"--ALT\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"Content-Transfer-Encoding: base64\r\n\r\n" +
		b64Wrapped(html) + "\r\n" +
		"--ALT--\r\n" +
		"--REL--\r\n"

	got := ExtractAndQuoteBody([]byte(raw), &Envelope{FromName: "Alice"}, "a@example.com")
	want := "> Hi team,\n>\n> Please review the plan \u2013 caf\u00e9 menu.\n> Thanks\n"
	if !strings.HasSuffix(got, want) {
		t.Fatalf("quote mismatch.\ngot:\n%q\nwant suffix:\n%q", got, want)
	}
	if !strings.Contains(got, "wrote:") {
		t.Fatalf("missing header: %q", got)
	}
	for _, bad := range []string{"--REL", "--ALT", "Content-Type", "Content-Transfer-Encoding", b64Wrapped(plain)[:20], "\r"} {
		if strings.Contains(got, bad) {
			t.Fatalf("quote leaked %q:\n%s", bad, got)
		}
	}
}

func TestExtractAndQuoteBodyQuotedPrintableLatin1(t *testing.T) {
	raw := "Content-Type: text/plain; charset=iso-8859-1\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
		"caf=E9 au lait=\r\n, ok\r\n\r\nbye\r\n"
	got := ExtractAndQuoteBody([]byte(raw), nil, "a@example.com")
	if !strings.Contains(got, "> caf\u00e9 au lait, ok\n>\n> bye\n") {
		t.Fatalf("unexpected quote: %q", got)
	}
}

func TestExtractAndQuoteBodyHTMLFallback(t *testing.T) {
	raw := "Content-Type: multipart/alternative; boundary=\"B\"\r\n\r\n" +
		"--B\r\nContent-Type: text/html; charset=utf-8\r\nContent-Transfer-Encoding: base64\r\n\r\n" +
		b64Wrapped("<style>p{}</style><p>Hello &amp; welcome</p><p>Line two</p>") + "\r\n--B--\r\n"
	got := ExtractAndQuoteBody([]byte(raw), nil, "a@example.com")
	if !strings.Contains(got, "> Hello & welcome\n> Line two\n") || strings.Contains(got, "<") || strings.Contains(got, "p{}") {
		t.Fatalf("unexpected html fallback quote: %q", got)
	}
}
