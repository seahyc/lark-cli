# Workspace desktop read research

Only `workspace.user-custom-status` is executable. Installed Lark 8.0.3
`StatusSettingModal` declares the response command, calls it as
`Z(ee.Sdg,{syncDataStrategy:X.lDc.FORCE_SERVER})`, and consumes the returned
`status` array. Its UI then reads each card's `orderWeight`, `type`,
`syncSettings`, `id`, `typeV2`, `iconKey`, `title`, and `eventName`. The same
file defines `FORCE_SERVER:3`; the descriptor has no caller parameters and
always sends `{syncDataStrategy:3}`. It excludes `syncSettings` because its
full shape is not required for a card listing.

Static evidence establishes this client request and response-consumer schema.
It does not establish availability, required account/container/realm context,
authorization, cache freshness, or successful execution. **Live calls were
not run.**

| Assigned domain | Catalog lead | Result |
| --- | --- | --- |
| Calendar | `GET_CALENDAR_SETTINGS` | Executable static-schema-backed read of timezone and event-recommendation settings. |
| Tasks | `GET_TODO_SETTING` | Executable static-schema-backed force-server settings read. |
| Profile | `GET_USER_CUSTOM_STATUS` | Executable static-schema-backed read. |
| Settings / Workplace navigation | `GET_NAVIGATION_APPS` | Declaration only; no executable wrapper. |
| Meetings | `GET_VC_MEETING_JOIN_STATUS`, `SYNC_MEETING_STATUS` | No wrapper: session/container context and synchronization effects unknown. |
| Global search | `UNIVERSAL_SEARCH` | No wrapper: query and pagination schema unknown. |
| Moments | `MOMENTS_LIST_TABS`, `MOMENTS_GET_TAB_FEED`, `MOMENTS_GET_POST_DETAIL` | Typed tab-list returns bounded IDs for the typed tab-feed page; post detail remains untraced. |
| Drive | `SUITE_DRIVE_TASK_GET_DOWNLOAD_PROCESS`, `SUITE_DRIVE_TASK_GET_UPLOAD_PROCESS` | No wrapper: task context and response schema unknown. |

The manifest contains archive-relative evidence for the literal wire command,
the force-server callsite, and the response consumer. There is no generic native
transport operation or raw settings output.
