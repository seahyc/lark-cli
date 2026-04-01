---
name: email
description: Read, search, send, reply, and manage emails from Lark Mail via IMAP/SMTP with local caching. Use when user asks about email, inbox, messages, sending, or replying.
---

# Email Management Skill

Read, search, send, reply, and manage emails from Lark Mail via the `lark` CLI using IMAP/SMTP with local caching.

## Setup

Before using mail commands, IMAP credentials must be configured:

```bash
lark mail setup
```

This will prompt for IMAP credentials. See: https://www.larksuite.com/hc/en-US/articles/378111206512-log-in-to-lark-mail-through-a-third-party-email-client

## Running Commands

Ensure `lark` is in your PATH, or use the full path to the binary. Set the config directory if not using the default:

```bash
lark mail <command>
# Or with explicit config:
LARK_CONFIG_DIR=/path/to/.lark lark mail <command>
```

## Commands Reference

### Check Status
```bash
lark mail status
```

Shows connection status and cache freshness. The `freshness` field indicates how stale the cached data is.

### List Mailboxes
```bash
lark mail list
```

### Sync Emails
Fetch new emails from the server into the local cache:

```bash
# Sync INBOX (default)
lark mail sync

# Sync with more parallel connections (faster for large mailboxes)
lark mail sync --workers 20

# Sync and cache full message bodies for later analysis
lark mail sync --include-bodies

# Sync specific mailbox
lark mail sync --mailbox Sent
```

Flags:
- `--mailbox`, `-m`: Mailbox to sync (default: INBOX)
- `--workers`, `-w`: Number of parallel connections (default: 10)
- `--include-bodies`: Also fetch and cache full RFC822 message bodies. Use when you need body-level analysis or plan to inspect many emails.

**Important**:
- Run sync before searching if you need fresh data
- Sync is resumable - if interrupted, running it again only fetches messages not already cached
- `--include-bodies` is intentionally opt-in because it is slower and stores more local data than header-only sync

### Search Emails
Search the local cache (fast, no network calls):

```bash
# List recent emails
lark mail search

# Filter by sender
lark mail search --from alice@example.com

# Filter by subject
lark mail search --subject "Q4 Report"

# Filter by body text (requires prior sync with --include-bodies)
lark mail search --body "sender authentication"

# Filter by date range
lark mail search --since 2025-01-01 --before 2025-01-15

# Combined filters
lark mail search --from boss@example.com --since 2025-01-01 --limit 10

# Search different mailbox
lark mail search --mailbox Sent
```

### View Email Content
```bash
lark mail show --uid <uid>
```

**Body search note**:
- `--body` only matches messages whose full RFC822 body was cached via `lark mail sync --include-bodies`
- messages synced without `--include-bodies` remain searchable by sender/subject/date only

The UID is obtained from search results.

### Download as .eml
```bash
lark mail fetch --uid <uid>
lark mail fetch --uid <uid> --output ./emails/
```

### Reply to Email
Reply to an email by UID, automatically setting threading headers:

```bash
# Simple reply
lark mail reply --uid 12345 --body "Thanks for the update"

# Reply with attachment
lark mail reply --uid 12345 --body "Please see attached" --attach report.pdf

# Reply from a different mailbox
lark mail reply --mailbox Archive --uid 12345 --body "Will review"

# Reply with body from file
lark mail reply --uid 12345 --body-file reply.txt --attach video.mp4

# Save reply as draft instead of sending
lark mail reply --uid 12345 --body "Will review" --draft
```

Flags:
- `--uid`: UID of email to reply to (required)
- `--mailbox`, `-m`: Mailbox containing the email (default: INBOX)
- `--body`: Reply body text
- `--body-file`: Read reply body from file
- `--attach`: File path(s) to attach (can be repeated)
- `--cc`: CC email address(es)
- `--draft`: Save as draft instead of sending

### Send Email
Send a new email via SMTP:

