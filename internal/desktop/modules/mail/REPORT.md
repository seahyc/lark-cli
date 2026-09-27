# Mail desktop module evidence

## Runnable, read-only operations

The implementation exposes only wrappers whose request transforms and response
shapes are present in the extracted Lark 8.0.3 mail static JavaScript:

| Operation | Native command | Exact request |
| --- | --- | --- |
| `mail.signatures` | `MAIL_GET_SIGNATURE` | `{accountId, fromSetting}` |
| `mail.blocked-sender` | `MAIL_GET_BLOCKED_ADDRESSES` | `{addresses:[address]}` |
| `mail.schedule-status` | `MAIL_GET_SCHEDULE_SEND_MESSAGE_COUNT` | `{}` |
| `mail.account-metadata` | `MAIL_GET_ACCOUNT` | `{fetchDb, fetchCurrentAccount:false}` |
| `mail.verified-auto-transfer-emails` | `MAIL_GET_AUTO_TRANSFER_EMAIL_REQUEST` | `{accountId}` |
| `mail.global-forwarding` | `MAIL_GET_RULES` | `{version:"2", isGlobalTransfer:true, accountId}` |

Evidence is recorded in `manifest.json`. The sources include the actual
`Vy(...)` wrapper calls, not just catalog declarations. Account projection
explicitly excludes `accountToken`; it also excludes vacation-responder body,
signature HTML, image tokens, and all unrecognized response fields.

Bounded static excerpts (all from `/tmp/lark-private-api-research/mail`):

```text
common/9a1f1c38eb.js module 206153:
  (0,r.Vy)(n.aM1,{reqOptions:{transform:(e,t)=>({accountId:e,fromSetting:t})
  resOptions:{transform:e=>e,logFormat:{signatures:[...],signatureUsages:[...]}}

common/41b8ba5175.js:
  (0,d.Vy)(s.H32,{reqOptions:{transform:e=>({addresses:[e]})
  resOptions:{transform:(e,[a])=>!!e.blockedAddresses?.some(e=>e===a)}})

common/92819f09f6.js:
  (0,l.Vy)(s.omk,{resOptions:{transform:e=>Number(e.scheduleSendMessageCount??"-1")}})

common/48be8628db.js module 6630:
  transform:r=>({accountId:r}); resOptions:{transform:({emails:r=[]})=>r}
  transform:r=>({version:String(2),isGlobalTransfer:!0,accountId:r})
```

## Planned mutations; not exposed

No mutation is exposed by this module. The following exact schemas are known
from the static wrappers and remain for an approval-bound implementation:

| Command | Exact wrapper request | Source |
| --- | --- | --- |
| `MAIL_ADD_SIGNATURE` | `{accountId, signature:{id,signatureType,signatureDevice,name,templateHtml,templateValueJson,images}}` | `common/9a1f1c38eb.js` module 206153 (`n.xHx`) |
| `MAIL_UPDATE_SIGNATURE` | `{accountId, signature:{id,signatureType,signatureDevice,name,templateHtml,templateValueJson,images}}` | same module (`n.Z9i`) |
| `MAIL_UPDATE_SIGNATURE_USAGE` | `{accountId, signatureUsage}` | same module (`n.rKc`) |
| `MAIL_ADD_USER_ALLOW_BLOCK` | `{multiFrom:[{address}],isAllow,scene:1}` | `common/bc90fef3a3.js` module 708241 (`_.Kgq`) |
| global-forwarding create/update | rule serialization and account context are described in `docs/research/lark-desktop-mail-protocol.md` | `common/48be8628db.js` module 6630 |

Signature write payloads include HTML and image fields. A safe command needs a
separate review/approval flow, bounded image handling, readback, and rollback
plan. Native block and allow-list writes have a shipped request transform, but are not
live-tested because the legacy server cleanup/list route failed before a
reliable cleanup proof could be established.

## Limits

Unit tests use synthetic static-code-shaped fixtures only. No native bridge,
desktop profile, credential, network, or live mailbox operation was tested.
Static schemas still need live confirmation for permission errors, account
context handling, pagination/cache behavior, and UI-visible state.


## Legacy sender-list/remove research status

`legacy.js` and `legacy-manifest.json` retain the exact three-part server wire
schemas for `SearchUserAllowBlock` and `DeleteUserAllowBlock`. They are not
auto-registered or executable. A direct live server search with valid
`cursor:"0"` and `lastTimestamp:"0"` returned `ErrCacheEmpty`. That error is
not an empty-list result; the cause is unresolved. The extracted module defines
those legacy wrappers as discarded expressions, while the current UI block/trust
flow uses `MAIL_ADD_USER_ALLOW_BLOCK`. Native block/allow-list writes therefore
remain schema-verified but live-untested until a cleanup-capable route is proven.
