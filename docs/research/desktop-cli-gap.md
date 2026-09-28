# Lark desktop / CLI coverage map

Inspected locally on 2026-09-22: installed Lark 8.0.3 (Chromium 147.0.7727.149),
current Go command source, and the installed `/opt/homebrew/bin/lark`.

## What the evidence establishes

The static inventory contains **1,726 SDK command declarations across
36 SDK namespaces**, plus **3,021 legacy server command declarations**. It scans JavaScript inside top-level installed ASAR archives.
It does not cover every native C++ feature, dynamically downloaded mini-app,
web-hosted editor, tenant feature flag, or UI interaction. This is a full inventory
of the declarations matched by that scan, not proof of complete desktop parity.

A declaration is a research lead. Calling it successfully also requires its
request schema, correct account/container context, permissions, and sometimes an
active native UI session. Names beginning with GET are not automatically safe
reads: some return credentials or initiate work.

Missing a dedicated command does **not** mean missing from Lark's public APIs.
`lark api` can reach supported endpoints with appropriate scopes. Prefer that
route when it provides the same user-visible behavior. Use the native bridge
for confirmed public-API limitations or desktop session/local-state features.

## Product capability matrix

“Candidate gap” below means absent from the dedicated CLI command surface plus
a matching shipped native declaration. Except where noted, request schemas and
live behavior still need verification. The command names are searchable through
`lark desktop catalog --search NAME --details`.

