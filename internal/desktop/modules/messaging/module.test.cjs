'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const moduleUnderTest = require('./module.js');

test('draft read requests use the desktop wrapper payloads', () => {
  const operations = moduleUnderTest.operations;
  assert.deepEqual(operations['messaging.drafts'].request(), {});
  assert.equal(operations['messaging.chat-draft'].command, operations['messaging.drafts'].command);
  assert.equal(operations['messaging.reply-draft'].command, operations['messaging.drafts'].command);
  assert.deepEqual(operations['messaging.chat-draft'].request({chatId: 'oc_1'}), {chatId: 'oc_1'});
  assert.deepEqual(operations['messaging.chat-draft'].request({threadId: 'omt_1'}), {threadId: 'omt_1'});
  assert.deepEqual(operations['messaging.chat-draft'].request({chatId: 'oc_1', threadId: 'omt_1'}), {chatId: 'oc_1', threadId: 'omt_1'});
  assert.deepEqual(operations['messaging.reply-draft'].request({messageId: 'om_1'}), {messageId: 'om_1'});
});

test('scoped draft reads filter the authoritative draft entity before bounding', () => {
  const operations = moduleUnderTest.operations;
  const text = value => JSON.stringify({innerText: value});
  const unrelated = Object.fromEntries(Array.from({length: 101}, (_, index) => [`u${index}`, {id: `u${index}`, chatId: 'other', type: 1, content: text('other')} ]));
  const matching = Object.fromEntries(Array.from({length: 101}, (_, index) => [`m${index}`, {id: `m${index}`, chatId: '7358778195685949471', type: 1, content: text(`match-${index}`)} ]));
  const response = {entity: {drafts: {...unrelated, '9': {chatId: '7358778195685949471', type: 1, content: text('fixture')}, ...matching}}};
  const chat = operations['messaging.chat-draft'].project(response, {chatId: '7358778195685949471'});
  assert.equal(chat.total, 102);
  assert.equal(chat.drafts.length, 100);
  assert.equal(chat.truncated, true);
  assert.equal(chat.drafts.some(value => value.id === 'u0'), false);
  assert.deepEqual(operations['messaging.chat-draft'].project({entity: {drafts: {'9': {chatId: '7358778195685949471', type: 1, content: text('fixture')}}}}, {chatId: '7358778195685949471'}), {drafts: [{id: '9', chatId: '7358778195685949471', type: 1, text: 'fixture'}], total: 1, truncated: false});
  assert.deepEqual(operations['messaging.reply-draft'].project({entity: {drafts: {'r': {id: 'r', messageId: 'native-message', type: 1, content: text('reply')}, 'other': {id: 'other', messageId: 'other'}}}}, {messageId: 'native-message'}), {drafts: [{id: 'r', messageId: 'native-message', type: 1, text: 'reply'}], total: 1, truncated: false});
  assert.deepEqual(operations['messaging.chat-draft'].project({}, {chatId: 'missing'}), {drafts: [], total: 0, truncated: false});
});

test('draft reads reject missing or unrecognised payload fields', () => {
  const operations = moduleUnderTest.operations;
  assert.throws(() => operations['messaging.drafts'].request({pageSize: 1}));
  assert.throws(() => operations['messaging.chat-draft'].request({}));
  assert.throws(() => operations['messaging.chat-draft'].request({chatId: 'oc_1', pageSize: 1}));
  assert.throws(() => operations['messaging.reply-draft'].request({}));
  assert.throws(() => operations['messaging.reply-draft'].request({messageId: 'om_1', chatId: 'oc_1'}));
  assert.throws(() => operations['messaging.chat-draft'].request({chatId: 'x'.repeat(513)}));
  assert.throws(() => operations['messaging.reply-draft'].request({messageId: 'x'.repeat(513)}));
});

test('message metadata uses MGET_MESSAGES and omits message content', () => {
  const operation = moduleUnderTest.operations['messaging.message-metadata'];
  assert.deepEqual(operation.request({messageId: '7522399349498757158'}), {messageIds: ['7522399349498757158']});
  assert.throws(() => operation.request({messageIds: ['7522399349498757158']}));
  assert.throws(() => operation.request({messageId: 'x'.repeat(513)}));
  assert.deepEqual(operation.project({entity: {messages: {
    '7522399349498757158': {id: '7522399349498757158', chatId: 'native-chat', threadId: 'native-thread', fromId: 'sender', content: {text: 'omit'}}
  }}}), {messages: [{messageId: '7522399349498757158', chatId: 'native-chat', threadId: 'native-thread', senderId: 'sender'}], total: 1, truncated: false});
});

