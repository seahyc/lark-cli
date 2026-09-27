'use strict';

// Read-only schemas traced from the installed desktop client's cloud-draft
// wrapper. Keep output finite because draft entities may contain rich content.
const chatDraftCommand = '2709|im.v1.FetchChatInputDraftRequest|im.v1.FetchChatInputDraftResponse|1|FETCH_CHAT_INPUT_DRAFT';
const replyDraftCommand = '2710|im.v1.FetchMessageReplyDraftRequest|im.v1.FetchMessageReplyDraftResponse|1|FETCH_MESSAGE_REPLY_DRAFT';
const allDraftsCommand = '2004|im.v1.GetAllDraftsRequest|im.v1.GetAllDraftsResponse|1|GET_ALL_DRAFTS';
const favoritesCommand = '2241|favorite.v1.GetFavoritesRequest|favorite.v1.GetFavoritesResponse|1|GET_FAVORITES';
const mgetMessagesCommand = '1|im.v1.MGetMessagesRequest|im.v1.MGetMessagesResponse|1|MGET_MESSAGES';
const mgetChatsCommand = '2|im.v1.MGetChatsRequest|im.v1.MGetChatsResponse|1|MGET_CHATS';
const inboxChatsCommand = '2480|feed.v1.GetFeedCardsV4Request|feed.v1.GetFeedCardsV4Response|1|GET_FEED_CARDS_V4';
const p2pChatsCommand = '1046|im.v1.GetP2PChatsByChatterIdsRequest|im.v1.GetP2PChatsByChatterIdsResponse|1|GET_P2P_CHATS_BY_CHATTER_IDS';
const chatMessagesCommand = '1020|im.v1.GetChatMessagesRequest|im.v1.GetChatMessagesResponse|1|GET_CHAT_MESSAGES';

function object(value) {
  return value && typeof value === 'object' && !Array.isArray(value) ? value : {};
}

function text(value, maximum = 16384) {
  return typeof value === 'string' ? value.slice(0, maximum) : undefined;
}

function identifier(value) {
  return typeof value === 'string' && value.length > 0 && value.length <= 512 ? value : undefined;
}

function textDraftContent(source) {
  if (source.type !== 1 || typeof source.content !== 'string' || source.content.length > 65536) return undefined;
  try {
    const richText = JSON.parse(source.content);
    return typeof richText.innerText === 'string' ? text(richText.innerText) : undefined;
  } catch (_) {
    return undefined;
  }
}

function draft(value) {
  const source = object(value);
  const result = {};
  for (const key of ['id', 'chatId', 'threadId', 'messageId', 'editMessageId', 'type', 'createTime', 'updateTime']) {
    if (typeof source[key] === 'string' || typeof source[key] === 'number') result[key] = source[key];
  }
  // For TEXT drafts, expose only bounded parsed innerText. Raw content is a
  // serialized rich-text graph and may be much larger than the visible draft.
  const innerText = textDraftContent(source);
  if (innerText !== undefined) result.text = innerText;
  return result;
}

function project(response) {
  const source = object(response);
  const result = {};
  if (source.draft) result.draft = draft(source.draft);
  const drafts = object(object(source.entity).drafts);
  const total = Object.keys(drafts).length;
  const entries = Object.entries(drafts).slice(0, 100).map(([id, value]) => ({id, ...draft(value)}));
  result.drafts = entries;
  result.total = total;
  result.truncated = total > entries.length;
  return result;
}

function scopedDraftProject(response, input) {
  const source = object(input);
  const drafts = object(object(object(response).entity).drafts);
  const keys = ['chatId', 'threadId', 'messageId'].filter(key => source[key] !== undefined);
  const matching = Object.entries(drafts).filter(([, value]) => {
    const candidate = object(value);
    return keys.every(key => candidate[key] === source[key]);
  });
  const entries = matching.slice(0, 100).map(([id, value]) => ({id, ...draft(value)}));
  return {drafts: entries, total: matching.length, truncated: matching.length > entries.length};
}

