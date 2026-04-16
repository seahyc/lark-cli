---
name: bitable
description: Full CRUD over Lark Bitable (multi-dimensional databases) - create Bitables, manage tables/fields, read/write/search records, and do batch operations. Use when the user wants to query, update, or build structured data stored in Lark Base (Bitable).
---

# Lark Bitable Skill

Access and manage Lark Bitable (database) content via the `lark` CLI.

## 🤖 Capabilities and Use Cases

- List, inspect, and read rows (records) from Bitable tables
- Create entire Bitable apps, tables within them, and fields (columns)
- Update and delete tables, fields, and records
- Batch create and batch delete records for bulk edits
- Search records with filter/sort expressions (advanced query)
- Accept raw app tokens OR full Bitable URLs (`https://xxx.larksuite.com/base/<token>`)

## 🚀 Quick Reference

**Browse / read:**
```bash
lark bitable tables <app_token>
lark bitable fields <app_token> <table_id>
lark bitable records <app_token> <table_id> --limit 100
lark bitable get <app_token>
```

**Create Bitable + tables + fields:**
```bash
lark bitable create --name "My Tracker"
lark bitable table create --app-token <t> --name "Tasks"
lark bitable field create --app-token <t> --table-id <tbl> --name "Status" --type 3 \
  --property '{"options":[{"name":"Open"},{"name":"Done"}]}'
```

**Write / delete records:**
```bash
lark bitable record create --app-token <t> --table-id <tbl> --fields '{"Name":"Alpha","Status":"Open"}'
lark bitable record update --app-token <t> --table-id <tbl> --record-id <r> --fields '{"Status":"Done"}'
lark bitable record delete --app-token <t> --table-id <tbl> --record-id <r>
```

**Batch:**
```bash
lark bitable record batch-create --app-token <t> --table-id <tbl> \
  --records '[{"Name":"A"},{"Name":"B"},{"Name":"C"}]'
lark bitable record batch-delete --app-token <t> --table-id <tbl> \
  --record-ids rec111,rec222,rec333
```

**Search (filter + sort):**
```bash
lark bitable record search --app-token <t> --table-id <tbl> \
  --filter '{"conjunction":"and","conditions":[{"field_name":"Status","operator":"is","value":["Open"]}]}' \
  --sort '[{"field_name":"Created","desc":true}]' \
  --limit 20
```

## Commands Reference

### Read

#### List Tables
```bash
lark bitable tables <app_token_or_url>
```
Output: `{ "app_token": "...", "tables": [...], "count": N }`

#### Get Bitable Metadata
```bash
lark bitable get <app_token_or_url>
```
Returns name, revision, icon, etc.

#### List Fields
```bash
lark bitable fields <app_token> <table_id>
```
Output: `{ "app_token": "...", "table_id": "...", "fields": [ {field_id, field_name, type, is_primary, property}, ...], "count": N }`

Field type numbers (from Lark):
| # | Type | # | Type |
|---|---|---|---|
| 1 | text | 13 | phone |
| 2 | number | 15 | url |
| 3 | single_select | 17 | attachment |
| 4 | multi_select | 18 | link (linked record) |
| 5 | date | 19 | lookup |
| 7 | checkbox | 20 | formula |
| 11 | person | 21 | duplex_link |

See the Lark docs for the full type table.

#### List Records
```bash
lark bitable records <app_token> <table_id> [--limit N] [--view <view_id>] [--filter <expression>]
```
Basic read. Use `record search` for complex filter/sort.

### Write

#### Create a Bitable
```bash
lark bitable create --name "Tracker" [--folder fldABC]
```
Creates a standalone Bitable app. `--folder` places it inside a specific Drive folder.

#### Create / Delete Table
```bash
lark bitable table create --app-token <t> --name "Tasks"
lark bitable table delete --app-token <t> --table-id <tbl>
```

#### Create / Update / Delete Field
```bash
lark bitable field create --app-token <t> --table-id <tbl> --name "Status" --type 3 \
  --property '{"options":[{"name":"Todo"},{"name":"Done"}]}'

lark bitable field update --app-token <t> --table-id <tbl> --field-id <f> \
  --name "State" --type 3 --property '{"options":[{"name":"New"},{"name":"Done"}]}'

lark bitable field delete --app-token <t> --table-id <tbl> --field-id <f>
```

