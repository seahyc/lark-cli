'use strict';

// Research-only descriptors. They are intentionally absent from module.js,
// mutations.js, and registered manifests. A direct live server search returned
// ErrCacheEmpty; do not treat that error as an empty allow/block collection.
function object(input) {
  if (!input || typeof input !== 'object' || Array.isArray(input)) throw new TypeError('input must be an object');
  return input;
}
function exactKeys(input, keys) {
  const value = object(input);
  for (const key of Object.keys(value)) if (!keys.includes(key)) throw new TypeError(`unsupported input field: ${key}`);
  return value;
}
function emailAddress(value, name) {
  if (typeof value !== 'string' || value.length === 0 || value.length > 254 || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) throw new TypeError(`${name} must be a valid email address`);
  return value;
}
function nonnegativeInt64(value, name, fallback) {
  if (value === undefined) return fallback;
  if (typeof value !== 'string' || !/^(0|[1-9][0-9]{0,18})$/.test(value) || (value.length === 19 && value > '9223372036854775807')) throw new TypeError(`${name} must be a nonnegative int64 decimal string`);
  return value;
}
function pageSize(value) {
  if (value === undefined) return 100;
  if (!Number.isInteger(value) || value < 1 || value > 100) throw new TypeError('size must be an integer from 1 to 100');
  return value;
}
function searchQuery(value) {
  if (value === undefined) return '';
  if (typeof value !== 'string' || value.length > 254) throw new TypeError('searchQuery must be a string of at most 254 characters');
  return value;
}
function records(source) {
  const list = Array.isArray(source) ? source : [];
  return { records: list.slice(0, 100).map((entry) => ({ record: typeof entry?.record === 'string' ? entry.record.slice(0, 254) : '', timestamp: typeof entry?.timestamp === 'string' ? entry.timestamp.slice(0, 128) : '' })), truncated: list.length > 100 };
}
function projectList(response, input) {
  const value = object(response);
  const request = exactKeys(input, ['reqType', 'lastTimestamp', 'cursor', 'size', 'searchQuery']);
  const reqType = request.reqType;
  if (!Number.isInteger(reqType) || reqType < 1 || reqType > 3) throw new TypeError('reqType must be an integer from 1 to 3');
  if (reqType === 1 || reqType === 2) {
    const listed = records(reqType === 1 ? value.allows : value.blocks);
    const nextTimestamp = typeof value.nextTimestamp === 'string' && value.nextTimestamp.length <= 128 ? value.nextTimestamp : '0';
    return { list: listed.records, listTruncated: listed.truncated, hasMore: nextTimestamp !== '0', nextTimestamp };
  }
  const allows = records(value.allows), blocks = records(value.blocks);
  return { allows: allows.records, allowsTruncated: allows.truncated, blocks: blocks.records, blocksTruncated: blocks.truncated };
}
module.exports = { operations: {
  'mail.sender-list': {
    transport: 'server', command: '3901|mails.SearchUserAllowBlockRequest|mails.SearchUserAllowBlockResponse',
    request(input) { const v = exactKeys(input, ['reqType', 'lastTimestamp', 'cursor', 'size', 'searchQuery']); if (!Number.isInteger(v.reqType) || v.reqType < 1 || v.reqType > 3) throw new TypeError('reqType must be an integer from 1 to 3'); return { reqType: v.reqType, lastTimestamp: nonnegativeInt64(v.lastTimestamp, 'lastTimestamp', '0'), cursor: nonnegativeInt64(v.cursor, 'cursor', '0'), size: pageSize(v.size), query: searchQuery(v.searchQuery) }; },
    project: projectList,
  },
  'mail.sender-remove': {
    transport: 'server', command: '3902|mails.DeleteUserAllowBlockRequest|mails.DeleteUserAllowBlockResponse',
    request(input) { const v = exactKeys(input, ['address']); return { multiFrom: [emailAddress(v.address, 'address')], isAllow: false }; },
    project(response, input) { object(response); const v = exactKeys(input, ['address']); return { address: emailAddress(v.address, 'address'), acknowledged: true, verified: false }; },
  },
} };
