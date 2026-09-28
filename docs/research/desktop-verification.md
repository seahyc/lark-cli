# Desktop capability verification ledger

Verified against Lark 8.0.3 on 2026-09-22. Three Terra agents traced separate
mail, messaging and workspace modules; the lead reviewed the implementations,
corrected unsupported assumptions, re-ran tests, installed the CLI and performed
live checks. **Full desktop parity is not achieved.**

## Installed surface

`desktop operations` lists 27 curated reads. `desktop mutations` lists 12 typed
writes. Dedicated `desktop set-forwarding` and `desktop rule-patch` add two
preserving mail-rule writes. Registration proves a traced schema, not a complete
user workflow. The static catalog separately contains 1,726 SDK and 3,021 legacy
server declarations; none becomes executable merely by appearing there.

## Live read results

| Operation | Evidence and limits |
|---|---|
| mail.account-metadata | Seven mailboxes; selected primary identity matched native user and UI |
| mail.blocked-sender | Known M1 sender returned false; positive membership not tested |
| mail.global-forwarding | Admin enabled; zero global rules, empty case only |
| mail.schedule-status | Count zero, empty case only |
| mail.signatures | Existing signature and usage; populated fixture also read back |
| mail.verified-auto-transfer-emails | Three verified addresses |
| mail.draft-item | Schema/local tests only; no matching native mail draft fixture |
| messaging.drafts | Exact populated text fixture read from native draft entity map; cleanup to zero |
| messaging.chat-draft | Corrected to filter GET_ALL_DRAFTS; exact populated fixture returned |
| messaging.reply-draft | Initial native lookup returned empty; current GET_ALL filter locally tested; populated reply case unverified |
| messaging.message-metadata | Real Favorite message resolved to native message/chat/sender IDs |
| messaging.chat-metadata | Native chat ID authenticated and resolved to a real contact |
| messaging.favorites | One Favorite and pagination returned |
| messaging.favorite-detail | Exact Favorite resolved |
| messaging.shortcuts | One shortcut returned; it was Knowledge AI, not a chat. Projection corrected to structural IDs/types only |
| messaging.feed-folder | Eleven used-folder IDs; folder contents/names not implemented |
| messaging.scheduled-chat | Real authenticated chat queried with force-server strategy; empty case only |
| messaging.scheduled-thread | Schema/local tests only; real thread target not tested |
| workspace.calendar-settings | Asia/Singapore timezone and recommendation setting |
| workspace.todo-setting | Populated personal task settings |
| workspace.user-custom-status | Seven configured status cards; not proof of the currently active status |
| workspace.moments-tabs | One real tab ID |
| workspace.moments-tab-feed | Three real post IDs and a next-page token; no post-body projection |

## Live write results

| Operation | Verification |
|---|---|
| mail.rule-enable | Disposable unique .invalid-sender rule enabled then disabled; native readback matched |
| mail.rule-reorder | Disabled fixture moved to first, original ordering restored; final original 75-rule sequence matched |
| desktop set-forwarding | Changed forwarding on disabled fixture to an already verified recipient; preserved conditions/non-forward actions; fixture deleted |
| desktop rule-patch | Changed fixture name and stop-processing flag; full preserved-rule readback matched; fixture deleted |
| mail.signature-create-text | Unique unused text signature created; native readback and Lark signature-list UI showed it |
| mail.signature-delete | Exact unused fixture removed; native metadata equaled pre-test state, refreshed UI showed absence |
| messaging.create-text-draft | Native storage create/readback passed after correcting the read path; UI composer rendering remains unproven, see below |
| messaging.delete-draft | Exact test ID removed, authoritative all-drafts read returned zero |
| mail.signature-usage-update | Schema and local guards only; actual reassignment not tested |
| mail.sender-block / mail.sender-unblock | Schema only. Unblock adds an allow-list entry. No live mutation without a proven complete cleanup path |
| messaging.set-chat-last-read | Schema only; existing user read positions were not changed as test fixtures |

### Draft correction and destination change

The first implementation used FETCH_CHAT_INPUT_DRAFT for preflight/readback. It
returned empty while GET_ALL_DRAFTS contained the newly created draft. It was
therefore unsafe as an overwrite guard and was replaced with authoritative
all-drafts filtering **before** the 100-item output limit. Matching existing
chat drafts now reject creation. A positive populated fixture and deletion
were verified through this corrected path.

Opening the chat did not display the fixture text in the desktop composer. The
composer showed unrelated local content, which was left untouched. Thus native
storage roundtrip is proven, desktop composer parity is **not**. On user direction,
the destination changed to Oswald, and subsequently to the authenticated **self-chat only**, never Shilin. No message was sent to
Shilin; the temporary native fixture is absent. The user subsequently approved repeated self-chat test messages until verification completes; Oswald is no longer a test destination.

