'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const { operations } = require('./legacy.js');
test('legacy server descriptors retain exact schema but are not registered', () => {
  assert.equal(operations['mail.sender-list'].transport, 'server');
  assert.deepEqual(operations['mail.sender-list'].request({ reqType: 3, searchQuery: 'x@example.test' }), { reqType: 3, lastTimestamp: '0', cursor: '0', size: 100, query: 'x@example.test' });
  assert.throws(() => operations['mail.sender-list'].request({ reqType: 3, cursor: '' }), /cursor.*int64/);
  assert.throws(() => operations['mail.sender-list'].request({ reqType: 3, lastTimestamp: '9223372036854775808' }), /lastTimestamp.*int64/);
  assert.deepEqual(operations['mail.sender-list'].project({ allows: [{ record: 'trusted@example.test', timestamp: '1' }], blocks: [{ record: 'blocked@example.test', timestamp: '2' }] }, { reqType: 3 }), { allows: [{ record: 'trusted@example.test', timestamp: '1' }], allowsTruncated: false, blocks: [{ record: 'blocked@example.test', timestamp: '2' }], blocksTruncated: false });
  assert.equal(operations['mail.sender-remove'].transport, 'server');
  assert.deepEqual(operations['mail.sender-remove'].request({ address: 'person@example.test' }), { multiFrom: ['person@example.test'], isAllow: false });
  assert.throws(() => operations['mail.sender-remove'].request({ address: 'invalid' }), /valid email/);
});