| Area | Existing CLI coverage | Candidate gaps / native evidence | Route and status |
|---|---|---|---|
| Ordinary chat messages | History, global search, get, send/reply, forward/merge-forward, recall, attachments, reactions | No blanket gap for these | Existing CLI; identity and scope limits still apply |
| Edit own user messages | `msg edit` is documented as bot-owned only | `EDIT_MESSAGE` | Native candidate; own-user edits need sender checks and approved exact content |
| Chat composer drafts | No dedicated draft commands | `FETCH_CHAT_INPUT_DRAFT`, `FETCH_MESSAGE_REPLY_DRAFT`, `CREATE_DRAFT`, `GET_ALL_DRAFTS`, `DELETE_DRAFT` | Native candidate, separate draft edits from sending |
| Scheduled chat messages | No dedicated chat scheduling commands | `GET_SCHEDULE_MESSAGES`, `PATCH_SCHEDULE_MESSAGE` | Native candidate; scheduling counts as an outbound send |
| Read/unread and inbox triage | Reading history is supported; changing feed read state is not a dedicated command | `CREATE_CHAT_LAST_READ_POSITION`, `UPDATE_FEED_CARD_UNREAD`, `BATCH_CLEAR_FEED_BADGE`, `IGNORE_ALL_UNREAD` | Native candidate; changes sync to the user's inbox |
| Sidebar / feed organization | Chat lookup and recent DMs only | Feed folders/groups/filters, shortcuts, mute, delayed cards, flags: `GET_FEED_GROUP_LIST`, `GET_FEED_FOLDER`, `UPDATE_FEED_FOLDER`, `BATCH_MUTE_FEED_CARDS`, `SET_FEED_CARD_DELAYED`, `GET_SHORTCUTS` | Native candidate |
| Saved items / Favorites | No dedicated Favorites command | `GET_FAVORITES`, `GET_FAVORITE_INFO`, `CREATE_FAVORITES`, `DELETE_FAVORITES` | Native candidate; Favorites are distinct from existing message pins |
| Chat/group administration | Create/update/disband, member management, pins, join link | Chat tabs, URL/announcement pins, group-specific controls: `ADD_CHAT_TAB`, `CREATE_URL_CHAT_PIN`, `CREATE_ANNOUNCEMENT_CHAT_PIN` | First inspect public API, then native if needed |
| Message translation / media assistance | Resources can be downloaded | Translation preferences/results, speech recognition, stickers: `TRANSLATE_MESSAGES_V3`, `GET_AUDIO_MESSAGE_RECOGNITION`, `GET_USED_REACTIONS`, `CREATE_CUSTOMIZED_STICKERS` | Native candidate; translations may trigger service-side processing |
| Mail basics | IMAP sync/search/read/fetch/drafts, SMTP send/reply, move/delete, folders | Native server-side search and richer native draft/thread state beyond cached IMAP view: `MAIL_ADVANCED_SEARCH_REQUEST`, `MAIL_GET_DRAFT_ITEM` | Native candidate; avoid duplicating existing mail basics |
| Scheduled mail | `mail schedule draft/update/send/cancel` exists in current source and installed build | Native count/list/status visibility and remaining scheduling semantics | Extend existing commands; not a missing scheduling feature wholesale |
| Mail filter forwarding | Public filter list/create/delete; public API rejected forwarding type 12 in live tests | Native forwarding updates, verification state | Bridge reads and same-state apply live-verified; changed write only unit-tested |
| Mail rule management | Basic public rule CRUD, native set-forwarding | General rule update, enable, reorder, preview, apply to existing mail: `MAIL_UPDATE_RULE`, `MAIL_ENABLE_RULE`, `MAIL_ADJUST_RULE_ORDER`, `MAIL_PREVIEW_RULE`, `MAIL_APPLY_RULE` | Exact native wrappers found; retroactive apply needs explicit bounded scope |
| Whole-mailbox forwarding | No dedicated native command | Global transfer rules and verification workflow | Need exact global-rule schema; must not confuse with per-filter forwarding |
| Mail signatures / preferences | No dedicated commands | `MAIL_GET_SIGNATURE`, `MAIL_ADD_SIGNATURE`, `MAIL_UPDATE_SIGNATURE`, `MAIL_UPDATE_SIGNATURE_USAGE` plus settings/account metadata | Native candidate; reads first |
| Mail sender blocking / labels | Folder commands exist; no dedicated native block/label controls | `MAIL_GET_BLOCKED_ADDRESSES`, `MAIL_ADD_USER_ALLOW_BLOCK`, `MAIL_ADD_LABEL_REQUEST` | Inspect public API first; native schema required |
| Calendar fundamentals | Event CRUD/search/list, RSVP, attendees, availability, conflict detection | No blanket gap for these | Existing CLI |
| Calendar desktop views / resources | Basic event locations/reminders | Calendar settings, colors, time zones, room/building/equipment discovery, room views, check-in, time blocks: `GET_CALENDAR_SETTINGS`, `GET_BUILDINGS`, `GET_RESOURCE_EQUIPMENTS`, `GET_TIME_BLOCKS_WITH_TIME_RANGE` | Mix of public API candidates and native UI state |
| Live meetings | History, metadata, notes, recordings, transcripts | Join/status, participants, lobby/host controls, live subtitles, recording, device settings: `JOIN_MEETING`, `GET_PARTICIPANT_LIST`, `HOST_MANAGE`, `PULL_SUBTITLES`, `RECORD_MEETING` | Native/RTC candidate; host permissions and actual device media state remain necessary |
| Meeting media UI | No generic native media automation | Camera/mic routing, screen-share picker, whiteboard, remote control | Protocol may cover part; OS/device UI interaction remains separate |
| Tasks | Task list/get/create/update/complete/reopen, assignees/followers, subtasks, comments, reminders and tasklists | Chat-derived tasks, comment drafts/history, activity and custom views/fields: `GET_CHAT_TODOS`, `GET_TODO_HISTORY_RECORDS`, `GET_TODO_COMMENT_DRAFT`, `UPDATE_TASK_VIEW` | Public API coverage check first |
| Documents / Drive / Wiki | Broad document blocks, editing, search, comments, upload/download, file operations, wiki navigation | Local offline/cache state, editor-specific state, native transfer queue pause/resume/progress, feed state: `SUITE_DRIVE_TASK_GET_DOWNLOAD_PROCESS`, `SUITE_DRIVE_TASK_PAUSE_DOWNLOAD_TASK`, `GET_DOCS_HISTORY` | Some public API, some native-only; scan does not cover whole web editor |
| Sheets / Slides / Bitable | Existing dedicated wrappers | Full desktop editor fidelity, specialized blocks/views and live UI state | Separate web-editor capability audit required; absence from native catalog proves nothing |
| Contacts / profile | Directory lookup, departments, search | User custom status, aliases/notes, Favorites/focus contacts, external relationship flows: `GET_USER_CUSTOM_STATUS`, `UPDATE_USER_CUSTOM_STATUS`, `SET_CHATTER_ALIAS`, `PATCH_USER_PROFILE_MEMO`, `GET_EXTERNAL_CONTACT_LIST` | Public API where supported; native candidates otherwise |
| Personal app preferences | CLI OAuth settings only | Notifications, translation, navigation ordering, user preferences: `GET_USER_SETTINGS`, `GET_DEVICE_NOTIFY_SETTING`, `MODIFY_NAVIGATION_ORDER`, `SET_TRANSLATE_LANGUAGES_SETTING` | Native candidate; typed individual settings rather than arbitrary blobs |
| Global desktop search | `lark find` aggregates public message, document, people, and chat search without Lark Desktop | Local cached search, search history, recommendations: `UNIVERSAL_SEARCH`, `UNIFORM_LOCAL_SEARCH`, `GET_SEARCH_INFO_HISTORY` | Main cross-surface workflow implemented through public APIs; desktop-local/history semantics remain native candidates |
| Moments / company feed | No dedicated commands | Posts, comments, categories, feed, reactions, follows: `MOMENTS_GET_TAB_FEED`, `MOMENTS_GET_POST_DETAIL`, `MOMENTS_CREATE_POST` | Native candidate; read surfaces first, publishing remains approved send |
| Workplace apps / cards | AnyCross, approvals, attendance already have wrappers | Navigation, app badges, interactive card actions, tenant-specific mini-app workflows: `GET_NAVIGATION_APPS`, `GET_APP_DETAIL`, `PUT_UNIVERSAL_CARD_ACTION_REQUEST` | Each app/card workflow needs its own schema and permissions; no universal action executor |
| Account/security/admin | CLI OAuth login/status/refresh/scopes | Native device/security/admin declarations exist | Not ordinary parity work: no credential export, arbitrary token getters, or bypass of server/admin controls |

