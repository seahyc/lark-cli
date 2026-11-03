package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yjwong/lark-cli/internal/api"
)

// fakeDirectory is an in-memory recipientDirectory; it records nothing and
// performs no network access.
type fakeDirectory struct {
	self        *api.UserInfo
	selfErr     error
	search      map[string][]api.SearchUserResult
	searchErr   error
	emails      map[string]string
	users       map[string]*api.ContactUser
	depts       map[string]string
	chats       map[string]*api.ChatDetail
	counterpart map[string][2]string
	appID       string
	searchCalls int
}

func (f *fakeDirectory) CurrentUser() (*api.UserInfo, error) { return f.self, f.selfErr }
func (f *fakeDirectory) SearchUsers(q string) ([]api.SearchUserResult, error) {
	f.searchCalls++
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	return f.search[q], nil
}
func (f *fakeDirectory) OpenIDByEmail(e string) (string, error) { return f.emails[e], nil }
func (f *fakeDirectory) User(id string) (*api.ContactUser, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}
func (f *fakeDirectory) DepartmentName(id string) string { return f.depts[id] }
func (f *fakeDirectory) Chat(id string, _ bool) (*api.ChatDetail, error) {
	if c, ok := f.chats[id]; ok {
		return c, nil
	}
	return nil, errors.New("chat not found")
}
func (f *fakeDirectory) P2PCounterpart(id string) (string, string) {
	c := f.counterpart[id]
	return c[0], c[1]
}
func (f *fakeDirectory) AppID() string { return f.appID }

func newFakeDirectory() *fakeDirectory {
	return &fakeDirectory{
		self: &api.UserInfo{Name: "Ying Cong", OpenID: "ou_self", Email: "yc@example.com"},
		search: map[string][]api.SearchUserResult{
			"Oswald Yeo": {
				{Name: "Oswald Yeo", OpenID: "ou_oswald", DepartmentIDs: []string{"od_mgmt"}},
				{Name: "Oswald Yeoh Tan", OpenID: "ou_oswald2"},
			},
			"Oswald": {
				{Name: "Oswald Yeo", OpenID: "ou_oswald"},
				{Name: "Oswald Lim", OpenID: "ou_oswald3", DepartmentIDs: []string{"od_eng"}},
			},
			"Alex": {
				{Name: "Alex", OpenID: "ou_alex1"},
				{Name: "alex", OpenID: "ou_alex2"},
			},
		},
		emails: map[string]string{"oswald@example.com": "ou_oswald"},
		users: map[string]*api.ContactUser{
			"ou_oswald":  {Name: "Oswald Yeo", EnName: "Oswald Yeo", Email: "oswald@example.com", DepartmentIDs: []string{"od_mgmt"}},
			"ou_oswald3": {Name: "Oswald Lim", Email: "olim@example.com"},
			"ou_self":    {Name: "Ying Cong", Email: "yc@example.com"},
		},
		depts: map[string]string{"od_mgmt": "Management", "od_eng": "Engineering"},
		chats: map[string]*api.ChatDetail{
			"oc_dm":    {ChatID: "oc_dm", ChatMode: "p2p", ChatType: "private", UserCount: 2},
			"oc_group": {ChatID: "oc_group", ChatMode: "group", ChatType: "private", Name: "Orion", UserCount: 5, BotCount: 1},
		},
		counterpart: map[string][2]string{"oc_dm": {"ou_oswald", "Oswald Yeo"}},
		appID:       "cli_test",
	}
}

