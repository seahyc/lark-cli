'use strict';

const MAX_INT64 = '9223372036854775807';

function object(value) {
  return value && typeof value === 'object' && !Array.isArray(value) ? value : {};
}

function chatId(value, name) {
  if (typeof value !== 'string' || !/^(?:0|[1-9][0-9]{0,18})$/.test(value) ||
      (value.length === 19 && value > MAX_INT64)) {
    throw new TypeError(`${name} must be a canonical non-negative native chat ID`);
  }
  return value;
}

function muteStateRequest(input) {
  const source = object(input);
  if (Object.keys(source).length !== 1 || !Object.prototype.hasOwnProperty.call(source, 'chatId')) {
    throw new TypeError('chatId is required');
  }
  // `chats.PullChatsByIdsRequest.chatIds` is repeated. Supplying exactly one
  // ID keeps this read scoped to one chat rather than loading a feed-sized set.
  return { chatIds: [chatId(source.chatId, 'chatId')] };
}

function muteStateProject(response) {
  const chats = object(object(response).chats);
  const ids = Object.keys(chats);
  if (ids.length !== 1) throw new TypeError('native chat-mute read must return exactly one chat');
  const id = chatId(ids[0], 'response.chats key');
  const chat = object(chats[id]);
  // Generated `entities.Chat.isRemind` defaults true. A muted chat therefore
  // has `isRemind:false`; an omitted generated default means unmuted.
  if (Object.prototype.hasOwnProperty.call(chat, 'isRemind') && typeof chat.isRemind !== 'boolean') {
    throw new TypeError('response chat isRemind must be Boolean');
  }
  return { chatId: id, muted: chat.isRemind === false ? 1 : 0 };
}

module.exports = {
  operations: {
    'inbox.chat-mute-state': {
      transport: 'server',
      command: '64|chats.PullChatsByIdsRequest|chats.PullChatsByIdsResponse',
      request: muteStateRequest,
      project: muteStateProject,
    },
  },
};