test('chat metadata uses the desktop MGET_CHATS defaults and omits chat content', () => {
  const operation = moduleUnderTest.operations['messaging.chat-metadata'];
  assert.deepEqual(operation.request({chatId: 'native-chat'}), {chatIds: ['native-chat'], shouldAuth: true, source: 1});
  assert.throws(() => operation.request({chatIds: ['native-chat']}));
  assert.deepEqual(operation.project({entity: {chats: {
    'native-chat': {id: 'native-chat', type: 2, name: 'Self chat', ownerId: 'owner', userCount: 1, lastMessage: {content: 'omit'}}
  }}}), {chats: [{chatId: 'native-chat', type: 2, name: 'Self chat', ownerId: 'owner', userCount: 1}], total: 1, truncated: false});
});

test('inbox chat cards use the feed first-page schema and omit preview snippets', () => {
  const operation = moduleUnderTest.operations['messaging.inbox-chats'];
  assert.deepEqual(operation.request({count: 20}), {feedCursor: {id: '0', rankTime: '9223372036854775807'}, filter: 1, count: 20});
  assert.throws(() => operation.request());
  assert.throws(() => operation.request({count: 0}));
  assert.deepEqual(operation.project({previews: [
    {feedId: 'native-chat', chatData: {name: 'Oswald', chatType: 1, unreadCount: 2, lastMessage: {content: 'omit'}}},
    {feedId: 'document', docData: {name: 'not a chat'}},
    {feedId: 'no-name', chatData: {chatType: 2, unreadCount: 0, draftPreview: {content: 'omit'}}}
  ]}), {chats: [
    {chatId: 'native-chat', name: 'Oswald', chatType: 1, unreadCount: 2},
    {chatId: 'no-name', chatType: 2, unreadCount: 0}
  ], total: 2, truncated: false});
});

test('P2P chat lookup uses one native chatter ID and returns only its chat ID', () => {
  const operation = moduleUnderTest.operations['messaging.p2p-chat'];
  assert.deepEqual(operation.request({userId: '7294093360141238304'}), {chatterIds: ['7294093360141238304']});
  assert.throws(() => operation.request({chatterIds: ['7294093360141238304']}));
  assert.throws(() => operation.request({userId: 'x'.repeat(513)}));
  assert.deepEqual(operation.project({chatterId2chat: {
    '7294093360141238304': {id: '7358778195685949471', name: 'omit', lastMessage: {content: 'omit'}},
    other: {id: 'other-chat'}
  }}, {userId: '7294093360141238304'}), {chats: [{userId: '7294093360141238304', chatId: '7358778195685949471'}], total: 1, truncated: false});
});

test('recent message matcher uses first-screen history and returns no message body', () => {
  const operation = moduleUnderTest.operations['messaging.recent-message-match'];
  const input = {chatId: 'native-chat', senderId: 'native-user', text: 'CLI self-test'};
  assert.deepEqual(operation.request(input), {chatId: 'native-chat', scene: 1, count: 20, strategy: 3, redundancyCount: 0, subscribChatEvent: false, needResponse: true});
  assert.throws(() => operation.request({...input, count: 0}));
  assert.throws(() => operation.request({chatId: 'native-chat', senderId: 'native-user', text: ''}));
  assert.deepEqual(operation.project({messageItems: [{itemId: 'match'}, {itemId: 'unrelated'}], entity: {messages: {
    match: {id: '735', chatId: 'native-chat', fromId: 'native-user', content: {richText: {innerText: 'CLI self-test'}}, extra: 'omit'},
    unrelated: {id: '736', chatId: 'native-chat', fromId: 'native-user', content: {richText: {innerText: 'other'}}}
  }}}, input), {matches: [{messageId: '735', chatId: 'native-chat', senderId: 'native-user'}], total: 1, truncated: false});
  assert.deepEqual(operation.project({messageItems: [{itemId: 'graph'}], entity: {messages: {
    graph: {id: '737', chatId: 'native-chat', fromId: 'native-user', content: {richText: {innerText: '', elementIds: ['one', 'two'], elements: {
      one: {tag: 1, property: {text: {content: 'CLI '}}}, two: {tag: 1, property: {text: {content: 'self-test'}}}
    }}}}
  }}}, input), {matches: [{messageId: '737', chatId: 'native-chat', senderId: 'native-user'}], total: 1, truncated: false});
});

test('projection preserves relevant drafts while bounding count and content', () => {
  const operations = moduleUnderTest.operations;
  const encoded = innerText => JSON.stringify({innerText});
  const drafts = Object.fromEntries(Array.from({length: 101}, (_, index) => [`d${index}`, {id: `d${index}`, type: 1, content: encoded('x'.repeat(20000)), ignored: 'no'}]));
  const result = operations['messaging.drafts'].project({draft: {id: 'primary', type: 1, content: encoded('hello'), secret: 'ignored'}, entity: {drafts}});
  assert.deepEqual(result.draft, {id: 'primary', type: 1, text: 'hello'});
  assert.equal(result.drafts.length, 100);
  assert.equal(result.total, 101);
  assert.equal(result.truncated, true);
  assert.equal(result.drafts[0].text.length, 16384);
  assert.equal(result.drafts[0].ignored, undefined);
});

