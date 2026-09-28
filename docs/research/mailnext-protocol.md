# Follow-up native mail protocol trace

Static evidence came from `/tmp/lark-private-api-research/mail` only. No live
bridge, mailbox, account/profile state, outbound mail, or credential was used.

## Reversible blank draft fixture

The active normal composer in `mail.asar/common/29fe6e09b2.js`, module
`586203`, creates a compose draft through this exact call:

```text
u.cB({threadId:"",originMessageId:"",action:m.bG.COMPOSE,
      needSignature:a,timeText:"",feedCardId:t})
```

`m.bG.COMPOSE` is `0`, shown in
`mail.asar/chunk-common/835b3b6848.js`:

```text
eS={COMPOSE:0,REPLY:1,REPLY_ALL:2,FORWARD:3}
```

The wrapper in `mail.asar/common/92819f09f6.js`, module `758292`, grounds the
native wire and response consumer:

```text
s.V5V (MAIL_CREATE_DRAFT) transform:e=>({...e,isFromLarkPlaintext:false})
response:e=>lS(e.draft,e.isNew)
```

The command is
`3606|email.client.v1.MailCreateDraftRequest|email.client.v1.MailCreateDraftResponse|1|MAIL_CREATE_DRAFT`.
`lS` in `mail.asar/71559/b4a48f3d52.js` maps the returned `id`, `threadId`,
`replyMessageId`, attachments, and images. The descriptor fixes
`needSignature:false`, omits the context-dependent `feedCardId`, and accepts
no subject/body/recipient/attachment input. This is deliberately a blank,
unsent fixture rather than an outbound email.

The deletion wrapper is in the same module:

```text
s.GUf (DELETE_MAIL_DRAFT) transform:(e,t,a)=>
  ({threadId:e,messageId:t,feedCardId:a})
```

Its command is
`3601|email.v1.DeleteMailDraftRequest|email.v1.DeleteMailDraftResponse|1|DELETE_MAIL_DRAFT`.
The normal composer records `F=r.threadId,A=r.replyMessageId` and its
`deleteDraft` helper calls `m.Dk(e,t,r)`, establishing `threadId` plus the
returned reply-message ID as the cleanup identity. A new compose can have an
empty reply-message ID, so an empty string is valid for this field; thread ID
must remain nonempty.

Parent live verification steps, with one generated fixture only:

1. Record the selected native account identity before creating anything.
2. Invoke `mailnext.draft-fixture-create` with `{}` once and retain its returned
   `draftId`, `threadId`, and `replyMessageId`.
3. Use `mailnext.draft-fixture-metadata` with that exact thread/reply pair.
   Confirm it identifies only the returned fixture and reports zero attachments
   and images. Do not inspect or alter any pre-existing draft.
4. Invoke `mailnext.draft-fixture-delete` with the exact returned thread/reply
   pair, once. Do not retry an uncertain result.
5. Read the same fixture identity again and capture the native missing/deleted
   result. Report a successful delete acknowledgement separately from readback.

Static schemas are verified; this sequence is **not live-tested**. In
particular, an environment might require a feed-card context or produce a
nonempty compose reply-message ID; the parent must stop and record that native
result rather than substitute guessed values.

## Rich draft update and attachment metadata

`MAIL_UPDATE_DRAFT` is command
`3607|email.client.v1.MailUpdateDraftRequest|email.client.v1.MailUpdateDraftResponse|1|MAIL_UPDATE_DRAFT`.
The active wrapper (`common/92819f09f6.js`, module `758292`) is:

```text
s.Ovw transform:(e,t,a)=>({draftId:e.id,payload:r.H5(e),isDelay:t,
  feedCardId:a,onlySaveLocal:e.isImageAndAttachmentOverLimit||e.isOverSize})
response:e=>r.lS(e.draft,false)
```

`r.H5` serializes recipient identity objects, HTML, images, attachments,
document permissions, timestamps, priority/calendar/read-receipt fields. The
related editor builder in `common/efae1e1bd1.js`, module `614120`, shows
attachments carry `fileName,fileSize,fileKey,type,largeFilePermission,
needConvertToLarge,expireTime,needCopyFileToken,originalFileToken`, and images
carry a file token. There is no safe partial-update schema: a CLI that accepts
only text would still have to re-send fresh complete token-bearing pre-state.
No update descriptor is exposed.

`mailnext.draft-fixture-metadata` uses `MAIL_GET_DRAFT_ITEM` and projects
bounded fixture attachment name/size/type/count plus image count. It exposes a
complete subject only up to 512 characters and complete body HTML only up to
16 KiB; larger values return empty fields with explicit truncation flags rather
than a partial expected-value snapshot. It strips recipients, sender details,
file keys/tokens, paths, and document permissions.

## Signature reassignment

`MAIL_UPDATE_SIGNATURE_USAGE` remains the grounded account-scoped reassignment
command:
`3727|email.client.v1.MailUpdateSignatureUsageRequest|email.client.v1.MailUpdateSignatureUsageResponse|1|MAIL_UPDATE_SIGNATURE_USAGE`.
In `mail.asar/common/9a1f1c38eb.js` at 18723 its wrapper is
`transform:(e,t)=>({accountId:e,signatureUsage:t})`. The only safe sequence is
fresh `MAIL_GET_SIGNATURE` for the selected account, retain the exact current
pair (`newMailSignatureId`, `replyMailSignatureId`), make one replacement, read
back, then send that retained pair as the inverse if cleanup is required.
This is already represented by the guarded mail mutation descriptor; no new
copy was registered here.

## Sender block and whole-mailbox forwarding blockers

