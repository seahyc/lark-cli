# Native request-schema audit

Date: 2026-09-22. This is a read-only comparison of registered descriptors with
`/Applications/LarkSuite.app/Contents/Frameworks/Lark Framework.framework/Versions/Current/Libraries/pb.desc`.
It uses the generated protobuf message named in each command, rather than a
higher-level JavaScript convenience wrapper. A matching field name is only a
wire-shape finding; it is not evidence of a live call.

`messages.*` server commands and the ShellAPI navigation call are not present
in that client descriptor set. They are explicitly marked as outside this
particular protobuf audit rather than treated as client-SDK messages with a
similar name.

## Findings requiring dispatch awareness

| Registered operation | Command and emitted request | Native descriptor finding | Result |
| --- | --- | --- | --- |
| `messaging.send-self-text` | `2003|im.v1.SendMessageRequest...`; descriptor emits composer data (`cid`, `channel`, `type`, `content`, `shouldNotify`) | `im.v1.SendMessageRequest` has only `cid` (required), `fileHandle`, `preprocessKey`, and `strictSendOrder`. | The descriptor output is **not** a direct 2003 request. It is valid only through the bridge's documented two-step composer path: `CreateQuasiMessageRequest` followed by 2003 with `{cid}`. Calling the generic SDK dispatcher with this descriptor would silently discard `channel`, `type`, `content`, and `shouldNotify`. |
| `messaging.send-self-post` | Same command and composer-shaped output | Same four-field `im.v1.SendMessageRequest`. | Same dispatch dependency and silent-drop risk as text send. |
| `triage.schedule-cancel-text` | `900034|messages.PatchScheduleMessageRequest...`; `{patchType,messageId,chatId,scheduleTime,type,sendImmediately}` | The installed *client* descriptor contains a different command: `39008|im.v1.PatchScheduleMessageRequest...`, whose fields are `chatId`, `patchType`, `patchObject`, `scheduleTime`, `messageType`, `content`, `isSendImmediately`. | Do **not** substitute the 39008 schema for 900034. A direct 39008 call needs `patchObject` (`ScheduleMessageItem` with `itemId` and `itemType`); the registered operation is intentionally a server transport. `messages.PatchScheduleMessageRequest` is absent from installed `pb.desc`, so its registered server payload cannot be re-certified from this file. Existing static server-helper evidence must remain the authority for 900034. |
| `triage.schedule-self-text` persistence leg | `900030|messages.PutScheduleMessageRequest...`; `putRequest` envelope | No `messages.PutScheduleMessageRequest` occurs in installed client `pb.desc`. | The `CreateQuasiMessageRequest` stage is schema-grounded below; the server persistence leg requires a server-schema/static-helper audit and must not be represented as directly checked here. |
| `workspacenext.navigation-layout` | `ShellAPI.app.navigation.getNavigationInfo`, `{}` | ShellAPI is not a protobuf transport. | Out of scope for `pb.desc`; its version-2 projection must be checked against live ShellAPI output separately. |

The second row is especially important: the successful two-step implementation
must preserve its special bridge branch. Registering these composer descriptors
as ordinary `callSdkApi(command, request(input))` requests is unsafe.

## Direct protobuf matches

