# Favorite create readback failure

This is a static trace plus the already-observed bridge result. No additional
native request, profile change, or Favorite mutation was made for this note.

## Observation

The guarded bridge attempted `CREATE_FAVORITES` for the authenticated user's
own self-chat TEXT message `<id>` in chat
`<id>`. The SDK promise did not reject, but four subsequent
`GET_FAVORITES` reads returned only the pre-existing Favorite
`<id>`; no newly-created entry referenced the message.

That is a failed workflow, not evidence that the item was saved locally or
that cleanup is needed. The bridge correctly returned a readback error and did
not reissue the create write.

## Exact desktop create path

The installed `favorite.asar:favorite/b736c5d828.js` contains the low-level
wrapper (module `43392`, extracted-file offset `4569721`):

```js
function c({messageIds:e,chatId:t="",originMergeForwardId:n}){
  return (0,r.px)(i.YNq,{messageIds:e,chatId:t,originMergeForwardId:n})
}
```

`i.YNq` is declared in the same archive at extracted-file offset `7387903` as:

```text
2240|favorite.v1.CreateFavoritesRequest|
favorite.v1.CreateFavoritesResponse|1|CREATE_FAVORITES
```

The ordinary message-menu caller (module containing `Ee`, extracted-file
offset `4353111`) supplies a default empty merge-origin string and shows a
success toast solely when the promise resolves:

```js
async function Ee(e,t,n="") {
  try {
    await I.f4({messageIds:e,chatId:t,originMergeForwardId:n});
    u.oR.success(...)
  } catch(e) { ... }
}
```

There is no `resOptions` transform, no examination of
`CreateFavoritesResponse`, and no state-store dispatch in this create path.
The only static response semantics established are transport rejection versus
promise resolution; the response field schema (including any logical-error
fields) is not consumed by the shipped caller and remains untraced.

## Consequences

- This is not a worklet or local-only operation. The feature calls `r.px`
  directly, as do `GET_FAVORITES`, detail, and delete wrappers. The create
  caller does not insert a synthetic Favorite into local state.
- A resolved `callSdkApi` promise is insufficient success evidence. The
  shipped UI's success toast has the same limitation.
- The descriptor used `originMergeForwardId: undefined`. The UI's public
  action path default is `originMergeForwardId: ""`. JavaScript/protobuf
  omission may be equivalent, but that has not been proven. This is the one
  concrete payload discrepancy and must be corrected before another fixture
  attempt.
- No static evidence supports adding arbitrary context, account IDs, a
  worklet persistence call, or retrying create. Those would be guesses.

## Safe correction and next verification

1. Change the typed create payload to use `originMergeForwardId: ""`, matching
   the caller's default exactly. Add a descriptor regression test for that
   exact payload. Do not widen inputs: merge-forward origins are outside the
   self-message fixture scope.
2. Keep the existing own-TEXT/self-chat, complete-list, duplicate, and
   one-write guards.
3. Before a new live attempt, capture the **bounded structural shape only** of
   raw create `response.data` and any transport error object: top-level own
   keys; scalar boolean/number/string lengths; array/object field names and
   lengths. Do not log message content, Favorite content, tokens, or complete
   raw response. This is required to map logical failure fields, if present.
4. Use a fresh unique self-chat TEXT fixture and pre-read a complete Favorites
   page. Invoke once with the empty-string origin. Re-read the full page and
   require exactly one new Favorite whose native `content.messageId` equals the
   fixture ID. If successful, delete that exact returned Favorite ID and
   re-read to prove the original IDs remain.
5. If the list still lacks an entry, retain the captured response shape and
   stop. The remaining concrete blocker is an accepted native command with no
   observed persisted state; it is not safe to infer a persistence companion
   or to retry.
