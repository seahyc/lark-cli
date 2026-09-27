'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const messaging = require('./modules/messaging/module.js').operations;
const triageReads = require('./modules/triage/module.js').operations;
const triage = require('./modules/triage/mutations.js').operations;

const accountMetadata = {
  command: 'test-mail-account-metadata',
  request() { return {}; },
  project() { return {accounts: [{isSelected: true, mailAccountId: USER, isShared: false}]}; },
};
const source = fs.readFileSync(__dirname + '/mail_bridge.js', 'utf8')
  .replace('__LARK_READ_OPERATIONS__', 'globalThis.readOperations')
  .replace('__LARK_MUTATION_OPERATIONS__', 'globalThis.mutationOperations')
  .replace('__LARK_LOCAL_SESSION__', JSON.stringify({port: 9337, token: 'test', expiresAt: Date.now() + 60000}));

const USER = 'self';
const CHAT = '22';
const MESSAGE = '900';
const TEXT = 'scheduled fixture';
const futureUnixSeconds = () => String(Math.floor(Date.now() / 1000) + 86400);

function scheduledResponse({present = true, time = futureUnixSeconds(), text = TEXT, status = 1, type = 4} = {}) {
  if (!present) return {messageItems: [], entity: {scheduleMessage: {}}};
  return {messageItems: [{itemId: MESSAGE}], entity: {scheduleMessage: {
    [MESSAGE]: {id: MESSAGE, chatId: CHAT, scheduleTime: time, status, message: {fromId: USER, type, content: {richText: {innerText: text}}}}
  }}};
}

function start({selfChatId = CHAT, stagedCid = 'same', serverFailure = false, schedules = []} = {}) {
  let peer; let stageCalls = 0; let serverCalls = 0; let cancelCalls = 0; let scheduleCalls = 0;
  const delays = []; const serverPayloads = []; const sdkPayloads = [];
  class Socket { constructor() { peer = this; } send(raw) { this.reply = JSON.parse(raw); } close() {} }
  const api = {
    passport: {getUserId: async () => USER},
    transport: {
      callSdkApi: async (command, payload) => {
        if (command === accountMetadata.command) return {data: {}};
        if (command === messaging['messaging.p2p-chat'].command) return {data: {chatterId2chat: selfChatId === null ? {} : {[USER]: {id: selfChatId}}}};
        if (command === triage['triage.schedule-self-text'].command) {
          stageCalls += 1; sdkPayloads.push({command, payload});
          return {data: {cid: stagedCid === 'same' ? payload.cid : stagedCid}};
        }
        if (command === triage['triage.schedule-cancel-text'].command) { cancelCalls += 1; sdkPayloads.push({command, payload}); return {data: {}}; }
        if (command === triageReads['triage.scheduled-items'].command) {
          const item = schedules[Math.min(scheduleCalls++, schedules.length - 1)] || scheduledResponse({present: false});
          return {data: item};
        }
        throw new Error(`unexpected SDK command ${command}`);
      },
      callServerApi: async (command, payload, options) => {
        serverCalls += 1; serverPayloads.push({command, payload, options});
        if (serverFailure) throw new Error('server persist failed');
        return {data: {message: {messageId: MESSAGE}}};
      }
    }
  };
  const context = {
    window: {LarkAPI: api}, LarkAPI: api, WebSocket: Socket, Date, console, queueMicrotask,
    readOperations: {...messaging, ...triageReads, 'mail.account-metadata': accountMetadata}, mutationOperations: triage,
    setTimeout(callback, delay) { if (delay === 250) { delays.push(delay); queueMicrotask(callback); } return delays.length; }
  };
  vm.runInNewContext(source, context);
  async function invoke(operation, input) {
    await peer.onmessage({data: JSON.stringify({id: operation, operation, params: {expectedUserId: USER, apply: true, input}})});
    return peer.reply;
  }
  return {invoke, get stageCalls() { return stageCalls; }, get serverCalls() { return serverCalls; }, get cancelCalls() { return cancelCalls; }, get scheduleCalls() { return scheduleCalls; }, delays, serverPayloads, sdkPayloads};
}

