package mail

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

const (
	verificationSourceScanLimit = 50
	verificationSourceMaxItems  = 5
	verificationSourceMaxRaw    = 1 << 20
	verificationSourceMaxRead   = 8 << 20
	verificationSourceTimeout   = 10 * time.Second
)

var (
	ErrVerificationSourceAccount = errors.New("ACCOUNT_MISMATCH")
	ErrVerificationSourceConfig  = errors.New("CONFIG_INVALID")
	ErrVerificationSourceNetwork = errors.New("IMAP_ERROR")
	ErrVerificationSourceLimit   = errors.New("SCAN_LIMIT")
	ErrVerificationSourceMatches = errors.New("MATCH_LIMIT")
	ErrVerificationSourceBody    = errors.New("MESSAGE_TOO_LARGE")
	ErrVerificationSourceInput   = errors.New("INVALID_INPUT")
)

// VerificationSourceMessage contains a private RFC822 message for a bounded
// verification workflow. Raw is JSON-encoded as base64 by encoding/json.
type VerificationSourceMessage struct {
	UID        uint32 `json:"uid"`
	ReceivedAt int64  `json:"received_at"`
	Raw        []byte `json:"raw"`
}

// VerificationSourceResult is intentionally limited to one configured
// account, INBOX, and a small number of exact-sender messages.
type VerificationSourceResult struct {
	Account     string                      `json:"account"`
	UIDValidity uint32                      `json:"uid_validity"`
	Messages    []VerificationSourceMessage `json:"messages"`
}

func allowedVerificationIMAPHost(host string) bool {
	switch strings.ToLower(strings.TrimSuffix(host, ".")) {
	case "imap.larksuite.com", "imap.feishu.cn":
		return true
	default:
		return false
	}
}

func allowedVerificationSourceFolder(folder string) bool {
	return folder == "INBOX" || folder == "Archive"
}

func validateVerificationSourceAccount(configured, requested string) error {
	if requested == "" || configured == "" || requested != configured {
		return ErrVerificationSourceAccount
	}
	return nil
}

func exactEnvelopeSender(envelope *imap.Envelope, sender string) bool {
	if envelope == nil {
		return false
	}
	for _, address := range envelope.From {
		if strings.EqualFold(address.Addr(), sender) {
			return true
		}
	}
	return false
}

type verificationCandidate struct {
	uid        imap.UID
	receivedAt time.Time
}

func selectVerificationCandidates(rows []*imapclient.FetchMessageBuffer, sender string, since time.Time) ([]verificationCandidate, error) {
	candidates := make([]verificationCandidate, 0, verificationSourceMaxItems)
	for _, row := range rows {
		if row == nil || row.UID == 0 || row.InternalDate.IsZero() || row.InternalDate.Before(since) || !exactEnvelopeSender(row.Envelope, sender) {
			continue
		}
		candidates = append(candidates, verificationCandidate{uid: row.UID, receivedAt: row.InternalDate})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].receivedAt.After(candidates[j].receivedAt) })
	if len(candidates) > verificationSourceMaxItems {
		return nil, ErrVerificationSourceMatches
	}
	return candidates, nil
}

func validateVerificationEnvelopeRows(start, end imap.UID, rows []*imapclient.FetchMessageBuffer) bool {
	if start == 0 || end < start || len(rows) > int(end-start)+1 {
		return false
	}
	seen := make(map[imap.UID]struct{}, len(rows))
	for _, row := range rows {
		if row == nil || row.UID == 0 || row.Envelope == nil || row.InternalDate.IsZero() {
			return false
		}
		if row.UID < start || row.UID > end {
			return false
		}
		if _, duplicate := seen[row.UID]; duplicate {
			return false
		}
		seen[row.UID] = struct{}{}
	}
	return true
}

func verificationUIDWindow(uidNext imap.UID) (start, end imap.UID, coversMailbox bool, err error) {
	if uidNext == 0 {
		return 0, 0, false, ErrVerificationSourceNetwork
	}
	if uidNext == 1 {
		return 0, 0, true, nil
	}
	end = uidNext - 1
	start = 1
	if uidNext > verificationSourceScanLimit+1 {
		start = uidNext - verificationSourceScanLimit
	}
	return start, end, start == 1, nil
}

func verificationWindowAmbiguous(coversMailbox bool, rows []*imapclient.FetchMessageBuffer, since time.Time) bool {
	if coversMailbox {
		return false
	}
	oldest := time.Time{}
	for _, row := range rows {
		if row != nil && (oldest.IsZero() || row.InternalDate.Before(oldest)) {
			oldest = row.InternalDate
		}
	}
	return oldest.IsZero() || !oldest.Before(since)
}

func readVerificationLiteral(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, verificationSourceMaxRaw+1))
	if err != nil {
		return nil, ErrVerificationSourceNetwork
	}
	if len(data) > verificationSourceMaxRaw {
		// go-imap's response decoder waits for the literal reader to reach EOF
		// before it can finish the FETCH response. The connection wrapper caps
		// this discard at a small total read budget.
		_, _ = io.Copy(io.Discard, reader)
		return nil, ErrVerificationSourceBody
	}
	return data, nil
}

type cappedReadConn struct {
	net.Conn
	remaining int64
}

func (conn *cappedReadConn) Read(p []byte) (int, error) {
	if conn.remaining <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > conn.remaining {
		p = p[:conn.remaining]
	}
	n, err := conn.Conn.Read(p)
	conn.remaining -= int64(n)
	return n, err
}

