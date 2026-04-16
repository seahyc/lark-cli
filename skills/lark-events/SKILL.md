---
name: events
description: Subscribe to real-time Lark events (messages, reactions, calendar, etc.) via long-polling, streamed as NDJSON - use when the user wants to watch live activity, tail events to a file, or build a reactive workflow. Runs until interrupted.
---

# Lark Events Skill

Subscribe to real-time Lark events as NDJSON using the `lark events` CLI. Best-effort long-polling fallback for the outbound events API — fine for watching live activity or piping into another tool.

## 🤖 Capabilities and Use Cases

- Stream live Lark events (messages, reactions, calendar changes, etc.) to stdout or a file
- Filter by event type (`--type`) or by regex on the JSON payload (`--match`)
- Emit compact single-line JSON for piping into `jq` or log pipelines
- Run until Ctrl-C — suitable for watching during a specific interaction or for lightweight reactive scripts

## 🚀 Quick Reference

**Stream all events to stdout:**
```bash
lark events subscribe
```

**Only new message events:**
```bash
lark events subscribe --type im.message.receive_v1
```

**Only events for a specific chat (regex match on payload):**
```bash
lark events subscribe --match '"chat_id":"oc_xxx"'
```

**Tail to file as compact NDJSON:**
```bash
lark events subscribe --output events.ndjson --compact
```

## Commands Reference

### `lark events subscribe`

Subscribes to the Lark outbound events endpoint and streams matching events as NDJSON. Runs until interrupted with Ctrl-C.

Available flags:
- `--type`: Filter by `event_type` in the callback header (e.g. `im.message.receive_v1`)
- `--match`: Filter events whose raw JSON payload matches this Go regex
- `--output`: Write events to file instead of stdout (appends)
- `--compact`: Emit single-line JSON (strips whitespace)

Output: one JSON event per line (NDJSON). Each event includes a `header` with `event_id`, `event_type`, `create_time`, `app_id`, `tenant_key`, and an `event` body specific to the event type.

## Common Event Types

| `event_type` | Meaning |
|---|---|
| `im.message.receive_v1` | New message in a chat the bot can see |
| `im.message.reaction.created_v1` | Reaction added |
| `im.message.reaction.deleted_v1` | Reaction removed |
| `im.chat.member.user.added_v1` | User joined a chat |
| `im.chat.member.user.deleted_v1` | User left a chat |
| `calendar.calendar.event.changed_v4` | Calendar event changed |
| `contact.user.updated_v3` | Contact changed |
| `task.task.update_v2` | Task updated |

See the Lark open-platform event docs for the full list — `lark events subscribe` will stream any event your app has subscribed to.

## Running in Background

Because `subscribe` runs until Ctrl-C, you typically run it in a background process or logging shell:

```bash
# Run in background, tee to file
lark events subscribe --compact --output events.ndjson &

# Pipe into jq for live pretty-printing
lark events subscribe --compact | jq '.'
```

## Output Format

NDJSON — one JSON object per line. Use `--compact` to guarantee no embedded newlines (useful when piping to `jq -c`, `awk`, or log collectors).

## Error Handling

Errors return JSON on stderr with a non-zero exit:
```json
{
  "error": true,
  "code": "ERROR_CODE",
  "message": "Description"
}
```

Common error codes:
- `AUTH_ERROR` — Need to run `lark auth login`
- `VALIDATION_ERROR` — Invalid `--match` regex
- `FILE_ERROR` — Could not open `--output` path for append
- `API_ERROR` — Events endpoint returned an error (often tenant doesn't have the outbound events API enabled)

## Required Permissions

Uses the tenant access token. Your Lark app must have the relevant event subscriptions enabled in the developer console (e.g. `im.message.receive_v1` must be subscribed) — the CLI does not set up subscriptions, it only reads delivered events.

## Notes

- This is a long-polling fallback; the "real" Lark event channel is a WebSocket at `wss://open.larksuite.com/open-apis/event/ws` which is not currently wired up.
- If the tenant returns no events, the subscribe loop sleeps briefly and retries — transient errors auto-retry with a 5-second backoff.
- To make one-off calls (e.g. "fetch the next batch of events and exit") use `lark api POST /open-apis/event/v1/outbound/events` directly.
- Filters are additive: if both `--type` and `--match` are set, both must match.
