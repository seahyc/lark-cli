# Recall-message static protocol trace

The ordinary UI handler in
`messenger.asar:topicCreateModal~4/51ac83bbf1.js` at offset `54257` resolves a
selected message and calls:

```js
await y.default.recallMessage({id: e})
```

The SDK adapter in `messenger.asar:44343/f127c4a4eb.js` at offset `283157`
preserves that one-field shape:

```js
recallMessage: function({id:e}) { return (0,c.px)(v.WUe,{id:e}) }
```

`v.WUe` maps to
`2079|im.v1.RecallMessageRequest|im.v1.RecallMessageResponse|1|RECALL_MESSAGE`
in the installed worklet command table at offset `3237600`. The handler at
offset `4414550` receives `e.id`, marks the local message recalled, and returns
`RecallMessageResponse` with `message="ok"`.

This proves the exact native wire request `{id: messageId}` and response use.
It does not establish ownership or destination safety; those are mandatory
parent preflight/readback guards for a self-chat fixture.

The message state contract is explicit: the shipped Message record declares
`isRecalled: !1` (`messenger.asar:10021/a6d2675328.js`, offset `25907`), and
the message-content renderer selects its recalled UI when `isRecalled` is true
(`messenger.asar:29184/d6793b5003.js`, offset `6304`). Readback must therefore
find the exact message record and require `isRecalled === true`; a missing
record is not recall proof.