function bounded(value, depth = 0) {
  if (typeof value === 'string') return value.slice(0, 16384);
  if (typeof value === 'number' || typeof value === 'boolean' || value === null) return value;
  if (depth >= 4 || !value || typeof value !== 'object') return undefined;
  if (Array.isArray(value)) return value.slice(0, 50).map(item => bounded(item, depth + 1)).filter(item => item !== undefined);
  const result = {};
  for (const [key, item] of Object.entries(value).slice(0, 50)) {
    const safe = bounded(item, depth + 1);
    if (safe !== undefined) result[key] = safe;
  }
  return result;
}

function favoritesProject(response) {
  const source = object(response);
  const favorites = object(object(source.entity).favorites);
  const ids = Array.isArray(source.favoritesIds) ? source.favoritesIds.slice(0, 100).filter(id => typeof id === 'string') : [];
  const result = {
    favorites: ids.map(id => ({id, favorite: bounded(favorites[id])}))
  };
  if (typeof source.hasMore === 'boolean') result.hasMore = source.hasMore;
  if (typeof source.minTime === 'string' || typeof source.minTime === 'number') result.minTime = source.minTime;
  return result;
}

function favoritesRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) {
    if (key !== 'count' && key !== 'time') throw new Error(`unsupported favorites parameter: ${key}`);
  }
  const count = source.count === undefined ? 15 : source.count;
  const time = source.time === undefined ? 0 : source.time;
  if (!Number.isSafeInteger(count) || count < 1 || count > 100) throw new Error('count must be an integer between 1 and 100');
  if (!Number.isSafeInteger(time) || time < 0) throw new Error('time must be a non-negative safe integer');
  return {time: String(time), count};
}

function favoriteDetailProject(response) {
  const favorites = object(object(object(response).entity).favorites);
  const entries = Object.entries(favorites).slice(0, 100).map(([id, favorite]) => ({id, favorite: bounded(favorite)}));
  return {favorites: entries, total: Object.keys(favorites).length, truncated: Object.keys(favorites).length > entries.length};
}

function favoriteDetailRequest(input) {
  const source = object(input);
  if (Object.keys(source).length !== 1 || !Object.prototype.hasOwnProperty.call(source, 'favoriteId')) throw new Error('favoriteId is required');
  const favoriteId = identifier(source.favoriteId);
  if (!favoriteId) throw new Error('favoriteId must be a native desktop identifier');
  return {favoriteIds: [favoriteId]};
}

function shortcutsRequest(input) {
  if (input !== undefined && (input === null || typeof input !== 'object' || Array.isArray(input) || Object.keys(input).length)) throw new Error('messaging.shortcuts accepts no parameters');
  // `syncDataStrategy: 1` is lDc.LOCAL in messenger~15/feef05be77.js.
  return {syncDataStrategy: 1};
}

function shortcutsProject(response) {
  const source = object(response);
  const shortcutMetadata = Array.isArray(source.shortcuts) ? source.shortcuts.slice(0, 100).map(item => {
    const value = object(item);
    const channel = object(value.channel);
    const result = {};
    for (const key of ['id', 'feedType']) if (typeof value[key] === 'string' || typeof value[key] === 'number') result[key] = value[key];
    const channelMetadata = {};
    for (const key of ['id', 'type']) if (typeof channel[key] === 'string' || typeof channel[key] === 'number') channelMetadata[key] = channel[key];
    if (Object.keys(channelMetadata).length) result.channel = channelMetadata;
    return result;
  }) : [];
  return {shortcuts: shortcutMetadata, total: Array.isArray(source.shortcuts) ? source.shortcuts.length : 0, truncated: Array.isArray(source.shortcuts) && source.shortcuts.length > shortcutMetadata.length};
}

function feedFolderRequest(input) {
  if (input !== undefined && (input === null || typeof input !== 'object' || Array.isArray(input) || Object.keys(input).length)) throw new Error('messaging.feed-folder accepts no parameters');
  return {};
}

function feedFolderProject(response) {
  const source = object(response);
  const setting = object(source.feedFolderSettings);
  const usedFolderIds = Array.isArray(setting.usedFolderIds) ? setting.usedFolderIds.slice(0, 100).filter(value => typeof value === 'string') : [];
  return {usedFolderIds, total: Array.isArray(setting.usedFolderIds) ? setting.usedFolderIds.length : 0, truncated: Array.isArray(setting.usedFolderIds) && setting.usedFolderIds.length > usedFolderIds.length};
}

function scheduledChatRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) if (key !== 'chatId' && key !== 'rootId' && key !== 'strategy') throw new Error(`unsupported scheduled-chat parameter: ${key}`);
  const chatId = identifier(source.chatId);
  const rootId = source.rootId === undefined ? undefined : identifier(source.rootId);
  if (!chatId || (source.rootId !== undefined && !rootId)) throw new Error('chatId and optional rootId must be native desktop identifiers');
  const strategy = source.strategy === undefined ? 3 : source.strategy;
  if (strategy !== 1 && strategy !== 3) throw new Error('strategy must be 1 (local) or 3 (force-server)');
  const result = {chatId, syncDataStrategy: strategy, scene: rootId ? 3 : 1};
  if (rootId) result.rootId = rootId;
  return result;
}

function scheduledThreadRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) if (key !== 'chatId' && key !== 'threadId' && key !== 'strategy') throw new Error(`unsupported scheduled-thread parameter: ${key}`);
  const chatId = identifier(source.chatId);
  const threadId = identifier(source.threadId);
  if (!chatId || !threadId) throw new Error('chatId and threadId must be native desktop identifiers');
  const strategy = source.strategy === undefined ? 3 : source.strategy;
  if (strategy !== 1 && strategy !== 3) throw new Error('strategy must be 1 (local) or 3 (force-server)');
  return {chatId, threadId, syncDataStrategy: strategy, scene: 2};
}

function scheduledProject(response) {
  const source = object(response);
  const entity = object(source.entity);
  const scheduleMessage = object(entity.scheduleMessage);
  const quasiScheduleMessage = object(entity.quasiScheduleMessage);
  const items = Array.isArray(source.messageItems) ? source.messageItems.slice(0, 100).map(item => {
    const safe = object(bounded(item));
    const itemId = object(item).itemId;
    if (typeof itemId === 'string') {
      if (scheduleMessage[itemId]) safe.scheduleMessage = bounded(scheduleMessage[itemId]);
      if (quasiScheduleMessage[itemId]) safe.quasiScheduleMessage = bounded(quasiScheduleMessage[itemId]);
    }
    return safe;
  }) : [];
  return {items, total: Array.isArray(source.messageItems) ? source.messageItems.length : 0, truncated: Array.isArray(source.messageItems) && source.messageItems.length > items.length};
}

function chatDraftRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) {
    if (key !== 'chatId' && key !== 'threadId') throw new Error(`unsupported chat-draft parameter: ${key}`);
  }
  const chatId = identifier(source.chatId);
  const threadId = identifier(source.threadId);
  if (!chatId && !threadId) throw new Error('chatId or threadId is required');
  const result = {};
  if (chatId) result.chatId = chatId;
  if (threadId) result.threadId = threadId;
  return result;
}

function replyDraftRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) {
    if (key !== 'messageId') throw new Error(`unsupported reply-draft parameter: ${key}`);
  }
  const messageId = identifier(source.messageId);
  if (!messageId) throw new Error('messageId is required');
  return {messageId};
}

function messageMetadataRequest(input) {
  const source = object(input);
  if (Object.keys(source).length !== 1 || !Object.prototype.hasOwnProperty.call(source, 'messageId')) throw new Error('messageId is required');
  const messageId = identifier(source.messageId);
  if (!messageId) throw new Error('messageId must be a native desktop identifier');
  return {messageIds: [messageId]};
}

function messageMetadataProject(response) {
  const messages = object(object(object(response).entity).messages);
  const entries = Object.entries(messages).slice(0, 100).map(([key, value]) => {
    const message = object(value);
    const result = {messageId: typeof message.id === 'string' ? message.id : key};
    for (const key of ['chatId', 'threadId']) if (typeof message[key] === 'string') result[key] = message[key];
    if (typeof message.fromId === 'string') result.senderId = message.fromId;
    return result;
  });
  return {messages: entries, total: Object.keys(messages).length, truncated: Object.keys(messages).length > entries.length};
}

function chatMetadataRequest(input) {
  const source = object(input);
  if (Object.keys(source).length !== 1 || !Object.prototype.hasOwnProperty.call(source, 'chatId')) throw new Error('chatId is required');
  const chatId = identifier(source.chatId);
  if (!chatId) throw new Error('chatId must be a native desktop identifier');
  // Matches mGetChats' default `shouldAuth: true, source: DD.UNKNOWN (1)`.
  return {chatIds: [chatId], shouldAuth: true, source: 1};
}

