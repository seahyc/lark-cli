'use strict';

const resourceEquipmentsCommand = '3052|calendar.v1.GetResourceEquipmentsRequest|calendar.v1.GetResourceEquipmentsResponse|1|GET_RESOURCE_EQUIPMENTS';

function object(value) {
  return value && typeof value === 'object' && !Array.isArray(value) ? value : {};
}

function noParameters(input) {
  if (input !== undefined && (input === null || typeof input !== 'object' || Array.isArray(input) || Object.keys(input).length)) {
    throw new TypeError('this read accepts no parameters');
  }
  return {};
}

function navigationDiagnosticKeys(value, limit) {
  return Object.keys(object(value)).slice(0, limit).map((key) => key.slice(0, 64));
}

function navigationEntryDiagnostic(value) {
  const source = object(value);
  const fields = ['id', 'key', 'type', 'visible', 'movable'];
  const types = fields.map((key) => `${key}:${typeof source[key]}`);
  const keys = navigationDiagnosticKeys(source, 32);
  const nested = [];
  for (const key of keys) {
    if (nested.length === 8) break;
    if (source[key] && typeof source[key] === 'object' && !Array.isArray(source[key])) {
      nested.push(`${key}:[${navigationDiagnosticKeys(source[key], 32).join(',')}]`);
    }
  }
  let type = '';
  if (typeof source.type === 'string') type = ` type=${JSON.stringify(source.type.slice(0, 64))}`;
  return `{keys:[${keys.join(',')}], ${types.join(', ')}, nested:{${nested.join(',')}}}${type}`;
}

function navigationLayoutRequest(input) {
  if (input !== undefined && (input === null || typeof input !== 'object' || Array.isArray(input) || Object.keys(input).length)) {
    throw new TypeError('this shell navigation read accepts no parameters');
  }
  return {};
}

function projectNavigationLayout(response) {
  const source = object(response);
  function projectList(value, name) {
    if (!Array.isArray(value)) throw new TypeError(`${name} is missing from the ShellAPI navigation snapshot`);
    if (value.length > 100) throw new RangeError(`${name} exceeds the safe reorder limit`);
    const seen = new Set();
    return value.map((entry, index) => {
      const app = object(entry);
      const diagnostic = navigationEntryDiagnostic(entry);
      if (typeof app.id !== 'string' || !app.id || app.id.length > 512) throw new TypeError(`${name}[${index}].id is invalid ${diagnostic}`);
      const typeIsString = typeof app.type === 'string' && app.type.length > 0 && app.type.length <= 128;
      const typeIsInteger = Number.isInteger(app.type) && app.type >= 0 && app.type <= 2147483647;
      if (!typeIsString && !typeIsInteger) throw new TypeError(`${name}[${index}].type is invalid ${diagnostic}`);
      if (Object.prototype.hasOwnProperty.call(app, 'key') && (typeof app.key !== 'string' || app.key.length > 512)) throw new TypeError(`${name}[${index}].key is invalid ${diagnostic}`);
      if (Object.prototype.hasOwnProperty.call(app, 'visible') && typeof app.visible !== 'boolean') throw new TypeError(`${name}[${index}].visible is invalid ${diagnostic}`);
      if (Object.prototype.hasOwnProperty.call(app, 'movable') && typeof app.movable !== 'boolean') throw new TypeError(`${name}[${index}].movable is invalid ${diagnostic}`);
      const identity = `${typeof app.type}:${app.type}:${app.id}`;
      if (seen.has(identity)) throw new TypeError(`${name} contains duplicate navigation ID ${identity}`);
      seen.add(identity);
      const projected = { id: app.id, type: app.type };
      if (typeof app.key === 'string') projected.key = app.key;
      if (typeof app.visible === 'boolean') projected.visible = app.visible;
      if (typeof app.movable === 'boolean') projected.movable = app.movable;
      return projected;
    });
  }
  return {
    schemaVersion: 2,
    mainNavigation: projectList(source.mainNavigation, 'mainNavigation'),
    shortcutNavigation: projectList(source.shortcutNavigation, 'shortcutNavigation'),
  };
}

function deviceNotifyRequest(input) {
  if (input !== undefined && (input === null || typeof input !== 'object' || Array.isArray(input) || Object.keys(input).length)) {
    throw new TypeError('this preference read accepts no parameters');
  }
  return { syncDataStrategy: 3 };
}

