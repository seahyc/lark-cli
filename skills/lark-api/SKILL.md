---
name: lark-api
description: Execute raw HTTP requests against any Lark Open API endpoint via the `lark api` passthrough - use when you need an endpoint that is not covered by the higher-level skills, want to prototype a new call, or need to upload/download files through the API.
---

# Lark Raw API Passthrough Skill

Call any Lark Open API endpoint directly using the `lark api` command. This is the escape hatch for when no dedicated skill covers what you need.

## 🤖 Capabilities and Use Cases

- Call any Lark Open API endpoint (GET, POST, PUT, PATCH, DELETE) with full control over headers and body
- Choose between bot token (`--as bot`) and user token (`--as user`) auth
- Send JSON bodies, query params, and multipart form data (including file uploads)
- Download binary files to disk with `--output`
- Prototype and debug new API integrations before wiring a dedicated command

## 🚀 Quick Reference

**GET with query params:**
```bash
lark api GET /open-apis/im/v1/chats --params '{"page_size":20}' --as user
```

**POST JSON:**
```bash
lark api POST /open-apis/task/v2/tasks --data '{"summary":"Ship it"}' --as user
```

**File upload (multipart):**
```bash
lark api POST /open-apis/im/v1/files \
  --form file=@./report.pdf \
  --form file_type=stream
```

**File download:**
```bash
lark api GET /open-apis/im/v1/messages/om_xxx/resources/file_xxx \
  --output ./download.pdf
```

**DELETE:**
```bash
lark api DELETE /open-apis/im/v1/messages/om_xxx --as bot
```

## Commands Reference

### `lark api <METHOD> <path>`

Executes a raw HTTP request against the Lark Open API.

Available flags:
- `<METHOD>` (positional, required): `GET`, `POST`, `PUT`, `PATCH`, `DELETE`
- `<path>` (positional, required): API path, typically starting with `/open-apis/...`. The host (`https://open.larksuite.com`) is added automatically.
- `--as`: Identity — `bot` (default) uses the tenant access token, `user` uses your user access token
- `--params`: JSON object of query parameters, e.g. `'{"page_size":20}'`
- `--data`: JSON request body as a string, e.g. `'{"summary":"Ship it"}'`
- `--form`: Multipart form field; repeatable. Use `key=value` for text fields or `key=@./path` for file uploads
- `--output`: Save response body to a file path (use for binary downloads)

## Choosing `--as user` vs `--as bot`

| Use `--as user` when... | Use `--as bot` (default) when... |
|---|---|
| The endpoint requires user scopes (email, drive, calendar) | The endpoint requires app/tenant scopes (`tenant_access_token`) |
| You want actions attributed to your user identity | The endpoint explicitly documents `tenant_access_token` |
| Bot identity restrictions block the call (e.g. DMs) | The bot has been added to the target resource |

If a call fails with a scope error, try the other identity first.

## Tips

- Use `lark api` to test a new endpoint before deciding whether to add a dedicated command
- `--params` values must be strings in JSON (e.g. `'{"page_size":"20"}'` also works; numbers are coerced to strings)
- Combining `--data` and `--form` is not supported — a request is either JSON or multipart
- Path should start with `/open-apis/...`; the client prepends the Lark host
- For pagination, look for `page_token` / `has_more` in responses and re-call with `--params '{"page_token":"..."}'`

## Output Format

Responses are returned as JSON on stdout. For binary payloads (file downloads), use `--output` to stream to a file — stdout will then contain a small JSON summary.

## Error Handling

Errors return JSON:
```json
{
  "error": true,
  "code": "ERROR_CODE",
  "message": "Description"
}
```

Common error codes:
- `AUTH_ERROR` — Need to run `lark auth login`
- `SCOPE_ERROR` — Missing the scope for this endpoint. Try the other identity (`--as user` vs `--as bot`), or add the scope with `lark auth login --add --scopes <group>`.
- `API_ERROR` — Lark API returned a non-zero code; the message includes Lark's own error text and request id