function chatMetadataProject(response) {
  const chats = object(object(object(response).entity).chats);
  const entries = Object.entries(chats).slice(0, 100).map(([key, value]) => {
    const chat = object(value);
    const result = {chatId: typeof chat.id === 'string' ? chat.id : key};
    for (const field of ['type', 'name', 'ownerId', 'userCount']) {
      if (typeof chat[field] === 'string' || typeof chat[field] === 'number') result[field] = chat[field];
    }
    return result;
  });
  return {chats: entries, total: Object.keys(chats).length, truncated: Object.keys(chats).length > entries.length};
}

function inboxChatsRequest(input) {
  const source = object(input);
  if (Object.keys(source).length !== 1 || !Object.prototype.hasOwnProperty.call(source, 'count')) throw new Error('count is required');
  // The feed reducer initializes its first-page cursor this way. `filter: 1`
  // is lJ.INBOX; its `count` is derived from the current viewport height.
  if (!Number.isSafeInteger(source.count) || source.count < 1) throw new Error('count must be a positive safe integer');
  return {feedCursor: {id: '0', rankTime: '9223372036854775807'}, filter: 1, count: source.count};
}

function inboxChatsProject(response) {
  const previews = Array.isArray(object(response).previews) ? object(response).previews : [];
  const chatCards = previews.filter(value => object(value).chatData);
  const chats = chatCards.slice(0, 100).map(value => {
    const card = object(value);
    const chat = object(card.chatData);
    const result = {};
    if (typeof card.feedId === 'string') result.chatId = card.feedId;
    if (typeof chat.name === 'string') result.name = text(chat.name, 512);
    for (const field of ['chatType', 'unreadCount']) {
      if (typeof chat[field] === 'string' || typeof chat[field] === 'number') result[field] = chat[field];
    }
    return result;
  });
  return {chats, total: chatCards.length, truncated: chatCards.length > chats.length};
}

function p2pChatRequest(input) {
  const source = object(input);
  if (Object.keys(source).length !== 1 || !Object.prototype.hasOwnProperty.call(source, 'userId')) throw new Error('userId is required');
  const userId = identifier(source.userId);
  if (!userId) throw new Error('userId must be a native desktop user identifier');
  return {chatterIds: [userId]};
}

function p2pChatProject(response, input) {
  const source = object(response);
  const requestedUserId = object(input).userId;
  const mappings = object(source.chatterId2chat);
  const entries = requestedUserId === undefined ? Object.entries(mappings) : [[requestedUserId, mappings[requestedUserId]]];
  const chats = entries.flatMap(([userId, value]) => {
    const chat = object(value);
    return typeof chat.id === 'string' ? [{userId, chatId: chat.id}] : [];
  });
  return {chats, total: chats.length, truncated: false};
}

function recentMessageMatchRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) if (!['chatId', 'senderId', 'text', 'count'].includes(key)) throw new Error(`unsupported recent-message-match parameter: ${key}`);
  const chatId = identifier(source.chatId);
  const senderId = identifier(source.senderId);
  if (!chatId || !senderId) throw new Error('chatId and senderId must be native desktop identifiers');
  if (typeof source.text !== 'string' || !source.text.length || source.text.length > 16384) throw new Error('text must be a non-empty string up to 16 KiB');
  const count = source.count === undefined ? 20 : source.count;
  if (!Number.isSafeInteger(count) || count < 1 || count > 100) throw new Error('count must be an integer between 1 and 100');
  // First-screen and synchronous-server defaults used by getChatMessages.
  return {chatId, scene: 1, count, strategy: 3, redundancyCount: 0, subscribChatEvent: false, needResponse: true};
}