func fetchVerificationMessage(client *imapclient.Client, uid imap.UID) (VerificationSourceMessage, error) {
	section := &imap.FetchItemBodySection{Peek: true, Partial: &imap.SectionPartial{Offset: 0, Size: verificationSourceMaxRaw + 1}}
	cmd := client.Fetch(imap.UIDSetNum(uid), &imap.FetchOptions{
		UID:          true,
		InternalDate: true,
		BodySection:  []*imap.FetchItemBodySection{section},
	})
	messageCount := 0
	var result VerificationSourceMessage
	gotUID, gotDate, gotBody := false, false, false
	for msg := cmd.Next(); msg != nil; msg = cmd.Next() {
		messageCount++
		for item := msg.Next(); item != nil; item = msg.Next() {
			switch item := item.(type) {
			case imapclient.FetchItemDataUID:
				result.UID = uint32(item.UID)
				gotUID = item.UID == uid
			case imapclient.FetchItemDataInternalDate:
				if !item.Time.IsZero() {
					result.ReceivedAt = item.Time.Unix()
					gotDate = true
				}
			case imapclient.FetchItemDataBodySection:
				if item.Literal == nil {
					return VerificationSourceMessage{}, ErrVerificationSourceNetwork
				}
				raw, err := readVerificationLiteral(item.Literal)
				if err != nil {
					return VerificationSourceMessage{}, err
				}
				result.Raw = raw
				gotBody = true
			}
		}
	}
	if err := cmd.Close(); err != nil || messageCount != 1 || !gotUID || !gotDate || !gotBody {
		return VerificationSourceMessage{}, ErrVerificationSourceNetwork
	}
	return result, nil
}

// VerificationSource performs a read-only bounded fetch from INBOX. It never
// writes mail data or credentials to disk and returns only fixed error codes.
func VerificationSource(account, sender string, since int64) (*VerificationSourceResult, error) {
	return VerificationSourceInMailbox(account, sender, since, "INBOX")
}

// VerificationSourceInMailbox performs the bounded read against one explicitly
// allowed mailbox. Folder names are closed over INBOX and Archive.
func VerificationSourceInMailbox(account, sender string, since int64, folder string) (*VerificationSourceResult, error) {
	if account == "" || sender == "" || len(account) > 254 || len(sender) > 254 || since <= 0 {
		return nil, ErrVerificationSourceInput
	}
	if !allowedVerificationSourceFolder(folder) {
		return nil, ErrVerificationSourceInput
	}
	creds, err := LoadCredentials()
	if err != nil {
		return nil, ErrVerificationSourceConfig
	}
	if err := validateVerificationSourceAccount(creds.Username, account); err != nil {
		return nil, err
	}
	if !allowedVerificationIMAPHost(creds.Host) || creds.Port != 993 || !creds.UseSSL || creds.Password == "" {
		return nil, ErrVerificationSourceConfig
	}

	deadline := time.Now().Add(verificationSourceTimeout)
	dialer := &net.Dialer{Timeout: verificationSourceTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(creds.Host, "993"), &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: creds.Host,
	})
	if err != nil {
		return nil, ErrVerificationSourceNetwork
	}
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return nil, ErrVerificationSourceNetwork
	}
	return verificationSourceOnConn(creds, account, sender, since, folder, conn)
}

// verificationSourceOnConn runs the bounded IMAP exchange over an already
// connected socket. Keeping the protocol exchange separate allows synthetic
// local transport tests without reaching a real mail service.
func verificationSourceOnConn(creds *Credentials, account, sender string, since int64, folder string, conn net.Conn) (*VerificationSourceResult, error) {
	client := imapclient.New(&cappedReadConn{Conn: conn, remaining: verificationSourceMaxRead}, nil)
	defer client.Close()
	if err := client.Login(creds.Username, creds.Password).Wait(); err != nil {
		return nil, ErrVerificationSourceNetwork
	}
	selected, err := client.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		return nil, ErrVerificationSourceNetwork
	}

	result := &VerificationSourceResult{
		Account:     account,
		UIDValidity: selected.UIDValidity,
		Messages:    make([]VerificationSourceMessage, 0, verificationSourceMaxItems),
	}
	if selected.NumMessages == 0 {
		return result, nil
	}
	startUID, endUID, coversMailbox, err := verificationUIDWindow(selected.UIDNext)
	if err != nil {
		return nil, err
	}
	if selected.UIDNext == 1 {
		return result, nil
	}
	var uidSet imap.UIDSet
	uidSet.AddRange(startUID, endUID)
	envelopeRows, err := client.Fetch(uidSet, &imap.FetchOptions{UID: true, Envelope: true, InternalDate: true}).Collect()
	if err != nil || !validateVerificationEnvelopeRows(startUID, endUID, envelopeRows) {
		return nil, ErrVerificationSourceNetwork
	}
	if verificationWindowAmbiguous(coversMailbox, envelopeRows, time.Unix(since, 0)) {
		return nil, ErrVerificationSourceLimit
	}
	candidates, err := selectVerificationCandidates(envelopeRows, sender, time.Unix(since, 0))
	if err != nil {
		return nil, err
	}
	for _, item := range candidates {
		message, err := fetchVerificationMessage(client, item.uid)
		if err != nil {
			return nil, err
		}
		result.Messages = append(result.Messages, message)
	}
	return result, nil
}
