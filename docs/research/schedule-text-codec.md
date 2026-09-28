# Scheduled TEXT server-content conversion

The installed Lark build is under:

```text
/Applications/LarkSuite.app/Contents/Frameworks/Lark Framework.framework/
Versions/147.0.7727.149/
```

The scheduler worklet does not pass `entities.TextContent` bytes to the
server. In module `78602` at
`jssdk/jssdk_worklet_messenger.js:2457638`, `eC` returns `[n, a, o]` for TEXT:

```js
n.text = content.text;
n.richText = r.ak(content.richText);
a.richText = content.richText;
o = s.uC(n);
return [n, a, o];
```

At offset `2520273`, the schedule UPDATE caller destructures
`const [e, t, s] = await h.eC(...)` and calls `U(E.YN({content: e, ...}))`.
Thus the server request takes the first object (`n`). The third item (`o`) is
an `entities.TextContent` encoding used for local scheduled-message state.
It must not be assigned to `messages.PutMessageRequest.Content`.

`r.ak` is module `13947` at offset `2309260`. For a plain TEXT element it
constructs `entities.RichText` and sets the element `property` to the
`Uint8Array` encoding of `entities.RichTextElement.TextProperty`. The exact
restricted server content is:

```js
{
  richText: {
    elementIds: [id], innerText: text,
    imageIds: [], atIds: [], anchorIds: [], mediaIds: [],
    atUserGroupIds: [], markdownIds: [],
    elements: {dictionary: {[id]: {tag: 1, property: textPropertyBytes}}}
  }
}
```

The installed `pb_server.desc` defines `messages.PutMessageRequest.Content`
with `rich_text = 14` of type `entities.RichText`; it separately defines
`RichTextElement.property = 3` as `bytes`. For `fixture`, that nested property
is exactly `0a0766697874757265`, the protobuf form of
`TextProperty { content: "fixture" }`.

`lark-api-injected.js:216656` proves `callServerApi` uses a descriptor encoder
before native transport:

```js
await Hy().load(request);
const packet = Hy().encode(request, params);
return i.transport.callServerApi({params: packet, isPacketBuffer: true, ...});
```

Therefore the descriptor receives the object above with its `Uint8Array`
property directly. Base64 would violate the declared `bytes` value and is not
supported by this call path.

`internal/desktop/modules/triage/text_codec.js` implements this constrained
conversion only. Its test runs `protoc --encode` and `--decode` on a minimal
server `Content { rich_text = 14 }` fixture and rejects empty, oversize,
malformed-surrogate, non-TEXT, and multiple-element input. This verifies the
static schema. No live request was made after the correction.

Parent independently encoded and decoded the fixture using the installed full
`pb_server.desc` via `protoc --descriptor_set_in=...
--encode=messages.PutMessageRequest.Content`. The resulting bytes were
`72280a04746578741207666978747572651a170a150a0474657874120d08011a090a0766697874757265`;
the decoded inner text and TextProperty matched. This validates the actual
shipped descriptor boundary, in addition to the isolated test schema.
