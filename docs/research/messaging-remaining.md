# Messaging native bridge: remaining gaps

Static review used the installed Lark 8.0.3 ASAR bundles only; no desktop
bridge calls were made. A native declaration was not treated as a schema.

## Mapped reads

`GET_FAVORITE_INFO` is mapped by the Favorites UI. Its helper builds
`{favoriteIds: ids}` and the detail view turns a single requested ID into
`[id]`; it consumes `entity.favorites` when rendering the result. It is
registered as `messaging.favorite-detail`.

## Mapped but unregistered mutations

`DELETE_DRAFT` takes `{draftId}`. The cloud-draft wrapper calls it and then
may update local chat/thread draft metadata. It is retained in `mutations.js`
because deletion changes synced user drafts. Before any apply, read the draft
again and require the user/parent runtime to guard the intended identity; an
empty or failed response is not proof that the target draft was deleted.

`CREATE_CHAT_LAST_READ_POSITION` takes `{chatId, position, offset}` where the
UI defaults position to `-1`, defaults offset to `0`, and sends
`Math.round(offset).toString()`. It changes inbox/read state and is likewise
unregistered. Verify using a fresh chat/feed read after a parent-controlled
apply; do not retry a timed-out write automatically.

`CREATE_DRAFT` is exposed only as `messaging.create-text-draft` in the separate
mutation registry. It accepts only native `chatId` plus nonempty plain text up
to 16 KiB UTF-8. Its exact nested rich-text graph, including UUID-v4 element
ID generation, is documented in `chat-draft-protocol.md`; it never accepts a
generic draft object. Native acknowledgement remains insufficient proof, so
the parent must fetch the same chat draft before treating an apply as verified.

`EDIT_MESSAGE` is not exposed. Static code establishes edit drafts but this
pass did not locate the final `EDIT_MESSAGE` request transform or sender/owner
validation. Do not infer that a public `om_` ID is accepted by the desktop
operation.

## Still unresolved reads

`GET_SCHEDULE_MESSAGES`, `GET_FEED_FOLDER`, and `GET_SHORTCUTS` are now
registered as bounded reads. The scheduled-message UI makes a local-cache read
followed by a force-server refresh; descriptors default to the second,
force-server request (`syncDataStrategy: 3`) and allow the explicitly labelled
local first pass (`syncDataStrategy: 1`). The UI maps scene values
`CHAT_ONLY:1`, `REPLY_IN_THREAD:2`, and `CHAT_THREAD:3`. Feed-folder uses an empty payload and consumes
`feedFolderSettings.usedFolderIds`. Shortcuts uses `{syncDataStrategy:1}` and
consumes `shortcuts` plus `entityPreview`. The registered projection intentionally
returns only shortcut `id`, `feedType`, and channel `id`/`type`; a channel is
not assumed to be a chat, and `entityPreview` can contain private content.

`GET_FEED_GROUP_LIST` has a concrete native call in
`messenger.asar:8147/56c4fe82ab.js`: `n.px(i.vn0,{pairs:e})`. That module is
adjacent to feed-group update code, but this pass did not recover a UI consumer
for its response or establish what a valid `pairs` element contains. It remains
unregistered: the request wrapper alone does not justify an arbitrary array
parameter. Feed mute/unread/folder writes likewise remain unmapped state
changes.

`MGET_MESSAGES` is registered as `messaging.message-metadata` for one native
message ID. The actual message service sends `{messageIds:e}` and consumers
read `entity.messages`. Its projection is intentionally limited to message,
chat, thread, and sender IDs; it excludes body content. This can establish a
real chat ID from a Favorite's message ID without treating feed shortcut
channels as chats.

`MGET_CHATS` is registered as `messaging.chat-metadata` for one native chat
ID. The shipped `mGetChats` wrapper sends
`{chatIds, shouldAuth: true, source: DD.UNKNOWN}`; `DD.UNKNOWN` is `1`.
The bounded projection returns only chat ID, type, name, owner ID, and
`userCount`. It does not return messages, chatters, or previews.

`FETCH_CHAT_INPUT_DRAFT` is not an authoritative enumeration/readback source.
Its actual wrapper in `messenger.asar:23985/a3c37f9fe1.js`, module `20208`,
reads only transport `draft` and `entity.drafts`, normalizes each, and for a
chat caches `entity` or a synthetic `{drafts:{[draft.id]:draft}}`. There is no
alternate root-level draft envelope to project. A live text-draft create was
visible through `GET_ALL_DRAFTS` while this fetch returned no draft, so parent
create preflight and readback must use `GET_ALL_DRAFTS`; do not treat an empty
fetch result as permission to overwrite or proof that create/delete completed.
Accordingly, `messaging.chat-draft` and `messaging.reply-draft` use
`GET_ALL_DRAFTS` and filter `entity.drafts` by their validated `chatId`,
`threadId`, or `messageId` before applying their 100-draft output limit.

## Own-message editing: read-only protocol findings