## When to Graduate from Raw API to a Skill

Raw API is ideal for prototyping. If you find yourself calling the same endpoint repeatedly, consider whether a dedicated skill already covers it:

- Messages, chats, reactions → `lark msg` / `lark chat` (see `lark-messages` skill)
- Tasks → `lark task` (see `lark-tasks` skill)
- Bitable databases → `lark bitable` (see `lark-bitable` skill)
- Docs, sheets, wiki → `lark doc` / `lark sheet` / `lark wiki`
- Calendar, contacts, mail → `lark cal` / `lark contact` / `lark mail`
- Meetings, minutes → `lark meetings` / `lark minutes`
- Event subscriptions → `lark events`

Use the raw `lark api` when the dedicated command doesn't yet expose what you need.

For a cross-surface lookup, use the public API aggregator before the native
desktop bridge. It works while Lark Desktop is closed and reports per-surface
scope errors without discarding other successful results:

```sh
lark find 'quarterly planning'
lark find 'incident' --type messages,docs --limit 10
```

For ordinary public `om_...` messages, prefer the user-token commands; they do
not require Lark Desktop:

```sh
lark msg edit --message-id om_xxx --text 'Updated text'       # --as user is default
lark msg recall om_xxx                                        # --as user is default
```

These routes cannot address native numeric personal-self-chat message IDs.

## Desktop-only capabilities

When the public API cannot perform a desktop action, check the installed CLI's
curated native operations before falling back to manual UI work:

```sh
lark desktop operations       # Typed reads and their inputs
lark desktop mutations        # Typed writes and their effects
lark desktop catalog --search SIGNATURE --details  # Research declarations only
```

The catalog is not executable coverage. Use only registered operations whose
schema and live verification fit the requested action. Current implementation
and evidence live in `/Users/yingcong/Code/lark-cli/docs/research/desktop-verification.md`.

A native session uses a temporary, version-pinned patch to the desktop Email
surface. `lark desktop start` installs it; restart Lark and open Email to load it.
Use `desktop identity` to check the native account. `desktop restore` restores
the original archive; restart unloads the in-memory bridge immediately, or its
30-minute expiry ends it. Do not expose the bridge beyond loopback or export
session credentials. Reuse existing authorization to patch/restart when given;
otherwise obtain it before interrupting the desktop.

```sh
lark desktop run OPERATION --input '{"documentedField":"value"}'
lark desktop mutate OPERATION --input '{"documentedField":"value"}' # preview
# Only after reviewing the exact change and its identity:
lark desktop mutate OPERATION --input '{"documentedField":"value"}' --apply --expect-user NATIVE_USER_ID
```

Native IDs are not interchangeable with public `oc_`/`om_` IDs. Shortcut channel
IDs can identify apps such as Knowledge AI, so they are not necessarily chat IDs.
A resolved native mutation is an acknowledgement; only a matching subsequent
readback establishes the new state. Do not retry an uncertain write blindly.
Outbound send/edits/scheduling follow the applicable communication rules. A
user-authorized bounded test conversation remains authorized until its stopping
condition; do not ask again for each in-scope test message.

### Native self-chat messaging

Do not treat a public `msg send --to <own open_id> --as user` as proof of personal
self-chat delivery: on the verified installation it routes into the user's app
conversation. Use the native self-chat resolver and guarded command instead:

```sh
lark desktop send-self --text 'Test text' # resolves native self-chat and previews
lark desktop send-self --text 'Test text' --apply --expect-user NATIVE_USER_ID
lark desktop run messaging.text-message --input '{"messageId":"NATIVE_MESSAGE_ID"}'
lark desktop mutate messaging.edit-text --input '{"messageId":"NATIVE_MESSAGE_ID","chatId":"NATIVE_CHAT_ID","expectedText":"Test text","text":"Edited test text"}' --apply --expect-user NATIVE_USER_ID
```

