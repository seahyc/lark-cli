'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const textRead = require('./modules/outbound/module.js').operations['messaging.text-message'];
const p2pRead = require('./modules/messaging/module.js').operations['messaging.p2p-chat'];
const recall = require('./modules/recall/mutations.js').operations['messaging.recall-self-text'];

const source = fs.readFileSync(__dirname + '/mail_bridge.js', 'utf8')
  .replace('__LARK_READ_OPERATIONS__', 'globalThis.readOperations')
  .replace('__LARK_MUTATION_OPERATIONS__', 'globalThis.mutationOperations')
  .replace('__LARK_LOCAL_SESSION__', JSON.stringify({port: 9338, token: 'test', expiresAt: Date.now() + 60000}));
const USER = 'self', CHAT = '22', MESSAGE = '11', TEXT = 'recall fixture';

function message(overrides = {}) {
  return {id: MESSAGE, fromId: USER, chatId: CHAT, type: 4, isRecalled: false, content: {richText: {
    innerText: TEXT, elementIds: ['plain'], elements: {plain: {tag: 1, property: {text: {content: TEXT}}}}, imageIds: [], atIds: [], anchorIds: [], mediaIds: [], docsIds: []
  }}, ...overrides};
}

function start({selfChatId = CHAT, reads = [message(), message({isRecalled: true})]} = {}) {
  let peer, recalls = 0, readCalls = 0, p2ps = 0; const delays = [], writes = [];
  class Socket { constructor() { peer = this; } send(raw) { this.reply = JSON.parse(raw); } close() {} }
  const api = {passport: {getUserId: async () => USER}, transport: {callSdkApi: async (command, payload) => {
    if (command === p2pRead.command) { p2ps += 1; return {data: {chatterId2chat: selfChatId === null ? {} : {[USER]: {id: selfChatId}}}}; }
    if (command === textRead.command) {
      const value = reads[Math.min(readCalls++, reads.length - 1)];
      return {data: {entity: {messages: value === null ? {} : {[MESSAGE]: value}}}};
    }
    if (command === recall.command) { recalls += 1; writes.push(payload); return {data: {message: 'ok'}}; }
    throw new Error(`unexpected command ${command}`);
  }}};
  const context = {window: {LarkAPI: api}, LarkAPI: api, WebSocket: Socket, Date, console, queueMicrotask,
    readOperations: {'messaging.p2p-chat': p2pRead, 'messaging.text-message': textRead}, mutationOperations: {'messaging.recall-self-text': recall},
    setTimeout(callback, delay) { if (delay === 250) { delays.push(delay); queueMicrotask(callback); } return delays.length; }};
  vm.runInNewContext(source, context);
  async function invoke(input = {messageId: MESSAGE, chatId: CHAT, expectedText: TEXT}) {
    await peer.onmessage({data: JSON.stringify({id: 'recall', operation: 'messaging.recall-self-text', params: {expectedUserId: USER, apply: true, input}})});
    return peer.reply;
  }
  return {invoke, get recalls() { return recalls; }, get readCalls() { return readCalls; }, get p2ps() { return p2ps; }, delays, writes};
}

test('recall writes once after own self TEXT preflight and requires isRecalled readback', async () => {
  const fixture = start();
  const reply = await fixture.invoke();
  assert.equal(reply.ok, true);
  assert.deepEqual(reply.data, {userId: USER, acknowledged: true, verified: true, data: {messageId: MESSAGE, chatId: CHAT}});
  assert.equal(fixture.p2ps, 1); assert.equal(fixture.readCalls, 2); assert.equal(fixture.recalls, 1);
  assert.deepEqual(fixture.writes, [{id: MESSAGE}]); assert.deepEqual(fixture.delays, []);
});

test('recall rejects changed ownership, destination, type, text, rich content, and self mapping before write', async () => {
  const invalid = [
    {message: message({fromId: 'other'})}, {message: message({chatId: 'other'})}, {message: message({type: 2})},
    {message: message({content: {richText: {...message().content.richText, innerText: 'other', elements: {plain: {tag: 1, property: {text: {content: 'other'}}}}}}})},
    {message: message({content: {richText: {...message().content.richText, imageIds: ['image']}}})}, {selfChatId: 'other'}
  ];
  for (const item of invalid) {
    const fixture = start({selfChatId: item.selfChatId, reads: [item.message || message()]});
    const reply = await fixture.invoke();
    assert.equal(reply.ok, false); assert.equal(fixture.recalls, 0); assert.equal(fixture.delays.length, 0);
  }
});

test('recall does not treat missing or unrecalled record as proof and never retries the write', async () => {
  for (const reads of [[message(), null], [message(), message()]]) {
    const fixture = start({reads});
    const reply = await fixture.invoke();
    assert.equal(reply.ok, false); assert.match(reply.error, /Recall readback/);
    assert.equal(fixture.recalls, 1); assert.equal(fixture.readCalls, 5);
    assert.deepEqual(fixture.delays, [250, 250, 250]);
  }
});
