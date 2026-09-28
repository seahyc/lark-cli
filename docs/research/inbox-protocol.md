# Self-chat mute protocol

Date: 2026-09-22. This workflow is static-schema verified only. No Lark call,
chat-setting mutation, or rollback ran during this research.

## Native path

The shipped generated protobuf registry, not a convenience wrapper, defines:

- `chats.PullChatsByIdsRequest` with repeated `chatIds` at field 1.
  `PullChatsByIdsResponse` returns a `chats` map of `entities.Chat` records.
  The read descriptor submits exactly one ID and rejects a response containing
  zero or more than one record.
- `chats.PatchChatSettingRequest` with required `chatId` field 1,
  `chatSetting` field 2, and repeated `updateChatSettingField` field 3.
- `entities.ChatSetting.isRemind` is Boolean field 1 with default `true`.
  `entities.ChatSetting.Field.IS_REMIND` is enum value `1`.

The desktop `updateChatSetting` call site builds precisely this narrow patch:
it sets `chatSetting.isRemind` and adds only `IS_REMIND` to
`updateChatSettingField`, then invokes command 60. Therefore native `isRemind`
is the inverse of the user-facing mute switch: `false` means muted and `true`
means reminders enabled.

| Descriptor | Exact command | Request | Bounded result |
| --- | --- | --- | --- |
| `inbox.chat-mute-state` | `64|chats.PullChatsByIdsRequest|chats.PullChatsByIdsResponse` | `{chatIds:[chatId]}` | `{chatId, muted: 0\|1}` |
| `inbox.set-self-chat-mute` | `60|chats.PatchChatSettingRequest|chats.PatchChatSettingResponse` | `{chatId,chatSetting:{isRemind: muted === 0},updateChatSettingField:[1]}` | acknowledgement only; parent must read back |

`expectedMuted` is a descriptor input guard only. It is intentionally omitted
from the native patch and must be checked against a fresh
`inbox.chat-mute-state` result before the write.

## Parent live-fixture protocol

1. Resolve the current signed-in user and obtain their exact self-chat ID using
   the existing P2P/self-chat identity path. Reject P2P chats whose peer is not
   the signed-in user.
2. Call `inbox.chat-mute-state` for that one ID. Retain its `muted` value as
   `expectedMuted` and the exact prior state for rollback.
3. Preview `inbox.set-self-chat-mute` with the opposite requested `muted`
   value. Confirm its payload contains only `chatId`, `chatSetting.isRemind`,
   and `[1]` as `updateChatSettingField`.
4. Apply once. Do not retry a failed native write.
5. Re-read the same one ID and require the requested state. An acknowledgement
   is not verification.
6. Apply one rollback patch with the original state and a fresh expected state;
   re-read and require the original state. Do not extend this sequence to any
   other chat.

The read response may omit generated default `isRemind:true`; its projection
correctly treats that as `muted:0`. A non-Boolean present value, an invalid
chat ID, or a response map with anything other than one entry fails closed.

## Static evidence

- `jssdk/jssdk_worklet_messenger.js:658418` registers
  `chats.PatchChatSettingRequest` with fields `chatId`, `chatSetting`, and
  `updateChatSettingField`.
- `jssdk/jssdk_worklet_messenger.js:718191` registers
  `entities.ChatSetting.isRemind` and enum `Field.IS_REMIND:1`.
- `jssdk/jssdk_worklet_messenger.js:1810200` is the desktop caller that builds
  the setting-only patch and calls the native command.
- `jssdk/jssdk_worklet_messenger.js:3354728` maps command 60 to
  `chats.PatchChatSettingRequest` and command 64 to
  `chats.PullChatsByIdsRequest`.
- `jssdk/jssdk_worklet_shared.js:532754` registers the generated
  `PullChatsByIdsResponse` `chats` map consumed by the scoped read.

The legacy `chats.*` generated registry is shipped in the JSSDK worklet and is
not included in the separate `Libraries/pb.desc` client descriptor. The
worklet registration is the native protobuf schema used by commands 60 and 64.
