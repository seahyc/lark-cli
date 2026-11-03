package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/yjwong/lark-cli/internal/api"
)

// Recipient resolution for `lark msg send --to`.
//
// Besides raw IDs (ou_/oc_/email/user_id), --to accepts a person's display
// name ("Oswald Yeo") or "me". A name is resolved through the read-only
// contact search API and must identify exactly one person: when several people
// match, resolution fails and lists the candidates rather than guessing, since
// a wrong guess delivers the message to the wrong person.

// recipientDirectory is the read-only Lark surface needed to resolve a --to
// value and describe the destination. It is an interface so tests can run the
// resolution and dry-run logic without network access.
type recipientDirectory interface {
	CurrentUser() (*api.UserInfo, error)
	SearchUsers(query string) ([]api.SearchUserResult, error)
	OpenIDByEmail(email string) (string, error)
	User(openID string) (*api.ContactUser, error)
	DepartmentName(deptID string) string
	Chat(chatID string, asUser bool) (*api.ChatDetail, error)
	P2PCounterpart(chatID string) (openID, name string)
	AppID() string
}

// maxNameSearchResults caps how many search hits a name lookup considers.
const maxNameSearchResults = 100

// maxCandidatesListed caps how many candidates an ambiguity error enriches and
// lists (each one costs a contact lookup).
const maxCandidatesListed = 10

// recipientPerson describes a resolved person recipient.
type recipientPerson struct {
	Name       string `json:"name,omitempty"`
	EnName     string `json:"en_name,omitempty"`
	Email      string `json:"email,omitempty"`
	OpenID     string `json:"open_id,omitempty"`
	Department string `json:"department,omitempty"`
}

// sendRecipient is the outcome of resolving --to.
type sendRecipient struct {
	Input         string           `json:"input"`
	ResolvedBy    string           `json:"resolved_by"` // name, me, email, open_id, chat_id, user_id, bot_open_id
	ReceiveIDType string           `json:"receive_id_type"`
	ReceiveID     string           `json:"receive_id"`
	Person        *recipientPerson `json:"person,omitempty"`
}

// recipientError is a resolution failure; Candidates is set for ambiguous or
// near-miss name lookups so the caller can pick an exact open_id.
type recipientError struct {
	Code       string
	Message    string
	Candidates []recipientPerson
}

func (e *recipientError) Error() string { return e.Message }

var (
	knownIDPrefixes = []string{"ou_", "oc_", "on_", "om_", "cli_"}
	// opaqueIDPattern matches Lark user_id-style tokens (lowercase alnum with at
	// least one digit, e.g. "3e3cf96b"). They keep the legacy open_id/user_id
	// handling instead of being treated as a name.
	opaqueIDPattern = regexp.MustCompile(`^[a-z0-9]*[0-9][a-z0-9]*$`)
)

// looksLikePersonName reports whether an un-typed --to value should be
// resolved as a display name rather than passed through as an ID.
func looksLikePersonName(to string) bool {
	t := strings.TrimSpace(to)
	if t == "" {
		return false
	}
	for _, p := range knownIDPrefixes {
		if strings.HasPrefix(t, p) {
			return false
		}
	}
	if strings.Contains(t, "@") {
		return false
	}
	if _, err := strconv.Atoi(t); err == nil {
		return false
	}
	if opaqueIDPattern.MatchString(t) {
		return false
	}
	return true
}

// isSelfRecipient reports whether --to refers to the authenticated user.
func isSelfRecipient(to, toType string) bool {
	return (toType == "" || toType == "me") && strings.EqualFold(strings.TrimSpace(to), "me")
}

