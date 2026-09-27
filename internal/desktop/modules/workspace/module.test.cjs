'use strict';
const assert = require('node:assert/strict');
const test = require('node:test');
const operation = require('./module').operations['workspace.user-custom-status'];
test('custom-status uses the shipped force-server request', () => {
  assert.deepEqual(operation.request(), {syncDataStrategy: 3});
  assert.deepEqual(operation.request({}), {syncDataStrategy: 3});
  assert.throws(() => operation.request({syncDataStrategy: 1}), /no parameters/);
});
test('projection matches only statically consumed scalar card fields', () => {
  const response = {status: [{id: 'status-1', type: 3, typeV2: 2, iconKey: 'coffee', title: 'Focus', eventName: 'focus', orderWeight: 5, syncSettings: {hidden: true}, token: 'no'}]};
  assert.deepEqual(operation.project(response), [{id: 'status-1', type: 3, typeV2: 2, iconKey: 'coffee', title: 'Focus', eventName: 'focus', orderWeight: 5}]);
});
test('projection is bounded', () => {
  assert.equal(operation.project({status: Array.from({length: 101}, (_, i) => ({id: String(i)}))}).length, 100);
  assert.deepEqual(operation.project({status: 'wrong'}), []);
});
test('todo setting uses the shipped force-server request', () => {
  const todo = require('./module').operations['workspace.todo-setting'];
  assert.deepEqual(todo.request(), {strategy: 3});
  assert.throws(() => todo.request({strategy: 1}), /no parameters/);
});
test('todo setting projects only consumed bounded fields', () => {
  const todo = require('./module').operations['workspace.todo-setting'];
  assert.deepEqual(todo.project({setting: {dueTimeDayOffset: 2, dueReminderOffset: 15, anotherNameFormat: 1, syncFromMeego: true, showOpenMeegoTip: false, showSyncFromMeego: true, enableDailyFollow: false, badgeConfig: {enable: true, type: 4, secret: 'no'}, meegoBrand: {private: 'no'}, token: 'no'}}), {dueTimeDayOffset: 2, dueReminderOffset: 15, anotherNameFormat: 1, syncFromMeego: true, showOpenMeegoTip: false, showSyncFromMeego: true, enableDailyFollow: false, badgeConfig: {enable: true, type: 4}});
});
test('calendar settings uses the shipped empty refresh request and bounded fields', () => {
  const calendar = require('./module').operations['workspace.calendar-settings'];
  assert.deepEqual(calendar.request(), {});
  assert.throws(() => calendar.request({timezone: 'UTC'}), /no parameters/);
  assert.deepEqual(calendar.project({settings: {timezone: 'Asia/Singapore', recommendEventFromChat: true, deviceTimezone: 'secret-local-context', token: 'no'}}), {timezone: 'Asia/Singapore', recommendEventFromChat: true});
});
test('moments feed validates traced request fields and feed-order enum', () => {
  const feed = require('./module').operations['workspace.moments-tab-feed'];
  assert.deepEqual(feed.request({tabId: 'tab-1', count: 20, feedOrder: 3}), {tabId: 'tab-1', count: 20, pageToken: '', useLocal: false, feedOrder: 3});
  assert.throws(() => feed.request({tabId: '', count: 20}), /tabId/);
  assert.throws(() => feed.request({tabId: 'tab-1', count: 20, feedOrder: 5}), /feedOrder/);
  assert.throws(() => feed.request({tabId: 'tab-1', count: 20, manageMode: 1}), /required fields/);
});
test('moments feed omits the unbounded entities graph', () => {
  const feed = require('./module').operations['workspace.moments-tab-feed'];
  assert.deepEqual(feed.project({entities: {posts: {secret: true}}, entryList: [{postId: 'p1', other: 'no'}], pinnedPostIds: ['p2', 3], nextPageToken: 'next', isRecommend: true, lastNewRecommendPostId: 'p3'}), {entryList: [{postId: 'p1'}], pinnedPostIds: ['p2'], nextPageToken: 'next', isRecommend: true, lastNewRecommendPostId: 'p3'});
});
test('moments tabs uses the shipped empty list request and exposes only consumed IDs', () => {
  const tabs = require('./module').operations['workspace.moments-tabs'];
  assert.deepEqual(tabs.request(), {});
  assert.throws(() => tabs.request({forceRemote: true}), /no parameters/);
  assert.deepEqual(tabs.project({tabs: [{id: 'tab-1', title: 'private'}, {id: 2}, {id: 'tab-2', iconKey: 'private'}]}), [{id: 'tab-1'}, {id: 'tab-2'}]);
  assert.equal(tabs.project({tabs: Array.from({length: 101}, (_, index) => ({id: String(index)}))}).length, 100);
});
