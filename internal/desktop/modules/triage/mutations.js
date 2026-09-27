'use strict';

// Persistent inbox actions. The parent owns preview/apply, self-chat identity
// checks, readback, and cleanup; these descriptors only reproduce proven UI
// wire shapes and refuse broad or unscoped requests.
const MAX_INT64 = '9223372036854775807';
const codec = typeof require === 'function' ? require('./text_codec.js') : scheduleTextCodec;

function inputObject(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new TypeError('input must be an object');
  return value;
}

function exactKeys(value, keys) {
  const source = inputObject(value);
  for (const key of Object.keys(source)) if (!keys.includes(key)) throw new TypeError(`unsupported input field: ${key}`);
  return source;
}

function int64(value, name) {
  if (typeof value !== 'string' || !/^(?:0|[1-9][0-9]{0,18})$/.test(value) ||
      (value.length === 19 && value > MAX_INT64)) throw new TypeError(`${name} must be a canonical non-negative int64 string`);
  return value;
}

function identifier(value, name) {
  if (typeof value !== 'string' || !value.length || value.length > 512) throw new TypeError(`${name} must be a non-empty native identifier up to 512 characters`);
  return value;
}

function utf8Bytes(value) {
  let bytes = 0;
  for (let index = 0; index < value.length; index += 1) {
    const unit = value.charCodeAt(index);
    if (unit >= 0xd800 && unit <= 0xdbff && index + 1 < value.length) {
      const next = value.charCodeAt(index + 1);
      if (next >= 0xdc00 && next <= 0xdfff) { bytes += 4; index += 1; continue; }
    }
    bytes += unit < 0x80 ? 1 : unit < 0x800 ? 2 : 3;
  }
  return bytes;
}

