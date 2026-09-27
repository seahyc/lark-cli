'use strict';
const assert = require('node:assert/strict');
const test = require('node:test');
const { operations } = require('./module.js');

test('one-chat mute read uses generated PullChatsByIds request shape', () => {
  const operation = operations['inbox.chat-mute-state'];
  assert.equal(operation.transport, 'server');
  assert.equal(operation.command, '64|chats.PullChatsByIdsRequest|chats.PullChatsByIdsResponse');
  assert.deepEqual(operation.request({ chatId: '7358778195685949471' }), { chatIds: ['7358778195685949471'] });
  assert.deepEqual(operation.project({ chats: { '7358778195685949471': { isRemind: false } } }), { chatId: '7358778195685949471', muted: 1 });
  // The generated default is isRemind:true, so an omitted default decodes to unmuted.
  assert.deepEqual(operation.project({ chats: { '7358778195685949471': {} } }), { chatId: '7358778195685949471', muted: 0 });
});

test('one-chat mute read rejects broad, malformed, or ambiguous results', () => {
  const operation = operations['inbox.chat-mute-state'];
  assert.throws(() => operation.request({ chatIds: ['7'] }));
  assert.throws(() => operation.request({ chatId: 'not-native' }));
  assert.throws(() => operation.project({ chats: {} }));
  assert.throws(() => operation.project({ chats: { '1': {}, '2': {} } }));
  assert.throws(() => operation.project({ chats: { '1': { isRemind: 1 } } }));
});
