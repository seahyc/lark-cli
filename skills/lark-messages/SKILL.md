---
name: messages
description: Chat and message operations in Lark - send/read/reply/recall/edit messages, react, download resources, and manage groups (create chats, add/remove members, pin/unpin messages, get share links, search messages across chats, forward or merge-forward messages). Use when the user asks about chat messages, conversation history, sending/searching messages, or managing Lark groups.
---

# Messages Skill

Full coverage for Lark chat + message operations via the `lark` CLI: send, read, react, edit, recall, search, forward, group-chat management, and resource downloads.

## 🤖 Capabilities and Use Cases

- Send markdown-lite messages with links and mentions
- **Send as user** (`--as user`, default) or as bot (`--as bot`)
- Send images with `--image` and `{{image}}` placement
- Send files (PDF, PPTX, DOCX, XLSX, etc.) with `--file`
- Reply in threads with `--parent-id` / `--root-id`
- Edit and recall sent messages
- Add/list/remove reactions; browse the emoji catalog
- Read chat or thread history
- **Search messages across chats** (`msg search`) with filters on chat, sender, time, type
- **Forward** or **merge-forward** messages to another recipient
- Download resources (images/files/audio/video) from messages
- **Group chat management**: create chats, update name/description, add/remove members, pin/unpin messages, get share links, disband chats
- Find chats by name/member or list all visible chats (`chat search`)
- Enumerate recent direct-message (P2P) chats (`chat list-dms`)

## 🚀 Quick Reference

**Send / reply:**
```bash
lark msg send --to user@example.com --text "Hello!"
lark msg send --to oc_12345 --parent-id om_abcdef --msg-type text --text "Replying here"
lark msg send --to oc_12345 --file ./report.pdf
```

**Read / search / forward:**
```bash
lark dm "Alice" --last 20
lark dm "Alice" --reply "On it" --compact
lark msg history --chat-id oc_12345 --limit 10
lark msg search "incident" --chat-id oc_xxx --limit 20
lark msg forward --message-id om_xxx --to oc_yyy
lark msg merge-forward --message-ids om_a,om_b,om_c --to oc_yyy
```

**Edit / recall / react:**
```bash
lark msg edit <message-id> --text "Fixed typo"
lark msg recall om_dc13264520392913993dd051dba21dcf
lark msg react --message-id om_xxx --reaction SMILE
```

**Group chat management:**
```bash
lark chat search "project team"
lark chat list-dms --limit 100                                         # enumerate recent P2P chats
lark chat dm-cache                                                     # inspect person -> chat_id cache
lark chat dm-cache --verify                                            # verify all entries, drop bad ones
lark chat create --name "Launch Room" --members alice@x,bob@x
lark chat create --name "1:1 w/ Alice" --to alice@x                    # 1:1 shortcut
lark chat update <chat-id> --name "Launch Room 2" --description "Updated"
lark chat member add <chat-id> --members carol@x
lark chat member remove <chat-id> --members ou_xxx
lark chat pin <chat-id> --message-id om_xxx
lark chat pins <chat-id>
lark chat unpin <chat-id> --message-id om_xxx
lark chat link <chat-id>
lark chat delete <chat-id>                                             # disband (owner only)
```

## Running Commands

Ensure `lark` is in your PATH, or use the full path to the binary. Set the config directory if not using the default:

```bash
lark msg <command>
lark chat <command>
# Or with explicit config:
LARK_CONFIG_DIR=/path/to/.lark lark msg <command>
```

## Commands Reference

### Send Messages

Send messages to users or group chats. By default sends **as the user** (your identity); use `--as bot` to send as the bot.

```bash
lark msg send --to ou_xxxx --text "Hello!"
lark msg send --to ou_xxxx --text "Hello!" --as bot
```

Reply in thread:

```bash
lark msg send --to oc_xxxx --parent-id om_xxxx --msg-type text --text "Replying here"
```

