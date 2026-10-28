---
name: lark-messages
description: Chat and message operations in Lark - send/read/reply/recall/edit messages, react, download resources, and manage groups (create chats, add/remove members, pin/unpin messages, get share links, search messages across chats, forward or merge-forward messages). Use when the user asks about chat messages, conversation history, sending/searching messages, or managing Lark groups.
---

# Messages Skill

Full coverage for Lark chat + message operations via the `lark` CLI: send, read, react, edit, recall, search, forward, group-chat management, and resource downloads.

## 🤖 Capabilities and Use Cases

- Send markdown-lite messages with links and mentions
- **Send as user** (`--as user`, default) or as bot (`--as bot`)
- Send images with `--image` and `{{image}}` placement
- Send files (PDF, PPTX, DOCX, XLSX, etc.) with `--file`; videos (`.mp4`/`.mov`/`.m4v`) go as playable media (auto cover + duration, or pass `--cover`), audio (`.opus`) as a voice message
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
lark msg send --to oc_12345 --file ./demo.mp4                 # playable video (auto cover + duration)
lark msg send --to oc_12345 --file ./demo.mp4 --cover ./poster.jpg
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
lark msg edit --message-id <message-id> --text "Fixed typo"
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
- `--file`: File path to send (repeatable; each file sent as a separate message). The CLI picks the message type from the extension: **video** (`.mp4`/`.mov`/`.m4v`) is sent as a playable **media** message, **audio** (`.opus`) as a **voice** message, and everything else (pdf, doc/docx, xls/xlsx, ppt/pptx, and any other type via `stream`) as a plain file. **Cannot be combined with `--text` or `--image`.** Max 30MB per file (Lark API limit). Images sent via `--image` are capped at 10MB each.
- `--cover`: Cover image for a video `--file` (optional). If omitted, a plain placeholder cover is generated automatically — Lark overlays its own play button. A video *must* have a cover or Lark stores it as `nonsupport` ("This message type is currently not supported"); the CLI handles this for you. Video duration is read from the mp4 and shown on the thumbnail automatically.
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

### Read a copied message link without foregrounding the browser

```bash
lark msg get 'https://applink.larksuite.com/client/message/link/open?token=...'
lark msg history --chat-id 'https://applink.larksuite.com/client/message/link/open?token=...' --format text
```

URL inputs automatically use `browserctrl` and the selected browser's signed-in
Lark web Messenger session to resolve the exact chat position. The CLI opens and
closes its own background tab without focusing the browser or launching Lark.
Existing ID inputs do not use a browser.

Chrome, Edge, Brave, Chromium, Arc, and Dia are supported when accessible through
browserctrl. When no browser override is supplied, the CLI reads the operating system’s
default HTTPS browser (macOS, Linux, or Windows). It does not choose whichever
browser happens to be accessible. Safari and Firefox are not supported by this
background resolver. Only Dia has been verified live so far.

```bash
lark msg get '<message-url>' --browser chrome --workspace-url https://your-team.larksuite.com
```

Settings, in precedence order: command flags; `LARK_MSG_LINK_BROWSER` and
`LARK_MSG_LINK_WORKSPACE_URL`; `message_links.browser` and
`message_links.workspace_url` in the existing CLI config. `BROWSERCTRL_BROWSER`
is also accepted as the final browser fallback.

With a workspace URL, no existing Messenger tab is required: the temporary tab
uses the browser's existing login cookies. Without one, the CLI discovers a
workspace from open Messenger tabs and rejects ambiguous workspaces. An expired
login fails explicitly; sign in through the browser and retry. Credentials stay
inside the browser.

`msg get` reads the linked message; `msg history` follows its `thread_id` to read
replies using user API access.

The resolver uses a private Lark read operation and may need updating after Lark
changes. Unsupported thread-position/multi-message links, denied access, and
messages beyond the bounded 10,000-message history scan fail explicitly. A
message without a thread can still be read with `msg get`.

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

> ⚠️ **P2P discovery is bounded.** `lark chat dm` returns an **empty `chat_id`** unless the DM appears in the recent-activity scan that backs `list-dms`. For an **inactive / old** DM it will say `"No prior DM found"` even though history exists. Search for a distinctive phrase and retry `lark dm` — see [Resolving a DM missed by automatic discovery](#resolving-a-dm-missed-by-automatic-discovery) below.

### Resolving a DM missed by automatic discovery

`lark dm` checks the local cache, then a bounded message-search scan. A missing
result means that scan did not discover the chat; it does not prove there is no
DM history.

1. Search a distinctive phrase from the conversation:
   `lark msg search "phrase" --limit 20`.
2. Search caches up to 12 distinct P2P chats after verifying their actual
   members. Retry `lark dm "Name" --last 20` to read the discovered chat.
3. If you already know the chat ID, use
   `lark msg history --chat-id oc_xxx --sort desc --limit 20`.

Search sender and chat filters are applied client-side. A sender or display-name
match alone is not enough to attribute a DM; membership verification prevents
replies going to the wrong person. Failed verification leaves the cache unchanged.

Sending a message also returns a chat ID, but notifies the recipient and requires
outbound approval. Do not send merely to discover a chat for a read-only request.

Inspect mappings with `lark chat dm-cache`; use `--verify` to check and prune them.

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

If automatic discovery misses a DM, follow the search-and-retry workflow above.
`--reply` sends a real message; it is not a read-only discovery option.

### Enumerate Recent DMs

```bash
lark chat list-dms                # last ~50 distinct DMs by recent activity
lark chat list-dms --limit 200    # scan deeper
```

Output: `{ count, dms: [{ chat_id, counterpart, last_message_at, last_sender_id }] }`.

Workaround for a Lark API limitation: `/im/v1/chats` only returns group chats, so P2P chats are not listed anywhere directly. This command scans recent messages across all visible chats and dedupes by `chat_id` where `is_p2p_chat=true`. Result count depends on how active your DMs are in the scanned window, so bump `--limit` if you expect more than a few. For a missed DM, see [Resolving a DM missed by automatic discovery](#resolving-a-dm-missed-by-automatic-discovery).

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

The default output is `pretty`. Use the global `--format` flag for `json`,
`ndjson`, `table`, `csv`, or agent-readable `text` where supported.

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
