# Desktop workspace next protocols

Static research against the installed Lark desktop archives only. No bridge,
UI, native manager, or live account action was performed.

## Supported: 12/24-hour display preference

The custom-status quick view calls `GET_USER_SETTING` with `{}` at
`main-window.asar/customStatusModal/41d68de6bf.js:157867`. It reads only
`timeFormat.timeFormat` and compares that value with the shipped enum
`TWELVE_HOUR: 1` and `TWENTY_FOUR_HOUR: 2`.
`workspacenext.time-format-preference` therefore returns only `hourCycle: 12`
or `hourCycle: 24`; it omits all other settings fields. Parent live workflow:
read it while signed in, confirm it matches the desktop clock format, and make
no change or cleanup.

## Supported: calendar resource equipment catalog

The calendar resource filter has a complete static chain. The command literal
is `3052|calendar.v1.GetResourceEquipmentsRequest|calendar.v1.GetResourceEquipmentsResponse|1|GET_RESOURCE_EQUIPMENTS`
in `calendar.asar/2216_e8bf809b.js:78439`. Its generated wrapper is exported
as `KN` in `common-calendar-packages_d38e796c.js:239479`. The resource filter
calls it with `{}` and returns `equipmentLists` plus `equipmentIds` at 286345.
The room-card UI consumes each equipment's `i18nName` at
`chunk_meeting-room-card_f0e9f31c.js:14399`.

`workspacenext.calendar-resource-equipments` sends `{}` and returns at most 100
response-order IDs with only `i18nName`. It excludes room
availability, booking state, resource/tenant graphs, and unconsumed fields.

Parent live fixture workflow: call this read with `{}` while signed in to a
known tenant, confirm response items are bounded equipment catalog entries,
and compare one returned ID/name with Calendar's room filter. This is a read
only operation: there is no before/after mutation and no cleanup.

## Traced but not exposed

| Area | What static source establishes | Why no descriptor or mutation exists |
| --- | --- | --- |
| Navigation apps | `GET_NAVIGATION_APPS` remains a declared settings command. Existing static scans did not establish its payload or a bounded consumer. | Empty input and raw app fields would be guesses. |
| User preferences | Broad `GET_USER_SETTINGS` is still declaration-only. A separate `GET_USER_SETTING` quick-view chain grounds only `timeFormat.timeFormat`, exposed as `workspacenext.time-format-preference`. | The broad settings blob can contain security, device, tenant, and account data; it remains excluded. |
| Custom status current/restore | The status modal grounds the force-server read already exposed by `workspace.user-custom-status`: `syncDataStrategy: FORCE_SERVER` and consumed status-card fields. Its edit modal reloads that list after confirmation. | The installed setting archive did not yield a complete `UPDATE_USER_CUSTOM_STATUS` UI request transform and response consumer. A restore mutation must retain and replay an exact pre-read status object, so it is not safe to infer from the command declaration. |
| Task comment draft/history/chat-linked tasks | The Todo command table declares `GET_TODO_COMMENT_DRAFT` and `GET_TODO_HISTORY_RECORDS`. | No caller established the required task/container/comment identifiers, cursor shape, or bounded response fields. |
| Meeting status | The existing runtime boundary traces participant info but leaves join-status request identity unresolved. | `GET_VC_MEETING_JOIN_STATUS` remains declaration-only; never synthesize an active meeting or device context. |
| Drive queue | Existing runtime boundary traces Rust `GetDownloadProcessList`/`GetUploadProcessList` calls. | They are not SDK bridge calls and emit raw local queue objects that can contain file/path metadata. A sanctioned native adapter with an allowlist is required. |

Any future custom-status restoration test must be performed only after a
specific supported update request exists: read the pre-status, apply one
approved change, read back, restore that exact pre-status, read back again, and
report both comparisons. That workflow is intentionally not executable now.

## Supported mutation: clear one selected custom status

The main-window custom-status card follows the selected status through its
click handler into `pa(status, false)`. The actual close branch calls
`UPDATE_USER_CUSTOM_STATUS` with `updateStatus` containing that status ID,
an interval of `startTime: "0"`, `endTime: "0"`, and `isShowEndTime: true`,
then `lastCustomizedEndTime: "0"`. The shipped `StatusField` enum assigns
`EFFECTIVE_INTERVAL` to `4` and `LAST_CUSTOMIZED_END_TIME` to `6`, which are
the sole `fields` values sent. The UI consumes `response.status` only to
select the currently active status after the change.