Flags:
- `--to` (required): Recipient identifier (user ID, open_id, email, or chat_id)
- `--to-type`: Explicitly specify ID type (`open_id`, `user_id`, `email`, `chat_id`) — auto-detected if omitted
- `--text`: Message text (markdown-lite). Use `{{image}}` to place images.
- `--image`: Image file path (repeatable)
- `--file`: File path to send (repeatable; each file sent as a separate message). Supported: pdf, doc/docx, xls/xlsx, ppt/pptx, mp4, opus, and any other file (sent as `stream`). **Cannot be combined with `--text` or `--image`.** Max 30MB per file (Lark API limit). Images sent via `--image` are capped at 10MB each.
- `--msg-type`: `text` (default) or `post` — auto-upgraded to `post` when `--text` contains `**`/`@{` mentions or `--image` is used
- `--as`: `bot` or `user` (default)
- `--parent-id`: Parent message ID for threaded reply
- `--root-id`: Root message ID for thread replies

### Code blocks / rich markdown (interactive card)

`lark msg send --text` (markdown-lite, `post`/`text`) does **NOT** render fenced
code blocks — triple-backticks show up literally in Lark. To get a real
rendered code block (line numbers + syntax highlighting), send an **interactive
card** with a `markdown` element via the `lark api` passthrough. Works `--as user`.

```bash
# Build the card body (content must be a JSON *string*), then POST it.
python3 - <<'PY' > /tmp/card.json
import json
md = """```bash
POST /api/auth/token
grant_type=refresh_token
```

```json
{ "activeRole": "COMPANY", "availableRoles": ["COMPANY"] }
```"""
card = {"config": {"wide_screen_mode": True},
        "elements": [{"tag": "markdown", "content": md}]}
print(json.dumps({"receive_id": "ou_xxx_or_oc_xxx",
                  "msg_type": "interactive",
                  "content": json.dumps(card)}))
PY

lark api POST /open-apis/im/v1/messages \
  --params '{"receive_id_type":"open_id"}' \
  --data "$(cat /tmp/card.json)" --as user
```