func TestLooksLikePersonName(t *testing.T) {
	cases := map[string]bool{
		"Oswald Yeo":         true,
		"oswald":             true,
		"王小明":                true,
		"ou_abc":             false,
		"oc_abc":             false,
		"oswald@example.com": false,
		"12345":              false,
		"3e3cf96b":           false, // user_id-shaped token keeps legacy handling
		"":                   false,
	}
	for in, want := range cases {
		if got := looksLikePersonName(in); got != want {
			t.Errorf("looksLikePersonName(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestResolveSendRecipientExactNameWins(t *testing.T) {
	dir := newFakeDirectory()
	r, err := resolveSendRecipient(dir, "Oswald Yeo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.ReceiveIDType != "open_id" || r.ReceiveID != "ou_oswald" || r.ResolvedBy != "name" {
		t.Fatalf("got %+v", r)
	}
	if r.Person.Email != "oswald@example.com" || r.Person.Department != "Management" {
		t.Fatalf("person not enriched: %+v", r.Person)
	}
}

func TestResolveSendRecipientAmbiguousNameListsCandidates(t *testing.T) {
	dir := newFakeDirectory()
	_, err := resolveSendRecipient(dir, "Oswald", "")
	var re *recipientError
	if !errors.As(err, &re) || re.Code != "AMBIGUOUS_RECIPIENT" {
		t.Fatalf("want AMBIGUOUS_RECIPIENT, got %v", err)
	}
	if len(re.Candidates) != 2 {
		t.Fatalf("want 2 candidates, got %+v", re.Candidates)
	}
	got := map[string]recipientPerson{}
	for _, c := range re.Candidates {
		got[c.OpenID] = c
	}
	if got["ou_oswald3"].Email != "olim@example.com" || got["ou_oswald3"].Department != "Engineering" {
		t.Fatalf("candidate missing email/department: %+v", got["ou_oswald3"])
	}
}

func TestResolveSendRecipientDuplicateExactNamesAreAmbiguous(t *testing.T) {
	dir := newFakeDirectory()
	_, err := resolveSendRecipient(dir, "Alex", "")
	var re *recipientError
	if !errors.As(err, &re) || re.Code != "AMBIGUOUS_RECIPIENT" || len(re.Candidates) != 2 {
		t.Fatalf("want ambiguity with 2 candidates, got %v", err)
	}
}

func TestResolveSendRecipientFuzzyOnlyHitIsNotTrusted(t *testing.T) {
	dir := newFakeDirectory()
	// Contact search is fuzzy: it may return a lone hit whose name does not
	// contain every query token. That hit must be listed, not used.
	dir.search["Tan Os"] = []api.SearchUserResult{{Name: "Joseph Osman", OpenID: "ou_joseph"}}
	_, err := resolveSendRecipient(dir, "Tan Os", "")
	var re *recipientError
	if !errors.As(err, &re) || re.Code != "RECIPIENT_NOT_FOUND" {
		t.Fatalf("want RECIPIENT_NOT_FOUND, got %v", err)
	}
	if len(re.Candidates) != 1 || re.Candidates[0].OpenID != "ou_joseph" {
		t.Fatalf("near-miss should be listed as a candidate: %+v", re.Candidates)
	}
}

func TestResolveSendRecipientNoHits(t *testing.T) {
	dir := newFakeDirectory()
	_, err := resolveSendRecipient(dir, "Nobody Here", "")
	var re *recipientError
	if !errors.As(err, &re) || re.Code != "RECIPIENT_NOT_FOUND" || len(re.Candidates) != 0 {
		t.Fatalf("want RECIPIENT_NOT_FOUND without candidates, got %v", err)
	}
}

func TestResolveSendRecipientMe(t *testing.T) {
	dir := newFakeDirectory()
	r, err := resolveSendRecipient(dir, "me", "")
	if err != nil || r.ReceiveID != "ou_self" || r.ResolvedBy != "me" {
		t.Fatalf("got %+v, %v", r, err)
	}
	if dir.searchCalls != 0 {
		t.Fatal("me must not hit contact search")
	}
}

func TestResolveSendRecipientRawIDsPassThrough(t *testing.T) {
	dir := newFakeDirectory()
	cases := []struct{ to, toType, wantType string }{
		{"ou_x", "", "open_id"},
		{"oc_x", "", "chat_id"},
		{"oswald@example.com", "", "email"},
		{"3e3cf96b", "user_id", "user_id"},
		{"Oswald Yeo", "open_id", "open_id"}, // explicit type disables name lookup
	}
	for _, c := range cases {
		r, err := resolveSendRecipient(dir, c.to, c.toType)
		if err != nil || r.ReceiveIDType != c.wantType || r.ReceiveID != c.to {
			t.Errorf("resolveSendRecipient(%q,%q) = %+v, %v", c.to, c.toType, r, err)
		}
	}
	if dir.searchCalls != 0 {
		t.Fatalf("raw IDs must not trigger contact search (calls=%d)", dir.searchCalls)
	}
}

func TestResolveSendRecipientExplicitNameType(t *testing.T) {
	dir := newFakeDirectory()
	dir.search["3e3cf96b"] = []api.SearchUserResult{{Name: "3e3cf96b", OpenID: "ou_odd"}}
	r, err := resolveSendRecipient(dir, "3e3cf96b", "name")
	if err != nil || r.ReceiveID != "ou_odd" {
		t.Fatalf("got %+v, %v", r, err)
	}
}

func TestResolveSendRecipientSearchError(t *testing.T) {
	dir := newFakeDirectory()
	dir.searchErr = errors.New("99991679 permission denied")
	_, err := resolveSendRecipient(dir, "Oswald Yeo", "")
	var re *recipientError
	if !errors.As(err, &re) || re.Code != "API_ERROR" {
		t.Fatalf("want API_ERROR, got %v", err)
	}
}

func makeSizedFile(t *testing.T, name string, size int64) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(size); err != nil { // sparse; no real disk use
		t.Fatal(err)
	}
	f.Close()
	return p
}

func TestInspectAttachmentLimits(t *testing.T) {
	ok := inspectAttachment("file", makeSizedFile(t, "a.pdf", 1024))
	if !ok.WithinLimit || ok.Problem != "" || ok.UploadFileType != "pdf" || ok.MsgType != "file" {
		t.Fatalf("small pdf: %+v", ok)
	}
	atLimit := inspectAttachment("file", makeSizedFile(t, "b.pptx", api.MaxIMFileBytes))
	if !atLimit.WithinLimit {
		t.Fatalf("file exactly at limit should pass: %+v", atLimit)
	}
	big := inspectAttachment("file", makeSizedFile(t, "deck.pptx", 38<<20))
	if big.WithinLimit || !strings.Contains(big.Problem, "lark doc upload") || !strings.Contains(big.Problem, "30.0 MiB") {
		t.Fatalf("38 MiB deck should fail with Drive hint: %+v", big)
	}
	img := inspectAttachment("image", makeSizedFile(t, "x.png", 11<<20))
	if img.WithinLimit || img.LimitBytes != api.MaxIMImageBytes {
		t.Fatalf("11 MiB image should exceed the 10 MiB image limit: %+v", img)
	}
	empty := inspectAttachment("file", makeSizedFile(t, "e.txt", 0))
	if empty.WithinLimit || !strings.Contains(empty.Problem, "empty") {
		t.Fatalf("empty file: %+v", empty)
	}
	missing := inspectAttachment("file", filepath.Join(t.TempDir(), "nope.pdf"))
	if missing.Problem == "" || !strings.Contains(missing.Problem, "not found") {
		t.Fatalf("missing file: %+v", missing)
	}
}

func TestFirstAttachmentProblemCodes(t *testing.T) {
	big := makeSizedFile(t, "deck.pptx", 38<<20)
	err := firstAttachmentProblem(inspectSendAttachments([]string{big}, nil))
	var ae *attachmentError
	if !errors.As(err, &ae) || ae.Code != "FILE_TOO_LARGE" {
		t.Fatalf("want FILE_TOO_LARGE, got %v", err)
	}
	err = firstAttachmentProblem(inspectSendAttachments([]string{filepath.Join(t.TempDir(), "nope")}, nil))
	if !errors.As(err, &ae) || ae.Code != "FILE_ERROR" {
		t.Fatalf("want FILE_ERROR, got %v", err)
	}
	if err := firstAttachmentProblem(inspectSendAttachments([]string{makeSizedFile(t, "ok.pdf", 10)}, nil)); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestBuildSendDryRunPersonFile(t *testing.T) {
	dir := newFakeDirectory()
	rcpt, err := resolveSendRecipient(dir, "Oswald Yeo", "")
	if err != nil {
		t.Fatal(err)
	}
	deck := makeSizedFile(t, "deck.pptx", 38<<20)
	r := buildSendDryRun(dir, sendDryRunInput{
		AsUser: true, Recipient: rcpt, ReceiveIDType: "open_id", ReceiveID: "ou_oswald",
		Attachments: inspectSendAttachments([]string{deck}, nil), FileCount: 1,
	})
	if r.OK || len(r.Problems) != 1 || !strings.Contains(r.Problems[0], "lark doc upload") {
		t.Fatalf("oversize deck must make the dry run not OK: %+v", r)
	}
	if r.Identity.As != "user" || r.Identity.OpenID != "ou_self" || r.Identity.Name != "Ying Cong" {
		t.Fatalf("identity: %+v", r.Identity)
	}
	if r.Chat == nil || r.Chat.ChatMode != "p2p" || r.Routing != "direct" {
		t.Fatalf("chat/routing: %+v %s", r.Chat, r.Routing)
	}
	if r.Message.MsgType != "file" || r.Message.Attachments[0].UploadFileType != "ppt" {
		t.Fatalf("message plan: %+v", r.Message)
	}
	// Output must be valid JSON with the documented keys.
	b, _ := json.Marshal(r)
	for _, k := range []string{`"dry_run":true`, `"ok":false`, `"identity"`, `"recipient"`, `"within_limit":false`} {
		if !strings.Contains(string(b), k) {
			t.Errorf("dry-run JSON missing %s: %s", k, b)
		}
	}
}

func TestBuildSendDryRunVerifiedDMRoute(t *testing.T) {
	dir := newFakeDirectory()
	rcpt, _ := resolveSendRecipient(dir, "Oswald Yeo", "")
	r := buildSendDryRun(dir, sendDryRunInput{
		AsUser: true, Recipient: rcpt, ReceiveIDType: "chat_id", ReceiveID: "oc_dm",
		MsgType: "text", Text: "hi",
	})
	if !r.OK || r.Routing != "verified_dm_cache" {
		t.Fatalf("got %+v", r)
	}
	if r.Chat.ChatMode != "p2p" || r.Chat.Counterpart == nil || r.Chat.Counterpart.OpenID != "ou_oswald" || r.Chat.Counterpart.Email != "oswald@example.com" {
		t.Fatalf("p2p counterpart not described: %+v", r.Chat)
	}
}

func TestBuildSendDryRunCounterpartMismatchIsProblem(t *testing.T) {
	dir := newFakeDirectory()
	dir.counterpart["oc_dm"] = [2]string{"ou_someone_else", "Someone"}
	rcpt, _ := resolveSendRecipient(dir, "Oswald Yeo", "")
	r := buildSendDryRun(dir, sendDryRunInput{
		AsUser: true, Recipient: rcpt, ReceiveIDType: "chat_id", ReceiveID: "oc_dm", MsgType: "text", Text: "hi",
	})
	if r.OK {
		t.Fatalf("a DM routed to the wrong person must be flagged: %+v", r)
	}
}

func TestBuildSendDryRunGroupChatAsBot(t *testing.T) {
	dir := newFakeDirectory()
	rcpt, _ := resolveSendRecipient(dir, "oc_group", "")
	r := buildSendDryRun(dir, sendDryRunInput{
		AsUser: false, Recipient: rcpt, ReceiveIDType: "chat_id", ReceiveID: "oc_group",
		MsgType: effectiveTextMsgType("text", "**hi**", 0), Text: "**hi**", ParentID: "om_p",
	})
	if !r.OK || r.Identity.As != "bot" || r.Identity.AppID != "cli_test" {
		t.Fatalf("identity: %+v", r)
	}
	if r.Chat.Name != "Orion" || r.Chat.UserCount != 5 || r.Chat.ChatMode != "group" {
		t.Fatalf("group chat: %+v", r.Chat)
	}
	if r.Message.MsgType != "post" || len(r.Warnings) == 0 {
		t.Fatalf("expected post upgrade and parent-id warning: %+v %v", r.Message, r.Warnings)
	}
}

func TestBuildSendDryRunEmailEnriched(t *testing.T) {
	dir := newFakeDirectory()
	rcpt, _ := resolveSendRecipient(dir, "oswald@example.com", "")
	r := buildSendDryRun(dir, sendDryRunInput{AsUser: true, Recipient: rcpt, ReceiveIDType: "email", ReceiveID: "oswald@example.com", MsgType: "text", Text: "x"})
	if !r.OK || r.Recipient.Person == nil || r.Recipient.Person.OpenID != "ou_oswald" {
		t.Fatalf("email not resolved: %+v", r.Recipient)
	}
	rcpt2, _ := resolveSendRecipient(dir, "ghost@example.com", "")
	r2 := buildSendDryRun(dir, sendDryRunInput{AsUser: true, Recipient: rcpt2, ReceiveIDType: "email", ReceiveID: "ghost@example.com", MsgType: "text", Text: "x"})
	if r2.OK {
		t.Fatal("unknown email must not be OK")
	}
}

func TestFlexIntDecodesStringOrNumber(t *testing.T) {
	var c api.ChatDetail
	if err := json.Unmarshal([]byte(`{"user_count":"7","bot_count":2}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.UserCount != 7 || c.BotCount != 2 {
		t.Fatalf("got %+v", c)
	}
}
