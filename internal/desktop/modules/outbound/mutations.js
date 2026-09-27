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

// Matches the installed UUID-v4 fallback used by the messaging rich-text
// builder and provides the client-generated cid required by EditMessage.
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

function plainTextRichText(text) {
  const elementId = uuidV4();
  return {
    elementIds: [elementId],
    innerText: text,
    elements: {[elementId]: {tag: 1, property: {text: {content: text}}}},
    imageIds: [],
    atIds: [],
    anchorIds: [],
    mediaIds: [],
    docsIds: []
  };
}

function editTextRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) {
    if (key !== 'messageId' && key !== 'chatId' && key !== 'text' && key !== 'expectedText') throw new Error(`unsupported messaging.edit-text parameter: ${key}`);
  }
  const messageId = int64(source.messageId, 'messageId');
  const chatId = int64(source.chatId, 'chatId');
  if (typeof source.text !== 'string' || !source.text.trim()) throw new Error('text must be non-empty plain text');
  if (utf8Bytes(source.text) > 16384) throw new Error('text must not exceed 16 KiB UTF-8');
  if (typeof source.expectedText !== 'string') throw new Error('expectedText must be the exact current plain text');
  if (utf8Bytes(source.expectedText) > 16384) throw new Error('expectedText must not exceed 16 KiB UTF-8');
  return {
    msgId: messageId,
    chatId,
    cid: uuidV4(),
    // entities.Message.Type.TEXT is 4 in the shipped generated schema.
    type: 4,
    content: {richText: plainTextRichText(source.text)}
  };
}

function sendSelfTextRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) {
    if (key !== 'chatId' && key !== 'text') throw new Error(`unsupported messaging.send-self-text parameter: ${key}`);
  }
  const chatId = int64(source.chatId, 'chatId');
  if (typeof source.text !== 'string' || !source.text.trim()) throw new Error('text must be non-empty plain text');
  if (utf8Bytes(source.text) > 16384) throw new Error('text must not exceed 16 KiB UTF-8');
  return {
    cid: uuidV4(),
    // The normal composer assigns `channel` from its resolved chat object.
    channel: {id: chatId, type: 1},
    type: 4,
    content: {richText: plainTextRichText(source.text)},
    shouldNotify: true
  };
}

module.exports = {
  operations: {
    'messaging.edit-text': {
      command: '4520|im.v1.EditMessageRequest|im.v1.EditMessageResponse|1|EDIT_MESSAGE',
      request: editTextRequest,
      project() { return {acknowledged: true, verified: false}; }
    },
    'messaging.send-self-text': {
      command: '2003|im.v1.SendMessageRequest|im.v1.SendMessageResponse|1|SEND_MESSAGE',
      request: sendSelfTextRequest,
      project(response) {
        const source = object(response);
        const result = {acknowledged: true, verified: false};
        try {
          if (typeof source.messageId === 'string') result.messageId = int64(source.messageId, 'messageId');
        } catch (_) {}
        return result;
      }
    }
  }
};
