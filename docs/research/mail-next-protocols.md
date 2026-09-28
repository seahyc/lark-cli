# Native mail protocols for guarded follow-up work

Research source: extracted Lark 8.0.3 static JavaScript under
`/tmp/lark-private-api-research/mail`. This note records static call paths only.
No request, desktop bridge, mailbox state, or profile was touched.

## General rule update: complete replacement, never a partial patch

`mail.asar/common/48be8628db.js`, module `6630`, contains both the serializer
and wrapper:

```text
function O(r){return{
  ruleIdString:r.ruleIdString,name:r.name,isEnable:r.isEnable,
  ignoreTheRestOfRules:r.ignoreTheRestOfRules,isInvisible:r.isInvisible,
  condition:{matchType:r.matchType,items:r.condition.map(N)},
  action:{items:r.action.map(R)}}}
function N(r){return{type:r.type,operator:r.operator,input:r.input}}
function R(r){return{type:r.type,authStatus:r.authStatus,
  enableAutoTransfer:r.enableAutoTransfer,input:r.input}}
...
n.p6L transform:(r,t)=>({rule:O(r),...t})
logFormat:{rule:p(),checkPwd:ORIGIN,accountId:ORIGIN}
response => {rule:m(r.rule,r.emailErrors),emailErrors:r.emailErrors??{}}
```

The native wire command is
`3691|email.client.v1.MailUpdateRuleRequest|email.client.v1.MailUpdateRuleResponse|1|MAIL_UPDATE_RULE`.

Actionable guarded chain:

1. Read current rules for the selected account using `MAIL_GET_RULES` and keep
   the exact targeted rule plus every condition/action.
2. Confirm the desktop-selected mailbox is the requested `accountId` and obtain
   the UI-required `checkPwd` context through its ordinary UI flow. Static code
   shows both fields pass through; it does not establish how a CLI can mint a
   password check safely.
3. Modify only the approved fields in the preserved rule; serialize **all**
   fields with `O` above. Do not synthesize omitted conditions, actions, or
   forwarding auth fields.
4. Issue once with `{rule: O(rule), accountId, checkPwd}` and inspect
   `emailErrors` plus the returned rule.
5. Re-read `MAIL_GET_RULES` and compare the exact rule. A timeout receives no
   automatic retry.

This is intentionally not exposed as a mutation descriptor: without a verified
account/check-password supplier and a live full-rule readback, a partial update
can erase unrelated user rules/actions.

## Sender allow/block: actual desktop API path

The catalog's `MAIL_ADD_USER_ALLOW_BLOCK` declaration is not the active
callsite. The action UI uses the older desktop API module instead:

```text
mail.asar/18918/b2ed03e9bb.js @42886 (module 635656)
  Kgq => "3891|mails.AddUserAllowBlockRequest|mails.AddUserAllowBlockResponse"
  a7j => "3902|mails.DeleteUserAllowBlockRequest|mails.DeleteUserAllowBlockResponse"
  LyB => "3901|mails.SearchUserAllowBlockRequest|mails.SearchUserAllowBlockResponse"

mail.asar/common/bc90fef3a3.js @850
  m.Kgq transform:e=>{
    let{fromList:t,isBlockSender:r,base:s}=e;
    return{multiFrom:t.map(e=>({address:e})),isAllow:!r,
      scene:u.Ii.BLOCK_SENDER,base:s}}
  m.a7j transform:e=>{
    let{fromList:t,isBlockSender:r,base:s}=e;
    return{multiFrom:t,isAllow:!r,base:s}}
  m.LyB transform:e=>({reqType:e.reqType,lastTimestamp:e.lastTimestamp,
    cursor:e.cursor,size:e.size,query:e?.searchQuery??"",base:e.base})
```

There is also a non-base UI wrapper in that same module:

```text
M=(0,T.Vy)(_.Kgq,{reqOptions:{transform:e=>{
  let{from:t,isBlockSender:r}=e;
  return{multiFrom:t,isAllow:!r,scene:c.pH.BLOCK_SENDER}},
  resOptions:{transform:()=>!0}})
```

The UI invokes it with one or two derived sender-address objects:

```text
await M({isBlockSender:u,from:t})
```

