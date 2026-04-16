---
name: slides
description: Create and edit Lark Slides presentations from the CLI - new deck, append/delete slides via XML, upload media for embedding - use when the user wants to build slide decks programmatically or script changes to an existing presentation.
---

# Lark Slides Skill

Build and edit Lark Slides presentations via the `lark slides` CLI.

## 🤖 Capabilities and Use Cases

- Create an empty presentation in a chosen folder
- Read the current state of a presentation (slide list, metadata)
- Append a new slide using Lark's XML slide definition
- Delete a specific slide by ID
- Upload images/media to Drive and get a `file_token` for embedding in slide XML

## 🚀 Quick Reference

**Create a deck:**
```bash
lark slides create --title "Q4 Plan" --folder fldcnxxxxx
```

**Read it back:**
```bash
lark slides get <presentation-id>
```

**Append a slide from an XML file:**
```bash
lark slides slide create <presentation-id> --xml-file slide1.xml
```

**Delete a slide:**
```bash
lark slides slide delete <presentation-id> --slide-id slide_xxx
```

**Upload an image to reference in slide XML:**
```bash
lark slides media upload cover.png
# → { "file_token": "boxcn..." }
```

## Commands Reference

### `lark slides create`

Creates a new empty presentation.

Flags:
- `--title` (required): title of the presentation
- `--folder`: destination `folder_token` (default: root drive)

### `lark slides get <presentation-id>`

Returns the raw XML-presentation metadata (slide list, dimensions, etc.).

### `lark slides slide create <presentation-id>`

Appends a new slide using Lark's XML slide definition.

Flags:
- `--xml`: inline XML string
- `--xml-file`: path to XML file (use `-` for stdin)

One of `--xml` or `--xml-file` is required.

### `lark slides slide delete <presentation-id>`

Deletes a slide by its ID.

Flags:
- `--slide-id` (required): the slide ID from `slides get`

### `lark slides media upload <file>`

Uploads a local file to Drive and returns a `file_token` suitable for embedding in slide XML.

Flags:
- `--parent-type`: parent type hint (default `slides_image`; also `docx_image` etc.)
- `--parent-node`: parent node token (optional)

## Error Handling

JSON errors on failure with non-zero exit:
```json
{"error": true, "code": "API_ERROR", "message": "..."}
```

Common codes:
- `AUTH_ERROR` — run `lark auth login --add --scopes slides`
- `SCOPE_ERROR` — missing slides:presentation:* scopes
- `VALIDATION_ERROR` — required flag missing (title, xml, slide-id)
- `FILE_ERROR` — cannot read `--xml-file` or upload target
- `API_ERROR` — Lark rejected the request (malformed XML, bad ID, etc.)

## Required Permissions

Scopes in the `slides` group:
- `slides:presentation:create`, `slides:presentation:read`
- `slides:presentation:write_only`, `slides:presentation:update`
- `docs:document.media:upload`
- `drive:drive`

Grant with:
```bash
lark auth login --add --scopes slides
```

## Notes

- The XML slide format follows Lark's Slides XML schema — refer to the open platform docs for the full element/attribute list.
- Uploaded media returns a `file_token`; reference it in XML via `<image file_token="..."/>` (or the equivalent Lark tag).
- For bulk edits, script a loop: `slides get` to enumerate, then `slide delete`/`slide create` per slide.
