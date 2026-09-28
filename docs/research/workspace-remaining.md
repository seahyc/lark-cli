# Desktop workspace remainder: static trace ledger

This ledger records what was actually traced in the installed Lark 8.0.3
bundles on 2026-09-22. A command declaration is not treated as a descriptor
schema. No live calls, patches, sends, credential reads, or mutations were run.

## Proven descriptor

`workspace.user-custom-status` is the sole supported workspace read. In
`setting.asar/StatusSettingModal/54c3b8b32d.js`, the literal command appears
at offset 207802. The enum `LOCAL:1,TRY_LOCAL:2,FORCE_SERVER:3` appears at
208687. The same modal's `getForceServerUserCustomStatus` calls
`Z(ee.Sdg,{syncDataStrategy:X.lDc.FORCE_SERVER})` at 116161. Its
`setStatusSettings` consumer at 115717 reads the `status` array and uses card
fields `orderWeight`, `type`, `syncSettings`, `id`, `typeV2`, `iconKey`,
`title`, and `eventName`. The descriptor fixes `syncDataStrategy` to `3` and
does not return `syncSettings` because the shape has not been bounded.

## Matrix and blockers

| Area | Native candidate and static location | Actual trace result | Current disposition |
| --- | --- | --- | --- |
| Calendar settings | `GET_CALENDAR_SETTINGS`, `js-worker.asar/worker_temperate.js:1462707` | The worker binds `gb` to `x.c0f`; at 980984 it calls `gb({})`, destructures `{settings}`, and merges it into its settings service. At 980689 its consumers read `timezone` and `recommendEventFromChat`. | Added `workspace.calendar-settings`; only those two fields are returned. |
| Calendar resources | `GET_BUILDINGS`, `GET_RESOURCE_EQUIPMENTS`, calendar command table near `calendar.asar/2216_e8bf809b.js:70799` | Resource identity, pagination, and response fields remain untraced. | No descriptor. |
| Task settings | `GET_TODO_SETTING`, `todo.asar/4418/7da59dcfbc.js:2255324` | `todo.asar/2800/62e7e57cfb.js:84434` forwards `i.dz0` with the supplied payload; its service calls `c.Bl({strategy:d.lDc.FORCE_SERVER})` at 87837 and consumes `{setting}`. The default setting shape at 87272 identifies the projected fields. `todo.asar/3859/cfa7295828.js:7474` defines `FORCE_SERVER:3`. | Added `workspace.todo-setting`. |
| Task drafts/history | `GET_TODO_COMMENT_DRAFT`, `GET_TODO_HISTORY_RECORDS` in the catalog | Draft/task/container identifiers and pagination are not established. | No descriptor. |
| Preferences/navigation | `GET_NAVIGATION_APPS`, `main-window.asar/157766/91f3b16f27.js:174692` | Declaration only. Do not return a raw settings object; app response fields were not traced. | No descriptor. |
| Unified search | `UNIVERSAL_SEARCH`, `main-window.asar/157766/91f3b16f27.js:220462` | Declaration only. Query text, entity filters, pagination, and response data are untraced. | No descriptor. |
| Moments reads | `MOMENTS_GET_TAB_FEED`, `MOMENTS_GET_POST_DETAIL` in main-window command table | Tab/post identifiers, cursors, and post field consumers are untraced. | No descriptor. |
| Drive transfer status | `SUITE_DRIVE_TASK_GET_DOWNLOAD_PROCESS`, `space.asar/js/14954.67cc8241e798e6b617c3.no-online.js:192572` | `space.asar/js/73336.604fe9312e2d4721653a.no-online.js:7078` proves the drive manager invokes `rust.callApi(...GetDownloadProcessList)` with no explicit payload and emits the whole result as `TasksAdded` at 963. It does not prove the LarkAPI bridge request wire or bound response task fields. | No descriptor. |
| Meeting status | `GET_VC_MEETING_JOIN_STATUS`, `main-window.asar/157766/91f3b16f27.js:141577` | Declaration only. Meeting/container/session identity is untraced; `SYNC_MEETING_STATUS` may synchronize state. | No descriptor. |

The next static pass should trace a UI module import through to a transport call
and consumer before adding a descriptor. It must preserve required
account/container/realm context rather than guessing defaults.
