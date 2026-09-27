# Triage protocol status

## Favorites: ready for parent orchestration

The installed Favorites add wrapper accepts
`{messageIds, chatId, originMergeForwardId}`. The selected-message helper in
`favorite.asar:favorite/b736c5d828.js` forwards that exact envelope to
`CREATE_FAVORITES`. The delete helper forwards `{ids}` to `DELETE_FAVORITES`.
The visible Favorites view consumes list/detail reads, but neither mutation
callsite consumes a response ID. The descriptors therefore report only a
resolved native transport acknowledgement, never an applied-state claim.

Parent live procedure for a self-chat fixture:

1. Resolve and identity-check the signed-in user's native self P2P chat, then
   obtain the native message ID from metadata/history.
2. Read `messaging.favorites` before applying. Refuse if the target message is
   already favorited or the comparison cannot identify a single new Favorite.
3. Apply `triage.favorite-add` with that one native message/chat pair.
4. Re-read `messaging.favorites`; find exactly one new Favorite whose consumed
   content references the fixture message. Its native Favorite ID is cleanup
   input, not an ID inferred from the create acknowledgement.
5. Apply `triage.favorite-delete` to that ID and re-read until it is absent.

## Remaining paths

Scheduled simple-text fixtures require two desktop calls. `CREATE_QUASI_MESSAGE`
creates only the local scheduled-quasi record. The worklet then calls legacy
server API `900030|messages.PutScheduleMessageRequest|messages.PutScheduleMessageResponse`
with a typed `{putRequest,scheduleTime,chatId,entityId}` envelope. The
`triage.schedule-self-text` descriptor exposes both steps: the parent must
require the returned local `cid`, build its `persistRequest(stage, prepared)`,
and use the server transport for `persistCommand`. The patch handler maps
`DELETE` to discriminator `3` and sends `{patchType,messageId,chatId,scheduleTime,
type,content,sendImmediately,partialReplyInfo}`. The restricted cancel descriptor
models only the UI's simple TEXT fixture path, whose delete call has no content
and type `4`.

For a far-future self-chat test: compute the target Unix time in seconds as a
canonical decimal string; the scheduled-message editor renders native values as
`1000 * Number(scheduleTime)`. Stage and persist once, read `triage.scheduled-items`
until the new exact fixture is pending and exposes its native message ID and
schedule timestamp, cancel using those exact values, then read again for absence
or delete status. Do not run the cancel against a pre-existing item.

The installed client has `CREATE_CHAT_LAST_READ_POSITION` and feed mute commands,
but this pass did not establish readback and rollback payloads. They remain
research items rather than generic setting mutations. Draft storage is proved,
while the cloud/local companion feature gate prevents claiming composer UI parity.
