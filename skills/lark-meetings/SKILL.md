---
name: meetings
description: Search Lark video meetings and fetch their notes, transcripts, and recording info - use when the user asks "what did we discuss in yesterday's call", wants a transcript of a meeting, or needs to look up past video meetings by organizer, participant, or meeting number.
---

# Lark Meetings (VC) Skill

Search meeting history and fetch meeting notes, recordings, and transcripts via the `lark` CLI.

## 🤖 Capabilities and Use Cases

- Search the user's meeting history by time range, organizer, participant, or meeting number
- Get metadata for a specific meeting (title, start/end, participants)
- Fetch meeting notes attached during the call
- Resolve recording info (URL + `minute_token`) for a meeting
- Export the meeting transcript directly (TXT or SRT), optionally with speakers and timestamps

## 🚀 Quick Reference

**Search recent meetings:**
```bash
lark meetings search --after 2026-04-01 --before 2026-04-15 --limit 10
```

**Get meeting metadata:**
```bash
lark meetings get <meeting-id>
```

**Fetch meeting notes:**
```bash
lark meetings notes <meeting-id>
```

**Recording info (URL + minute_token):**
```bash
lark meetings recording <meeting-id>
```

**Export transcript (resolves minute_token automatically):**
```bash
lark meetings transcript <meeting-id>
lark meetings transcript <meeting-id> --format srt --speaker --timestamp
lark meetings transcript <meeting-id> --output transcript.txt
```

## Commands Reference

### `lark meetings search`

Search meetings using time range and optional filters.

Available flags:
- `--after`: Start time (Unix timestamp or ISO 8601)
- `--before`: End time (Unix timestamp or ISO 8601)
- `--organizer`: Filter by organizer open_id
- `--participant`: Filter by participant open_id
- `--meeting-no`: Filter by meeting number (the numeric ID people dial into)
- `--limit`: Maximum number of meetings (`0` = no limit)

Output:
```json
{
  "meetings": [
    {
      "id": "70700000123",
      "topic": "Weekly Sync",
      "meeting_no": "1234567890",
      "start_time": "1712860800",
      "end_time": "1712864400",
      "host_user": { "id": "ou_xxx" }
    }
  ],
  "count": 1
}
```

### `lark meetings get <meeting-id>`

Return full metadata for a meeting.

### `lark meetings notes <meeting-id>`

Return the notes attached to a meeting, if any.

### `lark meetings recording <meeting-id>`

Return recording info, including the `minute_token` that can be fed into the `minutes` commands.

Output:
```json
{
  "meeting_id": "70700000123",
  "url": "https://...",
  "minute_token": "minxxxxxxx"
}
```

### `lark meetings transcript <meeting-id>`

Export the transcript for a meeting. Internally resolves the `minute_token` via the VC recording API, then delegates to the existing minutes transcript export.

Available flags:
- `--format`: Output format, `txt` (default) or `srt`
- `--speaker`: Include speaker names
- `--timestamp`: Include timestamps
- `--output`: Write transcript to a file instead of embedding in the JSON response

Output (when `--output` is NOT used):
```json
{
  "meeting_id": "70700000123",
  "minute_token": "minxxxxxxx",
  "format": "txt",
  "content": "..."
}
```

Output (when `--output` IS used):
```json
{
  "meeting_id": "70700000123",
  "minute_token": "minxxxxxxx",
  "format": "txt",
  "file": "transcript.txt"
}
```

## Workflow Example

```bash
# 1. Find meetings from last week that someone organized
lark meetings search --after 2026-04-08 --before 2026-04-15 --organizer ou_xxx

# 2. Pick a meeting and fetch its notes
lark meetings notes 70700000123

# 3. Export the transcript with speaker labels
lark meetings transcript 70700000123 --speaker --timestamp --output notes.txt
```

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
- `SCOPE_ERROR` — Missing meetings permissions. Run `lark auth login --add --scopes meetings`
- `NOT_FOUND` — Meeting or minute_token not available (e.g. the recording hasn't finished processing yet)
- `API_ERROR` — Lark API issue (permissions on the meeting, no recording, etc.)

## Required Permissions

This skill requires the `meetings` scope group. If you see a `SCOPE_ERROR`:

```bash
lark auth login --add --scopes meetings
```

Minutes/transcript export also requires the `minutes` scope — most users already have this through the `lark-minutes` skill.

## Notes

- Lark VC tenant semantics vary. If `meetings search` doesn't return what you expect, fall back to `lark api GET /open-apis/vc/v1/meeting_list --params '{...}' --as user` to iterate on filters.
- The `minute_token` returned by `meetings recording` is the same token format used by the `lark-minutes` skill — any minutes operation can be run directly once you have it.
- Meeting IDs are short numeric strings (10-12 digits), distinct from `meeting_no` (the dial-in number).
