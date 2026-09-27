'use strict';

// These descriptors are deliberately separate from module.js. Their native
// calls persist user state and must be previewed, identity-guarded, applied,
// and read back by the parent runtime.
const id = value => typeof value === 'string' && value.length > 0 && value.length <= 512 ? value : undefined;
const object = value => value && typeof value === 'object' && !Array.isArray(value) ? value : {};

function utf8Bytes(value) {
  let bytes = 0;
  for (let index = 0; index < value.length; index += 1) {
    const unit = value.charCodeAt(index);
    if (unit >= 0xd800 && unit <= 0xdbff && index + 1 < value.length) {
      const next = value.charCodeAt(index + 1);
      if (next >= 0xdc00 && next <= 0xdfff) {
        bytes += 4;
        index += 1;
        continue;
      }
    }
    bytes += unit < 0x80 ? 1 : unit < 0x800 ? 2 : 3;
  }
  return bytes;
}

// This reproduces the installed bundle's module 519480 + 470716 UUID-v4
// path: getRandomValues when present, otherwise 32-bit Math.random chunks.
function uuidV4() {
  const bytes = new Uint8Array(16);
  const random = typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function'
    ? crypto.getRandomValues.bind(crypto)
    : typeof msCrypto !== 'undefined' && typeof msCrypto.getRandomValues === 'function'
      ? msCrypto.getRandomValues.bind(msCrypto)
      : undefined;
  if (random) random(bytes);
  else {
    for (let index = 0; index < 16; index += 1) {
      if ((index & 3) === 0) {
        const word = Math.floor(0x100000000 * Math.random());
        bytes[index] = word & 255;
        bytes[index + 1] = (word >>> 8) & 255;
        bytes[index + 2] = (word >>> 16) & 255;
        bytes[index + 3] = (word >>> 24) & 255;
      }
    }
  }
  bytes[6] = (bytes[6] & 15) | 64;
  bytes[8] = (bytes[8] & 63) | 128;
  const hex = Array.from(bytes, value => value.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

function plainTextRichText(value) {
  const elementId = uuidV4();
  return {
    elementIds: [elementId],
    innerText: value,
    elements: {[elementId]: {tag: 1, property: {text: {content: value}}}},
    imageIds: [],
    atIds: [],
    anchorIds: [],
    mediaIds: [],
    docsIds: []
  };
}

function createTextDraftRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) if (key !== 'chatId' && key !== 'text') throw new Error(`unsupported create-text-draft parameter: ${key}`);
  const chatId = id(source.chatId);
  const value = source.text;
  if (!chatId) throw new Error('chatId must be a native desktop identifier');
  if (typeof value !== 'string' || !value.trim()) throw new Error('text must be non-empty plain text');
  if (utf8Bytes(value) > 16384) throw new Error('text must not exceed 16 KiB UTF-8');
  return {draft: {chatId, type: 1, content: JSON.stringify(plainTextRichText(value))}};
}

function deleteDraftRequest(input) {
  const source = object(input);
  if (Object.keys(source).length !== 1 || !Object.prototype.hasOwnProperty.call(source, 'draftId')) throw new Error('draftId is required');
  const draftId = id(source.draftId);
  if (!draftId) throw new Error('draftId must be a native desktop identifier');
  return {draftId};
}

function lastReadRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) if (!['chatId', 'position', 'offset'].includes(key)) throw new Error(`unsupported last-read parameter: ${key}`);
  const chatId = id(source.chatId);
  const position = source.position === undefined ? -1 : source.position;
  const offset = source.offset === undefined ? 0 : source.offset;
  if (!chatId) throw new Error('chatId must be a native desktop identifier');
  if (!Number.isSafeInteger(position)) throw new Error('position must be an integer');
  if (typeof offset !== 'number' || !Number.isFinite(offset)) throw new Error('offset must be a finite number');
  return {chatId, position, offset: Math.round(offset).toString()};
}

module.exports = {
  operations: {
    'messaging.delete-draft': {
      command: '2011|im.v1.DeleteDraftRequest|im.v1.DeleteDraftResponse|1|DELETE_DRAFT',
      request: deleteDraftRequest,
      project() { return {acknowledged: true, verified: false}; }
    },
    'messaging.create-text-draft': {
      command: '2005|im.v1.CreateDraftRequest|im.v1.CreateDraftResponse|1|CREATE_DRAFT',
      request: createTextDraftRequest,
      project() { return {acknowledged: true, verified: false}; }
    },
    'messaging.set-chat-last-read': {
      command: '1067|im.v1.CreateChatLastReadPositionRequest|im.v1.CreateChatLastReadPositionResponse|1|CREATE_CHAT_LAST_READ_POSITION',
      request: lastReadRequest,
      project() { return {acknowledged: true, verified: false}; }
    }
  }
};