Notes:
- `receive_id_type` is `open_id` for a DM (the recipient's `ou_…`) or `chat_id`
  for a group (`oc_…`); set `receive_id` to match.
- The API echoes the card back as flattened `text` runs with literal backticks
  — that's only Lark's internal storage form. The actual chat render is a proper
  code block. Don't be fooled by the echo.
- The card markdown element supports the usual fences incl. a language tag
  (` ```json `, ` ```bash `) for highlighting. Threading: cards can't use
  `--parent-id`; post the card to the chat directly (or as a normal reply +
  separate card).

### Read Chat History

```bash
lark msg history --chat-id oc_xxxxx --limit 50 --sort desc
```

Normal reads emulate the surrounding context visible in Lark itself. In JSON,
each message can include `context.reactions` (grouped emoji plus the people who
reacted) and `context.thread` (exact reply count plus the latest three decoded
replies). In `--format text`, the same information is rendered immediately
below the message as `reactions:` and `thread:` lines. The primary message
content is unchanged if either optional context endpoint is unavailable.

Flags:
- `--chat-id` (required): Chat ID or thread ID
- `--type`: `chat` (default) or `thread`
- `--as`: `bot` or `user` (default). User mode works for DMs and any group you're in.
- `--start` / `--end`: Time range (Unix timestamp or ISO 8601)
- `--sort`: `asc` (default) or `desc`
- `--limit`: Maximum messages (`0` = no limit)

#### Agent-readable transcript: `--format text`

```bash
lark msg history --chat-id oc_xxxxx --limit 20 --format text
```

The global `--format text` flag (also valid on `lark dm` and `lark msg search`) emits **one fully-decoded plain-text block per message** instead of raw JSON. No JSON-walking required — content is decoded deterministically and senders/mentions are resolved to display names:

```
[2026-05-29T10:11:12Z] Alice (thread:1a2b3c4d): kickoff, see docs (https://x.io) @Bob
[2026-05-29T10:11:18Z] ↳ bot: on it
      second line of a multi-line message is indented
```

Decoding rules: `text`→text · `a`→`label (url)` · `at`→`@Name` · `img`/`media`/`file`→`[image|video|file <key>]` · `code_block`→the code verbatim, newlines intact · `md`→markdown source · `hr`→`---` · `post`→title + paragraphs joined by newlines · `interactive` card→`[card] <title>` plus every text/markdown element · recalled→`[recalled]` · audio/sticker/shared→`[audio]`/`[sticker]`/`[shared chat]`. An unrecognized tag renders as `[tag:<name>] <text>` rather than being dropped, so content loss is always visible. Replies are prefixed `↳` and indented; thread messages carry a `thread:<short>` tag so you can follow threading. `pretty`/`json`/`ndjson` are unchanged.

> `code_block` matters more than it looks: Lark puts anything pasted as a code
> block (keys, configs, logs, stack traces) in that tag, and it is usually the
> payload the message is actually about. `--format text` and `dm --compact`
> share one decoder, so they always agree on what a message says.

### Search Messages Across Chats

```bash
lark msg search "incident" --chat-id oc_xxx --limit 20
lark msg search "report" --after 2026-04-01 --before 2026-04-15
lark msg search "design" --sender ou_xxx --type text
```

Searches all chats visible to the authenticated user. Always uses the user token (`im:message:search` scope).

Flags:
- `<query>` (positional, required): Search text
- `--chat-id`: Restrict to a specific chat (**client-side**, see note)
- `--sender`: Restrict to messages from a sender, open_id (**client-side**, see note)
- `--type`: Filter by message type (`text`, `image`, `file`, etc.)
- `--after` / `--before`: Time range
- `--limit`: Maximum results
- `--format text`: Compact one-line-per-hit transcript (`[ts] sender @ chat: snippet`)

> ⚠️ **`--sender` and `--chat-id` are filtered client-side.** Lark's search API silently ignores its `from_ids` / `chat_ids` parameters and returns global results mixing senders and chats, so the CLI applies these filters **after** fetching (matching `meta_data.from_id` / `meta_data.chat_id`) and prints a one-line note to **stderr** when it does. Because matching hits can be sparse across the global result stream, the CLI scans up to 50 pages looking for `--limit` matches before stopping. The JSON output includes `"client_side_filtered": true` when a client-side filter was applied.

### Forward Messages

```bash
lark msg forward --message-id om_xxx --to oc_yyy       # default --as bot (required by Lark API)
```

Forward a single existing message to another recipient. **Bot-only by Lark API design** — user token is not supported by the forward endpoint.

Flags:
- `--message-id` (required): Message to forward
- `--to` (required): Recipient ID
- `--to-type`: Auto-detected if omitted
- `--as`: `bot` (default) or `user`

### Merge-forward Messages

```bash
lark msg merge-forward --message-ids om_a,om_b,om_c --to oc_yyy
```

Merge multiple messages into a single combined message and forward.

Flags:
- `--message-ids` (required): Comma-separated message IDs
- `--to` (required): Recipient ID
- `--to-type`: Auto-detected if omitted
- `--as`: `bot` (default) or `user`

### Edit a Sent Message

```bash
lark msg edit --message-id om_xxx --text "Fixed typo"
```

**Bot-only by Lark API design** — only works on messages the bot itself sent (the `--as` flag is locked to `bot`). Messages you sent via `--as user` cannot be edited via the API; recall and resend instead.

### Recall Messages

```bash
lark msg recall om_dc13264520392913993dd051dba21dcf            # default --as user
lark msg recall om_xxx --as bot                                # for bot-sent messages
```

Recalls/deletes previously sent messages. Default identity is `--as user`. Switch to `--as bot` to recall messages the bot sent.

### React to a Message

```bash
lark msg react --message-id om_xxx --reaction SMILE
lark msg react list --message-id om_xxx --reaction SMILE --limit 50
lark msg react remove --message-id om_xxx --reaction-id ZCaCIjUBVVWSrm5L-3ZTw...
lark msg react emojis    # list supported emoji types
```

**Custom Emojis:** Configure org-specific emojis in `.lark/config.yaml`:

```yaml
custom_emojis:
  "7405453485858095136": "ez-pepe"
```

### Batch Fetch Messages

```bash
lark msg get om_a,om_b,om_c
```

`msg get` includes the same reaction and thread context as `msg history`; add
`--format text` for the compact, human-readable view.

Fetch full content for a comma-separated list of message IDs.

### Download Resources

```bash
lark msg resource --message-id om_xxx --file-key img_v3_xxx --output ./image.png
lark msg resource --message-id om_xxx --file-key file_v2_xxx --output ./doc.pdf
lark msg resource --message-id om_xxx --file-key file_v3_xxx --output ./video.mp4   # video / audio files
```

Flags: `--message-id`, `--file-key`, `--output` (all required). `--type` is inferred from the `file-key` prefix (`img_*` → image, `file_*` → file). Files covers PDFs, docs, video (mp4/mov), and audio (opus). Pass `--type image|file` only if you need to override the inferred type.

Limitations (Lark API):
- Max 100MB per download (error code 234037 if exceeded)
- Emoji resources can't be downloaded
- Resources from card, merged, or forwarded messages are not supported

## Group Chat Management

All `lark chat` subcommands use the user token by default — override with `--as bot`.

### DM Lookup

```bash
lark chat dm alice@example.com                       # by email
lark chat dm "Francis Goh"                            # by name (fuzzy)
lark chat dm ou_f8735159a11237cb442c3d72aee8b073      # passes through
```

Returns the user's `open_id`, name, and a ready-to-use `lark msg send --to <open_id>` command. Lark auto-creates the P2P chat on first message — to read DM history afterward, capture the `chat_id` from the send response and use `lark msg history --chat-id <oc_id>`.

> ⚠️ **P2P chat_id is only recoverable by sending.** `lark chat dm` returns an **empty `chat_id`** unless the DM appears in the recent-activity scan that backs `list-dms`. For an **inactive / old** DM it will say `"No prior DM found"` even though history exists. This is a hard Lark API limitation, not a CLI bug — see [Resolving an inactive DM's chat_id](#resolving-an-inactive-dms-chat_id) below.

### Resolving an inactive DM's chat_id

**The constraint.** Lark exposes **no read-only way** to get a 1:1 P2P `chat_id` from a person's `open_id`:
- `GET /im/v1/chats` returns **group/topic chats only** — never P2P (verified).
- `POST /im/v1/chats` with `chat_mode=p2p` is **not supported** — the create endpoint only accepts `chat_mode=group` (verified against the docs); there is no get-or-create-P2P call.
- `msg search --sender <ou_>` / the raw `im/v1/messages/search` `from_ids`+`chat_type` filters are **silently ignored** by the API — they return unrelated chats, so search cannot recover a specific person's P2P chat (verified).
- `lark chat list-dms` only finds P2P chats that have **recent messages** in the scanned window. Inactive DMs never surface.

**What works:**

| Goal | Reliable command | Notes |
|---|---|---|
| **Send** to a person (active or inactive DM) | `lark msg send --to <ou_ or email> --as user --text "…"` | Lark auto-resolves/creates the P2P chat and **reuses the existing one** if a DM history exists — same `chat_id`, no duplicate. The send **response carries `chat_id`** (`oc_…`). |
| **Read** a known P2P chat | `lark msg history --chat-id <oc_…>` | `--chat-id` requires an `oc_` id; an `open_id` is rejected with `invalid container_id` (code 230001). |
| **Read** an inactive DM whose `chat_id` you don't have | Send first → grab `chat_id` from the response → `lark msg history --chat-id <oc_…>` | ⚠️ Sending **notifies the user**. There is no notify-free way to obtain the id for an inactive DM. If you must not notify, you cannot read that DM's history via the API. **You only pay this once:** see the local DM cache below. |

**Local person→chat_id cache (notify-free after the first time).** Once a P2P `chat_id` becomes known by any means — a `msg send` response, a `lark dm`/`chat dm` that resolved one, or a `list-dms` scan — the CLI persists it to `dm_cache.json` in the config dir, keyed by the counterpart's `open_id`. On the next `lark dm "<name>"` or `lark chat dm "<name>"`, the cache is consulted **first**, so reading an inactive DM no longer needs a fresh send (no notification). The cache is best-effort and never blocks a command on a read/write error. When a `chat_id` is still genuinely unknown (never cached, no recent activity), the command returns the counterpart's `open_id` plus an explicit message that the `chat_id` is only obtainable by sending (which notifies) — it never sends implicitly.

**Every cached mapping is membership-verified before it is used.** A `chat_id` is
only attributed to a person after the chat's own member list confirms it is that
person's 1:1 chat (exactly us and them, no third party). This is not paranoia
about a hypothetical: the discovery signals are hints, not proof. A message *we*
sent inside someone else's DM carries **our** `open_id` as its sender, so the old
sender-based inference filed unrelated chats under our own id — a lookup for
"Ying Cong Seah" resolved to the DM with Leo Yan, and anything that then replied
through that `chat_id` would have messaged Leo.

Consequences worth knowing:

- Entries carry a `verified` flag. Unverified ones (written before verification
  existed) are checked on first use; any that fail are dropped and re-resolved.
- `lark chat dm-cache` lists the cache and each entry's state;
  `lark chat dm-cache --verify` checks every entry at once and drops
  mis-attributed ones. Use it after any incident, or to clean a stale cache.
- **Bot DMs can't be verified and are therefore not cached.** Lark omits bots
  from member lists, so a bot DM and your own note-to-self chat are both exactly
  `[self]`; `owner_id` separates them (yours is owned by you, a bot DM has no
  owner), but nothing ties a bot DM back to a specific bot `open_id` — bot
  messages are sent by `cli_<app_id>` and `contact/v3/users/<bot ou_>` returns
  `41050 no user authority`. So bot `chat_id`s are re-obtained from a send
  response each time rather than trusted from cache.

**The full recipe (resolve → read an inactive DM):**
```bash
# 1. Resolve the person to an open_id (read-only, no notification)
lark chat dm "Nadhilah Nur Talitha"        # → open_id ou_5d3f6e96...; chat_id may be empty

# 2. Send a real message — this is the ONLY way to obtain the P2P chat_id for an
#    inactive DM. The response includes chat_id (oc_...). This DOES notify them.
lark msg send --to ou_5d3f6e9664030745fb0c00908042be43 --as user --text "Hi Nadhilah"

# 3. Read history with the chat_id returned in step 2
lark msg history --chat-id <oc_id_from_step_2> --sort desc --limit 30
```
For DMs that are recently active, skip the send: `lark chat list-dms` (or `lark dm "<name>"`) already yields the `chat_id`. The send-to-obtain-id path is **only** needed for inactive DMs, and only when you actually intend to message the person.

### Unified DM Read/Reply
```bash
lark dm "Francis Goh" --last 20
lark dm "Francis Goh" --reply "Can you share logs?" --compact
lark dm "Francis Goh" --reply "test" --dry-run-reply        # preview, don't send
```

Single command workflow for DM:
- resolves person (open_id/email/exact name/fuzzy name)
- reads recent DM history (default `--last 20`)
- optionally sends `--reply` first, then returns updated history
- `--compact` returns parsed plain-text transcript optimized for agent consumption
- `--dry-run-reply` resolves the recipient and previews the reply target without sending

> ⚠️ `lark dm "<name>"` (read-only, no `--reply`) returns **"no existing DM history yet"** for an **inactive** DM even when history exists — it can only read DMs whose `chat_id` is recoverable from recent activity. To read an inactive DM you must send (which yields the `chat_id`); see [Resolving an inactive DM's chat_id](#resolving-an-inactive-dms-chat_id). Passing `--reply` does send a real message and then reads back, so `lark dm "<name>" --reply "…"` works for inactive DMs (it notifies).

### Enumerate Recent DMs

```bash
lark chat list-dms                # last ~50 distinct DMs by recent activity
lark chat list-dms --limit 200    # scan deeper
```

Output: `{ count, dms: [{ chat_id, counterpart, last_message_at, last_sender_id }] }`.

Workaround for a Lark API limitation: `/im/v1/chats` only returns group chats, so P2P chats are not listed anywhere directly. This command scans recent messages across all visible chats and dedupes by `chat_id` where `is_p2p_chat=true`. Result count depends on how active your DMs are in the scanned window, so bump `--limit` if you expect more than a few. **Inactive DMs will not appear at any `--limit`** — to reach those, see [Resolving an inactive DM's chat_id](#resolving-an-inactive-dms-chat_id).

### Search Chats

```bash
lark chat search "project" --limit 10
lark chat search "团队"            # supports internationalized names + pinyin / fuzzy match
```

Output: `chats[] { chat_id, name, description, owner_id, external, chat_status }`, `count`, `query`.

For listing direct messages, use `chat list-dms` (above) — `chat search` is geared toward group chats and requires a non-empty query in practice.

### Get Chat Details / Members

```bash
lark chat get <chat-id>
lark chat members <chat-id> --limit 100
```

### Create a Chat

```bash
lark chat create --name "Launch Room"                                 # empty chat
lark chat create --name "Launch Room" --description "Go-to-market" \
  --members alice@example.com,bob@example.com
lark chat create --name "1:1 w/ Francis" --to "Francis Goh"           # 1:1 shortcut
```

Flags:
- `--name` (required)
- `--description`
- `--members`: Comma-separated IDs, emails, or names
- `--to`: Shortcut for a single person (open_id, email, or name) — resolved before creation

Notes:
- Bot identity is required by the Lark API for `create`.
- Members must have interacted with the bot at least once before they can be added. If they haven't, create the chat empty and share the join link with `lark chat link <chat-id>`.

### Delete a Chat

```bash
lark chat delete <chat-id>                # default --as bot (chats created via the CLI are bot-owned)
lark chat delete <chat-id> --as user      # for chats you personally own
```

Disbands (deletes) a group chat. Owner privileges required. Irreversible.

### Update a Chat

```bash
lark chat update <chat-id> --name "New Name" --description "New Desc"
```

### Manage Members

```bash
lark chat member add <chat-id> --members carol@example.com,dave@example.com
lark chat member remove <chat-id> --members ou_xxx
```

### Pin / Unpin Messages

```bash
lark chat pin <chat-id> --message-id om_xxx
lark chat pins <chat-id>
lark chat unpin <chat-id> --message-id om_xxx
```

### Share Link

```bash
lark chat link <chat-id>
```

Returns a shareable join link for the chat.

## Tips

- Use `\n` for line breaks and `\t` for indentation in `--text`
- Use `@{ou_xxx}` to mention users in group chats
- Use `{{image}}` in text to place images in order
- Chat IDs start with `oc_`; thread IDs start with `thread_` or `omt_`; message IDs start with `om_`
- The CLI auto-detects recipient type; override with `--to-type` if needed
- `msg search` requires user token and the `im:message:search` scope — group chat membership determines what's visible
- `--as user` is the default for `msg send` and all `chat` commands since it bypasses bot availability restrictions

## Message Types

- `text` — Plain text
- `post` — Rich text post (default)
- `image`
- `file`
- `audio`
- `media` — Video/media
- `sticker`
- `interactive` — Card
- `share_chat`
- `share_user`

## Reading Thread Replies

If a message has a `thread_id`, it's part of a thread (or is the root):

```bash
lark msg history --chat-id omt_1a3b99f9d2cfd982 --type thread
```

Thread messages have `is_reply: true` for replies (root has `is_reply: false`).

## Output Format

All commands output JSON.

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
- `SCOPE_ERROR` — Missing messages permissions
- `VALIDATION_ERROR` — Missing required fields
- `API_ERROR` — Lark API issue (e.g., bot not in group, missing permissions)

## Required Permissions

This skill requires the `messages` scope group:

```bash
lark auth login --add --scopes messages
```

To check current permissions:
```bash
lark auth status
```

Additional requirements:

**For reading messages:**
- Bot must be in the group chat (bot mode)
- Group chat reads require "Read all messages in associated group chat"
- Private chat reads require `im:message:readonly`

**For sending messages:**
- Requires `im:message` or `im:message:send_as_bot`
- Bot must be in group chats before sending

**For reactions:**
- List requires `im:message.reactions:read`
- Add/remove requires `im:message.reactions:write_only`

**For search:**
- Requires user token with `im:message:search`

**For group chat management:**
- Create/update/member/pin require `im:chat` (or `im:chat:readonly` for read-only ops)

## Notes

- Default identity is **user** for `msg send` and all `chat` commands (bypasses bot restrictions)
- Messages are sorted by creation time ascending by default in `msg history`
- Time filters don't apply to thread container type
- Replies with `--parent-id` always create a thread
- Forwarded/merged messages carry original sender attribution visually, but are sent by the forwarder