// resolveSendRecipient turns a --to/--to-type pair into a Lark receive target.
// Names and "me" are resolved through dir; raw IDs pass through unchanged
// (callers enrich them separately when they need display details).
func resolveSendRecipient(dir recipientDirectory, to, toType string) (*sendRecipient, error) {
	to = strings.TrimSpace(to)
	switch {
	case isSelfRecipient(to, toType):
		u, err := dir.CurrentUser()
		if err != nil || u == nil || u.OpenID == "" {
			return nil, &recipientError{Code: "RECIPIENT_NOT_FOUND", Message: fmt.Sprintf("could not resolve --to me to your open_id (is `lark auth login` done?): %v", err)}
		}
		return &sendRecipient{
			Input: to, ResolvedBy: "me", ReceiveIDType: "open_id", ReceiveID: u.OpenID,
			Person: &recipientPerson{Name: u.Name, EnName: u.EnName, Email: u.Email, OpenID: u.OpenID},
		}, nil
	case toType == "name" || (toType == "" && looksLikePersonName(to)):
		p, err := resolvePersonByName(dir, to)
		if err != nil {
			return nil, err
		}
		return &sendRecipient{Input: to, ResolvedBy: "name", ReceiveIDType: "open_id", ReceiveID: p.OpenID, Person: p}, nil
	}
	idType := toType
	if idType == "" {
		idType = detectIDType(to)
	}
	return &sendRecipient{Input: to, ResolvedBy: idType, ReceiveIDType: idType, ReceiveID: to}, nil
}