function projectDeviceNotify(response) {
  const setting = object(object(response).setting);
  const result = {};
  for (const key of ['disableMobileNotify', 'stillNotifyAt', 'showMessageDetail', 'stillNotifySpecialNotice']) {
    if (typeof setting[key] === 'boolean') result[key] = setting[key];
  }
  return result;
}

function navigationAppsRequest(input) {
  if (input !== undefined && (input === null || typeof input !== 'object' || Array.isArray(input) || Object.keys(input).length)) {
    throw new TypeError('this desktop catalog read accepts no parameters');
  }
  // settings.v1.NavigationPlatform.NAV_PC in the bundled descriptor set.
  return { platform: 1 };
}

function projectNavigationApps(response) {
  const apps = Array.isArray(object(response).appInfo) ? object(response).appInfo : [];
  const result = [];
  for (const app of apps.slice(0, 100)) {
    const source = object(app);
    const item = {};
    if (typeof source.id === 'string') item.id = source.id;
    if (typeof source.key === 'string') item.key = source.key.slice(0, 512);
    if (Number.isInteger(source.appType)) item.appType = source.appType;
    if (typeof source.displayName === 'string') item.displayName = source.displayName.slice(0, 512);
    if (Object.keys(item).length) result.push(item);
  }
  return { apps: result, total: apps.length, truncated: apps.length > 100 };
}

function projectCustomStatusIntervals(response) {
  const statuses = Array.isArray(object(response).status) ? object(response).status : [];
  const result = [];
  for (const status of statuses.slice(0, 100)) {
    const source = object(status);
    if (typeof source.id !== 'string' || !source.id) continue;
    const interval = object(source.effectiveInterval);
    const item = { id: source.id };
    if (typeof interval.startTime === 'string') item.startTime = interval.startTime;
    if (typeof interval.endTime === 'string') item.endTime = interval.endTime;
    if (typeof interval.isShowEndTime === 'boolean') item.isShowEndTime = interval.isShowEndTime ? 1 : 0;
    if (typeof source.lastCustomizedEndTime === 'string') item.lastCustomizedEndTime = source.lastCustomizedEndTime;
    result.push(item);
  }
  return result;
}

function projectTimeFormat(response) {
  const value = object(object(response).timeFormat).timeFormat;
  if (value === 1) return { hourCycle: 12 };
  if (value === 2) return { hourCycle: 24 };
  return {};
}

function projectEquipment(response) {
  const source = object(response);
  const ids = Array.isArray(source.equipmentIds) ? source.equipmentIds : [];
  const equipmentLists = object(source.equipmentLists);
  const equipment = [];

  for (const id of ids.slice(0, 100)) {
    if (typeof id !== 'string' || !id) continue;
    const item = object(equipmentLists[id]);
    const projected = { id };
    if (typeof item.i18nName === 'string') projected.i18nName = item.i18nName.slice(0, 512);
    equipment.push(projected);
  }

  return { equipment, total: ids.length, truncated: ids.length > 100 };
}

module.exports = {
  operations: {
    'workspacenext.navigation-layout': {
      transport: 'shell-navigation',
      command: 'ShellAPI.app.navigation.getNavigationInfo',
      request: navigationLayoutRequest,
      project: projectNavigationLayout,
    },
    'workspacenext.device-notify-preferences': {
      command: '1039|device.v1.GetDeviceNotifySettingRequest|device.v1.GetDeviceNotifySettingResponse|1|GET_DEVICE_NOTIFY_SETTING',
      request: deviceNotifyRequest,
      project: projectDeviceNotify,
    },
    'workspacenext.navigation-apps': {
      command: '5231|settings.v1.GetNavigationAppsRequest|settings.v1.GetNavigationAppsResponse|1|GET_NAVIGATION_APPS',
      request: navigationAppsRequest,
      project: projectNavigationApps,
    },
    'workspacenext.custom-status-intervals': {
      command: '50301|contact.v1.GetUserCustomStatusRequest|contact.v1.GetUserCustomStatusResponse|1|GET_USER_CUSTOM_STATUS',
      request(input) { const payload = noParameters(input); return { ...payload, syncDataStrategy: 3 }; },
      project: projectCustomStatusIntervals,
    },
    'workspacenext.time-format-preference': {
      command: '2196|settings.v1.GetUserSettingRequest|settings.v1.GetUserSettingResponse|1|GET_USER_SETTING',
      request: noParameters,
      project: projectTimeFormat,
    },
    'workspacenext.calendar-resource-equipments': {
      command: resourceEquipmentsCommand,
      request: noParameters,
      project: projectEquipment,
    },
  },
};
