# Native triage protocol research

## Favorites

`favorite.asar:favorite/b736c5d828.js` at byte offset `12562774` exports the
desktop wrapper functions:

```js
function c({messageIds:e,chatId:t="",originMergeForwardId:n}) {
  return (0,r.px)(i.YNq,{messageIds:e,chatId:t,originMergeForwardId:n})
}
function s(e) { const t={ids:e}; return (0,r.px)(i.O2S,t) }
```

The command table in `messenger.asar:44343/f127c4a4eb.js` maps `YNq` to
`2240|favorite.v1.CreateFavoritesRequest|favorite.v1.CreateFavoritesResponse|1|CREATE_FAVORITES`
and `O2S` to
`2242|favorite.v1.DeleteFavoriteRequest|favorite.v1.DeleteFavoriteResponse|1|DELETE_FAVORITES`.

The favorite detail view calls deletion as `await B.C5(e)` and removes the IDs
locally only after resolution (`favorite.asar:favorite-detail-viewer/d7d0d432f9.js`,
offset `43374834`). It does not consume a server-returned ID. A controlled
fixture must therefore identify its favorite ID by pre/post `GET_FAVORITES`
comparison, then delete and re-read it.

## Scheduled simple-text path: typed two-call sequence

`jssdk/jssdk_worklet_messenger.js` offset `2518676` proves the schedule patch
handler. It resolves the stored scheduled item, maps plain TEXT to type `4`, and
for `DELETE` calls the native SDK with `patchType:3`, `messageId`, `chatId`,
`scheduleTime`, `type`, omitted content, `sendImmediately:false`, and omitted
reply data. The composer passes `{channel,type,content,cid,scheduleTime}` to
`CREATE_QUASI_MESSAGE` (`messenger.asar:app-chat~1/60390bf70c.js`, offset
`123826`). This creates a local `quasiScheduleMessage` record; it is not the
server persistence step. Worklet function `U` (`jssdk` offset `2521950`) loads
that record by `cid` and calls `g.YSk` with exactly:

```js
{ putRequest: e, scheduleTime: i.scheduleTime, chatId: i.chatId,
  entityId: i.threadId || i.chatId }
```

`g.YSk` is legacy server command
`900030|messages.PutScheduleMessageRequest|messages.PutScheduleMessageResponse`
(`jssdk` offset `3364369`). Its `putRequest` is module `9510` function `T`
(`jssdk` offset `2158100`): TEXT type, content, chatId, empty root/parent IDs,
cid, `isNotified`, version `1`, and the optional thread/sync/AI/reply fields.
On success `U` requires `response.message`, removes the local quasi record, and
saves the returned scheduled message (`jssdk` offset `2522350`).

The editor proves the unit: `messenger.asar:common/49dc2094d2.js` at offset
`30024` checks an existing item with
`dayjs(1e3 * Number(i.scheduleTime)).isBefore(dayjs())`. Thus `scheduleTime`
on the native wire and list record is a decimal Unix-seconds string. The picker
returns `n.scheduleTime` and passes it unchanged into the PATCH handler at
offsets `31910` and `32974`.

The resulting response identifiers are not consumed directly by the compose
callsite. `GET_SCHEDULE_MESSAGES` list readback is authoritative: it maps a
server item from `messageId`, `scheduleTime`, `status`, and `scheduleMessage`
(`jssdk` offset `2527735`).

## Other unresolved paths

`CREATE_CHAT_LAST_READ_POSITION`, `BATCH_MUTE_FEED_CARDS`, and
`QUERY_MUTE_FEED_CARDS` likewise require exact before/after and rollback caller
traces. The draft cloud companion remains feature-gated; native storage
readback does not establish composer UI parity.

## Favorite create request correction

The `CREATE_FAVORITES` wire is **not** the high-level merge helper shape
`{messageIds,chatId,originMergeForwardId}`. The installed `pb.desc`,
`favorite/v1/favorite.proto`, defines:

```text
favorite.v1.CreateFavoritesRequest
  favs = 1 repeated FavoritesTarget
FavoritesTarget
  id = 1 required string
  type = 2 required basic.v1.FavoritesType
  chat_id = 3 optional int64
  origin_merge_forward_id = 6 optional string
basic.v1.FavoritesType.FAVORITES_MESSAGE = 1
```

The corrected direct-SDK request for one verified self-chat message is:

```js
{favs:[{id: messageId, type: 1, chatId, originMergeForwardId: ""}]}
```

`MergeFavoriteRequest` is the separately declared request with
`message_ids`, `chat_id`, and `origin_merge_forward_id`; it must not be used
for `2240|favorite.v1.CreateFavoritesRequest|favorite.v1.CreateFavoritesResponse|1|CREATE_FAVORITES`.
No live Favorite write was made after this correction.
