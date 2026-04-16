---
name: attendance
description: Query Lark Attendance check-in records from the CLI - list clock-in/out tasks for the current user or a teammate over a date range - use when you need to see working hours or check-in history without opening the Lark mobile app.
---

# Lark Attendance Skill

Query Lark Attendance user tasks (clock-in/out records) via the `lark attendance` CLI.

## Capabilities

- List the current user's attendance records for the last N days
- List a specific teammate's records (by open_id) within an explicit date range
- Optionally include overtime results

## Quick Reference

**Last 7 days for the current user (default):**
```bash
lark attendance list
```

**Custom range:**
```bash
lark attendance list --after 2025-01-01 --before 2025-01-31
```

**Another user by open_id:**
```bash
lark attendance list --user ou_xxxxxxxxxxxxxxxxxxxx --after 2025-03-01
```

**Include overtime:**
```bash
lark attendance list --overtime
```

## Commands Reference

### `lark attendance list`

Queries `/attendance/v1/user_tasks/query` for one user over a date range.

Flags:
- `--after <date>`: start date (YYYY-MM-DD or YYYYMMDD). Default: 7 days ago.
- `--before <date>`: end date (YYYY-MM-DD or YYYYMMDD). Default: today.
- `--user <open_id>`: target user's open_id. Default: the current user (resolved via `/authen/v1/user_info`).
- `--overtime`: include overtime results in the response.

Output shape:
```json
{
  "user_id": "ou_xxx",
  "check_date_from": 20250401,
  "check_date_to":   20250407,
  "count": 5,
  "tasks": [
    { "user_id": "ou_xxx", "day": 20250401, "check_in_time": "...", "check_out_time": "...", ... }
  ]
}
```

## Error Handling

JSON errors on failure with non-zero exit:
```json
{"error": true, "code": "API_ERROR", "message": "..."}
```

Common codes:
- `AUTH_ERROR` — run `lark auth login --add --scopes attendance`
- `SCOPE_ERROR` — missing `attendance:task` scopes
- `VALIDATION_ERROR` — bad `--after`/`--before` date
- `API_ERROR` — Lark rejected the request (e.g. user not enrolled in attendance)

## Required Permissions

Scopes in the `attendance` group:
- `attendance:task`
- `attendance:task:readonly`

Grant with:
```bash
lark auth login --add --scopes attendance
```

## Notes

- The Lark endpoint uses tenant token and treats `user_ids[0]` as the query target. The CLI defaults that to the caller's own open_id for convenience.
- Date integers on the wire are YYYYMMDD (e.g. 20250401). The CLI accepts `YYYY-MM-DD` for ergonomics and converts on your behalf.
- For multi-user or bulk reporting, script a loop that invokes `lark attendance list --user <id>` per teammate.
