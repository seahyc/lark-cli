'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const mutations = require('./mutations.js').operations;

test('delete draft only accepts one bounded native identifier', () => {
  assert.deepEqual(mutations['messaging.delete-draft'].request({draftId: 'draft-native'}), {draftId: 'draft-native'});
  assert.throws(() => mutations['messaging.delete-draft'].request({draftId: 'x'.repeat(513)}));
  assert.throws(() => mutations['messaging.delete-draft'].request({draftId: 'd', chatId: 'c'}));
});

test('create text draft uses the shipped UUID-v4 rich-text graph', () => {
  const operation = mutations['messaging.create-text-draft'];
  const request = operation.request({chatId: 'native-chat', text: 'hello'});
  assert.equal(request.draft.chatId, 'native-chat');
  assert.equal(request.draft.type, 1);
  const richText = JSON.parse(request.draft.content);
  assert.match(richText.elementIds[0], /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  assert.deepEqual(richText, {
    elementIds: [richText.elementIds[0]],
    innerText: 'hello',
    elements: {[richText.elementIds[0]]: {tag: 1, property: {text: {content: 'hello'}}}},
    imageIds: [], atIds: [], anchorIds: [], mediaIds: [], docsIds: []
  });
  assert.throws(() => operation.request({chatId: 'native-chat', text: '   '}));
  assert.throws(() => operation.request({chatId: 'native-chat', text: 'é'.repeat(8193)}));
  assert.throws(() => operation.request({chatId: 'x'.repeat(513), text: 'hello'}));
  assert.throws(() => operation.request({chatId: 'native-chat', text: 'hello', type: 1}));
});

test('last-read matches the UI offset transform without sending it', () => {
  assert.deepEqual(mutations['messaging.set-chat-last-read'].request({chatId: 'chat-native', position: 8, offset: 1.6}), {chatId: 'chat-native', position: 8, offset: '2'});
  assert.deepEqual(mutations['messaging.set-chat-last-read'].request({chatId: 'chat-native'}), {chatId: 'chat-native', position: -1, offset: '0'});
  assert.throws(() => mutations['messaging.set-chat-last-read'].request({chatId: 'chat-native', unexpected: true}));
});
