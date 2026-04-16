---
name: api
description: Execute raw HTTP requests against any Lark Open API endpoint via the `lark api` passthrough - use when you need an endpoint that is not covered by the higher-level skills, want to prototype a new call, or need to upload/download files through the API.
---

# Lark Raw API Passthrough Skill

Call any Lark Open API endpoint directly using the `lark api` command. This is the escape hatch for when no dedicated skill covers what you need.

## 🤖 Capabilities and Use Cases

- Call any Lark Open API endpoint (GET, POST, PUT, PATCH, DELETE) with full control over headers and body
- Choose between bot token (`--as bot`) and user token (`--as user`) auth
- Send JSON bodies, query params, and multipart form data (including file uploads)
- Download binary files to disk with `--output`
- Prototype and debug new API integrations before wiring a dedicated command

## 🚀 Quick Reference

**GET with query params:**
```bash
lark api GET /open-apis/im/v1/chats --params '{"page_size":20}' --as user
```

**POST JSON:**
```bash
lark api POST /open-apis/task/v2/tasks --data '{"summary":"Ship it"}' --as user
```

**File upload (multipart):**
```bash
lark api POST /open-apis/im/v1/files \
  --form file=@./report.pdf \
  --form file_type=stream
```

**File download:**
```bash
lark api GET /open-apis/im/v1/messages/om_xxx/resources/file_xxx \
  --output ./download.pdf
```

**DELETE:**
```bash
lark api DELETE /open-apis/im/v1/messages/om_xxx --as bot
```

## Commands Reference

### `lark api <METHOD> <path>`

Executes a raw HTTP request against the Lark Open API.

Available flags:
- `<METHOD>` (positional, required): `GET`, `POST`, `PUT`, `PATCH`, `DELETE`
- `<path>` (positional, required): API path, typically starting with `/open-apis/...`. The host (`https://open.larksuite.com`) is added automatically.
- `--as`: Identity — `bot` (default) uses the tenant access token, `user` uses your user access token
- `--params`: JSON object of query parameters, e.g. `'{"page_size":20}'`
- `--data`: JSON request body as a string, e.g. `'{"summary":"Ship it"}'`
- `--form`: Multipart form field; repeatable. Use `key=value` for text fields or `key=@./path` for file uploads
- `--output`: Save response body to a file path (use for binary downloads)

## Choosing `--as user` vs `--as bot`

| Use `--as user` when... | Use `--as bot` (default) when... |
|---|---|
| The endpoint requires user scopes (email, drive, calendar) | The endpoint requires app/tenant scopes (`tenant_access_token`) |
| You want actions attributed to your user identity | The endpoint explicitly documents `tenant_access_token` |
| Bot identity restrictions block the call (e.g. DMs) | The bot has been added to the target resource |

If a call fails with a scope error, try the other identity first.

## Tips

- Use `lark api` to test a new endpoint before deciding whether to add a dedicated command
- `--params` values must be strings in JSON (e.g. `'{"page_size":"20"}'` also works; numbers are coerced to strings)
- Combining `--data` and `--form` is not supported — a request is either JSON or multipart
- Path should start with `/open-apis/...`; the client prepends the Lark host
- For pagination, look for `page_token` / `has_more` in responses and re-call with `--params '{"page_token":"..."}'`

## Output Format

Responses are returned as JSON on stdout. For binary payloads (file downloads), use `--output` to stream to a file — stdout will then contain a small JSON summary.

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
- `SCOPE_ERROR` — Missing the scope for this endpoint. Try the other identity (`--as user` vs `--as bot`), or add the scope with `lark auth login --add --scopes <group>`.
- `API_ERROR` — Lark API returned a non-zero code; the message includes Lark's own error text and request id

## When to Graduate from Raw API to a Skill

Raw API is ideal for prototyping. If you find yourself calling the same endpoint repeatedly, consider whether a dedicated skill already covers it:

- Messages, chats, reactions → `lark msg` / `lark chat` (see `lark-messages` skill)
- Tasks → `lark task` (see `lark-tasks` skill)
- Bitable databases → `lark bitable` (see `lark-bitable` skill)
- Docs, sheets, wiki → `lark doc` / `lark sheet` / `lark wiki`
- Calendar, contacts, mail → `lark cal` / `lark contact` / `lark mail`
- Meetings, minutes → `lark meetings` / `lark minutes`
- Event subscriptions → `lark events`

Use the raw `lark api` when the dedicated command doesn't yet expose what you need.