## Current implementation progress

This map records a selective desktop bridge implementation. It does not claim
full desktop parity, and a command declaration or generated protobuf schema is
not counted as implemented behavior. The [verification ledger](desktop-verification.md)
contains the authoritative parent-run live observations, failures, cleanup, and
remaining owner checks.

### Live and UI evidence obtained

- Signature fixture create, edit, and delete passed live verification.
- Mail draft create, read, update, and delete passed native verification; the
  saved subject and subsequent absence were also observed in the desktop.
- Styled-text POST send and edit passed exact native readback; bold and italic
  rendered in the UI. Native self-message recall also passed.
- Device notification preferences, time-format preference, and calendar
  resource-equipment reads passed live verification.
- Native self-send and own plain-text editing have live evidence with guarded
  readback; see the ledger for their identity, freshness, and retry limits.

### Additional bounded live results

- Scheduled self TEXT create/read/cancel passed live with the exact nonempty
  content. Every schedule fixture was cancelled; future delivery was not tested.
- Self-chat mute was changed, read back, and restored to the original unmuted
  state with a final independent read.
- `workspacenext.navigation-layout` has a live-observed v2 projection of the
  ShellAPI response, and its latest complete projection passed a live read. It deliberately does not map native navigation `type`
  values to settings reorder enums.
- Favorites add and delete passed exact native readback while preserving the
  pre-existing saved item.

### Still-open product gaps

The matrix above remains a list of real capability gaps rather than an
implementation backlog. In particular, active-meeting/media controls,
meeting-specific participant/status workflows, full task drafts/history and
chat-linked tasks, Drive transfer queues, unified/local search history,
Moments post/comment/feed workflows, whole-mailbox forwarding, complete
navigation ordering, editor-local state, and tenant-specific Workplace cards
still require a grounded schema and a reachable live workflow. Broad user
settings, credentials, security/admin controls, arbitrary card actions, and
raw local-state blobs remain intentionally excluded.


## Implementation stages and acceptance criteria

1. **Inventory and usable session lifecycle.** Searchable catalog; capability
   status explicit; hash-pinned archive patch; loopback authentication; restore
   original resources; no credential extraction. Catalog is implemented.
2. **Finish mail.** Typed settings/signature/blocklist reads; rule updates,
   enable/reorder; whole-mailbox forwarding and verification. Verify against UI
   and public API where possible. Test writes on a disabled, impossible-match
   fixture or an explicitly requested real change, then clean up.
3. **Messaging and triage.** Drafts, scheduled messages, Favorites, read/unread,
   mute, feed organization, own-user message edits. Preserve account/chat/message
   identity and present exact outbound content before sends.
4. **Calendar/tasks/profile/preferences/search.** Prefer public API wrappers;
   native bridge only where necessary. Test persistent settings with before/after
   readback and rollback. Document pagination and cache freshness.
5. **Live meetings, editors, Moments, Workplace.** Session/device-dependent work;
   validate one reachable workflow at a time. Active calls, recording, screen
   sharing, publishing, and admin/security actions are not safe generic probes.

A capability is complete only after its request and response schema are mapped,
identity/authorization behavior is established, errors and pagination are handled,
its specific UI-visible result is live-verified, and any fixture/patch is cleaned
up. A matched command name, mock test, or HTTP success alone is insufficient.

## Reproduce the inventory

```sh
python3 scripts/inventory_desktop.py --output internal/desktop/catalog.json
lark desktop catalog
lark desktop catalog --domain feed
lark desktop catalog --search SIGNATURE --details
```

Sources: Go commands under `internal/cmd/`; exact installed ASAR paths, inner JS
paths, and byte offsets in `internal/desktop/catalog.json`; native-mail wrapper
research in `docs/research/lark-desktop-mail-protocol.md`.

Native self-send and own plain-text editing are now live-verified; see the verification ledger for limits and the public self-recipient routing finding.
