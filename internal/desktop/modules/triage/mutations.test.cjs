'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const {operations} = require('./mutations.js');

test('favorite add reproduces the selected-message UI request', () => {
  const operation = operations['triage.favorite-add'];
  assert.deepEqual(operation.request({messageId: '7522399349498757158', chatId: '7358778195685949471'}), {
    favs: [{id: '7522399349498757158', type: 1, chatId: '7358778195685949471', originMergeForwardId: ''}]
  });
  assert.deepEqual(operation.project({}, {messageId: '7522399349498757158', chatId: '7358778195685949471'}), {
    messageId: '7522399349498757158', chatId: '7358778195685949471', acknowledged: true, verified: false
  });
  assert.throws(() => operation.request({messageIds: ['1'], chatId: '2'}));
  assert.throws(() => operation.request({messageId: '01', chatId: '2'}));
  assert.throws(() => operation.request({messageId: '1', chatId: 'x'.repeat(513)}));
});

test('scheduled self text has a local stage then a cid-bound legacy persist envelope', () => {
  const operation = operations['triage.schedule-self-text'];
  const oldRandom = Math.random;
  Math.random = () => 0;
  try {
    const stage = operation.request({chatId: '7', text: 'fixture', scheduleTime: '1790000000'});
    assert.equal(stage.channel.id, '7'); assert.equal(stage.channel.type, 1); assert.equal(stage.type, 4); assert.equal(stage.scheduleTime, '1790000000');
    assert.match(stage.cid, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
    const persisted = operation.persistRequest(stage, {cid: stage.cid});
    assert.equal(operation.persistCommand, '900030|messages.PutScheduleMessageRequest|messages.PutScheduleMessageResponse');
    assert.deepEqual({scheduleTime: persisted.scheduleTime, chatId: persisted.chatId, entityId: persisted.entityId}, {scheduleTime: '1790000000', chatId: '7', entityId: '7'});
    assert.equal(persisted.putRequest.cid, stage.cid); assert.equal(persisted.putRequest.type, 4); assert.equal(persisted.putRequest.isNotified, true);
    assert.deepEqual(persisted.putRequest.content.richText.elementIds, stage.content.richText.elementIds);
    assert.equal(persisted.putRequest.content.richText.innerText, 'fixture');
    assert.ok(persisted.putRequest.content.richText.elements.dictionary[stage.content.richText.elementIds[0]].property instanceof Uint8Array);
    assert.equal(Buffer.from(persisted.putRequest.content.richText.elements.dictionary[stage.content.richText.elementIds[0]].property).toString('hex'), '0a0766697874757265');
    assert.notEqual(persisted.putRequest.content, stage.content);
    assert.throws(() => operation.persistRequest(stage, {cid: 'other'}));
    stage.content.richText.innerText = '';
    assert.throws(() => operation.persistRequest(stage, {cid: stage.cid}), /must not be empty/);
  } finally { Math.random = oldRandom; }
});

test('favorite delete targets one exact UI id envelope', () => {
  const operation = operations['triage.favorite-delete'];
  assert.deepEqual(operation.request({favoriteId: '9'}), {ids: ['9']});
  assert.deepEqual(operation.project({}, {favoriteId: '9'}), {favoriteId: '9', acknowledged: true, verified: false});
  assert.throws(() => operation.request({ids: ['9']}));
  assert.throws(() => operation.request({favoriteId: ''}));
});

test('simple scheduled fixture cancellation uses the desktop DELETE patch', () => {
  const operation = operations['triage.schedule-cancel-text'];
  assert.equal(operation.transport, 'server');
  assert.equal(operation.command, '900034|messages.PatchScheduleMessageRequest|messages.PatchScheduleMessageResponse');
  assert.deepEqual(operation.request({messageId: '9', chatId: '7', scheduleTime: '1790000000', expectedText: 'fixture'}), {
    patchType: 3, messageId: '9', chatId: '7', scheduleTime: '1790000000', type: 4, sendImmediately: false
  });
  assert.throws(() => operation.request({messageId: '9', chatId: '7', scheduleTime: '1790000000'}));
});