`workspacenext.clear-custom-status` exposes exactly that branch. Parent live
workflow: force-server read `workspace.user-custom-status`; select an active
status ID belonging to the signed-in user; obtain an explicit approved change;
apply once; force-server read back and confirm that status's interval is
cleared. It does not attempt a restore because the shipped clear branch cannot
replay the full pre-change status safely. Any later restoration requires a
separately grounded request that copies only documented fields, followed by
another force-server comparison.

## Supported bounded restore sequence: custom-status interval

`workspacenext.custom-status-intervals` force-reads status IDs plus precisely
the `effectiveInterval` and `lastCustomizedEndTime` fields used by the
shipped active-status selector. The companion `restore-custom-status-interval`
mutation follows the shipped custom-end-time branch: it writes only the
pre-read interval and custom end fields with field IDs `[4, 6]`; it does not
replay titles, icons, timing presets, or sync settings. This preserves those
unrelated fields while undoing a prior supported clear.

Parent fixture workflow: force-read the intervals; retain one active signed-in
status item; with explicit approval clear that status once; force-read to
confirm it changed; apply the restore using the saved values for the same ID;
force-read to confirm all five captured values match the pre-read snapshot.
There is no automatic retry. This is a visible user-status mutation and must
not be used without the fixture-specific approval.

## Provider trace and remaining global boundaries

The main-window status modal's local `ra` adapter constructs SDK calls as
`LarkAPI.transport.callSdkApi(command, payload, {parentContextId, collectTrace,
extendParams, useFast})`; it is the same transport family used by the desktop
bridge. A full static text scan of the installed framework resources finds
`GET_NAVIGATION_APPS` and broad `GET_USER_SETTINGS` only in generated command
registries and re-export tables, not a UI call with a payload and bounded
response consumer. That makes the provider known but their request schemas
unknown; neither descriptor is exposed. The safe no-argument `GET_USER_SETTING`
time-format branch above is independently grounded and does not justify
calling broad settings APIs.

Drive transfer queues remain outside this provider: their runtime-boundary
calls are Rust manager methods and return local task/file-path graphs. Meeting
join status has a registered SDK command but no static caller yielding the
active meeting identity. Both remain excluded pending a typed adapter or an
active-context caller; no meeting or transfer state was created while tracing.

## Supported with schema-only UI verification pending: desktop navigation apps

The framework-native descriptor set at
`Lark Framework.framework/Versions/147.0.7727.149/Libraries/pb.desc` defines
`GetNavigationAppsRequest.platform` as field 1 of enum
`settings.v1.NavigationPlatform`, whose `NAV_PC` value is `1`; the request is
therefore exactly `{platform: 1}`, not `{}`. Its response contains repeated
`appInfo` values of `basic.v1.NavigationAppInfo`; the descriptor identifies
`id`, `key`, `appType`, `name`, and `displayName` fields. The descriptor exports
only safe, bounded `id`, `key`, `appType`, and `displayName` values and excludes
URLs, logos, extras, and platform detail. No shipped JS sidebar caller was
located, so parent must run this read while signed in and compare a sample to
the current desktop sidebar before marking it UI-verified. No cleanup applies.

The same descriptor proves broad `GetUserSettingsRequest` is **not** empty: it
requires `syncDataStrategy` and accepts a repeated `fields` selector, plus
`needDomain`, `ignoreTtl`, and `configPriority`. Its response is an arbitrary
string settings map and `DomainSettings`. Since the descriptor supplies no
safe field-name allowlist, this task does not synthesize a selected-preferences
call or return a settings blob.

## Drive queue route: separate ShellAPI SDK adapter, not LarkAPI

The extracted Drive task-manager implementation at
`space.asar/js/32424.434931606b9259c86d2f.no-online.js:700` proves the exact
native envelope. `callApi` maps a named Drive method to its numeric command,
serializes `params` with `JSON.stringify`, supplies protobuf names, creates a
random `contextId`, and calls `ShellAPI.app.sdk.invokeAsync(envelope)`. The
native response arrives as JSON in `result` and is parsed by that wrapper.

