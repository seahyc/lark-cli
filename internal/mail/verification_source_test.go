package mail

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func TestValidateVerificationSourceAccountRequiresExactConfiguredValue(t *testing.T) {
	if err := validateVerificationSourceAccount("owner@example.com", "owner@example.com"); err != nil {
		t.Fatalf("matching account rejected: %v", err)
	}
	for _, requested := range []string{"", "Owner@example.com", "other@example.com"} {
		if err := validateVerificationSourceAccount("owner@example.com", requested); !errors.Is(err, ErrVerificationSourceAccount) {
			t.Errorf("requested account %q: got %v, want account mismatch", requested, err)
		}
	}
}

func TestAllowedVerificationIMAPHost(t *testing.T) {
	for _, host := range []string{"imap.larksuite.com", "IMAP.LARKSUITE.COM", "imap.feishu.cn"} {
		if !allowedVerificationIMAPHost(host) {
			t.Errorf("expected allowed host %q", host)
		}
	}
	for _, host := range []string{"localhost", "imap.larksuite.com.attacker.test", "imap.feishu.cn:993", ""} {
		if allowedVerificationIMAPHost(host) {
			t.Errorf("unexpectedly allowed host %q", host)
		}
	}
}

func TestAllowedVerificationSourceFolder(t *testing.T) {
	for _, folder := range []string{"INBOX", "Archive"} {
		if !allowedVerificationSourceFolder(folder) {
			t.Errorf("expected allowed folder %q", folder)
		}
	}
	for _, folder := range []string{"archive", "INBOX/../Archive", "Other", ""} {
		if allowedVerificationSourceFolder(folder) {
			t.Errorf("unexpectedly allowed folder %q", folder)
		}
	}
}

func TestExactEnvelopeSender(t *testing.T) {
	envelope := &imap.Envelope{From: []imap.Address{
		{Mailbox: "other", Host: "example.com"},
		{Mailbox: "verify", Host: "example.net"},
	}}
	if !exactEnvelopeSender(envelope, "verify@example.net") {
		t.Fatal("exact sender address was not matched")
	}
	for _, sender := range []string{"verify@example.net.attacker.test", "example.net", "verify@example.com"} {
		if exactEnvelopeSender(envelope, sender) {
			t.Errorf("sender %q should not match", sender)
		}
	}
	if exactEnvelopeSender(nil, "verify@example.net") {
		t.Fatal("nil envelope matched")
	}
}

func TestSelectVerificationCandidatesFiltersAndCapsNewestFive(t *testing.T) {
	since := time.Unix(1_700_000_000, 0)
	rows := make([]*imapclient.FetchMessageBuffer, 0, 8)
	for i := 0; i < 5; i++ {
		rows = append(rows, &imapclient.FetchMessageBuffer{
			UID:          imap.UID(i + 1),
			InternalDate: since.Add(time.Duration(i) * time.Minute),
			Envelope:     &imap.Envelope{From: []imap.Address{{Mailbox: "verify", Host: "example.net"}}},
		})
	}
	rows = append(rows,
		&imapclient.FetchMessageBuffer{UID: 8, InternalDate: since.Add(time.Hour), Envelope: &imap.Envelope{From: []imap.Address{{Mailbox: "verify", Host: "attacker.test"}}}},
		&imapclient.FetchMessageBuffer{UID: 9, InternalDate: since.Add(-time.Second), Envelope: &imap.Envelope{From: []imap.Address{{Mailbox: "verify", Host: "example.net"}}}},
	)
	selected, err := selectVerificationCandidates(rows, "verify@example.net", since)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != verificationSourceMaxItems {
		t.Fatalf("got %d candidates, want %d", len(selected), verificationSourceMaxItems)
	}
	for i, wantUID := range []imap.UID{5, 4, 3, 2, 1} {
		if selected[i].uid != wantUID {
			t.Errorf("candidate %d UID=%d, want %d", i, selected[i].uid, wantUID)
		}
	}
}