### Legacy sender route rejected

Direct server search first rejected an empty cursor (int64 encoding). After
passing `"0"`, it returned `ErrCacheEmpty`. This is a failed request, not an empty
result. Tracing found discarded legacy wrappers and no proven active UI caller.
The cause remains unresolved. Search/remove descriptors were moved out of
executable registrations into `modules/mail/legacy.js` and a not-ready manifest.
No sender settings were changed during these tests.

## Guards and checks

Writes default to preview, require an explicit account on apply, and recheck
identity inside the native bridge. No automatic mutation retry. Rule-field
patches and forwarding retain fresh complete pre-state and reject stale inputs.
Reordering requires the full current ID set. Signature deletion checks live
assignment state. Draft creation refuses existing drafts; deletion verifies
presence and subsequent absence. Some commands provide native acknowledgement
only; `verified:false` must not be presented as success of the user workflow.

Latest settled-tree verification:

```sh
node --test internal/desktop/modules/*/*.test.cjs internal/desktop/*bridge.test.cjs internal/desktop/server_transport.test.cjs scripts/test_desktop_modules.cjs
# 79 passed, 0 failed
go test -race ./internal/desktop
# passed
go test ./internal/cmd ./internal/api ./internal/mail ./internal/desktop
# passed
go build -o /tmp/lark-private-api-research/lark ./cmd/lark
# passed; installed to /opt/homebrew/bin/lark
```

## Remaining work

The [gap matrix](desktop-cli-gap.md) is still open. Scheduling
writes, inbox mute/folders, richer mail editing, calendar resources, meeting
controls, document editor/transfer state, navigation and mini-app workflows
remain unimplemented or unverified. Research files distinguish missing tracing,
missing runtime context, and observed failures; those are not interchangeable.

## Session and cleanup

Every created mail rule and signature fixture was removed. Native draft fixture
cleanup was read back; unrelated local composer content was preserved. No test
message was sent, meeting joined, camera/microphone enabled, or recording started.
Private response fixtures remain mode 0600 outside the repository under
`/tmp/lark-private-api-research/`. Final archive restoration is recorded below
when the active verification session ends.

## Latest integration

Installed inbox chat-card lookup, text-message preflight, guarded own-text editing,
and `desktop status`. New message operations were subsequently loaded and verified live in the self-chat run below. Edit guards reject
wrong owner/chat/type, stale text, and rich content before writing; readback failure
does not retry. Missing message entities return an empty projection.

`mail server-search --from ebill@m1.com.sg --since 2026-08-01 --limit 3`
passed against authenticated IMAP: six matching UIDs, three envelopes fetched,
truncation explicitly true. No message bodies fetched or mailbox/cache writes.

Temporary archive restored and Lark restarted. Original archive SHA256 verified:
`fd2d495a7d8f4da81334a3695cdc996060aa16f7c20cd529d2c53c683e2f5c8c`.
Port 9330 had no listener after restoration. At that checkpoint no outbound message had been sent. Subsequent authorized self-chat tests are recorded below.

## Self-chat test authorization and public-route finding

The user authorized repeated messages to their own self-chat for this verification
workflow, without per-message approval. No further colleague test sends are in scope.
An initial public-API send addressed to the authenticated user's own open_id
returned success under the user's identity, but its chat history proved it landed
in the user's Glints MCP Access app conversation and triggered an app auto-reply.
It is not evidence of personal self-chat delivery. No colleague received a message.
A dedicated native self-send therefore verifies the destination using
GET_P2P_CHATS_BY_CHATTER_IDS before SEND_MESSAGE and rejects missing/mismatched
self-chat identity. Native send/edit live outcomes are recorded below.

## Native self-chat send and edit: live verified

User authorized repeated self-chat messages for the verification workflow. Native
passport identity was resolved through GET_P2P_CHATS_BY_CHATTER_IDS to the actual
self-chat. `messaging.inbox-chats`, `messaging.p2p-chat`,
`messaging.recent-message-match`, and `messaging.text-message` passed populated
live reads. The personal self-chat differs from the public API's app conversation.

The first direct SEND_MESSAGE failed with `quasi entity not found`; a fresh
server-sync history match found no message. This was corrected using the shipped
CREATE_QUASI_MESSAGE -> same-CID SEND_MESSAGE sequence, without retrying the
failed outbound call. The next native send appeared in the actual self-chat.
Native edit succeeded; initial immediate readback was stale. Separate readback
and the desktop UI both showed the exact edited text. Verification now polls
reads at most four times, 250ms apart, while performing the write only once.