function recentMessageMatchProject(response, input) {
  const source = object(response);
  const query = object(input);
  const messages = object(object(source.entity).messages);
  const itemIds = Array.isArray(source.messageItems) ? source.messageItems.map(value => object(value).itemId).filter(value => typeof value === 'string') : Object.keys(messages);
  const matches = itemIds.slice(0, 100).flatMap(messageId => {
    const message = object(messages[messageId]);
    const messageText = richTextText(object(message.content).richText);
    return message.chatId === query.chatId && message.fromId === query.senderId && messageText === query.text
      ? [{messageId: typeof message.id === 'string' ? message.id : messageId, chatId: message.chatId, senderId: message.fromId}]
      : [];
  });
  return {matches, total: matches.length, truncated: false};
}

// The desktop preview helper's gU decoder traverses elementIds in order and,
// for TEXT (tag 1), reads property.text.content. History messages sometimes
// omit innerText, so retain that exact ordered plain-text subset for matching.
function richTextText(value) {
  const richText = object(value);
  if (typeof richText.innerText === 'string' && richText.innerText.length) return richText.innerText;
  const elements = object(richText.elements);
  const walk = (elementId, seen) => {
    if (seen[elementId]) return '';
    const element = object(elements[elementId]);
    if (!Object.keys(element).length) return '';
    const nextSeen = {...seen, [elementId]: true};
    if (element.tag === 1) return text(object(element.property).text && object(element.property).text.content) || '';
    return Array.isArray(element.childIds) ? element.childIds.map(childId => typeof childId === 'string' ? walk(childId, nextSeen) : '').join('') : '';
  };
  return Array.isArray(richText.elementIds) ? richText.elementIds.map(elementId => typeof elementId === 'string' ? walk(elementId, {}) : '').join('') : '';
}

module.exports = {
  operations: {
    'messaging.drafts': {
      command: allDraftsCommand,
      request(input) {
        if (input !== undefined && (input === null || typeof input !== 'object' || Array.isArray(input) || Object.keys(input).length)) throw new Error('messaging.drafts accepts no parameters');
        return {};
      },
      project
    },
    'messaging.chat-draft': {
      command: allDraftsCommand,
      request: chatDraftRequest,
      project: scopedDraftProject
    },
    'messaging.reply-draft': {
      command: allDraftsCommand,
      request: replyDraftRequest,
      project: scopedDraftProject
    },
    'messaging.message-metadata': {
      command: mgetMessagesCommand,
      request: messageMetadataRequest,
      project: messageMetadataProject
    },
    'messaging.chat-metadata': {
      command: mgetChatsCommand,
      request: chatMetadataRequest,
      project: chatMetadataProject
    },
    'messaging.inbox-chats': {
      command: inboxChatsCommand,
      request: inboxChatsRequest,
      project: inboxChatsProject
    },
    'messaging.p2p-chat': {
      command: p2pChatsCommand,
      request: p2pChatRequest,
      project: p2pChatProject
    },
    'messaging.recent-message-match': {
      command: chatMessagesCommand,
      request: recentMessageMatchRequest,
      project: recentMessageMatchProject
    },
    'messaging.favorites': {
      command: favoritesCommand,
      request: favoritesRequest,
      project: favoritesProject
    },
    'messaging.favorite-detail': {
      command: '2244|favorite.v1.GetFavoriteInfoRequest|favorite.v1.GetFavoriteInfoResponse|1|GET_FAVORITE_INFO',
      request: favoriteDetailRequest,
      project: favoriteDetailProject
    },
    'messaging.shortcuts': {
      command: '2062|feed.v1.GetShortcutsRequest|feed.v1.GetShortcutsResponse|1|GET_SHORTCUTS',
      request: shortcutsRequest,
      project: shortcutsProject
    },
    'messaging.feed-folder': {
      command: '2538|feed.v1.GetFeedFolderRequest|feed.v1.GetFeedFolderResponse|1|GET_FEED_FOLDER',
      request: feedFolderRequest,
      project: feedFolderProject
    },
    'messaging.scheduled-chat': {
      command: '39007|im.v1.GetScheduleMessagesRequest|im.v1.GetScheduleMessagesResponse|1|GET_SCHEDULE_MESSAGES',
      request: scheduledChatRequest,
      project: scheduledProject
    },
    'messaging.scheduled-thread': {
      command: '39007|im.v1.GetScheduleMessagesRequest|im.v1.GetScheduleMessagesResponse|1|GET_SCHEDULE_MESSAGES',
      request: scheduledThreadRequest,
      project: scheduledProject
    }
  }
};
