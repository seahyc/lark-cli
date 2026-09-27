'use strict';

const command = '50301|contact.v1.GetUserCustomStatusRequest|contact.v1.GetUserCustomStatusResponse|1|GET_USER_CUSTOM_STATUS';
const todoCommand = '90024|todo.v1.GetTodoSettingRequest|todo.v1.GetTodoSettingResponse|1|GET_TODO_SETTING';
const calendarCommand = '3016|calendar.v1.GetCalendarSettingsRequest|calendar.v1.GetCalendarSettingsResponse|1|GET_CALENDAR_SETTINGS';
const momentsTabFeedCommand = '290037|moments.v1.GetTabFeedRequest|moments.v1.GetTabFeedResponse|1|MOMENTS_GET_TAB_FEED';
const momentsListTabsCommand = '290038|moments.v1.ListTabsRequest|moments.v1.ListTabsResponse|1|MOMENTS_LIST_TABS';

function request(input) {
  if (input !== undefined && (input === null || typeof input !== 'object' || Array.isArray(input) || Object.keys(input).length !== 0)) throw new TypeError('this read accepts no parameters');
  return {syncDataStrategy: 3};
}
function scalar(value) { return value === null || ['string', 'number', 'boolean'].includes(typeof value) ? value : undefined; }
function project(response) {
  const statuses = Array.isArray(response?.status) ? response.status : [];
  return statuses.slice(0, 100).map(status => {
    const out = {};
    for (const key of ['id', 'type', 'typeV2', 'iconKey', 'title', 'eventName', 'orderWeight']) {
      const candidate = scalar(status?.[key]);
      if (candidate !== undefined) out[key] = candidate;
    }
    return out;
  });
}
function todoRequest(input) {
  if (input !== undefined && (input === null || typeof input !== 'object' || Array.isArray(input) || Object.keys(input).length !== 0)) throw new TypeError('this read accepts no parameters');
  return {strategy: 3};
}
function todoProject(response) {
  const setting = response?.setting;
  if (!setting || typeof setting !== 'object' || Array.isArray(setting)) return {};
  const out = {};
  for (const key of ['dueTimeDayOffset', 'dueReminderOffset', 'anotherNameFormat', 'syncFromMeego', 'showOpenMeegoTip', 'showSyncFromMeego', 'enableDailyFollow']) {
    const candidate = scalar(setting[key]);
    if (candidate !== undefined) out[key] = candidate;
  }
  if (setting.badgeConfig && typeof setting.badgeConfig === 'object' && !Array.isArray(setting.badgeConfig)) {
    const badgeConfig = {};
    for (const key of ['enable', 'type']) { const candidate = scalar(setting.badgeConfig[key]); if (candidate !== undefined) badgeConfig[key] = candidate; }
    if (Object.keys(badgeConfig).length) out.badgeConfig = badgeConfig;
  }
  return out;
}
function emptyRequest(input) {
  if (input !== undefined && (input === null || typeof input !== 'object' || Array.isArray(input) || Object.keys(input).length !== 0)) throw new TypeError('this read accepts no parameters');
  return {};
}
function calendarProject(response) {
  const settings = response?.settings;
  if (!settings || typeof settings !== 'object' || Array.isArray(settings)) return {};
  const out = {};
  for (const key of ['timezone', 'recommendEventFromChat']) { const candidate = scalar(settings[key]); if (candidate !== undefined) out[key] = candidate; }
  return out;
}
function momentsTabFeedRequest(input) {
  if (!input || typeof input !== 'object' || Array.isArray(input) || Object.keys(input).some(key => !['tabId', 'count', 'pageToken', 'feedOrder'].includes(key))) throw new TypeError('tabId, count, optional pageToken, and optional feedOrder are required fields only');
  const {tabId, count, pageToken = '', feedOrder = 0} = input;
  if (typeof tabId !== 'string' || !tabId || tabId.length > 256) throw new TypeError('tabId must be a non-empty string');
  if (!Number.isInteger(count) || count < 1 || count > 100) throw new TypeError('count must be an integer from 1 to 100');
  if (typeof pageToken !== 'string' || pageToken.length > 1024) throw new TypeError('pageToken must be a string');
  if (!Number.isInteger(feedOrder) || feedOrder < 0 || feedOrder > 4) throw new TypeError('feedOrder must be one of 0, 1, 2, 3, 4');
  return {tabId, count, pageToken, useLocal: false, feedOrder};
}
function momentsTabFeedProject(response) {
  const entryList = Array.isArray(response?.entryList) ? response.entryList.slice(0, 100).map(entry => ({postId: typeof entry?.postId === 'string' ? entry.postId : undefined})).filter(entry => entry.postId) : [];
  const pinnedPostIds = Array.isArray(response?.pinnedPostIds) ? response.pinnedPostIds.filter(value => typeof value === 'string').slice(0, 100) : [];
  const out = {entryList, pinnedPostIds};
  for (const key of ['nextPageToken', 'isRecommend', 'lastNewRecommendPostId']) { const candidate = scalar(response?.[key]); if (candidate !== undefined) out[key] = candidate; }
  return out;
}
function momentsTabsProject(response) {
  const tabs = Array.isArray(response?.tabs) ? response.tabs : [];
  return tabs.slice(0, 100).map(tab => ({id: typeof tab?.id === 'string' ? tab.id : undefined})).filter(tab => tab.id);
}
module.exports = {operations: {
  'workspace.user-custom-status': {command, request, project},
  'workspace.todo-setting': {command: todoCommand, request: todoRequest, project: todoProject},
  'workspace.calendar-settings': {command: calendarCommand, request: emptyRequest, project: calendarProject},
  'workspace.moments-tab-feed': {command: momentsTabFeedCommand, request: momentsTabFeedRequest, project: momentsTabFeedProject},
  'workspace.moments-tabs': {command: momentsListTabsCommand, request: emptyRequest, project: momentsTabsProject}
}};
