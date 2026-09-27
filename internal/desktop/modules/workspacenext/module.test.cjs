'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');
const descriptor = require('./module.js').operations['workspacenext.calendar-resource-equipments'];

test('calendar resource equipment catalog accepts exactly no parameters', () => {
  assert.deepEqual(descriptor.request(), {});
  assert.deepEqual(descriptor.request({}), {});
  assert.throws(() => descriptor.request({ roomId: 'x' }), /no parameters/);
  assert.throws(() => descriptor.request([]), /no parameters/);
});

test('calendar resource equipment catalog projects only consumed names', () => {
  assert.deepEqual(descriptor.project({
    equipmentIds: ['projector', 'whiteboard', 3],
    equipmentLists: {
      projector: { i18nName: 'Projector', equipmentType: 'display', token: 'omit' },
      whiteboard: { i18nName: 'Whiteboard', bookingState: 'omit' },
    },
    roomAvailability: { should: 'omit' },
  }), {
    equipment: [
      { id: 'projector', i18nName: 'Projector' },
      { id: 'whiteboard', i18nName: 'Whiteboard' },
    ],
    total: 3,
    truncated: false,
  });
});

test('calendar resource equipment catalog caps response order at 100', () => {
  const ids = Array.from({ length: 101 }, (_, index) => `equipment-${index}`);
  const equipmentLists = Object.fromEntries(ids.map((id) => [id, { i18nName: id }]));
  const result = descriptor.project({ equipmentIds: ids, equipmentLists });
  assert.equal(result.equipment.length, 100);
  assert.equal(result.equipment[0].id, 'equipment-0');
  assert.equal(result.equipment[99].id, 'equipment-99');
  assert.equal(result.total, 101);
  assert.equal(result.truncated, true);
});

const clearStatus = require('./mutations.js').operations['workspacenext.clear-custom-status'];

test('clear custom status uses only the UI close-status shape', () => {
  assert.deepEqual(clearStatus.request({ statusId: '42' }), {
    updateStatus: [{
      id: '42',
      effectiveInterval: { startTime: '0', endTime: '0', isShowEndTime: true },
      lastCustomizedEndTime: '0',
      fields: [4, 6],
    }],
  });
  assert.deepEqual(clearStatus.project({ status: { ignored: true } }), { acknowledged: true, verified: false });
});

test('clear custom status rejects non-canonical and unsupported inputs', () => {
  for (const statusId of ['', '-1', '01', 1, '9223372036854775808']) {
    assert.throws(() => clearStatus.request({ statusId }), /statusId/);
  }
  assert.throws(() => clearStatus.request({ statusId: '42', fields: [4] }), /unsupported/);
});

const timeFormat = require('./module.js').operations['workspacenext.time-format-preference'];

test('time-format preference uses the exact no-arg user-setting branch and safe enum', () => {
  assert.deepEqual(timeFormat.request(), {});
  assert.deepEqual(timeFormat.project({ timeFormat: { timeFormat: 1, locale: 'omit' }, security: 'omit' }), { hourCycle: 12 });
  assert.deepEqual(timeFormat.project({ timeFormat: { timeFormat: 2 } }), { hourCycle: 24 });
  assert.deepEqual(timeFormat.project({ timeFormat: { timeFormat: 0 } }), {});
  assert.deepEqual(timeFormat.project({}), {});
});

const statusIntervals = require('./module.js').operations['workspacenext.custom-status-intervals'];
const restoreStatus = require('./mutations.js').operations['workspacenext.restore-custom-status-interval'];

test('custom status interval read bounds only values needed for exact interval restoration', () => {
  assert.deepEqual(statusIntervals.request(), { syncDataStrategy: 3 });
  assert.deepEqual(statusIntervals.project({ status: [{
    id: '42', effectiveInterval: { startTime: '100', endTime: '200', isShowEndTime: true },
    lastCustomizedEndTime: '200', title: 'excluded', syncSettings: { excluded: true },
  }, { id: 7 }] }), [{ id: '42', startTime: '100', endTime: '200', isShowEndTime: 1, lastCustomizedEndTime: '200' }]);
});

test('restore custom status interval replays exactly the fields changed by clear', () => {
  assert.deepEqual(restoreStatus.request({ statusId: '42', startTime: '100', endTime: '200', isShowEndTime: 1, lastCustomizedEndTime: '200' }), {
    updateStatus: [{ id: '42', effectiveInterval: { startTime: '100', endTime: '200', isShowEndTime: true }, lastCustomizedEndTime: '200', fields: [4, 6] }],
  });
  assert.throws(() => restoreStatus.request({ statusId: '42', startTime: '1', endTime: '2', isShowEndTime: 'true', lastCustomizedEndTime: '2' }), /isShowEndTime/);
});