```bash
# Simple send
lark mail send --to user@example.com --subject "Hello" --body "Hi there"

# With CC and attachment
lark mail send --to user@example.com --cc other@example.com --subject "Report" --body "See attached" --attach file.pdf

# Thread a reply manually
lark mail send --to user@example.com --subject "Re: Thread" --body "Reply" --in-reply-to "<msgid@server>"
```

Flags:
- `--to`: Recipient email address(es) (required)
- `--subject`: Email subject
- `--body`: Email body text
- `--body-file`: Read body from file
- `--attach`: File path(s) to attach
- `--cc`: CC email address(es)
- `--in-reply-to`: Message-ID for threading
- `--references`: Message-ID chain for threading

**Note**: SMTP server is derived from the IMAP host (imap.x.com → smtp.x.com) unless smtp_host/smtp_port are set in mail.json.

### Save Draft
Save an email as a draft in the Drafts folder:

```bash
lark mail draft --to user@example.com --subject "Hello" --body "Hi there"
lark mail draft --to user@example.com --subject "Re: Thread" --body-file msg.txt --attach file.pdf
```

Flags are the same as `send`.

### Delete Email
Flag an email as deleted and expunge:

```bash
lark mail delete --uid 12345
lark mail delete --mailbox Archive --uid 12345
```

### Move Email
Move an email to a different mailbox:

```bash
lark mail move --uid 12345 --dest Archive
lark mail move --mailbox INBOX --uid 12345 --dest Trash
```

## Output Formats

All commands output JSON.

### mail status Output
```json
{
  "configured": true,
  "host": "imap.larksuite.com",
  "port": 993,
  "username": "user@example.com",
  "connection": "ok",
  "cache": {
    "last_sync": "2025-01-14T10:30:00+08:00",
    "freshness": "15 minutes ago",
    "uidvalidity": 12345,
    "last_uid": 4521
  }
}
```

### mail search Output
```json
{
  "mailbox": "INBOX",
  "last_sync": "2025-01-14T10:30:00+08:00",
  "freshness": "15 minutes ago",
  "total_cached": 1523,
  "results": [
    {
      "uid": 4521,
      "message_id": "<abc123@mail.example.com>",
      "date": "2025-01-14T09:15:00+08:00",
      "from_addr": "alice@example.com",
      "from_name": "Alice",
      "subject": "Q4 Report"
    }
  ],
  "count": 1
}
```

### mail show Output
```json
{
  "uid": 4521,
  "from": {
    "email": "alice@example.com",
    "name": "Alice"
  },
  "subject": "Q4 Report",
  "date": "2025-01-14T09:15:00+08:00",
  "message_id": "<abc123@mail.example.com>",
  "body": "Hi,\n\nPlease find the Q4 report attached...\n\nBest,\nAlice"
}
```

### mail sync Output
```json
{
  "mailbox": "INBOX",
  "new_messages": 5,
  "total_cached": 1523,
  "bodies_cached": 1523,
  "message": "synced 5 new messages"
}
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
- `CONNECTION_ERROR` - IMAP connection failed (check credentials)
- `SCOPE_ERROR` - Missing mail permissions. Run `lark auth login --add --scopes mail`
- `SYNC_ERROR` - Failed to sync emails
- `SEARCH_ERROR` - Cache query failed
- `VALIDATION_ERROR` - Missing required fields (e.g., --uid)
- `IO_ERROR` - File system error

## Required Permissions

This skill requires the `mail` scope group. If you see a `SCOPE_ERROR`, the user needs to add mail permissions:

```bash
lark auth login --add --scopes mail
```

To check current permissions:
```bash
lark auth status
```

## Typical Workflow

1. **Check cache freshness**:
   ```bash
   lark mail status
   ```

2. **Sync if needed** (based on freshness):
   ```bash
   lark mail sync
   ```

3. **Search emails**:
   ```bash
   lark mail search --from boss@example.com --since 2025-01-01
   ```

4. **View email details** using UID from search:
   ```bash
   lark mail show --uid 4521
   ```

5. **Download if needed**:
   ```bash
   lark mail fetch --uid 4521
   ```