Self-send uses CREATE_QUASI_MESSAGE followed by SEND_MESSAGE with the same CID.
Never call SEND_MESSAGE directly with a fabricated CID: the pending local entity
must exist first. The bridge performs bounded read-only polling for delayed
readback, never retries the outbound write, and checks owner/chat/current text
before editing. The plain-text edit path excludes rich messages, mentions, and
attachments.

The verified self-chat surface also supports styled text posts, guarded recall,
and far-future plain-text scheduling. Inspect the operation schema before
constructing JSON; POST payloads are normalized JSON strings and support only a
title plus styled text runs (bold, italic, and underline).

```sh
lark desktop mutate messaging.send-self-post --input '{...}'
lark desktop run messaging.post-message --input '{"messageId":"NATIVE_MESSAGE_ID"}'
lark desktop mutate messaging.edit-post --input '{...}'
lark desktop mutate messaging.recall-self-text --input '{...}'

lark desktop schedule-self --text 'Scheduled test' --at 'RFC3339_TIME'
lark desktop scheduled-self
lark desktop cancel-scheduled-self NATIVE_MESSAGE_ID
```

All mutations preview by default. Apply with `--apply --expect-user
NATIVE_USER_ID`, then use the corresponding read to prove the result. A
scheduled-message acknowledgement is not delivery proof; for fixture tests,
read the pending item back and cancel it before it becomes due.

### Favorites and self-chat mute state

Favorites and notification state are covered for tightly scoped self-chat
fixtures:

```sh
lark desktop run messaging.favorites --input '{"count":15,"time":0}'
lark desktop mutate triage.favorite-add --input '{"messageId":"NATIVE_MESSAGE_ID","chatId":"NATIVE_CHAT_ID"}'
lark desktop mutate triage.favorite-delete --input '{"favoriteId":"NATIVE_FAVORITE_ID"}'

lark desktop run inbox.chat-mute-state --input '{"chatId":"NATIVE_CHAT_ID"}'
lark desktop mutate inbox.set-self-chat-mute --input '{...}'
```

For Favorites, diff the before/after list to identify only the fixture created
by the test, and delete only that Favorite. For mute tests, capture the original
state, apply the requested state, read it back, restore the original value, and
read it back again.

### Recipient-free mail fixtures

The native bridge can create, inspect, update, and delete recipient-free draft
fixtures, plus create/update/delete a temporary plain-text signature:

```sh
lark desktop mutate mailnext.draft-fixture-create --input '{...}'
lark desktop run mailnext.draft-fixture-metadata --input '{"draftId":"NATIVE_DRAFT_ID"}'
lark desktop mutate mailnext.draft-fixture-update --input '{...}'
lark desktop mutate mailnext.draft-fixture-delete --input '{...}'

lark desktop mutate mail.signature-create-text --input '{...}'
lark desktop mutate mailnext.signature-fixture-update-text --input '{...}'
lark desktop mutate mail.signature-delete --input '{...}'
```

These are fixture operations, not general mail composition. Drafts exclude
recipients and attachments. Signature update/delete require the exact current
fixture snapshot. Always delete the temporary object and verify its absence.

### Desktop state reads

The curated registry includes bounded reads that the public CLI did not expose:

```sh
lark desktop run workspacenext.navigation-layout --input '{}'
lark desktop run workspacenext.device-notify-preferences --input '{}'
lark desktop run workspacenext.time-format-preference --input '{}'
lark desktop run workspacenext.calendar-resource-equipments --input '{}'
```

Treat `verification: schema-verified` as protocol evidence, not proof that the
operation succeeded on the current client/account. Check the live-status notes
in `docs/research/desktop-verification.md`. In particular, do not use
`workspacenext.modify-navigation-order` until the navigation app-type mapping is
live-verified; a full-order write can corrupt the user's layout.

Registry counts are discovery metadata, not a parity claim. Enumerate the
installed version with `desktop operations` and `desktop mutations`. Consult
`/Users/yingcong/Code/lark-cli/docs/research/desktop-cli-gap.md` for remaining
desktop gaps and `/Users/yingcong/Code/lark-cli/docs/desktop-bridge.md` for the
session and threat model.
