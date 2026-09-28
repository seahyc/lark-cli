# Desktop own-message edit runtime

Static-bundle research only, 2026-09-22. No message edit, recall, send, or
other live call was made.

## Proven save path

The Messenger re-edit toolbar passes `handleSubmit` as its save callback in
`messenger.asar/common/ca9a351f13.js` (source offset 138177). The toolbar's
save button invokes that callback (`messenger.asar/common/aafb67268f.js`,
module 736721, offset 91626). The callback checks edit eligibility and then
calls `this.sendMessage()` at offset 129963 in the first archive. That method
builds and submits the actual edit request at offset 127101:

```js
let A = { title: l, richText: o };
(0, eQ.Ab)(A);
(0, eH.u_)({
  msgId: t.id,
  type: d,
  chatId: i?.id,
  content: { ...A, attachmentUpdate: (0, eL.Ls)(this.fileAttachmentScene) }
});
```

`t` is the current message item, so the message ID is sourced by the UI rather
than entered by the user. `o` is editor-produced rich text; it is not a plain
text string. `d` is selected as the text or post message type from the rendered
editor state, with a thread/post preservation condition. The code also derives
`attachmentUpdate` from the editor's attachment scene.

`handleSubmit` has an empty-editor branch before this request which offers a
recall action. Consequently a generic edit executor must not map an empty
replacement to this save flow.

## Runtime transport and command

The imported `eH.u_` resolves to export `u_:()=>g` in module 969997 of
`messenger.asar/31966/14196632fb.js` (module begins at offset 95219). At
offset 96399 it calls:

```js
async function g(e) {
  try {
    return await (0, i.px)(n.$Dn, { cid: (0, T.kk)(), ...e });
  } catch (e) {
    if (await (0, O.tR)(e?.code)) return;
    (0, c.U)(e, e.displayMessage);
    e.code === C.C.EDIT_MESSAGE_NOT_IN_VALID_TIME && (0, d.it)(!0);
  }
}
```

`$Dn` is explicitly exported as `eG` by module 601686 in
`messenger.asar/44343/f127c4a4eb.js` (offset 376), and `eG` is declared at
offset 31824 as:

```
4520|im.v1.EditMessageRequest|im.v1.EditMessageResponse|1|EDIT_MESSAGE
```

This is a compiled UI wrapper rather than a direct handwritten
`LarkAPI.transport` call. Its module 2483 shim in
`messenger.asar/45231/2465f4b0af.js` delegates `px` to module 929927. That
module in `messenger.asar/40040/7ee2cb9da7.js`, offset 11995, invokes
`n.transport.callSdkApi(e, t, i)`, setting `parentContextId`, `collectTrace`,
and optional transport flags. Thus the native runtime namespace is
`transport.callSdkApi`, with the `im.v1.EditMessageRequest` / response command
pair; the wrapper supplies a generated `cid` itself.

## Request and response boundary

The generated protobuf registry in
`main-window.asar/common/9f3a7ebf62.js` begins the edit request schema at
offset 52448. `messages.EditMessageRequest` registers the following fields:

| Field | Schema evidence |
| --- | --- |
| `msgId` | required `int64`, field 1 |
| `cid` | string, field 2 |
| `type` | `entities.Message.Type`, field 3 |
| `content` | nested `Content`, field 4 |
| `partialReplyInfo` | `entities.PartialReplyInfo`, field 5 |
| `content.richText` | `entities.RichText`, field 1 |
| `content.title` | string, field 2 |
| `content.lingoOption` | `entities.LingoOption`, field 3 |

The same registry declares `messages.EditMessageResponse` as a required
`message: entities.Message` (field 1), and registers both message types at
offset 79894. The wrapper normalizes `transport.callSdkApi`'s `data` into its
returned object. However, the proven re-edit save handler does not `await` or
inspect `eH.u_`'s returned response. Its observable local handling is the
wrapper's error path, including `EDIT_MESSAGE_NOT_IN_VALID_TIME`; no static
response-field consumer was found in this save chain. A bounded response
projection therefore cannot honestly be claimed from the UI caller alone.

## Implementation implications

This is a mutation, not a read descriptor. A future guarded mutation would
need the UI's canonical rich-text conversion, a known message item/chat
context, and the editor-derived `type` and attachment update semantics. It
must not accept arbitrary raw payloads, derive a target from text search, or
turn empty content into a recall. Live behavior, authorization checks, tenant
scope, attachment semantics, and server-side response behavior remain
unverified.