function uuidV4() {
  const bytes = new Uint8Array(16);
  const random = typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function'
    ? crypto.getRandomValues.bind(crypto)
    : typeof msCrypto !== 'undefined' && typeof msCrypto.getRandomValues === 'function'
      ? msCrypto.getRandomValues.bind(msCrypto) : undefined;
  if (random) random(bytes);
  else for (let index = 0; index < 16; index += 1) if ((index & 3) === 0) {
    const word = Math.floor(0x100000000 * Math.random());
    bytes[index] = word & 255; bytes[index + 1] = (word >>> 8) & 255;
    bytes[index + 2] = (word >>> 16) & 255; bytes[index + 3] = (word >>> 24) & 255;
  }
  bytes[6] = (bytes[6] & 15) | 64; bytes[8] = (bytes[8] & 63) | 128;
  const hex = Array.from(bytes, value => value.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

function richText(text) {
  const elementId = uuidV4();
  return {elementIds: [elementId], innerText: text, elements: {[elementId]: {tag: 1, property: {text: {content: text}}}}, imageIds: [], atIds: [], anchorIds: [], mediaIds: [], docsIds: []};
}

function stageScheduledTextRequest(input) {
  const source = exactKeys(input, ['chatId', 'text', 'scheduleTime']);
  const chatId = int64(source.chatId, 'chatId');
  if (typeof source.text !== 'string' || !source.text.trim() || utf8Bytes(source.text) > 16384) throw new TypeError('text must be non-empty plain text up to 16 KiB UTF-8');
  // The scheduled-message editor renders a native item with
  // dayjs(1000 * Number(scheduleTime)); native wire values are Unix seconds.
  const scheduleTime = int64(source.scheduleTime, 'scheduleTime');
  return {cid: uuidV4(), channel: {id: chatId, type: 1}, type: 4, content: {richText: richText(source.text)}, shouldNotify: true, scheduleTime};
}

function persistScheduledTextRequest(params, prepared) {
  const stage = inputObject(params), result = inputObject(prepared);
  const cid = identifier(stage.cid, 'cid');
  if (identifier(result.cid, 'prepared.cid') !== cid) throw new TypeError('prepared.cid must exactly match the staged cid');
  const channel = inputObject(stage.channel);
  const chatId = int64(channel.id, 'channel.id');
  const content = inputObject(stage.content);
  const richText = inputObject(content.richText);
  if (channel.type !== 1 || stage.type !== 4 || stage.shouldNotify !== true) throw new TypeError('params must be the exact staged self TEXT request');
  const encodedContent = codec.scheduledTextContent(richText);
  const scheduleTime = int64(stage.scheduleTime, 'scheduleTime');
  // The worklet passes eC()'s first result (server Content with a serialized
  // RichText) to module 9510 and then descriptor-aware server transport.
  return {putRequest: {type: 4, content: encodedContent, chatId, rootId: '', parentId: '', cid, isNotified: true, version: 1, isReplyInThread: undefined, syncToChat: undefined, aiInfoContext: undefined, partialReplyInfo: undefined, isShare: false, isSmartReply: undefined}, scheduleTime, chatId, entityId: chatId};
}

function addFavoriteRequest(input) {
  const source = exactKeys(input, ['messageId', 'chatId']);
  return {
    // favorite.v1.CreateFavoritesRequest accepts FavoritesTarget records, not
    // the high-level MergeFavorite {messageIds, chatId} wrapper shape.
    favs: [{id: int64(source.messageId, 'messageId'), type: 1, chatId: int64(source.chatId, 'chatId'), originMergeForwardId: ''}]
  };
}

function deleteFavoriteRequest(input) {
  const source = exactKeys(input, ['favoriteId']);
  // The desktop detail view deletes exactly the selected favorite via {ids}.
  return {ids: [identifier(source.favoriteId, 'favoriteId')]};
}

function cancelScheduledTextRequest(input) {
  const source = exactKeys(input, ['messageId', 'chatId', 'scheduleTime', 'expectedText']);
  if (typeof source.expectedText !== 'string' || source.expectedText.length > 16384) throw new TypeError('expectedText must be plain text up to 16 KiB');
  // The SDK 39008 request takes a local ScheduleMessageItem patchObject and
  // updates its local store. The worklet converts a persisted item into this
  // server request before sending 900034. The fixture has no reply context, so
  // content and partialReplyInfo are deliberately absent rather than guessed.
  return {patchType: 3, messageId: int64(source.messageId, 'messageId'), chatId: int64(source.chatId, 'chatId'), scheduleTime: int64(source.scheduleTime, 'scheduleTime'), type: 4, sendImmediately: false};
}

module.exports = {
  operations: {
    'triage.schedule-self-text': {
      command: '1001|im.v1.CreateQuasiMessageRequest|im.v1.CreateQuasiMessageResponse|1|CREATE_QUASI_MESSAGE',
      persistCommand: '900030|messages.PutScheduleMessageRequest|messages.PutScheduleMessageResponse',
      request: stageScheduledTextRequest,
      persistRequest: persistScheduledTextRequest,
      project(response) { inputObject(response || {}); return {acknowledged: true, verified: false}; }
    },
    'triage.favorite-add': {
      command: '2240|favorite.v1.CreateFavoritesRequest|favorite.v1.CreateFavoritesResponse|1|CREATE_FAVORITES',
      request: addFavoriteRequest,
      project(response, input) {
        inputObject(response || {});
        const source = exactKeys(input, ['messageId', 'chatId']);
        return {messageId: int64(source.messageId, 'messageId'), chatId: int64(source.chatId, 'chatId'), acknowledged: true, verified: false};
      }
    },
    'triage.favorite-delete': {
      command: '2242|favorite.v1.DeleteFavoriteRequest|favorite.v1.DeleteFavoriteResponse|1|DELETE_FAVORITES',
      request: deleteFavoriteRequest,
      project(response, input) {
        inputObject(response || {});
        const source = exactKeys(input, ['favoriteId']);
        return {favoriteId: identifier(source.favoriteId, 'favoriteId'), acknowledged: true, verified: false};
      }
    },
    'triage.schedule-cancel-text': {
      transport: 'server',
      command: '900034|messages.PatchScheduleMessageRequest|messages.PatchScheduleMessageResponse',
      request: cancelScheduledTextRequest,
      project(response, input) {
        inputObject(response || {});
        const source = exactKeys(input, ['messageId', 'chatId', 'scheduleTime', 'expectedText']);
        return {messageId: int64(source.messageId, 'messageId'), chatId: int64(source.chatId, 'chatId'), acknowledged: true, verified: false};
      }
    }
  }
};