func TestSelectVerificationCandidatesFailsClosedAboveFive(t *testing.T) {
	since := time.Unix(1_700_000_000, 0)
	rows := make([]*imapclient.FetchMessageBuffer, verificationSourceMaxItems+1)
	for i := range rows {
		rows[i] = &imapclient.FetchMessageBuffer{
			UID:          imap.UID(i + 1),
			InternalDate: since.Add(time.Duration(i) * time.Minute),
			Envelope:     &imap.Envelope{From: []imap.Address{{Mailbox: "verify", Host: "example.net"}}},
		}
	}
	if _, err := selectVerificationCandidates(rows, "verify@example.net", since); !errors.Is(err, ErrVerificationSourceMatches) {
		t.Fatalf("got %v, want match-limit error", err)
	}
}

func TestValidateVerificationEnvelopeRowsRequiresCompleteMetadata(t *testing.T) {
	base := func(uid imap.UID) *imapclient.FetchMessageBuffer {
		return &imapclient.FetchMessageBuffer{
			UID:          uid,
			Envelope:     &imap.Envelope{},
			InternalDate: time.Unix(1_700_000_000, 0),
		}
	}
	if !validateVerificationEnvelopeRows(41, 42, []*imapclient.FetchMessageBuffer{base(41), base(42)}) {
		t.Fatal("complete metadata rejected")
	}
	missingDate := base(42)
	missingDate.InternalDate = time.Time{}
	for name, rows := range map[string][]*imapclient.FetchMessageBuffer{
		"out of window UID": {base(41), base(43)},
		"duplicate UID":     {base(41), base(41)},
		"missing date":      {base(41), missingDate},
		"missing envelope":  {base(41), {UID: 42, InternalDate: time.Unix(1_700_000_000, 0)}},
		"nil row":           {base(41), nil},
	} {
		t.Run(name, func(t *testing.T) {
			if validateVerificationEnvelopeRows(41, 42, rows) {
				t.Fatal("incomplete metadata accepted")
			}
		})
	}
}

func TestVerificationUIDWindowAndGapValidation(t *testing.T) {
	start, end, covers, err := verificationUIDWindow(1001)
	if err != nil || start != 951 || end != 1000 || covers {
		t.Fatalf("window=(%d,%d,%v), err=%v", start, end, covers, err)
	}
	rows := []*imapclient.FetchMessageBuffer{
		{UID: 952, Envelope: &imap.Envelope{}, InternalDate: time.Unix(1_700_000_000, 0)},
		{UID: 999, Envelope: &imap.Envelope{}, InternalDate: time.Unix(1_700_000_001, 0)},
	}
	if !validateVerificationEnvelopeRows(start, end, rows) {
		t.Fatal("UID gaps in the bounded window should be allowed")
	}
	if _, _, _, err := verificationUIDWindow(0); !errors.Is(err, ErrVerificationSourceNetwork) {
		t.Fatal("zero UIDNEXT should fail closed")
	}
	start, end, covers, err = verificationUIDWindow(1)
	if err != nil || start != 0 || end != 0 || !covers {
		t.Fatalf("empty mailbox window=(%d,%d,%v), err=%v", start, end, covers, err)
	}
}

func TestVerificationWindowFailsClosedWhenOlderUIDsMayHideMatches(t *testing.T) {
	since := time.Unix(1_700_000_000, 0)
	newer := []*imapclient.FetchMessageBuffer{{InternalDate: since.Add(time.Second)}}
	older := []*imapclient.FetchMessageBuffer{{InternalDate: since.Add(-time.Second)}}
	if !verificationWindowAmbiguous(false, newer, since) {
		t.Fatal("incomplete window with a recent oldest result should be ambiguous")
	}
	if verificationWindowAmbiguous(false, older, since) {
		t.Fatal("incomplete window with an older result should not be ambiguous")
	}
	if verificationWindowAmbiguous(true, nil, since) {
		t.Fatal("full-mailbox window should not be ambiguous")
	}
	if !verificationWindowAmbiguous(false, nil, since) {
		t.Fatal("empty incomplete window should be ambiguous")
	}
}