func normalizePersonName(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// nameTokensMatch reports whether every token of query is a prefix of some
// token of name ("os yeo" matches "Oswald Yeo"; "oswald tan" does not).
func nameTokensMatch(query, name string) bool {
	nameTokens := strings.Fields(normalizePersonName(name))
	for _, q := range strings.Fields(normalizePersonName(query)) {
		found := false
		for _, n := range nameTokens {
			if strings.HasPrefix(n, q) {
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

// resolvePersonByName resolves a display name to exactly one person.
//
// Selection, from the contact search hits:
//  1. exactly one hit whose name equals the query (case/space-insensitive) wins;
//  2. otherwise exactly one hit whose name contains every query token wins;
//  3. otherwise fail: AMBIGUOUS_RECIPIENT (several plausible) or
//     RECIPIENT_NOT_FOUND (none plausible), listing candidates.
func resolvePersonByName(dir recipientDirectory, name string) (*recipientPerson, error) {
	query := strings.TrimSpace(name)
	if query == "" {
		return nil, &recipientError{Code: "VALIDATION_ERROR", Message: "--to name is empty"}
	}
	hits, err := dir.SearchUsers(query)
	if err != nil {
		return nil, &recipientError{Code: "API_ERROR", Message: fmt.Sprintf("contact search for %q failed (needs a user token with contact search scope): %v", query, err)}
	}
	seen := map[string]bool{}
	var unique []api.SearchUserResult
	for _, h := range hits {
		if h.OpenID == "" || seen[h.OpenID] {
			continue
		}
		seen[h.OpenID] = true
		unique = append(unique, h)
	}

	var exact, tokenMatches []api.SearchUserResult
	want := normalizePersonName(query)
	for _, h := range unique {
		if normalizePersonName(h.Name) == want {
			exact = append(exact, h)
		}
		if nameTokensMatch(query, h.Name) {
			tokenMatches = append(tokenMatches, h)
		}
	}

	switch {
	case len(exact) == 1:
		return describePerson(dir, exact[0].OpenID, exact[0].Name, exact[0].DepartmentIDs), nil
	case len(exact) == 0 && len(tokenMatches) == 1:
		return describePerson(dir, tokenMatches[0].OpenID, tokenMatches[0].Name, tokenMatches[0].DepartmentIDs), nil
	}

	if len(exact) > 1 || len(tokenMatches) > 1 {
		pool := exact
		if len(pool) == 0 {
			pool = tokenMatches
		}
		return nil, &recipientError{
			Code:       "AMBIGUOUS_RECIPIENT",
			Message:    fmt.Sprintf("%d people match %q; re-run with --to set to the intended person's ou_ open_id or email", len(pool), query),
			Candidates: describeCandidates(dir, pool),
		}
	}
	msg := fmt.Sprintf("no Lark user matches %q; try `lark contact search`, an email, or an ou_ open_id", query)
	if len(unique) > 0 {
		msg = fmt.Sprintf("no Lark user name matches %q exactly; similar people are listed; re-run with --to set to an ou_ open_id or email", query)
	}
	return nil, &recipientError{Code: "RECIPIENT_NOT_FOUND", Message: msg, Candidates: describeCandidates(dir, unique)}
}

func describeCandidates(dir recipientDirectory, hits []api.SearchUserResult) []recipientPerson {
	if len(hits) > maxCandidatesListed {
		hits = hits[:maxCandidatesListed]
	}
	out := make([]recipientPerson, 0, len(hits))
	for _, h := range hits {
		out = append(out, *describePerson(dir, h.OpenID, h.Name, h.DepartmentIDs))
	}
	return out
}

// describePerson fills in email/en_name/department for an open_id. Lookups
// are best-effort: missing scopes leave fields empty rather than failing.
func describePerson(dir recipientDirectory, openID, name string, deptIDs []string) *recipientPerson {
	p := &recipientPerson{Name: name, OpenID: openID}
	if u, err := dir.User(openID); err == nil && u != nil {
		if u.Name != "" {
			p.Name = u.Name
		}
		p.EnName = u.EnName
		p.Email = u.Email
		if p.Email == "" {
			p.Email = u.EnterpriseEmail
		}
		if len(deptIDs) == 0 {
			deptIDs = u.DepartmentIDs
		}
	}
	if len(deptIDs) > 0 {
		p.Department = dir.DepartmentName(deptIDs[0])
	}
	return p
}

// enrichRecipient fills in person details for raw-ID recipients (open_id,
// email) so a dry run can show who an ID belongs to.
func enrichRecipient(dir recipientDirectory, r *sendRecipient) error {
	if r.Person != nil {
		return nil
	}
	switch r.ReceiveIDType {
	case "open_id", "bot_open_id":
		r.Person = describePerson(dir, r.ReceiveID, "", nil)
	case "email":
		openID, err := dir.OpenIDByEmail(r.ReceiveID)
		if err != nil {
			return &recipientError{Code: "API_ERROR", Message: fmt.Sprintf("email lookup for %s failed: %v", r.ReceiveID, err)}
		}
		if openID == "" {
			return &recipientError{Code: "RECIPIENT_NOT_FOUND", Message: fmt.Sprintf("no Lark user has email %s in this tenant", r.ReceiveID)}
		}
		r.Person = describePerson(dir, openID, "", nil)
		if r.Person.Email == "" {
			r.Person.Email = r.ReceiveID
		}
	}
	return nil
}

// apiRecipientDirectory implements recipientDirectory against the Lark API.
type apiRecipientDirectory struct {
	client    *api.Client
	appID     string
	deptCache map[string]string
}

func newAPIRecipientDirectory(client *api.Client, appID string) *apiRecipientDirectory {
	return &apiRecipientDirectory{client: client, appID: appID, deptCache: map[string]string{}}
}

func (d *apiRecipientDirectory) CurrentUser() (*api.UserInfo, error) {
	return d.client.GetCurrentUser()
}

func (d *apiRecipientDirectory) SearchUsers(query string) ([]api.SearchUserResult, error) {
	var all []api.SearchUserResult
	pageToken := ""
	for {
		users, more, next, err := d.client.SearchUsers(query, 50, pageToken)
		if err != nil {
			return nil, err
		}
		all = append(all, users...)
		if !more || next == "" || len(all) >= maxNameSearchResults {
			break
		}
		pageToken = next
	}
	return all, nil
}

func (d *apiRecipientDirectory) OpenIDByEmail(email string) (string, error) {
	users, err := d.client.LookupUsers(api.UserLookupOptions{Emails: []string{email}})
	if err != nil {
		return "", err
	}
	for _, u := range users {
		if strings.EqualFold(u.Email, email) && u.UserID != "" {
			return u.UserID, nil
		}
	}
	return "", nil
}

func (d *apiRecipientDirectory) User(openID string) (*api.ContactUser, error) {
	return d.client.GetUser(openID, "open_id")
}

func (d *apiRecipientDirectory) DepartmentName(deptID string) string {
	if deptID == "" {
		return ""
	}
	if name, ok := d.deptCache[deptID]; ok {
		return name
	}
	name := ""
	if dept, err := d.client.GetDepartment(deptID); err == nil && dept != nil {
		name = dept.Name
	}
	d.deptCache[deptID] = name
	return name
}

func (d *apiRecipientDirectory) Chat(chatID string, asUser bool) (*api.ChatDetail, error) {
	return d.client.GetChat(chatID, asUser)
}

func (d *apiRecipientDirectory) P2PCounterpart(chatID string) (string, string) {
	return counterpartOfP2PChat(d.client, chatID)
}

func (d *apiRecipientDirectory) AppID() string { return d.appID }
