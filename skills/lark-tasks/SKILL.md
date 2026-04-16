---
name: tasks
description: Create, update, and manage Lark Tasks - list tasks, view details, create/update tasks with due dates, complete/reopen, assign to people, manage subtasks, add comments, set reminders, and organize into tasklists. Use when the user asks about their todo list or wants to create/update/track tasks in Lark.
---

# Lark Tasks Skill

Full CRUD over Lark Tasks (v2) via the `lark` CLI.

## 🤖 Capabilities and Use Cases

- List your assigned tasks (incomplete by default, `--completed` to include done)
- Get full details of a specific task by GUID
- Create new tasks with optional due date (all-day or timed)
- Update an existing task's summary, description, or due date
- Complete/reopen tasks
- Assign or remove assignees/followers by open_id or email
- Manage subtasks (create, list)
- Add plain-text comments to a task
- Set reminders relative to the task's due time
- Manage tasklists: create new tasklists and add tasks to them

## 🚀 Quick Reference

**List/view:**
```bash
lark task list                         # incomplete only
lark task list --completed --limit 50  # include completed
lark task get <task-guid>
```

**Create / update:**
```bash
lark task create --summary "Ship v2" --due 2026-04-20
lark task update <guid> --summary "Ship v2.1" --due 2026-04-22
```

**Complete / reopen:**
```bash
lark task complete <guid>
lark task reopen <guid>
```

**Assign / follow:**
```bash
lark task assign <guid> --assignee alice@example.com,bob@example.com
lark task assign <guid> --follower ou_yyy
lark task assign <guid> --assignee ou_xxx --remove
```

**Subtasks:**
```bash
lark task subtask create <parent-guid> --summary "Substep 1"
lark task subtask list <parent-guid>
```

**Comments / reminders:**
```bash
lark task comment <guid> --content "Blocked on legal review"
lark task remind <guid> --minutes 15   # 15 min before due
```

**Tasklists:**
```bash
lark task tasklist list
lark task tasklist create --name "Q2 Goals"
lark task tasklist add <tasklist-guid> --task-guid <task-guid>
```

## Commands Reference

### List Tasks

```bash
lark task list [--limit N] [--completed]
```

Lists tasks assigned to you. By default, only shows incomplete tasks.

Flags:
- `--limit`: Maximum number of tasks (`0` = no limit)
- `--completed`: Include completed tasks

Output:
```json
{
  "tasks": [
    {
      "guid": "d300e75f-c56a-4be9-80d6-e47653b3e1a9",
      "summary": "Review Q4 budget proposal",
      "description": "Check the numbers and approve",
      "due_date": "2026-02-05T00:00:00+08:00",
      "is_all_day": true,
      "status": "todo",
      "creator_id": "ou_xxx",
      "creator_name": "John Doe",
      "created_at": "1704067200000"
    }
  ],
  "count": 1,
  "has_more": false
}
```

### Get Task Details

```bash
lark task get <task-guid>
```

Returns the same shape as a list item, with the full `description`.

### Create Task

```bash
lark task create --summary "Ship v2" --due 2026-04-20
lark task create --summary "Daily standup" --due 2026-04-17T09:30:00 --all-day=false
```

Flags:
- `--summary` (required): Task summary
- `--due`: Due date (Unix timestamp or ISO 8601)
- `--all-day`: Treat due date as all-day (default `true`). Set to `false` for a timed reminder.

Output includes the new `guid`.

### Update Task

```bash
lark task update <task-guid> --summary "New title" --due 2026-05-01 --description "Updated"
```

Flags:
- `--summary`: New summary
- `--description`: New description
- `--due`: New due date (Unix timestamp or ISO 8601)

### Complete / Reopen

```bash
lark task complete <task-guid>
lark task reopen <task-guid>
```

### Assign / Follow

```bash
lark task assign <task-guid> --assignee alice@example.com
lark task assign <task-guid> --assignee ou_xxx --remove
lark task assign <task-guid> --follower ou_yyy
```

Flags:
- `--assignee`: Assignee IDs or emails (comma-separated, repeatable). Emails are resolved automatically.
- `--follower`: Follower IDs or emails (comma-separated, repeatable)
- `--remove`: Remove the listed members instead of adding them

### Subtasks

```bash
lark task subtask create <parent-guid> --summary "Step 1"
lark task subtask create <parent-guid> --summary "Step 2" --due 2026-04-18
lark task subtask list <parent-guid> --limit 50
```

### Comment

```bash
lark task comment <task-guid> --content "Waiting on design"
```

### Reminder

```bash
lark task remind <task-guid> --minutes 15
lark task remind <task-guid> --minutes 0   # at due time
lark task remind <task-guid> --minutes -60 # 60 minutes AFTER due
```

The task must have a due time set for reminders to fire.

### Tasklists

```bash
lark task tasklist list
lark task tasklist create --name "Q2 Goals"
lark task tasklist add <tasklist-guid> --task-guid <task-guid>
```

Tasklists group related tasks and can be shared. Use `tasklist add` to move or link an existing task into a list.

## Task Status Values

| Status | Description |
|---|---|
| `todo` | Task is incomplete |
| `completed` | Task has been completed |

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
- `SCOPE_ERROR` — Missing tasks permissions. Run `lark auth login --add --scopes tasks`
- `NOT_FOUND` — Task or tasklist GUID not found
- `VALIDATION_ERROR` — Missing required flag (e.g. `--summary` on create)
- `API_ERROR` — Lark API issue

## Required Permissions

This skill requires the `tasks` scope group:

```bash
lark auth login --add --scopes tasks
```

To check current permissions:
```bash
lark auth status
```

## Tips

- The Tasks v2 API uses Unix milliseconds internally, but `--due` accepts either Unix seconds or ISO 8601 — the CLI converts.
- `--assignee` / `--follower` accept emails or `ou_*` open IDs interchangeably; internal mail resolution uses contact lookup.
- For all-day tasks, set the due date to a plain date (`2026-04-20`). For timed tasks, add a time (`2026-04-20T14:30:00`) and set `--all-day=false`.
- Combining `--assignee` and `--follower` in one call is supported — both sets are updated together.
- Subtasks are full Task v2 records; they appear in `task list` only if also assigned to you.

## Output Format

All commands output JSON.

## Best Practices

- Use `task list` at the start of a session to orient to outstanding work
- Use `task create --due ... --all-day=false` for timed items and pair with `task remind --minutes N` to avoid missing them
- Keep long-lived initiatives in tasklists and day-to-day items in the default inbox
- For large one-off operations, a script that calls `lark task create` per row from a CSV is the pragmatic path
