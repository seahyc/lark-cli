# Lark CLI Feature Expansion Design

**Date:** 2026-04-16
**Status:** Approved

## Overview

Expand the lark-cli with a raw API passthrough for all 2500+ Lark endpoints, plus ergonomic shortcuts for 7 high-value domains: tasks, bitable, group chat, message search, meetings, wiki, and real-time events.

## Architecture

### Approach: Raw API Foundation + Ergonomic Shortcuts

The raw API passthrough (`lark api`) provides instant coverage of every Lark endpoint. Ergonomic shortcuts are layered on top for the 7 most-used domains, providing a friendlier CLI experience with input parsing, auto-detection, and structured output.

Both layers reuse the existing `client.go` infrastructure:
- `doRequest` for user token (`--as user`)
- `doRequestWithTenantToken` for bot token (`--as bot`)

### Skill File Organization

One skill file per domain (existing pattern). Updated/new skills:

| Skill | Status | Action |
|-------|--------|--------|
| `lark-api/` | New | Raw API passthrough + discovery |
| `lark-messages/` | Update | Add group chat ops, message search, forwarding |
| `lark-tasks/` | Update | Expand from read-only to full CRUD |
| `lark-bitable/` | Update | Expand from read-only to full CRUD |
| `lark-docs/` | Update | Add wiki aliases for backwards compat |
| `lark-meetings/` | New | Meeting search, notes, recordings |
| `lark-events/` | New | Real-time WebSocket event subscription |

New top-level `wiki` command (separate from `doc`). Existing `doc wiki*` commands remain as aliases.

---

## Feature 1: Raw API Passthrough

New `lark api` command that can hit any Lark endpoint directly.

### Commands

```bash
# List all API domains (fetches from open.larksuite.com/llms.txt, cached locally)
lark api list

# List endpoints for a domain
lark api list messaging

# HTTP methods
lark api GET /open-apis/im/v1/chats --params '{"page_size":20}' --as user
lark api POST /open-apis/task/v2/tasks --data '{"summary":"Ship it"}' --as user
lark api PUT /open-apis/im/v1/chats/oc_xxx --data '{"name":"New Name"}' --as user
lark api PATCH /open-apis/bitable/v1/apps/xxx --data '{"name":"Updated"}' --as user
lark api DELETE /open-apis/im/v1/messages/om_xxx --as bot

# File upload (multipart)
lark api POST /open-apis/im/v1/files --form file=@./report.pdf --form file_type=stream --as user

# File download (binary response saved to file)
lark api GET /open-apis/im/v1/messages/om_xxx/resources/file_xxx --output ./download.pdf --as user
```

### Flags

| Flag | Description |
|------|-------------|
| `--as` | `bot` (default) or `user` |
| `--params` | JSON query parameters |
| `--data` | JSON request body |
| `--form` | Multipart form fields (repeatable). `key=@./path` for file uploads |
| `--output` | Save response body to file (for binary downloads) |

### Design Details

- Accepts full URLs (`https://open.larksuite.com/open-apis/...`) or path shorthand (`/open-apis/...`)
- No schema validation — errors come from the Lark API directly
- `lark api list` fetches `https://open.larksuite.com/llms.txt` and caches locally
- `lark api list <domain>` fetches the domain-specific sub-index

### Implementation

- New file: `internal/cmd/api.go` — cobra command with method/path args and flags
- New file: `internal/api/raw.go` — generic request execution (JSON, multipart, download)
- New scope group: none needed (uses existing user/tenant tokens)

---

## Feature 2: Tasks CRUD

Expand the existing read-only `task` command to full lifecycle management.

### Commands

