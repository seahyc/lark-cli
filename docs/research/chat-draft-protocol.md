# Chat draft protocol from the desktop client

This is static analysis of the installed Lark 8.0.3 desktop bundle. It does
not establish that any request succeeds in a particular account or chat.

## Native create request

The cloud-draft wrapper in
`messenger.asar:23985/a3c37f9fe1.js` calls the native operation as:

```js
let o = await transport(u.TZ_, {draft: r});
```

The declaration is `2005|im.v1.CreateDraftRequest|im.v1.CreateDraftResponse|1|CREATE_DRAFT`.
It logs and consumes `response.draft`, `response.draftId`, and the returned
draft's `chatId`, `messageId`, `threadId`, `editMessageId`, and `type`. The
same wrapper can subsequently update local chat/thread draft metadata, so a
successful transport response alone is not a safe verification signal.

## Exact draft type enum

`messenger.asar:62148/81b0288138.js` exports `UfT` as:

```js
{TEXT:1, POST:2, INDIVIDUAL_TOPIC:3, EDIT_MESSAGE:4,
 MSG_THREAD:5, SCHEDULE_MESSAGE:6}
```

Only types `TEXT`, `POST`, and `MSG_THREAD` are accepted by the cloud-content
encoder. A basic chat composer draft is therefore `type: 1`; this says nothing
about the authorization to create it in any target chat.

## TEXT content transform

The encoder at `messenger.asar:44343/f127c4a4eb.js`, module `373506`, performs:

```js
let E = JSON.parse(content);
let richText = draftType === UfT.TEXT ? normalizeRichText(E) : ...;
if (!richText.innerText?.trim()) return;
return {title, richText};
```

For `TEXT`, `content` must be a JSON string encoding the complete rich-text
object itself, not a JSON string containing a plain text scalar. The encoder
does not create text elements. Its `normalizeRichText` implementation (module
`770167`) traverses the supplied `elementIds` and `elements`, sanitizes known
TEXT/MEDIA element fields, and returns the object with an `innerText` value.
It returns the input unchanged when `elementIds` is absent.

The UI save layer in `messenger.asar:43865/01157626a7.js` passes editor output
through unchanged: for a chat it supplies `{chatId, content, type, ...}` to
`createDraft`; for a message it supplies `{messageId, content, type, chatId,
threadId, ...}`. The nearest wrapper does not expose a plain-string-to-rich-
text builder.

## Canonical minimal plain-text rich-text builder

The installed client has a concrete text-to-rich-text constructor at
`messenger.asar:common/ac7134248d.js`, module `812382`, offset 38846:

```js
function textElement(content) {
  return {tag: RichTextElementTag.TEXT, property: {text: {content}}};
}
function plainTextRichText(content = "") {
  let elementIds = [], elements = {};
  if (content) {
    let id = uuid();
    elementIds.push(id);
    elements[id] = textElement(content);
  }
  return {
    elementIds, innerText: content, elements,
    imageIds: [], atIds: [], anchorIds: [], mediaIds: [], docsIds: []
  };
}
```

The source calls the bundled default export of module `470716` to obtain the
per-element `id`; it is shown as `uuid()` above only to make that call legible.
Its literal implementation is `let i=n()();`, where `n=a.n(a(470716))`.
That module is bundled UUID v4: it obtains 16 random bytes, sets byte 6's
version nibble to `4` and byte 8's RFC variant bits to `10`, then stringifies
the bytes. Its definition is at
`messenger.asar:45633/b313b33393.js`, module `470716`, offset 1628.

`RichTextElementTag.TEXT` is numeric `1`. This is defined in
`messenger.asar:62148/81b0288138.js`, offset 28897, as
`Ea={UNKNOWN:0,TEXT:1,IMG:2,...}` and is exported as `CUq`. Thus a nonempty
plain-text serialization emitted by this shipped builder is exactly:

```js
{
  elementIds: [elementId],
  innerText: plainText,
  elements: {
    [elementId]: {tag: 1, property: {text: {content: plainText}}}
  },
  imageIds: [], atIds: [], anchorIds: [], mediaIds: [], docsIds: []
}
```

The ID appears in both `elementIds[0]` and the `elements` key. It must be the
single UUID-v4 value returned by the bundled helper; a fixed `"1"` ID is not
what this constructor emits. An empty string is also deliberately rejected by
the draft encoder's `innerText.trim()` guard.

This is not merely an unreachable declaration. The same `dP` export is used
in `messenger.asar:45633/b313b33393.js`, module `447942`, offset 59085, to
turn a post title or summary into `{richText: dP(value)}` before it becomes a
message-content object. It is also used by the rich-content component at
`messenger.asar:58188/0e5ee4e5f8.js`, module `926941`, offset 907, when it
needs a fallback plain-text rich-text value. Those consumers demonstrate the
complete graph is accepted by current shipped rich-text rendering/content
paths. They do not prove a live `CREATE_DRAFT` request.

## Controlled plain-text fixture: known envelope and remaining boundary

The mapped native envelope for a potential basic draft is structurally:

```js
{draft: {chatId: nativeChatId, type: 1, content: JSON.stringify(richText)}}
```

This now has a shipped canonical rich-text construction rule above. The
remaining boundary is implementation provenance: a standalone native fixture
cannot invoke the webpack-private module `470716`. It can reproduce the
identified UUID-v4 rule only if the parent deliberately implements it and
tests the exact emitted schema. Do not substitute an invented
`{innerText:"..."}` object or a fixed `"1"` ID.

Before enabling a controlled fixture, use a disposable self-chat composer or
implement the demonstrated UUID-v4 graph with local schema tests, then create
once, read back through `FETCH_CHAT_INPUT_DRAFT`, and delete only the returned
native `draftId` with no automatic retry.

## Cloud companion used by current UI

The current `createDraft` wrapper does not always send only `content`. In `messenger.asar:23985/a3c37f9fe1.js` at offset `1903867`, it first conditionally applies a cloud representation, then invokes `TZ_` as `{draft:r}`. The condition is feature-gated (`_L()`) and skips unsupported draft types, crypto chats, and scheduled-message editing. The companion is produced by `373506.AX(content,type)` (`messenger.asar:44343/f127c4a4eb.js`, offset `6172598`): for TEXT it parses the content JSON and returns `{title: undefined, richText: 770167.T(parsedContent)}` only if `richText.innerText.trim()` is non-empty.

`770167.T` (`messenger.asar:44343/f127c4a4eb.js`, offset `6171454`) walks `elementIds`, rebuilds the reachable `elements` map, and applies sanitizers only to special text-link/media nodes. A minimal plain text graph is structurally retained. This is a concrete wire-envelope difference from a direct native create that provides only `content`. It is not evidence that the observed composer mismatch is caused by the missing companion: the UI condition is feature-controlled, and a previously cached local draft can also override what is displayed. Do not claim GUI parity from native `GET_ALL_DRAFTS` alone.
