# Scheduled self-message persistence produced an empty pending item

This note traces the installed worklet after an already-completed live fixture
attempt. No additional SDK/server call, cancellation, or other mutation was
made while writing it.

## Observed persisted item

`GET_SCHEDULE_MESSAGES` now returns one pending server item with:

```text
messageId:    <id>
chatId:       <id>
senderId:     <id>
message type: 4 (TEXT)
scheduleTime: 1790149791
status:       1 (PENDING)
text:         absent from the joined entity.messages record
```

The item belongs to the just-created self-chat fixture and is therefore a
cleanup obligation. It is not valid evidence that the requested text was
persisted. The direct legacy persist call created a real scheduled item whose
TEXT content decoded empty or was omitted.

## Corrected serialization boundary

The two completed direct persistence attempts created pending TEXT items whose
joined message had no text. The second attempt used an outer
`entities.TextContent` `Uint8Array`; that result is also the wrong value for
the server request. This is a schema mismatch, not evidence that the native
transport requires base64.

The installed worklet module `78602` at
`jssdk/jssdk_worklet_messenger.js:2457638` returns a three-element tuple for
TEXT:

```js
n.text = content.text;
n.richText = r.ak(content.richText); // server RichText, properties are bytes
a.richText = content.richText;       // local display model
o = s.uC(n);                         // local entities.TextContent bytes
return [n, a, o];
```

Its schedule UPDATE path destructures the tuple and sends **the first item**
into `U`:

```js
const [serverContent, localContent, localBytes] = await h.eC(...);
await U(E.YN({fieldType: l, content: serverContent, chatId: n, cid: o, ...}));
```

`U` at offset `2521950` sends this value unchanged as
`putRequest.content` to
`900030|messages.PutScheduleMessageRequest|messages.PutScheduleMessageResponse`.
The installed server descriptor fixes that field as
`messages.PutMessageRequest.Content`, whose `rich_text = 14` has type
`entities.RichText`. `s.uC(n)` is a different `entities.TextContent` message;
it is the third result retained for local scheduled-message storage.

`r.ak` (module `13947`, offset `2309260`) maps a constrained CLI TEXT element
to this server shape:

```js
{
  richText: {
    elementIds: [id], innerText: text,
    imageIds: [], atIds: [], anchorIds: [], mediaIds: [],
    atUserGroupIds: [], markdownIds: [],
    elements: {dictionary: {[id]: {tag: 1, property: Uint8Array([0x0a, len, ...utf8(text)])}}}
  }
}
```

The nested property is `entities.RichTextElement.TextProperty { content = 1 }`.
For `fixture`, its exact bytes are `0a0766697874757265`. The public
`callServerApi` wrapper in `lark-api-injected.js:216656` calls its descriptor
encoder (`Hy().encode(request, params)`) before native transport, so the
server content object and its `Uint8Array` property must be passed directly;
there is no JSON/base64 conversion at that boundary.

`internal/desktop/modules/triage/text_codec.js` now reproduces only this
single-TEXT `r.ak` conversion and rejects all other staged shapes. The
associated test uses `protoc --encode/--decode` against a minimal
`messages.Content { rich_text = 14 }` schema to validate the nested byte
property. No additional live call was made after deriving this correction.

A future one-shot fixture attempt still needs the existing identity, staging,
readback, and cancellation guards. It can be considered successful only if
the pending list's joined message contains the exact supplied text.

## Safe cleanup of the existing empty fixture

The current cancellation descriptor models the desktop DELETE patch and sends
no content. The present bridge guard requires non-empty `expectedText`, which
is too strict for this owned malformed fixture. A narrowly scoped cleanup
amendment may accept `expectedText: ""` **only when all of these are true**:

1. the session retained the stage CID and the resulting scheduled message ID;
2. a fresh force-server list read has exactly the IDs above, sender equal to
the authenticated user, chat equal to the authenticated self-chat, type `4`,
status `1`, and schedule time `1790149791`;
3. the joined message has no decodable text (missing `text`, not a truncated
   string); and
4. the deletion request remains the exact desktop DELETE patch with
   `patchType:3`, the returned message/chat/time IDs, type `4`, omitted
   content, and `sendImmediately:false`.

After one delete, re-read until the exact ID is absent or status is DELETE. No
retry of either create or delete should occur. This is a cleanup path for the
known owned malformed fixture, not a general ability to cancel empty user
schedules.

## Cancellation transport correction

A direct call to the worklet-facing SDK command
`39008|im.v1.PatchScheduleMessageRequest|im.v1.PatchScheduleMessageResponse|1|PATCH_SCHEDULE_MESSAGE`
failed with “req no patch object”. That is expected for the former CLI payload:
`pb.desc` defines this SDK request as a local-store control envelope with
`patch_object` (`basic.ScheduleMessageItem`) at field 3, `message_type` at
field 5, and `is_send_immediately` at field 7. A persisted schedule must have
`patchObject:{itemId:messageId,itemType:2}` for that path.

The active worklet at `jssdk_worklet_messenger.js:2518724` consumes that SDK
shape, looks up the scheduled item in its local store, and for DELETE calls its
server helper. At `2519735` the helper constructs:

```js
{patchType, messageId, chatId, scheduleTime, type,
 content, sendImmediately, partialReplyInfo}
```

and calls `g.VvJ`. The command table at `3364743` resolves this to the
**authoritative server operation**:

```text
900034|messages.PatchScheduleMessageRequest|messages.PatchScheduleMessageResponse
```

`900031` is instead `messages.PullChatScheduleMessagesRequest`; it is a read,
not a patch command.

`pb_server.desc` (`messages.proto`) independently fixes the server field names:
`patch_type=1`, `message_id=2`, `chat_id=3`, `schedule_time=20`, `type=21`,
`content=22`, `send_immediately=23`, and `partial_reply_info=24`. For the
existing session-created plain self-TEXT fixture, the corrected descriptor sends
only the grounded DELETE subset:

```js
{patchType:3, messageId, chatId, scheduleTime, type:4, sendImmediately:false}
```

It deliberately omits `content` and `partialReplyInfo`; the descriptor has no
trusted reply-context snapshot and must not invent one. It is marked
`transport:"server"`, so the bridge's selected-primary mailbox guard applies.
No live cancellation was made after this correction. A fresh pending-fixture
read, one server patch, and post-patch scheduled-item absence/DELETE-status
readback are still required before cleanup can be called verified.