```bash
# List
lark task list                                        # pending tasks
lark task list --completed                            # completed tasks

# Get
lark task get <task-id>

# Create
lark task create --summary "Ship feature" --due 2026-04-20
lark task create --summary "Review PR" --assignee ou_xxx --due 2026-04-18T17:00

# Update
lark task update <task-id> --summary "Updated title"
lark task update <task-id> --due 2026-04-25

# Complete / Reopen
lark task complete <task-id>
lark task reopen <task-id>

# Assign
lark task assign <task-id> --to ou_xxx

# Subtasks
lark task subtask create <parent-id> --summary "Sub-item"
lark task subtask list <parent-id>

# Comments
lark task comment <task-id> --text "Done, needs review"

# Reminders
lark task remind <task-id> --at 2026-04-19T09:00

# Tasklists
lark task tasklist list
lark task tasklist create --name "Sprint 42"
lark task tasklist add <list-id> <task-id>
```

### Design Details

- All commands default to `--as user`
- Timestamps accept ISO 8601 or date-only (`2026-04-20` -> midnight local TZ)
- Assignees accept `ou_xxx` open IDs or emails (auto-resolved via contacts API)
- Task IDs are GUIDs from the v2 tasks API

### New Scopes

`task:task`, `task:task:readonly`, `task:tasklist`, `task:tasklist:readonly`

### Implementation

- Expand: `internal/cmd/task.go` — add create/update/complete/reopen/assign/subtask/comment/remind/tasklist subcommands
- Expand: `internal/api/tasks.go` — add write methods (POST/PATCH/DELETE to `/task/v2/tasks/*`)
- Update: `internal/scopes/scopes.go` — add write scopes to tasks group

---

## Feature 3: Bitable CRUD

Expand the existing read-only `bitable` command to full CRUD.

### Commands

```bash
# Base management
lark bitable create --name "Project Tracker"
lark bitable get --app-token <token>

# Table management
lark bitable table create --app-token <token> --name "Tasks"
lark bitable table delete --app-token <token> --table-id <id>

# Field management
lark bitable field create --app-token <token> --table-id <id> --name "Status" --type select
lark bitable field update --app-token <token> --table-id <id> --field-id <id> --name "Priority"
lark bitable field delete --app-token <token> --table-id <id> --field-id <id>

# Record CRUD
lark bitable record create --app-token <token> --table-id <id> --fields '{"Name":"Task 1","Status":"Open"}'
lark bitable record update --app-token <token> --table-id <id> --record-id <id> --fields '{"Status":"Done"}'
lark bitable record delete --app-token <token> --table-id <id> --record-id <id>

# Batch operations (up to 500 per call)
lark bitable record batch-create --app-token <token> --table-id <id> --file ./records.json
lark bitable record batch-delete --app-token <token> --table-id <id> --records "rec1,rec2,rec3"

# Search/filter
lark bitable record search --app-token <token> --table-id <id> --filter 'CurrentValue.[Status]="Open"'

# File attachments
lark bitable record upload --app-token <token> --table-id <id> --field-id <id> --file ./photo.png
```

### Design Details

- All commands default to `--as user`
- Token handling: accepts full URLs (`/base/XXX` or `/wiki/XXX`), extracts token automatically
- Wiki URLs resolved via wiki API to get the underlying bitable token
- Batch record operations capped at 500 per call (Lark API limit)
- Field types for create: `text`, `number`, `select`, `multi_select`, `date`, `checkbox`, `person`, `url`, `phone`, `email`, `attachment`, `formula`, `lookup`

### New Scopes

`bitable:app` (write), expanding existing `bitable:app:readonly`

### Implementation

- Expand: `internal/cmd/bitable.go` — add table/field/record subcommands with CRUD operations
- Expand: `internal/api/bitable.go` — add write methods
- Update: `internal/scopes/scopes.go` — add write scope

---

## Feature 4: Group Chat Operations

Expand `lark chat` from search-only to full management.

### Commands