test('draft projection does not expose raw or malformed non-TEXT content', () => {
  const project = moduleUnderTest.operations['messaging.drafts'].project;
  assert.deepEqual(project({draft: {id: 'post', type: 2, content: '{"innerText":"not-text"}'}}).draft, {id: 'post', type: 2});
  assert.deepEqual(project({draft: {id: 'broken', type: 1, content: '{not-json'}}).draft, {id: 'broken', type: 1});
});

test('empty draft entity has an explicit empty result', () => {
  const result = moduleUnderTest.operations['messaging.drafts'].project({entity: {drafts: {}}});
  assert.deepEqual(result, {drafts: [], total: 0, truncated: false});
});

test('favorites uses its UI cursor transform and projects consumed fields', () => {
  const favorites = moduleUnderTest.operations['messaging.favorites'];
  assert.deepEqual(favorites.request(), {time: '0', count: 15});
  assert.deepEqual(favorites.request({count: 2, time: 42}), {time: '42', count: 2});
  assert.throws(() => favorites.request({count: 1.5}));
  assert.throws(() => favorites.request({time: '42'}));
  assert.throws(() => favorites.request({cursor: 1}));
  const result = favorites.project({favoritesIds: ['fav-1'], hasMore: true, minTime: 41, entity: {favorites: {'fav-1': {type: 2, content: {messageId: 'native-message-id'}}}}});
  assert.deepEqual(result, {favorites: [{id: 'fav-1', favorite: {type: 2, content: {messageId: 'native-message-id'}}}], hasMore: true, minTime: 41});
});

test('favorite detail uses the UI single-id array request', () => {
  const detail = moduleUnderTest.operations['messaging.favorite-detail'];
  assert.deepEqual(detail.request({favoriteId: 'native-favorite'}), {favoriteIds: ['native-favorite']});
  assert.throws(() => detail.request({favoriteId: 'x'.repeat(513)}));
  assert.throws(() => detail.request({favoriteIds: ['native-favorite']}));
  assert.deepEqual(detail.project({entity: {favorites: {one: {type: 1, content: {messageId: 'm'}}}}}), {favorites: [{id: 'one', favorite: {type: 1, content: {messageId: 'm'}}}], total: 1, truncated: false});
});

test('shortcuts uses the UI local-cache strategy and structural metadata only', () => {
  const shortcuts = moduleUnderTest.operations['messaging.shortcuts'];
  assert.deepEqual(shortcuts.request(), {syncDataStrategy: 1});
  assert.throws(() => shortcuts.request({syncDataStrategy: 3}));
  assert.deepEqual(shortcuts.project({shortcuts: [{id: 'shortcut', feedType: 3, channel: {id: 'native-channel', type: 14, private: 'omit'}, private: 'omit'}], entityPreview: {one: {title: 'Preview'}}}), {shortcuts: [{id: 'shortcut', feedType: 3, channel: {id: 'native-channel', type: 14}}], total: 1, truncated: false});
});

test('feed folder and scheduled reads reproduce UI payload forms', () => {
  const ops = moduleUnderTest.operations;
  assert.deepEqual(ops['messaging.feed-folder'].request(), {});
  assert.deepEqual(ops['messaging.feed-folder'].project({feedFolderSettings: {usedFolderIds: ['one']}}), {usedFolderIds: ['one'], total: 1, truncated: false});
  assert.deepEqual(ops['messaging.scheduled-chat'].request({chatId: 'chat'}), {chatId: 'chat', syncDataStrategy: 3, scene: 1});
  assert.deepEqual(ops['messaging.scheduled-chat'].request({chatId: 'chat', rootId: 'root', strategy: 1}), {chatId: 'chat', rootId: 'root', syncDataStrategy: 1, scene: 3});
  assert.deepEqual(ops['messaging.scheduled-thread'].request({chatId: 'chat', threadId: 'thread'}), {chatId: 'chat', threadId: 'thread', syncDataStrategy: 3, scene: 2});
  const result = ops['messaging.scheduled-chat'].project({messageItems: [{itemId: 'i'}], entity: {scheduleMessage: {i: {scheduleTime: 1, status: 2}}}});
  assert.deepEqual(result, {items: [{itemId: 'i', scheduleMessage: {scheduleTime: 1, status: 2}}], total: 1, truncated: false});
});

test('missing metadata maps produce explicit empty results',()=>{for(const name of ['messaging.chat-metadata','messaging.message-metadata']){const out=moduleUnderTest.operations[name].project({});assert.equal(out.total,0);assert.equal(out.truncated,false)}});