func TestVerificationSourceTransportFetchesBoundedPeekBytesReadOnly(t *testing.T) {
	raw := []byte("From: verify@example.net\r\nSubject: synthetic\r\n\r\nverification body")
	result, commands, bytesWritten, err := runSyntheticVerificationSource(t, raw)
	if err != nil {
		t.Fatalf("synthetic IMAP transport failed: %v", err)
	}
	if bytesWritten != len(raw) {
		t.Fatalf("transport read %d bytes, want %d", bytesWritten, len(raw))
	}
	if len(result.Messages) != 1 || !bytes.Equal(result.Messages[0].Raw, raw) {
		t.Fatalf("unexpected fetched message: %#v", result.Messages)
	}
	if result.Messages[0].UID != 42 || result.Messages[0].ReceivedAt != time.Date(2026, time.November, 1, 12, 0, 0, 0, time.UTC).Unix() {
		t.Fatalf("unexpected message metadata: %#v", result.Messages[0])
	}
	joined := strings.Join(commands, "\n")
	if !strings.Contains(joined, "EXAMINE INBOX") {
		t.Fatalf("mailbox was not opened read-only: %q", joined)
	}
	if !strings.Contains(joined, "BODY.PEEK[]<0.1048577>") {
		t.Fatalf("fetch was not the expected bounded BODY.PEEK partial: %q", joined)
	}
	if !strings.Contains(joined, "UID FETCH 1:42") || strings.Contains(joined, "UID SEARCH") {
		t.Fatalf("expected bounded UID range FETCH without SEARCH: %q", joined)
	}
	for _, command := range commands {
		upper := strings.ToUpper(command)
		if strings.HasPrefix(upper, "STORE ") || strings.HasPrefix(upper, "UID STORE ") || strings.HasPrefix(upper, "EXPUNGE") || strings.HasPrefix(upper, "UID EXPUNGE") {
			t.Fatalf("unexpected mutating command: %q", command)
		}
	}
}

func TestVerificationSourceTransportRejectsBodyOverOneMiB(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 20<<20)
	_, _, bytesWritten, err := runSyntheticVerificationSource(t, body)
	if !errors.Is(err, ErrVerificationSourceBody) {
		t.Fatalf("got %v, want message-too-large error", err)
	}
	if bytesWritten >= len(body) || bytesWritten > verificationSourceMaxRead {
		t.Fatalf("transport read exceeded bound: server wrote %d of %d bytes (connection cap %d)", bytesWritten, len(body), verificationSourceMaxRead)
	}
}

func runSyntheticVerificationSource(t *testing.T, body []byte) (*VerificationSourceResult, []string, int, error) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	if err := clientConn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	type serverResult struct {
		commands []string
		bytes    int
		err      error
	}
	serverCh := make(chan serverResult, 1)
	go func() {
		defer serverConn.Close()
		commands, n, err := serveSyntheticIMAP(serverConn, body)
		serverCh <- serverResult{commands: commands, bytes: n, err: err}
	}()
	creds := &Credentials{Username: "owner@example.com", Password: "synthetic"}
	result, err := verificationSourceOnConn(creds, creds.Username, "verify@example.net", time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC).Unix(), "INBOX", clientConn)
	server := <-serverCh
	if err != nil {
		t.Logf("synthetic commands: %q; server error: %v", server.commands, server.err)
	}
	if server.err != nil && err == nil {
		t.Fatalf("synthetic IMAP server failed: %v", server.err)
	}
	return result, server.commands, server.bytes, err
}

