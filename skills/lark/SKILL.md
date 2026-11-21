---
name: lark
description: Work in Lark (Feishu) with the `lark` CLI - messages and group chats, docs, sheets, Bitable bases, calendar and meetings, email, tasks, approvals, attendance, contacts, minutes and transcripts, slides, event subscriptions, and raw Open API calls. Use whenever the user mentions Lark or Feishu, a Lark link, or a colleague's message, doc, sheet, meeting or task there.
---

# Lark

Everything goes through the `lark` CLI. Find the area below, read its reference, and follow it.

## Before any command

- `lark` must be on PATH. If a project names a config directory, prefix commands with it, e.g.
  `LARK_CONFIG_DIR=/path/to/.lark lark ...`; otherwise the default config is used.
- Run `lark auth status` first when a command fails or on first use, to tell a missing login from a
  missing scope or a wrong ID.
- Sending, replying, reacting, editing, creating or deleting reaches real colleagues. Follow the
  host's rules for outbound messages and approvals before any such command.

## Areas

| Area | Covers | Read |
|---|---|---|
| api | Execute raw HTTP requests against any Lark Open API endpoint via the `lark api` passthrough | [references/api.md](references/api.md) |
| approval | Manage Lark approval workflows - list pending tasks, view instance details, approve/reject/transfer tasks, cancel instances, cc additional users | [references/approval.md](references/approval.md) |
| attendance | Query Lark Attendance check-in records from the CLI - list clock-in/out tasks for the current user or a teammate over a date range | [references/attendance.md](references/attendance.md) |
| bitable | Full CRUD over Lark Bitable (multi-dimensional databases) - create Bitables, manage tables/fields, read/write/search records, and do batch operations | [references/bitable.md](references/bitable.md) |
| calendar | Manage Lark calendar - view schedule, create/update/delete events, check availability, find meeting slots, RSVP to invitations | [references/calendar.md](references/calendar.md) |
| contacts | Look up employee information via Lark - find colleagues by ID, list department members, search users by name, search departments | [references/contacts.md](references/contacts.md) |
| docs | Read and write Lark documents - get content as markdown or blocks, create new documents, append content (text, headings, lists, code), download images, list folders | [references/docs.md](references/docs.md) |
| email | Read, search, send, reply, schedule, and manage emails from Lark Mail via IMAP/SMTP and the native Lark Mail API | [references/email.md](references/email.md) |
| events | Subscribe to real-time Lark events (messages, reactions, calendar, etc.) via long-polling, streamed as NDJSON | [references/events.md](references/events.md) |
| meetings | Search Lark video meetings and fetch their notes, transcripts, and recording info | [references/meetings.md](references/meetings.md) |
| messages | Chat and message operations in Lark - send/read/reply/recall/edit messages, react, download resources, and manage groups (create chats, add/remove members, pin/unpin messages, get share links, search messages across chats, forward or merge-forward messages) | [references/messages.md](references/messages.md) |
| minutes | Access Lark Minutes recordings - get metadata, export transcripts, download audio/video | [references/minutes.md](references/minutes.md) |
| sheets | Read and write Lark Sheets (spreadsheets) - list sheets, read cell data, write values, create spreadsheets, add tabs, apply formatting, resize columns | [references/sheets.md](references/sheets.md) |
| slides | Create and edit Lark Slides presentations from the CLI - new deck, append/delete slides via XML, upload media for embedding | [references/slides.md](references/slides.md) |
| tasks | Create, update, and manage Lark Tasks - list tasks, view details, create/update tasks with due dates, complete/reopen, assign to people, manage subtasks, add comments, set reminders, and organize into tasklists | [references/tasks.md](references/tasks.md) |

When an endpoint is missing from the CLI, use the api reference for a raw call before concluding it
cannot be done.
