# Mail search protocol research

Research source: extracted Lark 8.0.3 mail static JavaScript under
`/tmp/lark-private-api-research/mail`. This is static analysis only. No desktop
request, profile, cache, or mailbox was touched.

## The two search transports are selected by runtime state

The shipped search API contains both a native SDK wrapper and a legacy server
wrapper:

```text
mail.asar/common/6b0f43cfaa.js @2379, module 198366
m = Vy(n.E1c, {
  transform: (e,t) => ({
    keyword:e.keyword, offset:e.nextBegin||"0", searchSession:e.sid,
    isOffline:e.isOffline, ...e.searchFilter, ...Gg(e.searchType),
    ...e.relatedMailSearch, searchSortRule:e.searchSortRule
  })
})
f = eH(l.gY6, {
  transform: e => ({
    keyword, searchSession:sid, isFolderEnabled,
    isConversationModeEnable, ...searchFilter, ...Gg(searchType),
    ...relatedMailSearch, searchSortRule
  })
})
```

`n.E1c` is the native SDK wire command:

```text
3687|email.client.v1.MailAdvancedSearchRequest|
email.client.v1.MailAdvancedSearchResponse|1|MAIL_ADVANCED_SEARCH_REQUEST
```

`l.gY6` is the legacy server wire:

```text
3658|mails.MailAdvancedSearchRequest|mails.MailAdvancedSearchResponse
```

The caller chooses the native wrapper only for a tripartite client or when the
runtime computes `isOffline`; otherwise it chooses the legacy server wrapper:

```text
mail.asar/common/6b0f43cfaa.js @~4800
if (tripartiteClient || e.isOffline) {
  return T({keyword, sid, searchFilter, nextBegin, isOffline, searchType,
            isFolderEnabled, searchRangeLimit:false, searchSortRule}, labels)
}
return S({keyword, sid, searchFilter, isConversationModeEnable,
          isFolderEnabled, relatedMailSearch, searchType,
          searchRangeLimit, searchSortRule}, labels)
```

The `isOffline` decision is not a caller-selected simple-query default. In the
state thunk it depends on feature flags and runtime client/search state:

```text
mail.asar/common/ae001966e4.js @77464
L = tripartiteClient ? clientOffline : configuredOffline;
fetchSearchList invokes Hz({isOffline:L, keyword, ...})
```

A CLI must not set that flag merely to force the native command. Doing so would
change the desktop-selected search backend. This is the current public-CLI
cache-vs-server search gap: the native request schema is exact, but a simple
search needs the desktop's selected backend context before it is runnable.

## Proven simple-query construction

For the native branch, the empty filter constructor is exact:

```text
mail.asar/common/6b0f43cfaa.js @1669, module 868342
N5() => {
  fromList:[], toList:[], subjectList:[], folder:"", label:"",
  startDate:"0", endDate:"0", excludedKeywordList:[],
  hasAttachment:false, priorityType:UNKNOWN_PRIORITY
}

mail.asar/chunk-common/835b3b6848.js @19835
UNKNOWN_SEARCH_SORT_RULE:0

mail.asar/chunk-common/835b3b6848.js @21909
searchScene enum: Default:0, SearchTrashAndSpam:1, DefaultVersion2:2

mail.asar/common/41b8ba5175.js @4701
Gg(searchType) => {searchScene: searchType is relatedTrashAndSpam or
trashAndSpam ? SearchTrashAndSpam : DefaultVersion2}
```

Therefore the static native payload for a first-page, normal text query is:

```json
{
  "keyword": "<non-empty text>",
  "offset": "0",
  "searchSession": "",
  "isOffline": "<desktop-derived boolean; not a CLI default>",
  "fromList": [],
  "toList": [],
  "subjectList": [],
  "folder": "",
  "label": "",
  "startDate": "0",
  "endDate": "0",
  "excludedKeywordList": [],
  "hasAttachment": false,
  "priorityType": 0,
  "searchScene": 2,
  "searchSortRule": 0
}
```

`offset` and `searchSession` are returned pagination/session state and must be
fed back only from the immediately previous result. A future CLI descriptor
should bound `keyword` (for example 1–256 characters), use a fixed first-page
payload, and expose a separate continuation descriptor that requires the
returned strings. It must not accept a generic search-filter JSON object.

## Result fields that support a bounded list

The native result wrapper maps `msgSummary` with `pz`:

```text
mail.asar/common/6b0f43cfaa.js @2379
threads: (e.msgSummary ?? []).map(a => pz(a, {
  searchType, isFolderEnabled, labels, searchSession:e.searchSession||""
}))

mail.asar/common/41b8ba5175.js @4701, pz
{threadId:e.threadId||"", messageId:e.messageId||"",
 subject:e.subjectSummary||"", timestamp:Number(e.lastMessageTimestamp),
 messageCount:Number(e.messageCount), hasAttachment:!!e.hasAttachment,
 ...}
```

