'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const operation = require('./mutations.js').operations['messaging.recall-self-text'];

test('recall reproduces the desktop ordinary-recall request', () => {
  assert.equal(operation.command, '2079|im.v1.RecallMessageRequest|im.v1.RecallMessageResponse|1|RECALL_MESSAGE');
  assert.deepEqual(operation.request({messageId: '7522399349498757158', chatId: '7358778195685949471', expectedText: 'CLI fixture'}), {id: '7522399349498757158'});
  assert.deepEqual(operation.project({message: 'ok'}), {acknowledged: true, verified: false});
});

test('recall rejects broad IDs, incomplete preflight guards, and oversized text', () => {
  assert.throws(() => operation.request({messageId: '01', chatId: '7', expectedText: 'x'}));
  assert.throws(() => operation.request({messageId: '7', chatId: 'x', expectedText: 'x'}));
  assert.throws(() => operation.request({messageId: '7', chatId: '8'}));
  assert.throws(() => operation.request({messageId: '7', chatId: '8', expectedText: 'x', ids: ['7']}));
  assert.throws(() => operation.request({messageId: '7', chatId: '8', expectedText: 'x'.repeat(16385)}));
});
