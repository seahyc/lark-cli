---
name: contacts
description: Look up employee information via Lark - find colleagues by ID, list department members, search users by name, search departments. Use when user asks about a person, colleague, job title, department, or org structure.
---

# Contacts Lookup Skill

Look up employee information via the `lark` CLI.

## Running Commands

Ensure `lark` is in your PATH, or use the full path to the binary. Set the config directory if not using the default:

```bash
lark contact <command>
# Or with explicit config:
LARK_CONFIG_DIR=/path/to/.lark lark contact <command>
```

## Commands Reference

### Get User by ID
```bash
# Look up by open_id (default)
lark contact get ou_xxxx

# Look up by user_id
lark contact get 12345 --id-type user_id
```

Output:
```json
{
  "user_id": "ou_xxx",
  "open_id": "ou_xxx",
  "name": "Jane Doe",
  "email": "jane@example.com"
}
```

Email is returned only when the `contact:user.email:readonly` scope is granted (part of the `contacts` scope group as of the latest CLI). The Glints tenant does not expose `job_title` / `department` / `en_name` to this app's user token — the CLI surfaces whatever Lark returns, which is name + email for cross-department lookups.

### List Users in Department
```bash
# List users in root department
lark contact list-dept

# List users in specific department
lark contact list-dept od_xxxx

# Specify page size (max 50, default 50)
lark contact list-dept od_xxxx --page-size 25
```

Available flags: `--page-size` (number of results per page, max 50)

Output:
```json
{
  "contacts": [
    {
      "user_id": "ou_xxx",
      "name": "Alice"
    }
  ],
  "count": 1
}
```

**Tenant restriction:** the Glints workspace gates department member enumeration with admin policy. Calling `list-dept` against most department IDs returns `API_ERROR (code 40004): no dept authority error`. Root department (`0`) returns empty. If you need to find someone, prefer `lark contact search "Name"` followed by `lark contact get <open_id>`.

### Search Users by Name
```bash
lark contact search "Jane"
lark contact search "John Smith"
```

**IMPORTANT — strip diacritics / use the ROMANIZED name.** Lark's directory stores
ASCII/romanized names, and the search endpoint does NOT fold accents. Searching an
accented form returns **0 results** even when the person exists:
`lark contact search "Mai Thị Đỗ"` → `count: 0`, but
`lark contact search "Mai Thi Do"` → the match. So for Vietnamese/accented names,
search the de-accented spelling (Thị→Thi, Đỗ→Do, é→e, ñ→n). If a full-name search
misses, retry with a single de-accented token (e.g. just `"Mai"`).

Output:
```json
{
  "contacts": [
    {
      "user_id": "ou_xxx",
      "open_id": "ou_xxx",
      "name": "Jane Doe"
    }
  ],
  "count": 1
}
```

The `search` endpoint returns minimal identity (open_id + name). To get email, follow up with `lark contact get <open_id>`.

### Search Departments
```bash
lark contact search-dept "Engineering"
```

Output:
```json
{
  "departments": [
    {
      "department_id": "od_xxx",
      "name": "Engineering",
      "member_count": 42
    }
  ],
  "count": 1
}
```

## Integration with Calendar

When showing calendar events with attendees, you can enrich attendee info:

1. Get attendee `open_id` from calendar event
2. Use `contact get <open_id>` to fetch name + email
3. Present enriched attendee info to user

Example workflow:
```bash
# Get event with attendees
lark cal show <event_id>
# Returns attendees with open_id

# Look up each attendee
lark contact get ou_attendee_id
# Returns name and email
```

## Output Format

All commands output JSON. Format appropriately when presenting to user.

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
- `AUTH_ERROR` - Need to run `lark auth login`
- `SCOPE_ERROR` - Missing contacts permissions. Run `lark auth login --add --scopes contacts`
- `NOT_FOUND` - User or department not found
- `API_ERROR` - Lark API issue

## Required Permissions

This skill requires the `contacts` scope group. If you see a `SCOPE_ERROR`, the user needs to add contacts permissions:

```bash
lark auth login --add --scopes contacts
```

To check current permissions:
```bash
lark auth status
```

## Notes

- All four commands (`get`, `search`, `list-dept`, `search-dept`) use the user token; run `lark auth login` (or `lark auth login --add --scopes contacts`) before use.
- `list-dept` is heavily restricted in the Glints tenant (`no dept authority error`); prefer `search` + `get` to find people.
- Department IDs typically start with `od_` (or `od-`).
- User open_ids typically start with `ou_`.