### Records

#### Create / Update / Delete
```bash
lark bitable record create --app-token <t> --table-id <tbl> --fields '{"Name":"Alpha"}'
lark bitable record update --app-token <t> --table-id <tbl> --record-id <r> --fields '{"Status":"Done"}'
lark bitable record delete --app-token <t> --table-id <tbl> --record-id <r>
```

`--fields` is a JSON object keyed by **field name** (not field_id), matching what the Lark API expects. Use `bitable fields` first to see the exact names.

#### Batch

```bash
lark bitable record batch-create --app-token <t> --table-id <tbl> \
  --records '[{"Name":"A"},{"Name":"B"}]'

lark bitable record batch-delete --app-token <t> --table-id <tbl> \
  --record-ids rec111,rec222,rec333
```

#### Search (advanced filter/sort)

```bash
lark bitable record search --app-token <t> --table-id <tbl> \
  --filter '{"conjunction":"and","conditions":[{"field_name":"Status","operator":"is","value":["Open"]}]}' \
  --sort '[{"field_name":"Created","desc":true}]' \
  --field Name --field Status \
  --limit 100
```

Flags:
- `--filter`: JSON object in Lark filter format (`conjunction` + `conditions[]`)
- `--sort`: JSON array of sort specs
- `--field`: Field names to return (repeatable) — narrows the result payload
- `--view`: View ID
- `--limit`: Maximum records

## Extracting IDs from URLs

| URL | Extracted |
|---|---|
| `https://xxx.larksuite.com/base/ABC123xyz` | app_token = `ABC123xyz` |
| `https://xxx.larksuite.com/base/ABC123xyz?table=tblXYZ789` | app_token = `ABC123xyz`, table_id = `tblXYZ789` |

The `--app-token` flag accepts either the raw token or the full URL; the CLI extracts the token via regex.

## Workflow Example: Build a Tracker from Scratch

```bash
# 1. Create the Bitable
lark bitable create --name "Q2 Roadmap"
# → { "app_token": "ABC123xyz", ... }

# 2. Add a table
lark bitable table create --app-token ABC123xyz --name "Items"
# → { "table_id": "tblXYZ789" }

# 3. Add fields
lark bitable field create --app-token ABC123xyz --table-id tblXYZ789 --name "Title" --type 1
lark bitable field create --app-token ABC123xyz --table-id tblXYZ789 --name "Status" --type 3 \
  --property '{"options":[{"name":"Todo"},{"name":"Doing"},{"name":"Done"}]}'

# 4. Bulk insert
lark bitable record batch-create --app-token ABC123xyz --table-id tblXYZ789 \
  --records '[{"Title":"Ship auth v2","Status":"Todo"},{"Title":"Write docs","Status":"Doing"}]'

# 5. Query
lark bitable record search --app-token ABC123xyz --table-id tblXYZ789 \
  --filter '{"conjunction":"and","conditions":[{"field_name":"Status","operator":"is","value":["Todo"]}]}'
```

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
- `SCOPE_ERROR` — Missing bitable permissions. Run `lark auth login --add --scopes bitable`
- `API_ERROR` — Lark API issue (permissions on the specific Bitable, invalid field type, etc.)
- `VALIDATION_ERROR` — Bad `--fields` / `--records` / `--filter` JSON

## Required Permissions

Read-only uses `bitable:app:readonly`. Write commands require full `bitable:app` scope.

```bash
lark auth login --add --scopes bitable
```

To check current permissions:
```bash
lark auth status
```

## Tips

- **Field names, not IDs**, in `--fields` / `--records` / filter conditions.
- Date fields are returned and accepted as **Unix timestamps in milliseconds**.
- Select fields are passed by **option text**, not option ID.
- Person fields accept `[{"id":"ou_xxx"}]`; resolve names later via `lark contact get`.
- `--property` on `field create/update` is the raw Lark `property` object — select options go inside `options[]`, number formats go in `formatter`, etc. See Lark field type docs.
- Batch operations are per-request limited — very large datasets should be chunked (Lark limit is typically 500 records per call).
- Always use `--limit` when you're exploring unfamiliar tables; some tables have hundreds of thousands of rows.

## Output Format

All commands output JSON.
