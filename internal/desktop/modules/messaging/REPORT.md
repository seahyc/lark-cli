# Messaging desktop module research

The installed Lark 8.0.3 bundles provide concrete wrapper calls for cloud
drafts, Favorites, shortcuts, feed folders, scheduled messages, and a bounded
message metadata lookup. Every registered read has both a request callsite and
a response consumer; local tests cover its strict payload and projection.
Draft projections always include `drafts`, `total`, and `truncated`, including
when no drafts are returned.

Favorites is also schema-verified. Its Favorites UI wrapper transforms
`{time, count}` to `{time: String(time), count}` and destructures
`favoritesIds`, `entity`, `hasMore`, and `minTime`. The module returns those
pagination fields plus the matching `entity.favorites` records, bounded by 100
ids and finite nested content. The UI then separately resolves message details
for message-type Favorites; this operation deliberately reports the Favorites
records only and does not claim that extra message hydration.

The wrappers run only in the signed-in desktop application's active account and
are feature-gated by the client before calling native transport. The traced
payloads contain no account, container, or realm field; those contexts are
selected by the desktop transport/session and have not been independently
live-verified. The module therefore does not guess or inject them.

`GET_SHORTCUTS` uses its UI's local-cache request. Its result is limited to
shortcut `id`, `feedType`, and channel `id`/`type`; shortcut channels are not
assumed to be chats and raw `entityPreview` is omitted. `MGET_MESSAGES` accepts
one native message ID and returns metadata only (`messageId`, `chatId`,
`threadId`, `senderId`), with no body projection.

Scheduled reads reproduce the UI's request scenes and default to its
force-server refresh. Feed folders return only `usedFolderIds`. The evidence
and remaining protocol gaps are in `docs/research/messaging-remaining.md`.

Persistent descriptors are separate in `mutations.js` and require the parent
runtime's preview, apply, identity guard, and readback. They include delete
draft, set last read, and narrowly scoped `CREATE_DRAFT` for plain text only.
The create-draft builder reproduces the shipped UUID-v4 rich-text graph and
caps text at 16 KiB UTF-8. An acknowledgement reports `verified:false`; no
live desktop execution has been performed by this module.

`PATCH_SCHEDULE_MESSAGE`, Favorites creation/deletion, feed mute/folder
changes, and `EDIT_MESSAGE` remain unexposed because their complete safe
request and readback schemas have not been established.
