# Mail draft cleanup: acknowledgement, stale detail read, and list-based absence

## Observed runtime result

The corrected native delete request (`DELETE_MAIL_DRAFT`, command `3601`) with
its `{threadId, messageId: draft.id}` identity acknowledged successfully. An
immediate `MAIL_GET_DRAFT_ITEM` (`3612`) with `{draftId}` still returned the
same blank draft identity. This record is deliberately concise: no user
identifiers, draft contents, or account data are retained here.

That result **does not prove that the server still has the draft**. It also does
not prove deletion. The post-delete `3612` result is an unresolved client-side
readback condition.

## Why `MAIL_GET_DRAFT_ITEM` is not deletion proof

The active wrapper in `mail.asar/common/92819f09f6.js`, module `758292`, has
no freshness or source argument for the item read:

```js
p = Vy(s.RYy, {
  reqOptions: { transform: e => ({ draftId: e }) },
  resOptions: { transform: e => lS(e.draft) }
})
```

`RYy` is exactly
`3612|email.client.v1.MailGetDraftItemRequest|email.client.v1.MailGetDraftItemResponse|1|MAIL_GET_DRAFT_ITEM`.
The caller can supply only `draftId`; there is no native `forceServer`,
`syncDataStrategy`, mailbox context, or cache-bypass flag in this wrapper.
Therefore, a blank same-ID response after deletion is compatible with a stale
client record. Do not treat it as positive evidence that cleanup failed.

The native composer also updates UI state before awaiting the server delete.
In `mail.asar/common/220298db94.js`, its close handler is:

```js
deleteDraft = async (draft, threadId) => {
  const { feedCardId } = this.props;
  this._deleteLocalDraft(true);
  await this.props.deleteDraft(threadId, draft.id, feedCardId);
}
```

The related Redux reducer in `mail.asar/common/41b8ba5175.js` removes a matching
`draftId` from `messageList.drafts`, or from a message's nested `drafts`, by
splicing the in-memory list. This confirms that editor/list disappearance is a
local state transition and cannot independently prove server deletion.

The actual delete wrapper remains grounded: `92819f09f6.js` transforms the
composer arguments as:

```js
s.GUf transform: (threadId, messageId, feedCardId) =>
  ({ threadId, messageId, feedCardId })
```

with `GUf = 3601|email.v1.DeleteMailDraftRequest|email.v1.DeleteMailDraftResponse|1|DELETE_MAIL_DRAFT`.
The successful acknowledgement is thus valid evidence that this exact server
operation accepted the identity; it is not a readback.

## Native DRAFT-list route for positive absence

The desktop DRAFT view reloads through
`3603|email.client.v1.MailGetThreadListRequest|email.client.v1.MailGetThreadListResponse|1|MAIL_GET_THREAD_LIST`.
This is a real request/consumer chain, not a declaration-only command:

* `92819f09f6.js`, module `88147`, wraps `s.Jho` with
  `{labelId, newestTimestamp:String(timestamp), length:String(length),
  filterType}` and returns `threadItems`, `isLastPage`, and `isFromDb`.
* `s.Jho` is the `MAIL_GET_THREAD_LIST` command in
  `mail.asar/18918/b2ed03e9bb.js`.
* `mail.asar/common/ae001966e4.js`, module `47175`, calls that wrapper from the
  `fetchThreadList` thunk and replaces the current label's Redux thread list
  with the returned groups.

The installed protobuf descriptor `email/client/v1/mail_client.proto` fixes
the request shape and enum:

```text
MailGetThreadListRequest
  filter_type = 1;  ALL_MAIL = 2
  label_id = 2;     DRAFT is the desktop DRAFT label string
  newest_timestamp = 3
  length = 4

MailGetThreadListResponse
  thread_items = 3; is_from_db = 4
ThreadItem.thread = 1
Thread.id = 1
```

For a generated fixture, the exact native first-page request used by the
DRAFT UI model is consequently:

```json
{
  "labelId": "DRAFT",
  "newestTimestamp": "0",
  "length": "20",
  "filterType": 2
}
```

A **bounded positive-absence procedure** is to read every DRAFT page using
that schema, advance `newestTimestamp` with the returned page's oldest native
thread timestamp, stop only at `isLastPage:true`, and require that no
`threadItems[*].thread.id` equals the session-created fixture `threadId`.
The check is by `threadId`, not `draftId`: `ThreadItem` contains a `Thread`,
and the delete operation identifies the same draft using both its thread ID
and its draft/message ID.

## Remaining limit before calling it verified cleanup

`MAIL_GET_THREAD_LIST` returns the explicit `isFromDb` flag. Its static wrapper
has no caller-provided force-network parameter. Until a disposable runtime
run establishes the expected post-delete source behavior, the list route is
an exact, safer absence candidate but not a completed server-deletion proof.
The cleanup report should capture every page's `isFromDb`, `isLastPage`, and
whether the retained thread ID appeared. A page with `isFromDb:true`, an
incomplete pagination sequence, a list error, or an unrecognized response
must remain inconclusive; do not retry deletion automatically.

No live calls or mailbox changes were made for this trace.