| Area / operation | Generated request schema | Registered request result | Audit result |
| --- | --- | --- | --- |
| outbound `messaging.text-message` | `im.v1.MGetMessagesRequest`: `messageIds` (repeated string), optional `syncDataStrategy` default `TRY_LOCAL` | `{messageIds:[messageId]}` | Matches. |
| outbound `messaging.edit-text` | `im.v1.EditMessageRequest`: `msgId`, `cid`, `type`, `content`, `chatId`, optional `extraData` | `{msgId,chatId,cid,type:4,content}` | Top-level field shape matches. `content` is `basic.v1.EditMessageContent`; the rich-text nesting still needs the existing message-content evidence, not a name-only conclusion. |
| rich `messaging.edit-post` | Same `im.v1.EditMessageRequest` | `{msgId,chatId,cid,type:2,content}` | Same top-level match and nested-content qualification. |
| triage `favorite-add` | `favorite.v1.CreateFavoritesRequest`: `favs` repeated `FavoritesTarget`; target fields `id`, `type`, `chatId`, `translationLanguage`, `originMergeForwardId` | `{favs:[{id,message type:1,chatId,originMergeForwardId:''}]}` | Matches current code. `FavoritesType.FAVORITES_MESSAGE` is generated value `1`. The obsolete wrapper shape `{messageIds,chatId,originMergeForwardId}` would have been wrong, but is not the current descriptor output. |
| triage `favorite-delete` | `favorite.v1.DeleteFavoriteRequest`: `ids` repeated string | `{ids:[favoriteId]}` | Matches. |
| triage `scheduled-items` | `im.v1.GetScheduleMessagesRequest`: `syncDataStrategy`, `scene`, `chatId` | `{chatId,syncDataStrategy:3,scene:1}` | Matches. |
| triage scheduled creation stage | `im.v1.CreateQuasiMessageRequest`: required `type`, `chatId`, `content`; optional `channel`, `cid`, `shouldNotify`, `scheduleTime` | `{cid,channel,type,content,shouldNotify,scheduleTime}` | Field names and nesting match. `content.richText` is generated by `basic.v1.QuasiContent.richText`. |
| mailnext draft metadata | `email.client.v1.MailGetDraftItemRequest`: `draftId`, optional `params` | `{draftId}` | Matches; omitting optional render params is valid. |
| mailnext draft create | `email.client.v1.MailCreateDraftRequest`: `originMessageId`, `action`, `threadId`, `smartReplyText`, `timeText`, `needSignature`, `isFromLarkPlaintext`, `feedCardId` | `{threadId:'',originMessageId:'',action:0,needSignature:false,timeText:'',isFromLarkPlaintext:false}` | Matches supplied fields; omitted fields are optional. |
| mailnext draft delete | `email.v1.DeleteMailDraftRequest`: `messageId`, `threadId`, optional `feedCardId` | `{threadId,messageId}` | Matches. |
| mailnext draft update top level | `email.client.v1.MailUpdateDraftRequest`: `draftId`, `payload`, `isDelay`, optional `ffiEvent`, `feedCardId`, `onlySaveLocal` | `{draftId,payload,isDelay:false,onlySaveLocal:false}` | Top-level fields match. |
| mailnext signature update | `email.client.v1.MailUpdateSignatureRequest`: `signature`, required `accountId` | `{accountId,signature}` | Top-level fields match. `MailSignature` defines `id,name,templateHtml,templateValueJson,images,signatureType,signatureDevice`; the descriptor requires all seven before forwarding its snapshot. |
| workspacenext device preferences | `device.v1.GetDeviceNotifySettingRequest`: `syncDataStrategy` | `{syncDataStrategy:3}` | Matches. |
| workspacenext navigation apps | `settings.v1.GetNavigationAppsRequest`: `platform` | `{platform:1}` | Matches field shape. |
| workspacenext custom-status read | `contact.v1.GetUserCustomStatusRequest`: `syncDataStrategy` | `{syncDataStrategy:3}` | Matches. |
| workspacenext time-format read | `settings.v1.GetUserSettingRequest`: optional `isFromServer` default true and `syncDataStrategy` default `FORCE_SERVER` | `{}` | Matches default omission. |
| workspacenext resource equipment read | `calendar.v1.GetResourceEquipmentsRequest` has no fields | `{}` | Matches. |
| workspacenext navigation-order mutation | `settings.v1.ModifyNavigationOrderRequest`: `mainNavigation`, `shortcutNavigation`, `platform`, optional `isNewTabContainer`; nested `NavigationUniqueId` requires `id`,`appType` | `{mainNavigation:[{id,appType}],shortcutNavigation:[{id,appType}],platform:1}` | Matches. This does not prove that a ShellAPI layout's string `type` maps to `NavigationAppType`; the caller-side guard must continue rejecting unproven conversions. |
| workspacenext custom-status interval mutations | `contact.v1.UpdateUserCustomStatusRequest`: `updateStatus`; nested meta includes `id`, `effectiveInterval`, `lastCustomizedEndTime`, `fields` | `{updateStatus:[{id,effectiveInterval,lastCustomizedEndTime,fields:[4,6]}]}` | Matches field shape. Generated enum values 4 and 6 are `EFFECTIVE_INTERVAL` and `LAST_CUSTOMIZED_END_TIME`. |

## Qualified nested checks

The following have valid top-level schemas but cannot be fully re-certified
from their names alone.

- `mailnext.draft-fixture-update` forwards a nested `Payload`. The descriptor
  confirms `from`, recipient arrays, `bodyHtml`, images, attachments,
  `docsPermissions`, `threadId`, `isSendSeparately`, calendar/read-receipt
  fields, `bodySummary`, and `isFromLarkPlaintext` in generated data. One
  field-name string is redacted/obfuscated in the shipped descriptor output at
  field 7; therefore this audit does not call the descriptor's `subject`
  spelling protobuf-proven solely from `pb.desc`. Its live/UI fixture evidence
  is separate evidence.
- `messaging.edit-text` and `messaging.edit-post` rely on the generated
  `EditMessageContent` nested content format. The protobuf type confirms the
  `richText` container, but a byte-exact rich-text element graph requires the
  existing static UI serializer evidence.
- `mailnext.signature-fixture-update-text` preserves a bounded
  `MailSignature` snapshot and alters `templateHtml`. The seven generated
  `MailSignature` fields match the required snapshot vocabulary. Because the
  implementation spreads the raw snapshot, any additional UI-only keys would
  be silently ignored by protobuf serialization; a strict allowlist or a
  snapshot-key assertion is still needed before calling it schema-complete.
  A response projection/readback remains necessary to prove persistence.

## Required interpretation

No current registered direct request was found emitting the old favorite
`messageIds` top-level wrapper. The only client-SDK patch request in the
installed descriptor is 39008 and requires `patchObject`; it must remain
separate from the registered 900034 server patch. The material remaining risk
is dispatch: composer-shaped send descriptors must never bypass their
CreateQuasi/Send bridge, and legacy `messages.*` server requests cannot be
validated by the installed client `pb.desc` alone.