A privacy-bounded projection can return at most 50 values of
`{threadId,messageId,subject,timestamp,messageCount,hasAttachment}` plus
`hasMore`, `searchSession`, and `nextBegin`. Do not return body summaries,
addresses, recipient lists, labels, or attachment names. The static mapper
proves those fields exist but it does not establish that a direct CLI call has
the correct runtime `isOffline` context.

## Status

Do not register a native or server search operation yet. The exact request and
response wrapper paths are known, but the runtime chooses native versus legacy
server search from desktop state. The previous legacy sender-list research also
observed `ErrCacheEmpty`; it does not prove this distinct legacy search command
will fail, but it reinforces that direct server calls need a proven active UI
route and context. No live search was attempted here.

## Existing public IMAP path: local search only

The existing `lark mail search` command is not a server-side IMAP query. Its
user-facing description is explicit:

```text
internal/cmd/mail.go @274
Use: "search"
Short: "Search emails in local cache"
Long: "Search cached email metadata locally (no network calls)."
```

It parses `--from`, `--subject`, `--body`, `--since`, `--before`, and
`--limit`, then calls `mail.Search`:

```text
internal/cmd/mail.go @290
mail.ParseSearchOptions(...)
mail.Search(mailSearchMailbox, opts)

internal/mail/search.go @7
Search(mailbox, opts) opens Cache and returns cache.Search(mailbox, opts)
```

`Cache.Search` is SQLite only. It queries the cached `envelopes` table and
joins `message_bodies` only when `--body` is supplied:

```text
internal/mail/cache.go @338
SELECT ... FROM envelopes e
[JOIN message_bodies mb ... only when opts.Body != ""]
WHERE e.mailbox = ?
AND e.from_addr LIKE ?
AND e.subject LIKE ?
AND CAST(mb.body AS TEXT) LIKE ?
ORDER BY e.date DESC LIMIT <limit>
```

The cache is populated by `lark mail sync`. Sync connects to IMAP, selects one
mailbox, runs a UID search only to enumerate *all* UIDs, and fetches missing
envelopes into SQLite:

```text
internal/mail/sync.go @46
client := Connect(); client.SelectMailbox(mailbox)
serverUIDs, err := client.GetAllUIDs()

internal/mail/client.go @199
GetAllUIDs creates SearchCriteria{UID: [1:*]}
and calls c.imap.UIDSearch(criteria, nil)
```

Thus existing IMAP support is a real server read during sync, but it does not
perform a server-side text, sender, subject, or date query. `mail search` can
be stale, is limited to a previously synced mailbox, and body matches require
previous `sync --include-bodies`. It does not avoid the native desktop search
backend choice for fresh mailbox-wide query results.

## Bounded IMAP server-search implementation path

The checked-in IMAP dependency supports the needed standard criteria:

```text
$GOPATH/pkg/mod/github.com/emersion/go-imap/v2@v2.0.0-beta.4/search.go @38
SearchCriteria has SentSince, SentBefore,
Header []SearchCriteriaHeaderField, Body []string, and Text []string.

.../imapclient/search.go @81
UIDSearch(criteria, options) sends a UID SEARCH command.

.../imapclient/search.go @197
Header fields FROM and SUBJECT are encoded as their standard IMAP search keys.
```

A narrowly scoped public alternative could add a distinct `lark mail
server-search` command, rather than changing the meaning of existing local
`mail search`:

1. Require exactly one text predicate and bound it to 1–256 UTF-8 characters:
   `--from` maps to `Header{Key:"From"}`, `--subject` maps to
   `Header{Key:"Subject"}`, or `--body` maps to `Body`.
2. Validate `--since`/`--before` as dates and map them to `SentSince` and
   `SentBefore`, matching the existing cache's envelope-date semantics.
3. Select exactly one requested mailbox, call `UIDSearch`, retain at most the
   newest 50 returned UIDs, then call the existing `FetchEnvelopesByUID`.
   Sort the returned envelopes by date descending and return only
   `{uid,message_id,date,from_addr,from_name,subject}`.
4. Report that `UID SEARCH` itself can return an unbounded UID set: standard
   IMAP SEARCH has no portable result-limit field. The CLI can bound follow-up
   envelope fetch/output, but server response size still depends on the query.
   A capability check for ESEARCH/PARTIAL would be required before claiming a
   fully bounded server query.

This would be an IMAP-equivalent for simple sender/subject/body/date searches
when the user has separately configured IMAP credentials. It would not be
identical to the desktop search: the desktop supports richer labels, folders,
related-message searches, and Lark-specific search scenes. No IMAP connection,
credential read, or live request was made in this research.
