# Next desktop workspace protocols

Static-only research on installed Lark 8.0.3. This file does not authorize a native call and no module was added. Offsets are archive-file byte offsets.

## Moments tab feeds: request and bounded response are traced

`moments.asar/moments/42732361ce.js` declares `MOMENTS_GET_TAB_FEED` as `290037|moments.v1.GetTabFeedRequest|moments.v1.GetTabFeedResponse|1|MOMENTS_GET_TAB_FEED` at 852456. The command table export is `EeL` at 837189. Module 556988 exposes `_D`, whose wrapper calls `(0,s.px)(r.EeL,e)` at 558394. The feed model imports that helper as `Pt._D` and proves both payload and response use:

* Initial load at 101187 passes the caller's `{tabId, useLocal, feedOrder}` object through unchanged. It destructures `entities`, `entryList`, `pinnedPostIds`, `nextPageToken`, `isRecommend`, and `lastNewRecommendPostId`.
* Pagination at 103126 passes `{tabId, pageToken, useLocal:false, count, feedOrder, manageMode, clientPinnedPostIds}` and consumes `entities`, `entryList`, `nextPageToken`, and `pinnedPostIds`.
* Refresh at 104659 passes `{count, useLocal:false, pageToken:"", tabId, feedOrder, manageMode}` and consumes the same bounded feed envelope.

The `feedOrder` enum is traced in `moments.asar/2408/9414fe693c.js:3499`: `UNSPECIFIED:0`, `LAST_REPLIED:1`, `LAST_PUBLISH:2`, `RECOMMEND:3`, and `RECOMMEND_V2:4`. `workspace.moments-tab-feed` now requires caller-supplied `tabId` and `count`, validates that enum, fixes `useLocal:false`, and returns a capped list of `entryList[].postId`, `pinnedPostIds`, `nextPageToken`, `isRecommend`, and `lastNewRecommendPostId`; it excludes the `entities` graph. `manageMode` and `clientPinnedPostIds` are intentionally omitted because the initial load path shows them optional.

## Moments tab list: exact empty request and ID consumer are traced

`MOMENTS_LIST_TABS` is declared as `290038|moments.v1.ListTabsRequest|moments.v1.ListTabsResponse|1|MOMENTS_LIST_TABS` in `moments.asar/moments/42732361ce.js:853431`. The command export `eWD` is forwarded by `F=e=>(0,s.px)(r.eWD,e)` at 558425. The category store calls that wrapper as `const{tabs:n}=await(0,Pt.bl)(t);e.updateAllTabs(n)` at 91906. The startup service calls the store action with `i({})` at 552024, proving the request payload is `{}`. `getTabById` later finds an all-tabs item by `e.id===t` at 91253. `workspace.moments-tabs` therefore returns a capped `[{id}]` list and deliberately omits labels, icons, and every other tab field, because this chain does not prove their consumers.

## Moments post detail

The same command table contains `MOMENTS_GET_POST_DETAIL`, but this pass did not locate its helper export/caller pair. Do not infer a post-id payload from the name alone.

## Task history and comment draft

`todo.asar/4418/7da59dcfbc.js:2255095` contains declarations for `GET_TODO_COMMENT_DRAFT` (`90208`) and `GET_TODO_HISTORY_RECORDS` (`90016`), but no literal `getTodoCommentDraft` or `getTodoHistoryRecords` frontend caller exists in the scanned Todo bundle. The declarations occur in the shared command table, so their parameter identities and response envelopes remain unproven. In particular, task/container/comment identifiers must not be guessed.

## Global search history

The shared tables declare search-history operations, and `search.asar/search-command-bar/fd89dccf1f.js:479015` proves the UI consumes a `searchHistorySection` with `histories`, `sliceCount`, and `showMore`. This is not yet a native request schema: the provider call and its request parameters were not in the inspected static chunk. No history descriptor is supported.