func serveSyntheticIMAP(conn net.Conn, body []byte) ([]string, int, error) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	write := func(s string) error {
		if _, err := io.WriteString(writer, s); err != nil {
			return err
		}
		return writer.Flush()
	}
	if err := write("* OK synthetic IMAP ready\r\n"); err != nil {
		return nil, 0, err
	}
	commands := make([]string, 0, 4)
	bodyBytesWritten := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if len(commands) > 0 && err == io.EOF {
				return commands, bodyBytesWritten, nil
			}
			return commands, bodyBytesWritten, err
		}
		line = strings.TrimRight(line, "\r\n")
		commands = append(commands, line)
		fields := strings.SplitN(line, " ", 3)
		if len(fields) < 2 {
			return commands, bodyBytesWritten, fmt.Errorf("malformed command")
		}
		tag, command := fields[0], strings.ToUpper(fields[1])
		switch command {
		case "CAPABILITY":
			if err := write("* CAPABILITY IMAP4rev1\r\n" + tag + " OK CAPABILITY completed\r\n"); err != nil {
				return commands, bodyBytesWritten, err
			}
		case "LOGIN":
			if err := write(tag + " OK LOGIN completed\r\n"); err != nil {
				return commands, bodyBytesWritten, err
			}
		case "EXAMINE":
			if err := write("* FLAGS (\\Seen \\Answered \\Flagged \\Deleted \\Draft)\r\n* 1 EXISTS\r\n* OK [UIDVALIDITY 77] valid\r\n* OK [UIDNEXT 43] next\r\n" + tag + " OK [READ-ONLY] EXAMINE completed\r\n"); err != nil {
				return commands, bodyBytesWritten, err
			}
		case "UID":
			if len(fields) < 3 {
				return commands, bodyBytesWritten, fmt.Errorf("missing UID subcommand")
			}
			subcommand := strings.ToUpper(strings.Fields(fields[2])[0])
			switch subcommand {
			case "SEARCH":
				if err := write("* SEARCH 42\r\n" + tag + " OK UID SEARCH completed\r\n"); err != nil {
					return commands, bodyBytesWritten, err
				}
			case "FETCH":
				received := "01-Nov-2026 12:00:00 +0000"
				if strings.Contains(line, "ENVELOPE") {
					response := fmt.Sprintf("* 1 FETCH (UID 42 INTERNALDATE %q ENVELOPE (%q %q ((%q NIL %q %q)) NIL NIL NIL NIL NIL NIL %q))\r\n%s OK UID FETCH completed\r\n",
						received, "Sun, 01 Nov 2026 12:00:00 +0000", "synthetic", "Verify", "verify", "example.net", "<synthetic@example.net>", tag)
					if err := write(response); err != nil {
						return commands, bodyBytesWritten, err
					}
				} else {
					if err := write(fmt.Sprintf("* 1 FETCH (UID 42 INTERNALDATE %q BODY[]<0> {%d}\r\n", received, len(body))); err != nil {
						return commands, bodyBytesWritten, err
					}
					n, err := conn.Write(body)
					bodyBytesWritten += n
					if err != nil {
						if len(body) > verificationSourceMaxRaw {
							return commands, bodyBytesWritten, nil
						}
						return commands, bodyBytesWritten, err
					}
					if err := write(")\r\n" + tag + " OK UID FETCH completed\r\n"); err != nil {
						return commands, bodyBytesWritten, err
					}
				}
			default:
				return commands, bodyBytesWritten, fmt.Errorf("unexpected UID command")
			}
		default:
			return commands, bodyBytesWritten, fmt.Errorf("unexpected command")
		}
	}
}

func TestVerificationSourceResultJSONShapeAndBase64Raw(t *testing.T) {
	result := VerificationSourceResult{
		Account:     "owner@example.com",
		UIDValidity: 19,
		Messages: []VerificationSourceMessage{{
			UID:        23,
			ReceivedAt: time.Unix(123, 0).Unix(),
			Raw:        []byte("Subject: example\r\n\r\nbody"),
		}},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got["account"] == nil || got["uid_validity"] == nil || got["messages"] == nil {
		t.Fatalf("unexpected result keys: %s", data)
	}
	var messages []map[string]json.RawMessage
	if err := json.Unmarshal(got["messages"], &messages); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || len(messages[0]) != 3 || messages[0]["uid"] == nil || messages[0]["received_at"] == nil || messages[0]["raw"] == nil {
		t.Fatalf("unexpected message keys: %s", got["messages"])
	}
	var raw string
	if err := json.Unmarshal(messages[0]["raw"], &raw); err != nil {
		t.Fatal(err)
	}
	if raw != "U3ViamVjdDogZXhhbXBsZQ0KDQpib2R5" {
		t.Fatalf("raw JSON field is not base64 encoded: %q", raw)
	}
}

func TestVerificationSourceLimitsAreBounded(t *testing.T) {
	if verificationSourceScanLimit != 50 || verificationSourceMaxItems != 5 || verificationSourceMaxRaw != 1<<20 {
		t.Fatalf("unexpected limits: scan=%d items=%d raw=%d", verificationSourceScanLimit, verificationSourceMaxItems, verificationSourceMaxRaw)
	}
}