```bash
# Existing
lark chat search "project team"

# Create
lark chat create --name "Sprint Planning" --members ou_xxx,ou_yyy
lark chat create --name "1:1 with Francis" --members ou_f8735159...

# Get chat info
lark chat get <chat-id>

# Update
lark chat update <chat-id> --name "New Chat Name"
lark chat update <chat-id> --description "Weekly sync for backend team"

# Members
lark chat members <chat-id>
lark chat member add <chat-id> --members ou_xxx,ou_yyy
lark chat member remove <chat-id> --members ou_xxx

# Pin messages
lark chat pin <chat-id> --message-id om_xxx
lark chat unpin <chat-id> --message-id om_xxx
lark chat pins <chat-id>

# Chat link
lark chat link <chat-id>
```

### Design Details

- All commands default to `--as user`
- Members accept `ou_xxx` open IDs or emails (auto-resolved)
- Chat create returns the new `chat_id`

### New Scopes

`im:chat`, `im:chat:readonly`

### Implementation

- Expand: `internal/cmd/chat.go` — add create/get/update/members/pin/link subcommands
- New file: `internal/api/chats.go` — chat CRUD and member management methods
- Update: `internal/scopes/scopes.go` — add chat scopes to messages group

---

## Feature 5: Message Search & Forwarding

Adding search, batch fetch, and forwarding to `lark msg`.

### Commands

```bash
# Search across all chats (user-token only)
lark msg search "deployment failed" --limit 20
lark msg search "API change" --chat-id oc_xxx --limit 10
lark msg search "budget" --sender ou_xxx
lark msg search "outage" --after 2026-04-01 --before 2026-04-15
lark msg search "report" --type file

# Batch fetch messages by ID
lark msg get om_xxx,om_yyy,om_zzz

# Forward a message
lark msg forward --message-id om_xxx --to oc_yyy

# Merge-forward multiple messages
lark msg merge-forward --message-ids om_xxx,om_yyy --to oc_yyy
```

### Design Details

- Message search is user-token only (Lark API restriction), always uses `--as user`
- Search supports combining filters (chat, sender, time range, message type)
- Batch fetch accepts comma-separated message IDs, max 50 per call
- Forward/merge-forward default to `--as user`

### New Scopes

`im:message:search` (for search)

### Implementation

- Expand: `internal/cmd/msg.go` — add search/get/forward/merge-forward subcommands
- Expand: `internal/api/messages.go` — add search, batch-get, forward, merge-forward methods

---

## Feature 6: Meetings

New `lark meetings` command for video conferencing records and notes.

### Commands

```bash
# Search past meetings
lark meetings search "sprint review" --limit 10
lark meetings search --after 2026-04-01 --before 2026-04-15
lark meetings search --organizer ou_xxx
lark meetings search --participant ou_xxx

# Get meeting details
lark meetings get <meeting-id>

# Get meeting notes (AI-generated summary, action items)
lark meetings notes <meeting-id>

# Get transcript
lark meetings transcript <meeting-id>                 # plain text
lark meetings transcript <meeting-id> --format srt    # SRT with timestamps + speakers

# Get recording download URL (24-hour validity)
lark meetings recording <meeting-id>
```

### Design Details

- All commands default to `--as user`
- `notes` returns structured JSON with summary, action items, chapters
- `transcript` reuses existing minutes infrastructure (resolves meeting-id -> minute_token)
- `recording` returns a time-limited download URL, not the file itself

### New Scopes

`vc:meeting:readonly`, `vc:record:readonly`

### Implementation

- New file: `internal/cmd/meetings.go` — cobra commands for search/get/notes/transcript/recording
- New file: `internal/api/meetings.go` — API methods for VC endpoints
- Update: `internal/scopes/scopes.go` — add meetings scope group
- New skill file: `~/.claude/skills/lark-meetings/skill.md`

---

## Feature 7: Wiki Write

New `lark wiki` top-level command for wiki space and node management.

### Commands

```bash
# Space management
lark wiki spaces
lark wiki space get <space-id>

# Node management
lark wiki create --space-id <id> --title "Runbook" --parent <node-token>
lark wiki create --space-id <id> --title "API Docs"   # create at root
lark wiki move <node-token> --parent <target-node> --space-id <id>
lark wiki delete <node-token>
lark wiki get <node-token>
lark wiki children <node-token>
```

