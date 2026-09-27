'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');
const operation = require('./mutations.js').operations['messaging.edit-text'];

test('edit-text creates a TEXT request with canonical plain rich text and cid', () => {
  const request = operation.request({messageId: '123', chatId: '456', text: 'hello', expectedText: 'before'});
  assert.equal(request.msgId, '123');
  assert.equal(request.chatId, '456');
  assert.equal(request.type, 4);
  assert.equal(Object.hasOwn(request, 'expectedText'), false);
  assert.match(request.cid, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  const id = request.content.richText.elementIds[0];
  assert.deepEqual(request.content.richText, {
    elementIds: [id],
    innerText: 'hello',
    elements: {[id]: {tag: 1, property: {text: {content: 'hello'}}}},
    imageIds: [], atIds: [], anchorIds: [], mediaIds: [], docsIds: []
  });
});

test('edit-text rejects noncanonical IDs, empty or oversized text, and extra fields', () => {
  for (const input of [
    {messageId: '01', chatId: '2', text: 'x', expectedText: ''},
    {messageId: '9223372036854775808', chatId: '2', text: 'x', expectedText: ''},
    {messageId: '1', chatId: '-2', text: 'x', expectedText: ''},
    {messageId: '1', chatId: '2', text: '  ', expectedText: ''},
    {messageId: '1', chatId: '2', text: 'x'},
    {messageId: '1', chatId: '2', text: 'x', expectedText: 1},
    {messageId: '1', chatId: '2', text: 'x', expectedText: 'x'.repeat(16385)},
    {messageId: '1', chatId: '2', text: 'x', expectedText: '', content: {}},
    {messageId: '1', chatId: '2', text: '😀'.repeat(4097), expectedText: ''}
  ]) assert.throws(() => operation.request(input));
});

test('edit-text response is acknowledgement-only', () => {
  assert.deepEqual(operation.project({message: {content: 'sensitive'}}), {acknowledged: true, verified: false});
});

test('edit-text permits an empty expected current body without sending it', () => {
  const request = operation.request({messageId: '1', chatId: '2', text: 'replacement', expectedText: ''});
  assert.equal(Object.hasOwn(request, 'expectedText'), false);
});

test('send-self-text uses the native SEND_MESSAGE channel payload', () => {
  const send = require('./mutations.js').operations['messaging.send-self-text'];
  const request = send.request({chatId: '22', text: 'self test'});
  assert.equal(request.type, 4);
  assert.deepEqual(request.channel, {id: '22', type: 1});
  assert.equal(request.shouldNotify, true);
  assert.match(request.cid, /^[0-9a-f-]{36}$/);
  assert.equal(request.content.richText.innerText, 'self test');
  assert.throws(() => send.request({chatId: 'self', text: 'x'}));
  assert.throws(() => send.request({chatId: '22', text: ' '}));
  assert.throws(() => send.request({chatId: '22', text: 'x', channel: {}}));
});


test('send-self-text exposes only a canonical native response message ID', () => {
  const send = require('./mutations.js').operations['messaging.send-self-text'];
  assert.deepEqual(send.project({messageId: '7358778195685949471', sensitive: 'omit'}), {acknowledged: true, verified: false, messageId: '7358778195685949471'});
  assert.deepEqual(send.project({messageId: '01'}), {acknowledged: true, verified: false});
  assert.deepEqual(send.project({messageId: 1}), {acknowledged: true, verified: false});
});