Native sender block is schema-grounded, but the removal chain is not complete.
The older delete/read route was statically traced as server commands
`3902|mails.DeleteUserAllowBlockRequest|mails.DeleteUserAllowBlockResponse` and
`3901|mails.SearchUserAllowBlockRequest|mails.SearchUserAllowBlockResponse`.
A direct exact-transport attempt observed `ErrCacheEmpty`. This does not prove
an empty list or a route cause. `isAllow:true` writes an allow-list entry and
cannot serve as fixture cleanup.

Whole mailbox forwarding read is `MAIL_GET_RULES` with
`{version:"2",isGlobalTransfer:true,accountId}`. Its update is a full
`MAIL_UPDATE_RULE` replacement. `common/48be8628db.js` passes the serialized
complete rule and account context; one UI wrapper also carries `checkPwd`.
A safe write needs the currently selected mailbox, a fresh complete rule,
verified transfer-email state, and UI-derived check-password context. No
arbitrary settings blob or forwarding mutation is exposed.

### Concrete guarded rich-save bridge contract

The runtime prerequisite is now exact, rather than an untraced blocker. A
parent-owned workflow may support a rich fixture edit only when it performs all
of these in one guarded operation:

1. It receives the prior create result and only accepts its returned draft ID,
   thread ID, and reply-message ID as the target identity.
2. It obtains a fresh raw `MAIL_GET_DRAFT_ITEM` response for that identity
   without exposing it through the normal CLI projection, then verifies it is
   the retained fixture and that its attachment/image/recipient/document arrays
   have the expected shape before mutation.
3. It replaces only approved body/subject fields in that in-memory object and
   serializes the complete object exactly as `H5` above. `from`, aliases,
   recipient identity metadata, attachments, image tokens, document
   permissions, thread ID, priority, cover/calendar fields, and read-receipt
   fields come from that fresh snapshot, never caller defaults.
4. It invokes once with `{draftId: snapshot.id, payload: H5(snapshot),
   isDelay: false, onlySaveLocal: false}`. `feedCardId` remains absent unless
   the current native context itself supplied one.
5. It reads the fixture again, verifies the intended subject/body and retained
   attachment/image counts, then deletes it with the same retained
   thread/reply-message identity. An uncertain write has no automatic retry.

This is a bridge-level transaction prerequisite, because descriptor `project`
correctly filters sensitive raw draft fields and cannot feed them back into a
later ordinary CLI invocation. The condition is executable with the two native
commands traced above; it does not require credentials, an arbitrary user
settings object, or a UI patch.

### Implemented restricted fixture update

`internal/desktop/modules/mailnext/mutations.js` now exports
`mailnext.draft-fixture-update`. Its ordinary `request()` intentionally throws:
the parent must call `requestFromSnapshot(rawSnapshot, input)` after its own
identity/account guard. The method accepts exact `draftId`, `threadId`, and
`replyMessageId`, desired `subject`/`bodyHtml`, and expected current
`subject`/`bodyHtml`.

The method normalizes the native `MAIL_GET_DRAFT_ITEM` `draft` with the same
field defaults used by `lS`, preserves the complete sender identity required by
`H5`, and emits this exact `MAIL_UPDATE_DRAFT` outer shape:

```text
{draftId, payload:{from,to:[],cc:[],bcc:[],bodyHtml,subject,images:[],
 attachments:[],docsPermissions:[],threadId,isSendSeparately:false,
 priorityType:3,coverInfo:"",calendarEvent:undefined,needReadReceipt:false,
 bodySummary:"",isFromLarkPlaintext:false}, isDelay:false, onlySaveLocal:false}
```

It rejects snapshots with recipients, attachments, images, document
permissions, calendar data, body summary, read receipts, separate sends,
non-normal priority, cover data, plaintext mode, stale expected body/subject,
or mismatched fixture identity. The desired HTML is at most 16 KiB and permits
only text and attribute-free `p`, `br`, `strong`, `em`, and `u` tags. This keeps
scripts, links, images, styles, and external resources out of the fixture path.
The response projection keeps only draft/thread IDs, subject, and body length;
it never returns the snapshot, sender metadata, recipients, or file tokens.

## Session-created signature fixture text update

The signature settings action in `mail.asar/common/9a1f1c38eb.js` at offset
`9090` calls its update helper with the selected complete signature object:

```text
m=(e,t)=>async a=>{await (0,o.Q_)(e,t);a(s.o.setConfigChange())}
```

Its wrapper at offset `18543` sends exactly `{accountId, signature}` and the
`n.Z9i` command declaration in `mail.asar/18918/b2ed03e9bb.js` at `42886` is:

```text
3725|email.client.v1.MailUpdateSignatureRequest|
email.client.v1.MailUpdateSignatureResponse|1|MAIL_UPDATE_SIGNATURE
```

`mailnext.signature-fixture-update-text` is intentionally snapshot-only. Its
ordinary `request()` rejects. `requestFromSnapshot(rawGetSignatures, input)`
requires `{accountId, signatureId, text, expectedTemplateHtml}`, selects exactly
one raw signature with that ID, and requires the complete current HTML to equal
the expected value. It preserves the full selected native signature object,
including `id`, `name`, `signatureType`, `signatureDevice`,
`templateValueJson`, `images`, and unknown native fields, replacing only
`templateHtml` with text escaped inside `<div>...</div>`.

The bridge must restrict this to a signature it created in the current session,
read raw `MAIL_GET_SIGNATURE` data immediately before the write, then read it
again and compare the ID, name, type, device, template JSON, image records and
exact escaped HTML. Native acknowledgement is not readback proof. This path has
static-schema and isolated-node coverage only; no live signature update was run.
