'use strict';
const assert = require('node:assert/strict');
const test = require('node:test');
const { operations } = require('./mutations.js');

test('mute setter changes only native ChatSetting.IS_REMIND', () => {
  const operation = operations['inbox.set-self-chat-mute'];
  assert.equal(operation.transport, 'server');
  assert.equal(operation.command, '60|chats.PatchChatSettingRequest|chats.PatchChatSettingResponse');
  assert.deepEqual(operation.request({ chatId: '7358778195685949471', muted: 1, expectedMuted: 0 }), {
    chatId: '7358778195685949471', chatSetting: { isRemind: false }, updateChatSettingField: [1],
  });
  assert.deepEqual(operation.request({ chatId: '7358778195685949471', muted: 0, expectedMuted: 1 }), {
    chatId: '7358778195685949471', chatSetting: { isRemind: true }, updateChatSettingField: [1],
  });
});

test('mute setter rejects broad payloads and invalid state before native write', () => {
  const operation = operations['inbox.set-self-chat-mute'];
  assert.throws(() => operation.request({ chatId: '7', muted: 1 }));
  assert.throws(() => operation.request({ chatId: '7', muted: true, expectedMuted: 0 }));
  assert.throws(() => operation.request({ chatId: '7', muted: 2, expectedMuted: 0 }));
  assert.throws(() => operation.request({ chatId: '7', muted: 1, expectedMuted: 0, chatSetting: {} }));
  assert.deepEqual(operation.project({ chatSetting: { isRemind: false } }), { acknowledged: true, verified: false, muted: 1 });
});