The catalog defines `4520|im.v1.EditMessageRequest|im.v1.EditMessageResponse|1|EDIT_MESSAGE`,
but no final request wrapper or response consumer was recovered in the shipped
messenger chunks. It must remain unregistered. The client does demonstrate an
edit-draft path: `messenger.asar:43865/01157626a7.js` offset 115293 calls
`createDraft({content, messageId, chatId, editMessageId: messageId,
type: UfT.EDIT_MESSAGE, attachmentSnapshot})`; that is not evidence for the
later `EDIT_MESSAGE` payload.

The desktop error map makes ownership and scope mandatory preflight facts:
`EDIT_MESSAGE_USERID_NOT_SAME_FROM_REQUEST`,
`EDIT_MESSAGE_USER_NOT_IN_MESSAGE_CHAT`, and
`OPERATE_MESSAGE_NOT_FROM_SAME_CHAT` appear at
`messenger.asar:15312/8b65914177.js` offset 25786. The message metadata read
maps a native message's `fromId` to `senderId`, while chat metadata maps its
native chat. Before any future edit descriptor, the parent must compare that
sender ID with the active account user ID, establish the message/chat relation,
and then trace the actual edit request transform and response/readback. No
ownership check can turn a declaration or edit-draft envelope into a verified
message-edit write schema.

For every unresolved operation, find the request wrapper, the renderer/store
consumer, and the account/container context before registering it. Then perform
a bounded live read. Persistent operations need before-state, explicit apply,
and readback against the same native identifier.

## Inbox chat-card read (2026-09-22)

The desktop feed wrapper in `messenger.asar:common/7d5f20dbb7.js` module `542377`, offset `28749352`, exposes `C$` as `px(a.xrd,{...e},{parentContextId:t})`. The catalog binds `a.xrd` to `2480|feed.v1.GetFeedCardsV4Request|feed.v1.GetFeedCardsV4Response|1|GET_FEED_CARDS_V4` (`messenger.asar:44343/f127c4a4eb.js`, offset `5963044`).

The list flow calls it with `{feedCursor,filter,count,secondaryFilter,boxId,isPreload}` (`messenger.asar:messenger~4/b84e4ef239.js`, offset `58607765`). Its reducer initializes the first page as `{id:"0",rankTime:s.ab}` (offset `58632069`); `s.ab` is `"9223372036854775807"` (`messenger.asar:57062/9ed5ba9b39.js`, offset `7169100`). The page size is intentionally an explicit CLI input: the UI derives it from `Math.max(Math.ceil(window.innerHeight/a.Q1)+1,a.Mi+2)` (`messenger.asar:messenger~7/6b9fbb075f.js`, offset `58818784`), so there is no static UI default to invent. `lJ.INBOX` is enum 1 (`messenger.asar:messenger-chat~1/31cdd9feac.js`, offset `57501860`).

The response consumer passes `previews` through module `996260.S` (offset `58572725`). That transform selects a union member (`chatData` maps to CHAT), assigns `id: feedId`, then merges the chat data before the feed UI uses the resulting `name`, `chatType`, and `unreadCount`. `messaging.inbox-chats` therefore filters only raw cards with `chatData` and returns only those four metadata fields. It deliberately omits message/draft previews and other union card types. This request has static evidence only; no live call was made in this slice.

## Native P2P chat lookup (2026-09-22)

`GET_P2P_CHATS_BY_CHATTER_IDS` is a read-only resolver for an already-associated P2P chat. In `messenger.asar:common/aafb67268f.js`, offset `34649414`, the shipped Nexus bot UI calls `px(d.Vgt,{chatterIds:[s]})` and consumes `response.chatterId2chat[s].id`. `d.Vgt` is the native command `1046|im.v1.GetP2PChatsByChatterIdsRequest|im.v1.GetP2PChatsByChatterIdsResponse|1|GET_P2P_CHATS_BY_CHATTER_IDS` (`messenger.asar:44343/f127c4a4eb.js`, offset `5976207`). The create-P2P fallback is separate and deliberately not exposed. `messaging.p2p-chat` sends exactly `{chatterIds:[userId]}` and returns only the requested user ID and mapped native chat ID.

## Exact text matcher over first-screen history (2026-09-22)

`GET_CHAT_MESSAGES` is wrapped by `633951.Y` in `messenger.asar:44343/f127c4a4eb.js`, offset `6177089`. Its defaults are `redundancyCount: 0`, `count: 20`, `needResponse: true`, `strategy: RgN.SYNC_SERVER_DATA`, and `subscribChatEvent: false`; it calls native `zSv` with the resulting `{chatId,scene,count,position,strategy,redundancyCount,subscribChatEvent,needResponse,...}` object. `GET_CHAT_MESSAGES` is command `1020` at offset `5956250`. Enums establish `FIRST_SCREEN: 1` and `SYNC_SERVER_DATA: 3` (`messenger.asar:62148/81b0288138.js`, offset `7978575`).

The response normalization maps `messageItems[].itemId` through `entity.messages` (same wrapper, offset `6177089`). `messaging.recent-message-match` uses that actual response shape but returns a message ID only when all three local predicates match: native chat ID, native sender ID, and exact `content.richText.innerText`. It does not return the text or any unrelated message content.