### Design Details

- All commands default to `--as user`
- Created wiki nodes are documents by default
- Content is edited with existing `doc append` / `doc update` commands
- Existing `doc wiki` / `doc wiki-search` remain as aliases for backwards compatibility

### New Scopes

`wiki:wiki` (write), expanding existing `wiki:wiki:readonly`

### Implementation

- New file: `internal/cmd/wiki.go` — cobra commands for spaces/create/move/delete/get/children
- New file: `internal/api/wiki.go` — wiki space and node API methods (some may already exist in documents.go)
- Update: `internal/scopes/scopes.go` — add write scope to documents group

---

## Feature 8: Real-time Events

New `lark events` command for WebSocket event subscription.

### Commands

```bash
# Subscribe to all events
lark events subscribe

# Filter by event type
lark events subscribe --type im.message.receive_v1
lark events subscribe --type calendar.event.changed_v1

# Filter by regex on payload
lark events subscribe --match "urgent|critical"

# Output to file
lark events subscribe --output ./events.jsonl

# Compact format (agent-friendly)
lark events subscribe --compact

# Combine filters
lark events subscribe --type im.message.receive_v1 --match "deploy" --compact
```

### Design Details

- Bot-only (`--as bot` always) — WebSocket subscriptions use tenant_access_token
- Output is NDJSON (one JSON object per line) to stdout by default
- Auto-reconnects on disconnect
- Common event types: `im.message.receive_v1`, `im.chat.member.user.added_v1`, `contact.user.updated_v3`, `calendar.event.changed_v1`, `task.task.update_v1`, `approval.instance.status_changed`

### New Scopes

None — uses existing tenant token. Event types are gated by app permissions configured in Lark developer console.

### Implementation

- New file: `internal/cmd/events.go` — cobra command with filter flags
- New file: `internal/api/events.go` — WebSocket connection, reconnect logic, NDJSON output
- New skill file: `~/.claude/skills/lark-events/skill.md`

---

## New Scope Groups Summary

Update `internal/scopes/scopes.go`:

```go
"messages": {
    Scopes: []string{
        "im:message:readonly", "im:message", "im:message:send_as_bot",
        "im:message.send_as_user", "im:resource",
        "im:message.reactions:read", "im:message.reactions:write_only",
        "im:chat", "im:chat:readonly",       // NEW: group chat management
        "im:message:search",                  // NEW: message search
    },
},
"tasks": {
    Scopes: []string{
        "task:task", "task:task:readonly",     // EXPANDED: was read-only
        "task:tasklist", "task:tasklist:readonly", // NEW: tasklist management
    },
},
"bitable": {
    Scopes: []string{
        "bitable:app:readonly",
        "bitable:app",                        // NEW: write access
    },
},
"documents": {
    Scopes: []string{
        // ... existing scopes ...
        "wiki:wiki",                          // NEW: wiki write (was readonly only)
    },
},
"meetings": {                                 // NEW GROUP
    Name:        "meetings",
    Description: "Video conferencing records and notes",
    Scopes:      []string{"vc:meeting:readonly", "vc:record:readonly"},
    Commands:    []string{"meetings"},
},
```

---

## Implementation Order

Recommended build order (each is independently shippable):

1. **Raw API passthrough** — foundation, unblocks everything
2. **Group chat ops** — small scope, extends existing chat.go
3. **Message search** — small scope, extends existing msg.go
4. **Tasks CRUD** — medium scope, high daily value
5. **Wiki write** — small scope, extends existing doc infrastructure
6. **Bitable CRUD** — large scope, many subcommands
7. **Meetings** — medium scope, new domain
8. **Real-time events** — medium scope, WebSocket is architecturally different

Each feature: implement API methods -> add cobra commands -> update scopes -> update skill file -> build and test.