Final installed build:
- `desktop send-self` returned acknowledged=true and verified=true.
- `messaging.edit-text` returned acknowledged=true and verified=true.
- A stale expectedText attempt was rejected; readback proved the intended final
  text remained unchanged.
- 79 Node tests passed; Go cmd/mail/api tests and desktop race tests passed.
- Two clearly labelled native self-chat test messages remain as evidence. One
  earlier public-API test message remains in the user's app conversation, with
  its automated reply. Nothing was sent to Oswald or Shilin.

Full desktop parity is still not achieved. This run verifies plain self-send and
own plain-text edit, not rich editing, scheduled sends, or recipient delivery.

After the self-chat run, the original archive hash was verified again, port 9330 had no listener, and Lark was restarted to unload the temporary bridge. `desktop status` confirms session_present=false. The installed CLI retains 27 reads and 12 mutations plus dedicated rule commands.


## Terra integration pass: 2026-09-22

Parent-run live results (supersede the older 79-test milestone above):

- **Styled self POST:** send with bold/italic/underline returned `verified:true`;
  a prior POST was edited with exact stale-snapshot checks and returned
  `verified:true`. The desktop visibly rendered the title and bold/italic body.
  The reader now handles all 112 fields of the native `basic.v1.Content`
  protobuf union, including map fields and explicit nonzero enum defaults.
- **Recall:** a fresh native self TEXT was sent, recalled once, and the same
  native message returned `isRecalled:true`.
- **Favorites:** one self-test message was added, the exact new Favorite was
  read back, then deleted; readback retained the pre-existing Favorite only.
  The SDK requires `favs:[{id,type,chatId,originMergeForwardId}]`; a high-level
  `messageIds` wrapper payload silently produced no persistent item.
- **Signature fixture:** create, exact text update preserving the full native
  signature, and delete each returned `verified:true`. No signature usage was
  changed and no email was sent.
- **Recipient-free mail draft:** create, metadata read, formatted body/subject
  update, and delete each passed native verification in the final full cycle.
  A prior cycle's subject appeared in Drafts and the list became empty after
  deletion. Metadata reads use `{draftId}`; deletion uses the draft ID as
  `messageId`, not the empty reply-message ID. Post-delete `Normal(404)` is a
  successful absence read; an error from the write itself remains a failure.
- **Read-only desktop state:** device notification booleans, 12-hour clock
  preference, six room-equipment records, and the complete Shell navigation
  layout were read successfully. The layout uses `id/type/movable/visible`,
  rather than the older generated `ID/appType/unmovable` shape. Reordering stays
  disabled pending an independently proven native type mapping.
- **Scheduled self TEXT:** two persistence attempts produced pending items with
  empty/unreadable text. Both were cancelled through authoritative server
  command 900034 and a force-server schedule read returned an empty list.
  No test is left scheduled. Correct nonempty creation remains unverified;
  transport serialization of the encoded TextContent is still being traced.

These are bounded workflow results, not full desktop parity. The full remaining
capability gaps are maintained in `desktop-cli-gap.md`. Live sends in this pass
were exclusively to the authenticated user's personal self-chat.


### Resumed pass: scheduling and mute closed for the tested self-chat scope

- The corrected worklet conversion sends the server `PutMessageRequest.Content`
  object with `richText.elements.dictionary` and encoded TEXT properties, not
  the local `TextContent` byte wrapper. Parent verified the actual installed
  `pb_server.desc` encode/decode boundary independently with `protoc`.
- `desktop schedule-self --apply` returned `verified:true`; `scheduled-self`
  returned the exact nonempty fixture text, native sender, self-chat, future
  time, and pending status. `cancel-scheduled-self --apply` returned
  `verified:true`; the next force-server list was empty. All three scheduling
  fixtures from this pass are cancelled. Future delivery itself was not tested.
- `inbox.chat-mute-state` returned unmuted. The guarded self-chat mutation
  changed it to muted with authoritative readback, then restored unmuted with
  another successful readback. The final independent read returned unmuted.
- The complete settled tree passed **151 JavaScript tests**, desktop Go race
  tests, and Go cmd/mail/api tests. An additional Go test executes the actual
  embedded module closure without CommonJS `require`, verifying that the
  schedule helper is available in the browser bundle.
- Installed CLI now registers **37 curated reads and 27 curated mutations**.
  These counts are not a desktop-parity percentage. Several registered
  operations remain schema-only or intentionally blocked, including navigation
  reorder. Refer to the coverage map rather than treating the registry as a
  list of fully verified features.

Final cleanup: the original mail archive hash matches the pinned original, Lark
was restarted to unload the bridge, session_present=false, and port 9330 has
no listener. All schedule/Favorite/signature/mail-draft fixtures were removed;
self-chat mute was restored. Labelled delivered self-test posts remain as
verification evidence.