`M` uses `_.Kgq` from native module `236171`, not the older `m.Kgq` server
wrapper. It resolves to the five-part `MAIL_ADD_USER_ALLOW_BLOCK` command. The
enum is exact in `mail.asar/chunk-common/835b3b6848.js @15873`:
`pH={BLOCK_WEB_IMAGE:0,BLOCK_SENDER:1}`. Its typed native payload is
`{multiFrom:[{address}],isAllow:false,scene:1}` to block one validated address.
The companion `isAllow:true` request creates or changes an allow-list entry;
it is not evidence that a blocked entry was removed and must never be used as
fixture cleanup.

The actual removal wrapper is the older three-part command, with a materially
different transport:

```text
mail.asar/common/bc90fef3a3.js @850
  m.a7j transform:e=>{
    let{fromList:t,isBlockSender:r,base:s}=e;
    return{multiFrom:t,isAllow:!r,base:s}}

mail.asar/chunk-common/835b3b6848.js @144641 (module 531015)
  Eq(e,t): i.transport.callServerApi(`${id}|${request}|${response}`, t, ...)

mail.asar/chunk-common/835b3b6848.js @201959 (module 841196)
  Zy(e,t): return (0,i.Eq)(e,t)
```

For deleting a blocked sender, that transform requires
`{multiFrom:[address],isAllow:false}` through
`3902|mails.DeleteUserAllowBlockRequest|mails.DeleteUserAllowBlockResponse`.
The schema is retained only in the unregistered research files
`legacy.js` and `legacy-manifest.json`. A direct call through the exact server
transport and valid int64 pagination returned `ErrCacheEmpty`; this is an
observed failure, not an empty collection, and its cause remains unresolved.
The static bundle defines the legacy server wrappers as discarded expressions,
while its active block/trust UI uses the native SDK wrapper. Do not execute the
legacy search or delete descriptors until an active, working route is proven.

A complete future fixture chain is therefore:

1. Search the legacy allow/block collections for the exact address with
   `3901|mails.SearchUserAllowBlockRequest|mails.SearchUserAllowBlockResponse`.
   Its request transform is `{reqType,lastTimestamp,cursor,size,query}` and its
   response consumers use `{allows,blocks,nextTimestamp}`. The research-only
   descriptor bounds pages to 100 records, but cannot currently prove absence.
2. Block once via the native SDK payload above.
3. Confirm the address is blocked with `MAIL_GET_BLOCKED_ADDRESSES`; that read
   does not prove whether an allow-list entry also exists.
4. This cleanup chain is blocked: do not call the legacy delete command or
   claim cleanup until the legacy search route works and can prove absence.

No live call was made. A `.invalid` domain avoids delivery but does not avoid a
mailbox-settings mutation; it can leave a server-side allow-list record unless
the server delete/readback chain is implemented and verified.

## Update an existing text-only signature without losing images

The settings actions show both the caller and native wrapper:

```text
mail.asar/common/9a1f1c38eb.js @9090
  m=(e,t)=>async a=>{ await (0,o.Q_)(e,t); ... }

mail.asar/common/9a1f1c38eb.js @18543, module 206153
  p=(0,r.Vy)(n.Z9i,{reqOptions:{transform:(e,t)=>({accountId:e,signature:t}),
    logFormat:{accountId:ORIGIN,signature:s()}},
    resOptions:{transform:e=>e,logFormat:{signature:s()}}})

signature logging schema s():
  {id,signatureType,signatureDevice,name,templateHtml,templateValueJson,images}
```

`n.Z9i` resolves to native command
`3725|email.client.v1.MailUpdateSignatureRequest|email.client.v1.MailUpdateSignatureResponse|1|MAIL_UPDATE_SIGNATURE`.

Safe update chain:

1. `MAIL_GET_SIGNATURE` for the exact account. Select the signature by ID.
2. Reject it if the fresh record contains any image that cannot be preserved
   byte-for-byte in the outgoing `images` list. Do not replace images with an
   empty list during an update.
3. Escape the approved plain text into an inert `div`; preserve the original
   `id`, `signatureType`, `signatureDevice`, `templateValueJson`, and `images`.
4. Send `{accountId, signature}` via `MAIL_UPDATE_SIGNATURE` once.
5. Re-read signatures and compare the selected signature's preserved metadata,
   images, and escaped HTML. Restore from the retained complete pre-state only
   with a separately approved follow-up if readback shows damage.

The existing text-only fixture create/delete path is safer for a first live
roundtrip because it deliberately has `images: []` from creation. Existing
signature update remains a research protocol until real response shape and
image preservation are verified.