test('scheduled self text stages locally, persists once on server, then verifies the pending item', async () => {
  const time = futureUnixSeconds();
  const fixture = start({schedules: [scheduledResponse({time})]});
  const reply = await fixture.invoke('triage.schedule-self-text', {chatId: CHAT, text: TEXT, scheduleTime: time});
  assert.equal(reply.ok, true);
  assert.deepEqual(reply.data, {userId: USER, acknowledged: true, verified: true, data: {messageId: MESSAGE, chatId: CHAT, scheduleTime: time}});
  assert.equal(fixture.stageCalls, 1); assert.equal(fixture.serverCalls, 1); assert.equal(fixture.scheduleCalls, 1);
  assert.equal(fixture.serverPayloads[0].command, triage['triage.schedule-self-text'].persistCommand);
  assert.equal(fixture.serverPayloads[0].options.parentContextId, '');
  assert.equal(fixture.serverPayloads[0].options.collectTrace, false);
  assert.equal(fixture.serverPayloads[0].payload.scheduleTime, time);
  assert.equal(fixture.serverPayloads[0].payload.chatId, CHAT);
  assert.equal(fixture.serverPayloads[0].payload.putRequest.cid, fixture.sdkPayloads[0].payload.cid);
  assert.deepEqual(fixture.delays, []);
});

test('scheduled creation rejects self destination and future-time bounds before staging', async () => {
  for (const input of [
    {chatId: 'other', text: TEXT, scheduleTime: futureUnixSeconds()},
    {chatId: CHAT, text: TEXT, scheduleTime: String(Math.floor(Date.now() / 1000) + 59)}
  ]) {
    const fixture = start();
    const reply = await fixture.invoke('triage.schedule-self-text', input);
    assert.equal(reply.ok, false);
    assert.equal(fixture.stageCalls, 0); assert.equal(fixture.serverCalls, 0); assert.equal(fixture.scheduleCalls, 0);
  }
});

test('scheduled creation rejects a mismatched staged cid before server persistence', async () => {
  const fixture = start({stagedCid: 'wrong-cid'});
  const reply = await fixture.invoke('triage.schedule-self-text', {chatId: CHAT, text: TEXT, scheduleTime: futureUnixSeconds()});
  assert.equal(reply.ok, false); assert.match(reply.error, /prepared\.cid/);
  assert.equal(fixture.stageCalls, 1); assert.equal(fixture.serverCalls, 0); assert.equal(fixture.scheduleCalls, 0);
});

test('scheduled server persistence failure is not retried', async () => {
  const fixture = start({serverFailure: true});
  const reply = await fixture.invoke('triage.schedule-self-text', {chatId: CHAT, text: TEXT, scheduleTime: futureUnixSeconds()});
  assert.equal(reply.ok, false); assert.match(reply.error, /server persist failed/);
  assert.equal(fixture.stageCalls, 1); assert.equal(fixture.serverCalls, 1); assert.equal(fixture.scheduleCalls, 0); assert.deepEqual(fixture.delays, []);
});

test('scheduled cancellation checks its exact pending TEXT snapshot then verifies absence', async () => {
  const time = futureUnixSeconds();
  const fixture = start({schedules: [scheduledResponse({time}), scheduledResponse({present: false})]});
  const reply = await fixture.invoke('triage.schedule-cancel-text', {messageId: MESSAGE, chatId: CHAT, scheduleTime: time, expectedText: TEXT});
  assert.equal(reply.ok, true);
  assert.deepEqual(reply.data, {userId: USER, acknowledged: true, verified: true, data: {messageId: MESSAGE, chatId: CHAT, scheduleTime: time}});
  assert.equal(fixture.cancelCalls, 0); assert.equal(fixture.serverCalls, 1); assert.equal(fixture.scheduleCalls, 2);
  assert.equal(fixture.serverPayloads[0].command, '900034|messages.PatchScheduleMessageRequest|messages.PatchScheduleMessageResponse');
  assert.deepEqual(fixture.serverPayloads[0].payload, {patchType: 3, messageId: MESSAGE, chatId: CHAT, scheduleTime: time, type: 4, sendImmediately: false});
});

test('scheduled cancellation rejects a wrong text, time, or status before mutation', async () => {
  const time = futureUnixSeconds();
  for (const response of [
    scheduledResponse({time, text: 'other'}),
    scheduledResponse({time: String(Number(time) + 1)}),
    scheduledResponse({time, status: 2})
  ]) {
    const fixture = start({schedules: [response]});
    const reply = await fixture.invoke('triage.schedule-cancel-text', {messageId: MESSAGE, chatId: CHAT, scheduleTime: time, expectedText: TEXT});
    assert.equal(reply.ok, false);
    assert.equal(fixture.cancelCalls, 0); assert.equal(fixture.serverCalls, 0); assert.equal(fixture.scheduleCalls, 1);
  }
});