const navigationApps = require('./module.js').operations['workspacenext.navigation-apps'];

test('navigation app catalog uses the descriptor-proven PC platform request', () => {
  assert.deepEqual(navigationApps.request(), { platform: 1 });
  assert.throws(() => navigationApps.request({ platform: 2 }), /no parameters/);
  assert.deepEqual(navigationApps.project({ appInfo: [{
    id: '42', key: 'calendar', appType: 7, displayName: 'Calendar', url: 'exclude', extra: { exclude: true },
  }, { id: 9, key: 'unsafe-number' }] }), {
    apps: [{ id: '42', key: 'calendar', appType: 7, displayName: 'Calendar' }, { key: 'unsafe-number' }],
    total: 2,
    truncated: false,
  });
});

const deviceNotify = require('./module.js').operations['workspacenext.device-notify-preferences'];

test('device notification preferences use the force-server schema and bounded booleans', () => {
  assert.deepEqual(deviceNotify.request(), { syncDataStrategy: 3 });
  assert.throws(() => deviceNotify.request({ syncDataStrategy: 1 }), /no parameters/);
  assert.deepEqual(deviceNotify.project({ setting: {
    disableMobileNotify: true, stillNotifyAt: false, showMessageDetail: true, stillNotifySpecialNotice: false,
    notificationSoundSetting: { excluded: true },
  } }), { disableMobileNotify: true, stillNotifyAt: false, showMessageDetail: true, stillNotifySpecialNotice: false });
});

const modifyNavigation = require('./mutations.js').operations['workspacenext.modify-navigation-order'];

test('modify navigation order accepts only complete typed main and shortcut lists', () => {
  assert.deepEqual(modifyNavigation.request({
    mainNavigationJson: '[{"id":"calendar","appType":1},{"id":"wiki","appType":3}]',
    shortcutNavigationJson: '[{"id":"help","appType":6}]',
  }), {
    mainNavigation: [{ id: 'calendar', appType: 1 }, { id: 'wiki', appType: 3 }],
    shortcutNavigation: [{ id: 'help', appType: 6 }],
    platform: 1,
  });
  assert.throws(() => modifyNavigation.request({ mainNavigationJson: '[]', shortcutNavigationJson: '[{"id":"x","appType":7}]' }), /appType/);
  assert.throws(() => modifyNavigation.request({ mainNavigationJson: '[{"id":"x","appType":1}]', shortcutNavigationJson: '[{"id":"x","appType":1}]' }), /duplicate/);
});

const navigationLayout = require('./module.js').operations['workspacenext.navigation-layout'];

test('shell navigation layout v2 keeps complete observed native lists without URLs or extras', () => {
  assert.equal(navigationLayout.command, 'ShellAPI.app.navigation.getNavigationInfo');
  assert.deepEqual(navigationLayout.request(), {});
  assert.deepEqual(navigationLayout.project({
    mainNavigation: [{
      id: 'calendar', key: 'calendar', name: 'Calendar', type: 'native', visible: true, movable: false,
      extra: { excluded: true }, urlPrimaryLogoSvg: 'excluded', pathPrimaryLogoSvg: 'excluded',
    }],
    shortcutNavigation: [{ id: 'help', type: 6, visible: false, movable: true, urlPrimaryLogoSelectedPng: 'excluded' }],
  }), {
    schemaVersion: 2,
    mainNavigation: [{ id: 'calendar', key: 'calendar', type: 'native', visible: true, movable: false }],
    shortcutNavigation: [{ id: 'help', type: 6, visible: false, movable: true }],
  });
  assert.throws(() => navigationLayout.project({ mainNavigation: [{ ID: 'calendar', appType: 'native', unmovable: false }], shortcutNavigation: [] }), (error) => {
    assert.match(error.message, /mainNavigation\[0\]\.id is invalid/);
    assert.match(error.message, /keys:\[ID,appType,unmovable\]/);
    assert.match(error.message, /id:undefined, key:undefined, type:undefined, visible:undefined, movable:undefined/);
    return true;
  });
  assert.throws(() => navigationLayout.project({ mainNavigation: [{ id: 7, type: 'native', appInfo: { type: 'nested', key: 'calendar' } }], shortcutNavigation: [] }), (error) => {
    assert.match(error.message, /nested:\{appInfo:\[type,key\]\}/);
    return true;
  });
  assert.throws(() => navigationLayout.project({ mainNavigation: [{ id: 'calendar', type: 'x'.repeat(129) }], shortcutNavigation: [] }), /type is invalid/);
  assert.throws(() => navigationLayout.project({ mainNavigation: Array.from({ length: 101 }, (_, i) => ({ id: String(i), type: 'native' })), shortcutNavigation: [] }), /safe reorder limit/);
});