The no-argument queue reads are `GetDownloadProcessList` (numeric command
6034, protobuf `space.drive.v1.GetDownloadProcessListRequest/Response`) and
`GetUploadProcessList` (6047, corresponding protobuf pair). The native
`pb.desc` confirms both requests are empty. It also shows why a raw result is
unsafe: records contain paths, names, tokens, parent tokens, mount points,
error messages, response data, and extra info. A future **separate ShellAPI
adapter** could project only `uuid`, `status`, `size`, transferred size,
`progress`, `finishedCount`, `totalCount`, and `errorType`, with a bounded
list. It cannot use `LarkAPI.transport.callSdkApi`, which is a different
transport surface, and is not exposed by this module.

## Supported with schema-only UI verification pending: device notification preferences

`pb.desc` defines `GetDeviceNotifySettingRequest.syncDataStrategy`, defaulting
to `FORCE_SERVER`; the shared `SyncDataStrategy` enum assigns that value `3`.
The response wraps `basic.v1.DeviceNotifySetting`, whose scalar fields are
`disableMobileNotify`, `stillNotifyAt`, `showMessageDetail`, and
`stillNotifySpecialNotice`. `workspacenext.device-notify-preferences` returns
only those booleans and excludes nested `notificationSoundSetting`. Parent must
compare a live result with desktop notification settings before treating it as
UI-verified. It is read only and needs no cleanup.

## Mutation pending ShellAPI snapshot integration: navigation order

`ModifyNavigationOrderRequest` has a complete protobuf schema: it accepts both
full `mainNavigation` and `shortcutNavigation` lists of `{id, appType}` plus
desktop `platform: 1`. `workspacenext.modify-navigation-order` validates two
bounded JSON arrays, rejects duplicates across both lists, and emits that exact
transport payload. The JSON-string inputs keep the descriptor manifest within
the scalar parameter vocabulary while still enforcing every list item.

It must only be used after the dedicated `globalThis.ShellAPI.app.navigation
.getNavigationInfo()` bridge read has supplied both lists. Parent preflight
must compare membership and types with the snapshot, allow only the approved
reorder, read back using the same ShellAPI method, restore the exact saved
lists, then compare again. The mutation’s transport acknowledgement is not
verification; it never retries automatically.

## Supported through the dedicated ShellAPI bridge: navigation layout

`workspacenext.navigation-layout` uses the parent-owned `shell-navigation`
transport, which invokes exactly `globalThis.ShellAPI.app.navigation
.getNavigationInfo()` without arguments. A live read established the current
Shell record shape as `id`, `key`, `name`, `type`, `visible`, `movable`,
`extra`, logo URLs, and logo paths. The descriptor returns schema version 2
with only required `id` and bounded native `type` (string or non-negative
32-bit integer), plus optional bounded `key`, `visible`, and `movable`.
It excludes names, URLs, paths, logos, and extras.

Both main and shortcut arrays remain complete and fail on a missing list,
invalid required field, duplicate identity within a list, or more than 100
items. It never translates Shell `type` into a settings enum and therefore
does not enable navigation reorder. Validation errors expose only bounded
property and structural key names and types, never record values except a
64-character capped native type string.

## ShellAPI navigation app-type conversion remains unproven

The generated main-window registry at `main-window.asar/common/5cec861619.js:2915`
defines numeric `entities.NavigationAppType` values named
`APP_TYPE_LARK_NATIVE` through `APP_TYPE_URL`. It does **not** establish that
`ShellAPI.app.navigation.getNavigationInfo()` serializes `AppInfo.appType` as
those names rather than a different native string representation. There is no
converter exposed in `workspacenext`; navigation reorder remains fail-closed
until a live ShellAPI snapshot or a shipped caller conversion proves the link.

## Main-window extension patch surface

`larklet.config.json` registers the `main-window` package as an ASAR package
and maps its `userPageModal` extension to `main-window/userPageModal`. The
corresponding concrete page is
`main-window.asar/userPageModal/en-US.html` (archive offset `54073554`). It
loads `userPageModal/d85698636c.js`, whose module `258638` calls
`ShellAPI.app.navigation.getNavigationInfo()`; the page also loads Lark's
native before/after JS bootstrap resources. A concrete shipped opener is
`main-window.asar/245301/2a490d777d.js:110650`: `toUserPage(userId)` measures
the viewport and calls the modal navigator with
`{key:"userPageModal", margin:{left:width,top:height}, payload:{userId},
closeAllModals:true}`. That supplies a controlled user-profile modal route
for a future interactive surface test. This establishes a named
main-window package page with the needed ShellAPI capability for any future
patch design. The manifest has no primary main-window HTML entry, so a primary
window injection point is not established here and no patch was made.
