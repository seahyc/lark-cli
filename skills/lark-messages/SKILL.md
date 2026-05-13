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
- **Group chat management**: create chats, update name/description, add/remove members, pin/unpin messages, get share links
- Find chats by name or member (`chat search`)

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
lark chat create --name "Launch Room" --members alice@x,bob@x
lark chat update <chat-id> --name "Launch Room 2" --description "Updated"
lark chat member add <chat-id> --members carol@x
lark chat member remove <chat-id> --members ou_xxx
lark chat pin <chat-id> --message-id om_xxx
lark chat pins <chat-id>
lark chat unpin <chat-id> --message-id om_xxx
lark chat link <chat-id>
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
- `--file`: File path to send (repeatable; each file sent as a separate message). Supported: pdf, doc/docx, xls/xlsx, ppt/pptx, mp4, opus, and any other file (sent as `stream`). **Cannot be combined with `--text` or `--image`.** Max 30MB.
- `--msg-type`: `post` (default) or `text`
- `--as`: `bot` or `user` (default)
- `--parent-id`: Parent message ID for threaded reply
- `--root-id`: Root message ID for thread replies

### Read Chat History

```bash
lark msg history --chat-id oc_xxxxx --limit 50 --sort desc
```

Flags:
- `--chat-id` (required): Chat ID or thread ID
- `--type`: `chat` (default) or `thread`
- `--as`: `bot` or `user` (default). User mode works for DMs and any group you're in.
- `--start` / `--end`: Time range (Unix timestamp or ISO 8601)
- `--sort`: `asc` (default) or `desc`
- `--limit`: Maximum messages (`0` = no limit)

### Search Messages Across Chats

```bash
lark msg search "incident" --chat-id oc_xxx --limit 20
lark msg search "report" --after 2026-04-01 --before 2026-04-15
lark msg search "design" --sender ou_xxx --type text
```

Searches all chats visible to the authenticated user. Always uses the user token (`im:message:search` scope).

Flags:
- `<query>` (positional, required): Search text
- `--chat-id`: Restrict to a specific chat
- `--sender`: Restrict to messages from a sender (open_id)
- `--type`: Filter by message type (`text`, `image`, `file`, etc.)
- `--after` / `--before`: Time range
- `--limit`: Maximum results

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

Fetch full content for a comma-separated list of message IDs.

### Download Resources

```bash
lark msg resource --message-id om_xxx --file-key img_v3_xxx --type image --output ./image.png
lark msg resource --message-id om_xxx --file-key file_v2_xxx --type file --output ./doc.pdf
```

Flags: `--message-id`, `--file-key`, `--type` (`image` or `file`), `--output` — all required.

Limitations:
- Max 100MB
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

### Unified DM Read/Reply
```bash
lark dm "Francis Goh" --last 20
lark dm "Francis Goh" --reply "Can you share logs?" --compact
```

Single command workflow for DM:
- resolves person (open_id/email/exact name/fuzzy name)
- reads recent DM history
- optionally sends `--reply` first, then returns updated history
- `--compact` returns parsed plain-text transcript optimized for agent consumption

### Search Chats

```bash
lark chat search "project" --limit 10
```

Output: `chats[] { chat_id, name, description, owner_id, external, chat_status }`, `count`, `query`.

### Get Chat Details / Members

```bash
lark chat get <chat-id>
lark chat members <chat-id> --limit 100
```

### Create a Chat

```bash
lark chat create --name "Launch Room" --description "Go-to-market" \
  --members alice@example.com,bob@example.com
```

Flags:
- `--name` (required)
- `--description`
- `--members`: Comma-separated IDs or emails

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
