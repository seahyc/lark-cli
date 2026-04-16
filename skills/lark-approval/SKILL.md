---
name: approval
description: Manage Lark approval workflows - list pending tasks, view instance details, approve/reject/transfer tasks, cancel instances, cc additional users - use when the user needs to act on approvals from the CLI or automate approval flows.
---

# Lark Approval Skill

Drive the Lark approval workflow from the CLI: list pending tasks, fetch instance details, and act on them (approve, reject, transfer, cancel, cc).

## 🤖 Capabilities and Use Cases

- List approval tasks waiting on the current user
- Fetch the full form data / timeline of a specific approval instance
- Approve or reject pending tasks with an optional comment
- Transfer a task to a different approver
- Cancel an approval instance the user started
- CC an instance to additional users (for awareness, not action)

## 🚀 Quick Reference

**List my pending tasks:**
```bash
lark approval list
```

**Get details of one instance:**
```bash
lark approval get <instance-code>
```

**Approve a task (requires all IDs — typically grabbed from `approval list`):**
```bash
lark approval approve <task-id> \
  --approval-code ABC123 \
  --instance-code DEF456 \
  --user ou_abc... \
  --comment "LGTM"
```

**Reject:**
```bash
lark approval reject <task-id> \
  --approval-code ABC123 --instance-code DEF456 --user ou_abc... \
  --comment "Budget not approved"
```

**Transfer to another approver:**
```bash
lark approval transfer <task-id> \
  --approval-code ABC123 --instance-code DEF456 --user ou_abc... \
  --to ou_xyz...
```

**Cancel an instance you started:**
```bash
lark approval cancel <instance-code> --approval-code ABC123 --user ou_abc...
```

**CC users on an instance:**
```bash
lark approval cc <instance-code> --approval-code ABC123 --user ou_abc... \
  --to ou_xyz...,ou_pqr...
```

## Commands Reference

### `lark approval list`

Lists approval tasks pending for the acting user.

Flags:
- `--user`: user open_id to query for (defaults to current user from tenant token)
- `--page-size`: tasks per page (default 20)
- `--page-token`: pagination token
- `--locale`: `en-US` (default), `zh-CN`, `ja-JP`

Output:
```json
{
  "tasks": [
    {
      "task_id": "...",
      "instance_code": "...",
      "approval_code": "...",
      "title": "Leave Request",
      "status": "PENDING"
    }
  ],
  "count": 1,
  "has_more": false,
  "page_token": ""
}
```

### `lark approval get <instance-code>`

Fetches full detail of an approval instance including form data, timeline, task list, comments.

### `lark approval approve|reject|transfer <task-id>`

Acts on a pending approval task. All three require `--approval-code`, `--instance-code`, and `--user` (the acting approver's open_id). `--comment` is optional for approve/reject, recommended for reject. `transfer` also requires `--to` with the new approver's open_id.

### `lark approval cancel <instance-code>`

Cancels an approval instance. Requires `--approval-code` and `--user` (must be the instance initiator).

### `lark approval cc <instance-code>`

Adds CC recipients to an approval instance. Requires `--approval-code`, `--user`, and `--to` with a comma-separated list of open_ids.

## Error Handling

Errors return JSON with a non-zero exit:
```json
{"error": true, "code": "API_ERROR", "message": "..."}
```

Common codes:
- `AUTH_ERROR` — run `lark auth login --add --scopes approval`
- `SCOPE_ERROR` — app is missing approval:* scopes
- `VALIDATION_ERROR` — required flags missing
- `API_ERROR` — Lark rejected the request (often: wrong approval_code/instance_code/user combination, or the task has already been acted on)

## Required Permissions

Scopes in the `approval` group:
- `approval:approval`, `approval:approval:readonly`
- `approval:instance:write`
- `approval:task:write`

Grant with:
```bash
lark auth login --add --scopes approval
```

## Notes

- All commands use the tenant token. Lark's approval action endpoints require a tenant token even for user-driven actions — the `--user` flag supplies the acting identity.
- The `approval_code` is the template ID (per form definition); the `instance_code` is the specific running instance. Both come back in `lark approval list` output.
- A `task_id` is single-use: once approved/rejected/transferred, acting on it again will return an API error.
