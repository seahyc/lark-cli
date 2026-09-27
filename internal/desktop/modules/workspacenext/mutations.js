'use strict';

const MAX_INT64 = '9223372036854775807';

function object(value) {
  return value && typeof value === 'object' && !Array.isArray(value) ? value : {};
}

function int64(value, name) {
  if (typeof value !== 'string' || !/^(?:0|[1-9][0-9]{0,18})$/.test(value) ||
      (value.length === 19 && value > MAX_INT64)) {
    throw new Error(`${name} must be a canonical non-negative int64 string`);
  }
  return value;
}

function intervalFields(source) {
  const startTime = int64(source.startTime, 'startTime');
  const endTime = int64(source.endTime, 'endTime');
  const lastCustomizedEndTime = int64(source.lastCustomizedEndTime, 'lastCustomizedEndTime');
  if (!Number.isInteger(source.isShowEndTime) || (source.isShowEndTime !== 0 && source.isShowEndTime !== 1)) throw new Error('isShowEndTime must be 0 or 1');
  return { startTime, endTime, isShowEndTime: source.isShowEndTime === 1, lastCustomizedEndTime };
}

function clearCustomStatusRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) {
    if (key !== 'statusId') throw new Error(`unsupported workspacenext.clear-custom-status parameter: ${key}`);
  }
  const id = int64(source.statusId, 'statusId');
  return {
    updateStatus: [{
      id,
      effectiveInterval: { startTime: '0', endTime: '0', isShowEndTime: true },
      lastCustomizedEndTime: '0',
      // StatusField.EFFECTIVE_INTERVAL and LAST_CUSTOMIZED_END_TIME in the shipped enum.
      fields: [4, 6],
    }],
  };
}

function restoreCustomStatusIntervalRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) {
    if (!['statusId', 'startTime', 'endTime', 'isShowEndTime', 'lastCustomizedEndTime'].includes(key)) throw new Error(`unsupported workspacenext.restore-custom-status-interval parameter: ${key}`);
  }
  const interval = intervalFields(source);
  return { updateStatus: [{
    id: int64(source.statusId, 'statusId'),
    effectiveInterval: { startTime: interval.startTime, endTime: interval.endTime, isShowEndTime: interval.isShowEndTime },
    lastCustomizedEndTime: interval.lastCustomizedEndTime,
    fields: [4, 6],
  }] };
}

function navigationList(serialized, name, seen) {
  if (typeof serialized !== 'string' || serialized.length > 65536) throw new Error(`${name} must be a bounded JSON array`);
  let values;
  try { values = JSON.parse(serialized); } catch (_) { throw new Error(`${name} must be valid JSON`); }
  if (!Array.isArray(values) || values.length > 100) throw new Error(`${name} must be an array of at most 100 entries`);
  return values.map((value, index) => {
    const entry = object(value);
    if (Object.keys(entry).some(key => key !== 'id' && key !== 'appType')) throw new Error(`${name}[${index}] has unsupported fields`);
    const id = entry.id;
    if (typeof id !== 'string' || !id || id.length > 512) throw new Error(`${name}[${index}].id must be a non-empty string`);
    if (!Number.isInteger(entry.appType) || entry.appType < 1 || entry.appType > 6) throw new Error(`${name}[${index}].appType must be a NavigationAppType integer`);
    const identity = `${entry.appType}:${id}`;
    if (seen.has(identity)) throw new Error(`duplicate navigation ID: ${identity}`);
    seen.add(identity);
    return { id, appType: entry.appType };
  });
}

function modifyNavigationOrderRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) {
    if (key !== 'mainNavigationJson' && key !== 'shortcutNavigationJson') throw new Error(`unsupported workspacenext.modify-navigation-order parameter: ${key}`);
  }
  const seen = new Set();
  return {
    mainNavigation: navigationList(source.mainNavigationJson, 'mainNavigationJson', seen),
    shortcutNavigation: navigationList(source.shortcutNavigationJson, 'shortcutNavigationJson', seen),
    platform: 1,
  };
}

module.exports = {
  operations: {
    'workspacenext.modify-navigation-order': {
      command: '5221|settings.v1.ModifyNavigationOrderRequest|settings.v1.ModifyNavigationOrderResponse|1|MODIFY_NAVIGATION_ORDER',
      request: modifyNavigationOrderRequest,
      project() { return { acknowledged: true, verified: false }; },
    },
    'workspacenext.restore-custom-status-interval': {
      command: '50302|contact.v1.UpdateUserCustomStatusRequest|contact.v1.UpdateUserCustomStatusResponse|1|UPDATE_USER_CUSTOM_STATUS',
      request: restoreCustomStatusIntervalRequest,
      project() { return { acknowledged: true, verified: false }; },
    },
    'workspacenext.clear-custom-status': {
      command: '50302|contact.v1.UpdateUserCustomStatusRequest|contact.v1.UpdateUserCustomStatusResponse|1|UPDATE_USER_CUSTOM_STATUS',
      request: clearCustomStatusRequest,
      project() {
        return { acknowledged: true, verified: false };
      },
    },
  },
};
