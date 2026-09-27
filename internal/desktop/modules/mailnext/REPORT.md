# Mail-next descriptors

`mailnext.draft-fixture-create` and `mailnext.draft-fixture-delete` are a paired, unregistered draft mutation sequence. The creator has no recipient, subject, body, attachment, signature, or feed-card input. It uses the active normal-compose defaults observed in `mail.asar/common/29fe6e09b2.js`. `needSignature:false` is intentional: the fixture must not apply or write a user signature.

The caller must capture the returned `draftId`, `threadId`, and `replyMessageId`. Readback uses exactly `{draftId}` with `MAIL_GET_DRAFT_ITEM`; deletion then uses the separate `threadId` and `replyMessageId` identity and deletes exactly that fixture. `replyMessageId` is permitted to be empty because the normal compose callsite uses the returned draft value as its deletion message ID and a new compose has no source message. An acknowledgement is not deletion proof: use `mailnext.draft-fixture-metadata` before cleanup when the bridge can surface the returned identifiers, and treat a failed post-delete read as an expected outcome only after checking the native error code.

No descriptor creates rich recipients/body/attachments or updates an existing draft. The shipped update wrapper serializes a complete draft payload including recipient objects, image tokens, attachment tokens, document permissions, timestamps, and several flags. The bounded registry project intentionally withholds those fields, so accepting an arbitrary replacement object would risk data loss or token exposure.

No live bridge, mail account, or mailbox operation was run. Sender removal remains blocked by the legacy server route's observed `ErrCacheEmpty`; `isAllow:true` is an allow-list write, not block removal. Whole-mailbox forwarding still requires a current full rule plus account/check-password context. Signature reassignment is already represented by the guarded `mail.signature-usage-update` path and needs fresh account-scoped readback before change and inverse restore.

For a rich fixture body save, the parent needs a trusted in-memory raw
`MAIL_GET_DRAFT_ITEM` snapshot of the returned fixture. It must preserve every
field in the `MAIL_UPDATE_DRAFT` `H5` serializer, alter only approved body and
subject, save once, reread, and delete. This is an actual bridge prerequisite:
the public bounded projection cannot safely transport attachment/image tokens,
recipient identities, and document permissions back into a write request.

`mailnext.draft-fixture-update` is now available to the parent bridge through
`requestFromSnapshot(rawSnapshot, input)`. It requires exact returned fixture
identifiers and expected current subject/body. It rejects recipients,
attachments, images, document permissions, calendar data, non-normal priority,
read receipts, signatures, and other nonblank source state. HTML is limited to
plain text plus un-attributed `p`, `br`, `strong`, `em`, and `u` tags. Its
projection contains only IDs, subject, and body HTML length.
