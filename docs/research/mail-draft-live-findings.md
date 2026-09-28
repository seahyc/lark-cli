# Mail blank-compose draft failure: static reconciliation

Scope: read-only comparison of the observed 2026-09-22 `NotFound` result with
Lark Desktop's shipped mail bundle. No mailbox, bridge, or UI calls were made
for this note.

## The normal blank-compose create request

The ordinary compose helper in
`mail.asar/common/29fe6e09b2.js`, module `586203`, offset `5252`, calls:

```js
(0,u.cB)({
  threadId:"", originMessageId:"", action:m.bG.COMPOSE,
  needSignature:a, timeText:"", feedCardId:t
})
```

The enum source is `mail.asar/chunk-common/835b3b6848.js`, module `164375`,
offset `22373`:

```js
eS={COMPOSE:0,REPLY:1,REPLY_ALL:2,FORWARD:3}
```

The `cB` wrapper is `mail.asar/common/92819f09f6.js`, module `758292`, offset
`2901`. It supplies the only additional wire field:

```js
reqOptions:{transform:e=>({...e,isFromLarkPlaintext:!1})}
```

Therefore a normal no-context compose produces this request shape:

```js
{
  threadId: "",
  originMessageId: "",
  action: 0,
  needSignature: !G7("newSignature"),
  timeText: "",
  // feedCardId is context-dependent; absent for no feed-card context
  isFromLarkPlaintext: false
}
```

There is no initial-thread sentinel in this caller. The blank IDs and action
`0` are correct.

`needSignature` is **not statically fixed**. In the same normal-compose helper
it is computed as `!(0,l.G7)("newSignature")`. Consequently, the previous
literal `false` is equivalent to the desktop UI only when that account's
`newSignature` feature is enabled. Static archive inspection cannot establish
the current feature value. A next live attempt must obtain it from the active
mail runtime or use the exact request captured from a normal UI compose; it
must not infer it from the action enum.

## The observed read failure has a concrete schema mismatch

The 3612 command is:

```text
3612|email.client.v1.MailGetDraftItemRequest|
email.client.v1.MailGetDraftItemResponse|1|MAIL_GET_DRAFT_ITEM
```

The same shipped wrapper module maps it as follows (offset `2901`):

```js
p=(0,l.Vy)(s.RYy,{
  reqOptions:{transform:e=>({draftId:e})},
  resOptions:{transform:e=>(0,r.lS)(e.draft)}
})
```

`RYy` resolves to the 3612 command in `mail.asar/18918/b2ed03e9bb.js`, offset
`10081`. A UI push/reload consumer in
`mail.asar/common/85e9a41919.js`, offset `57576`, invokes it as:

```js
let t=await (0,et.tH)(e);
```

where `e` is `draftId`.

Thus the actual 3612 wire request is exactly:

```js
{draftId: createdDraftId}
```

It is not `{threadId, messageId}`. The bridge's immediate 3612 request used
the latter shape, so its `NotFound` result cannot establish that
`MAIL_CREATE_DRAFT` failed or that no draft was persisted. It establishes only
that the readback used an unshipped request schema.

## Returned shape consumed by the desktop client

For create, the `cB` wrapper consumes `response.draft` and `response.isNew`:

```js
resOptions:{transform:e=>(0,r.lS)(e.draft,e.isNew)}
```

`lS`, in `mail.asar/71559/b4a48f3d52.js`, module `757633`, offset `7426`,
reads at least these draft fields:

```js
threadId, id, replyMessageId, from, to, cc, bcc, subject, coverInfo,
calendarEvent, bodyHtml, lastUpdatedTimestamp, createdTimestamp, images,
isSendSeparately, replyToAddress, priorityType, docsPermissions, attachments,
needReadReceipt, isFromLarkPlaintext, bodySummary
```

The required identity for immediate readback is `response.draft.id`.
The 3612 response also contains `response.draft`, which the desktop maps with
`lS(e.draft)`.

Deletion remains a different schema: `s.GUf` in module `758292` maps
arguments to `{threadId, messageId, feedCardId}`. That cleanup shape should
not be reused for the 3612 lookup.

## Blank-composer discard uses the draft ID, not replyMessageId

The actual editor discard path is in
`mail.asar/common/220298db94.js`, at offset `40458`:

```js
this.deleteDraft=async(e,t)=>{
  let{feedCardId:r}=this.props;
  this._deleteLocalDraft(!0),await this.props.deleteDraft(t,e.id,r)
}
this.handleEditorClose=async()=>{
  let{draft:e,onClose:t,threadId:r}=this.props;
  // discard path calls a() -> this.deleteDraft(e,r)
}
```

Here `e` is the draft object and `r` is its thread ID. The resulting 3601 wire
request is therefore:

```js
{threadId: draft.threadId, messageId: draft.id, feedCardId /* only when present */}
```

`replyMessageId` identifies the source mail for reply/forward drafts; it is
not the message ID sent by the blank-composer discard call. The live blank
fixture had an empty `replyMessageId`, so the prior
`{threadId: fixture.threadId, messageId:""}` call could only address a missing
message and its `NotFound` is expected. For the observed fixture, `draft.id`
and `draft.threadId` are both the generated UUID, so the correct cleanup uses
that UUID in both required fields.

## Evidence-led next live procedure

1. Capture the current `newSignature` feature value or one normal UI compose
   request, before any new create attempt.
2. Invoke `MAIL_CREATE_DRAFT` once with the exact blank normal-compose shape,
   using `needSignature` from that active value. Retain `response.draft.id`,
   `response.draft.threadId`, and `response.draft.replyMessageId` if present.
3. Read it once via 3612 as `{draftId: response.draft.id}`. Treat the returned
   `response.draft.id` equality as creation readback.
4. Only after that readback, perform cleanup through the separate 3601 schema
   with the retained `threadId` and the returned `draftId` as `messageId`; do
   not substitute either the 3612 lookup shape or `replyMessageId`.

The source trace resolves the normal create shape and the failed immediate-read
schema. It does not prove why the create call itself returned `NotFound` if
that error originated before a response containing `draft`; that requires a
fresh, exact live request/response capture.
