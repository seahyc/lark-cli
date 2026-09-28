# Desktop runtime boundaries: meetings, Drive queue, and settings

Static-only research against installed Lark 8.0.3 webcontent archives. No
native bridge call, meeting action, transfer action, or settings change was run.
This covers the named workflows only; it does not claim every dynamically loaded
bundle or desktop feature was exhausted.

## Meetings: a concrete participant read is meeting-scoped

`video-conference-page.asar/extensions-modal/static/js/calendar-setting.js`
contains a real participant-information transport call at byte offset 317747:

```js
Fu("89403|videoconference.v1.PullParticipantInfoRequest|videoconference.v1.PullParticipantInfoResponse|1", {byteviewUsers:[e.byteviewUser],meetingId:e.meetingId})
  .then(e => e.userInfos.map(...))
```

The following consumer maps `e.user.userId`, `e.displayName || e.fullName`,
and `e.avatarKey` at offset 318080. This grounds one command, outer request
keys, and a narrow response use. It is still not a general read descriptor:
the syncer input contains both `meetingId` and a structured `byteviewUser`, and
this caller does not establish a complete `byteviewUser` schema or a safe source
for a meeting ID. A real call needs active/selected meeting context plus meeting
and tenant authorization. Neither identifier may be synthesized.

`GET_VC_MEETING_JOIN_STATUS` is declared in
`main-window.asar/157766/91f3b16f27.js:141577`, but static search found only
its command/export, not a caller with a request transform and response consumer.
`SYNC_MEETING_STATUS` is similarly declaration-only (for example,
`video-conference-live-page.asar/logic/static/js/main.js:422312`) and is
excluded because it may synchronize current meeting state.

**Actionable boundary:** participant detail needs a traced provider for the
complete active-meeting `byteviewUser` shape, followed by permitted read-only
validation in an authorized meeting. Join status, lobby, host, recording,
subtitles, device, and sync workflows remain unimplemented.

## Drive transfer queue: native Rust manager, not SDK bridge

The queue has grounded no-payload native-manager reads:

* Download manager at
  `space.asar/js/73336.604fe9312e2d4721653a.no-online.js:7078` calls
  `this.rust.callApi(y.si.GetDownloadProcessList)`. Its ready callback emits
  the untouched result as `TasksAdded` at offset 1004.
* Upload manager at
  `space.asar/js/10657.0f26cde666705be67cff.no-online.js:6656` calls
  `this.rust.callApi(L.si.GetUploadProcessList)`. Its ready callback likewise
  emits the untouched result at offset 843.

`space.asar/js/14954.67cc8241e798e6b617c3.no-online.js` maps these to
`SUITE_DRIVE_TASK_GET_DOWNLOAD_PROCESS` at 192569 and
`SUITE_DRIVE_TASK_GET_UPLOAD_PROCESS` at 192875. The managers register push
events for additions, removals, changes, speed, and subtask failures. This is
local desktop transfer runtime state, likely including file/path metadata.

No inspected manager caller supplies a `LarkAPI.transport.callSdkApi` route, a
bounded task-record schema, or a response consumer that proves non-secret
fields. The whole Rust result is emitted. Do not route these commands through
the shared SDK bridge or expose the raw queue. A future read needs a sanctioned
Rust adapter with an explicit safe-field allowlist, or a separately traced
browser bridge wrapper.

## Preferences and workplace navigation: declarations only

`main-window.asar/157766/91f3b16f27.js` declares:

* `GET_USER_SETTINGS` at 131102 as
  `5110|settings.v1.GetUserSettingsRequest|settings.v1.GetUserSettingsResponse|1|GET_USER_SETTINGS`,
  exported as `TjS` at 66019.
* `GET_NAVIGATION_APPS` at 174709 as
  `5231|settings.v1.GetNavigationAppsRequest|settings.v1.GetNavigationAppsResponse|1|GET_NAVIGATION_APPS`,
  exported as `JTJ` at 61480.

Searching installed static webcontent JavaScript for those names, exports, and
readable API aliases found declarations/re-exports but no concrete call with a
request object and response-field consumer. That does not prove absence: a
caller may be unreferenced, dynamic, or outside this static path.

**Actionable boundary:** do not infer an empty request or return a raw settings
blob. It can contain account, security, device, and tenant configuration.
Navigation apps are tenant-dependent and need a traced consumer-driven
allowlist before a typed read is safe.

## Verification boundary

Static evidence does not prove account access, an active meeting/transfer,
initialized device state, or request success. Any later validation must be
read-only in a known authorized tenant/meeting/device context. Meeting actions,
sync operations, and queue-control operations remain excluded.
