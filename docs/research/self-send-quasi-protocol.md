# Native self-send quasi protocol

Static evidence from the installed Lark desktop build shows that `SEND_MESSAGE`
is not a standalone composer request. It resolves a local quasi entity by CID.
Calling it before that entity exists produces the observed `quasi entity not
found` failure.

For plain TEXT only, the bridge uses this bounded, single-attempt sequence after
its signed-in-user and self-P2P-chat guards:

1. Build the validated composer request:
   `{cid, channel:{id:chatId,type:1}, type:4, content:{richText}, shouldNotify:true}`.
2. Invoke `1001|im.v1.CreateQuasiMessageRequest|im.v1.CreateQuasiMessageResponse|1|CREATE_QUASI_MESSAGE`
   with that complete request. Require `response.data.cid === request.cid`.
3. Invoke `2003|im.v1.SendMessageRequest|im.v1.SendMessageResponse|1|SEND_MESSAGE`
   with exactly `{cid: response.data.cid}`.
4. Treat the response as acknowledgement only. Its canonical `messageId`, when
   present, is a bounded readback target; use the separate native message lookup
   and own-text comparison before reporting delivery. Any create, CID-match,
   send, or readback failure stops the sequence; it is never retried.

The request/response link is grounded in
`jssdk/jssdk_worklet_messenger.js`: offset 2448580 creates and persists a
quasi message with `cid`, offset 2387913 writes that same `cid` into
`CreateQuasiMessageResponse`, and offset 2234199 reads only `e.cid` and fails
when the quasi message cannot be found before invoking the send preparation.
The generated schema at offset 3870157 defines `SendMessageResponse` with
`messageId` and `netCost`; the module projects only a canonical native messageId.
This is static verification only; no additional live mutation was performed.
