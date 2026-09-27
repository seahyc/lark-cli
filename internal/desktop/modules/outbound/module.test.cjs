'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');

const operation = require('./module.js').operations['messaging.text-message'];

test('text-message reads exactly one native message ID', () => {
  assert.deepEqual(operation.request({messageId: '42'}), {messageIds: ['42']});
  for (const input of [{}, {messageId: 'om_public'}, {messageId: '01'}, {messageId: '1', extra: true}]) {
    assert.throws(() => operation.request(input));
  }
});

test('text-message projects only preflight fields and bounds text', () => {
  const response = {entity: {messages: {'42': {
    id: '42', chatId: '77', fromId: '99', type: 4,
    content: {richText: {innerText: 'body', elements: {secret: {}}}},
    createTime: 'private'
  }}}};
  assert.deepEqual(operation.project(response), {chatId: '77', senderId: '99', type: 4, text: 'body'});
  assert.equal(operation.project({entity: {messages: {'42': {content: {richText: {innerText: 'x'.repeat(17000)}}}}}}).text.length, 16384);
  assert.deepEqual(Object.keys(require('./manifest.json')), ['0']);
});

test('text-message reconstructs a plain graph when innerText is blank', () => {
  const result = operation.project({entity: {messages: {'42': {
    content: {richText: {innerText: '', elementIds: ['a', 'b'], elements: {
      a: {tag: 1, property: {text: {content: 'hello '}}},
      b: {tag: 1, property: {text: {content: 'world'}}}
    }}}
  }}}});
  assert.equal(result.text, 'hello world');
});

test('missing text-message entity projects an empty result', () => { assert.deepEqual(require('./module.js').operations['messaging.text-message'].project({}), {}); });
