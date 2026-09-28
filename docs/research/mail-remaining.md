# Remaining native mail protocol gaps

Static source root: `/tmp/lark-private-api-research/mail` from the installed
Lark 8.0.3 archive. This is static evidence only; no desktop bridge, profile,
network, or mailbox request was run.

## Implemented read wrappers

`internal/desktop/modules/mail/module.js` now includes the existing typed
reads plus native `mail.draft-item`:

```text
common/92819f09f6.js @4284
  s.b2d transform:(e,t)=>({threadId:e,messageId:t})
  response transform:e=>lS(e.draft,true)

71559/b4a48f3d52.js @7426
  lS reads id, threadId, replyMessageId, subject, timestamps, bodyHtml,
  images, and attachments.
```

The descriptor returns only draft metadata/counts and `hasBody`; it excludes
body HTML, recipients, attachment/image tokens, and document permissions.

Native advanced search has a concrete wrapper but is not exposed. Its request
is a composed object with `keyword`, cursor, session, offline flag, search
filter, label/search type mapping, related-mail search, and sort rule:

```text
common/6b0f43cfaa.js @2378
  transform:(e,t)=>({keyword:e.keyword,offset:e.nextBegin||"0",
  searchSession:e.sid,isOffline:e.isOffline,...e.searchFilter,
  ...(0,c.Gg)(e.searchType),...e.relatedMailSearch,searchSortRule:e.searchSortRule})
```

No narrow, public CLI input schema was established for the required nested
filter/search context, so implementing it would be arbitrary payload pass-through.

## Typed mutation plans, deliberately unregistered

`mutations.js` supplies guarded descriptors for rule enable/disable, full rule
order, signature-usage update, and a disposable text-only signature lifecycle.

```text
common/48be8628db.js module 6630 @111
  n.$uT transform:(r,t)=>({ruleIdString:r,isEnable:t})
  n.Iz_ transform:r=>({ruleOrder:r.map(r=>({...r,order:String(r.order)}))})

common/9a1f1c38eb.js module 206153 @18723
  n.rKc transform:(e,t)=>({accountId:e,signatureUsage:t})
```

Rule enable/reorder receive account context from the selected native mailbox,
not request fields. Before apply, the parent must prove that selected identity
matches the user-selected account and must read back rules. Reorder requires a
fresh complete rule list; it must include every existing ID exactly once.
Signature usage requires a fresh `mail.signatures` read for the same account;
both requested signature IDs must be present for its address.

The text-only fixture signature lifecycle is traced from the setting UI:

```text
common/9a1f1c38eb.js @9090
  create: {name:t.name,images:t.images,templateHtml:t.templateHtml,id:void 0}
  delete: o._c({signatureId:t,accountId:e})

common/9a1f1c38eb.js @18281 and @18461
  add transform:(e,t)=>({accountId:e,signature:t})
  delete transform:e=>e
```

The exposed fixture creation accepts plain `text` only, HTML-escapes it into a
simple inert `div`, and forces `images: []`, so markup cannot execute and it
cannot overwrite or reuse an image-backed signature. A live test must create a unique `.invalid`
fixture, read it back by its returned ID, delete that exact ID, then confirm it
is absent. Deletion requires a fresh zero `signatureUsageCount` preflight so an
assigned/in-use signature is rejected. Mutation acknowledgements deliberately report `verified:false`;
only the subsequent read proves state.

## Exact but unsupported mutation protocol

`MAIL_UPDATE_RULE` is fully traced, but unsafe to offer as a partial patch:

```text
common/48be8628db.js @111
  transform:(r,t)=>({rule:O(r),...t})
  O(r)=>{ruleIdString,name,isEnable,ignoreTheRestOfRules,isInvisible,
    condition:{matchType,items},action:{items}}
```

Its account context can include `checkPwd` and `accountId`. A safe integration
must begin with a fresh `MAIL_GET_RULES` result, preserve every condition and
non-target action, present the complete reconstructed rule, and read back the
same rule after one guarded update. The module does not manufacture missing
fields or defaults.

`MAIL_ADD_SIGNATURE` and `MAIL_UPDATE_SIGNATURE` have wrappers at
`common/9a1f1c38eb.js @18281` and `@18543`, both transforming to
`{accountId,signature}`. Signature has `id`, `signatureType`,
`signatureDevice`, `name`, `templateHtml`, `templateValueJson`, and `images`.
Writing one without preserving the image fields can destroy image-backed
signatures, so neither is exposed.

`MAIL_ADD_USER_ALLOW_BLOCK` is present in `catalog.json` as command 4021, but
no request-transform callsite was found in the inspected static chunks. It is
not implemented. The separate generic public API wrapper observed in
`common/bc90fef3a3.js` uses a different command and is not evidence for this
native command’s schema.

No send, schedule-send, delete, forwarding-auth-mail, rule create/delete,
apply-rule, or account update mutation is exposed. Effects range from mailbox
settings to outbound email, and need a dedicated approval and readback design.
