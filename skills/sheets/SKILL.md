---
name: sheets
description: Read and query Lark Sheets (spreadsheets) - list sheets in a spreadsheet, read cell data. Use when user asks about a spreadsheet, wants to read data from a Lark sheet, or mentions a spreadsheet URL/ID.
---

# Lark Sheets Skill

Read and query Lark Sheets (spreadsheets) via the `lark` CLI.

## Running Commands

Ensure `lark` is in your PATH, or use the full path to the binary. Set the config directory if not using the default:

```bash
lark sheet <command>
# Or with explicit config:
LARK_CONFIG_DIR=/Users/yingcong/Code/lark-cli/.lark lark sheet <command>
```

## Commands Reference

### List Sheets in a Spreadsheet

```bash
lark sheet list <spreadsheet_token>
```

Lists all sheets (tabs) within a Lark spreadsheet. Returns sheet IDs, titles, dimensions, and hidden status.

Output:
```json
{
  "spreadsheet_token": "T4mHsrFyzhXrj0tVzRslUGx8gkA",
  "sheets": [
    {
      "sheet_id": "abc123",
      "title": "Sheet1",
      "index": 0,
      "row_count": 100,
      "column_count": 10
    },
    {
      "sheet_id": "def456",
      "title": "Sheet2",
      "index": 1,
      "hidden": true,
      "row_count": 50,
      "column_count": 5
    }
  ],
  "count": 2
}
```

Fields:
- `sheet_id`: The unique ID of the sheet (use this with `--sheet` flag)
- `title`: Display name of the sheet
- `index`: Position of the sheet (0-indexed)
- `hidden`: Whether the sheet is hidden in the UI
- `row_count` / `column_count`: Dimensions of the sheet

### Read Sheet Data

```bash
lark sheet read <spreadsheet_token> [--sheet <sheet_id>] [--range A1:Z100]
```

Reads cell values from a Lark spreadsheet.

Options:
- `--sheet`: Sheet ID to read from (default: first sheet by index)
- `--range`: Cell range to read (e.g., `A1:Z100`). Default: all data up to 1000 rows

Output:
```json
{
  "spreadsheet_token": "T4mHsrFyzhXrj0tVzRslUGx8gkA",
  "sheet_id": "abc123",
  "range": "abc123!A1:D10",
  "row_count": 10,
  "column_count": 4,
  "values": [
    ["Header1", "Header2", "Header3", "Header4"],
    ["Value1", "Value2", 123, true],
    ["Row2Val1", null, 456, false]
  ]
}
```

**Note:** Cell values preserve their types (string, number, boolean). Empty cells may appear as `null` or be omitted from rows. Some cells with rich formatting may return structured objects instead of plain values.

## Extracting IDs from URLs

The spreadsheet_token is from the spreadsheet URL:
- URL: `https://xxx.larksuite.com/sheets/T4mHsrFyzhXrj0tVzRslUGx8gkA`
- Token: `T4mHsrFyzhXrj0tVzRslUGx8gkA`

## Which Command to Use

| Use Case | Command | Notes |
|----------|---------|-------|
| Browse sheets/tabs | `sheet list` | See all sheets and dimensions |
| Read specific data | `sheet read --range` | Target specific cells |
| Read full sheet | `sheet read --sheet` | Up to 1000 rows |
| Read first sheet | `sheet read` | Auto-selects first by index |

## Workflow Examples

### Get Overview of a Spreadsheet

```bash
# List all sheets first
lark sheet list T4mHsrFyzhXrj0tVzRslUGx8gkA

# Then read specific sheet
lark sheet read T4mHsrFyzhXrj0tVzRslUGx8gkA --sheet abc123 --range A1:D20
```

### Read Header Row Only

```bash
lark sheet read T4mHsrFyzhXrj0tVzRslUGx8gkA --range A1:Z1
```

### Read First 50 Rows of Specific Sheet

```bash
lark sheet read T4mHsrFyzhXrj0tVzRslUGx8gkA --sheet def456 --range A1:Z50
```

## Efficient Extraction with jq

For large spreadsheets, use `jq` to extract specific data without loading everything into context.

### Get Column Headers

```bash
lark sheet read <token> --range A1:Z1 | jq '.values[0]'
```

### Get Row Count

```bash
lark sheet read <token> | jq '.row_count'
```

### Extract Specific Column (Column B)

```bash
lark sheet read <token> | jq '[.values[] | .[1]]'
```

### Find Rows Matching a Value

```bash
lark sheet read <token> | jq '[.values[] | select(.[0] == "SearchValue")]'
```

### Get First N Rows

```bash
lark sheet read <token> | jq '.values[:10]'
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
- `SCOPE_ERROR` - Missing documents permissions. Run `lark auth login --add --scopes documents`
- `API_ERROR` - Lark API issue (often permissions)
- `NO_SHEETS` - Spreadsheet has no sheets

## Required Permissions

This skill requires the `documents` scope group (uses `drive:drive:readonly`). If you see a `SCOPE_ERROR`, the user needs to add documents permissions:

```bash
lark auth login --add --scopes documents
```

To check current permissions:
```bash
lark auth status
```

## Limitations

- Maximum 1000 rows read by default (use `--range` for specific cells)
- Column letters limited to A-Z (26 columns) when auto-detecting range
- Rich text cells may return structured objects instead of plain strings
- Some merged cells may have unexpected value placement
