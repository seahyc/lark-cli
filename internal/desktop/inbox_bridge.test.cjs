'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const messaging = require('./modules/messaging/module.js').operations;
const inbox = require('./modules/inbox/module.js').operations;
const inboxMutations = require('./modules/inbox/mutations.js').operations;

const bridgeSource = fs.readFileSync(__dirname + '/mail_bridge.js', 'utf8')
  .replace('__LARK_LOCAL_SESSION__', JSON.stringify({ port: 9330, token: 'test', expiresAt: Date.now() + 60000 }));
const readOperations = { 'messaging.p2p-chat': messaging['messaging.p2p-chat'], 'inbox.chat-mute-state': inbox['inbox.chat-mute-state'] };
const mutationOperations = { 'inbox.set-self-chat-mute': inboxMutations['inbox.set-self-chat-mute'] };

function start({ selfChatId = '11', initialMuted = 0, afterWriteMuted = 1, writeError = null } = {}) {
  const peers = [], timers = [], sdk = [], server = [];
  let written = false;
  const api = {
    passport: { getUserId: () => 'user-1' },
    transport: {
      callSdkApi: async (command, payload) => {
        sdk.push({ command, payload });
        if (command === messaging['messaging.p2p-chat'].command) {
          return { data: { chatterId2chat: { 'user-1': { id: selfChatId } } } };
        }
        throw new Error(`unexpected SDK command ${command}`);
      },
      callServerApi: async (command, payload) => {
        server.push({ command, payload: JSON.parse(JSON.stringify(payload)) });
        if (command === inbox['inbox.chat-mute-state'].command) {
          const muted = written ? afterWriteMuted : initialMuted;
          return { data: { chats: { '11': muted ? { isRemind: false } : { isRemind: true } } } };
        }
        if (command === inboxMutations['inbox.set-self-chat-mute'].command) {
          if (writeError) throw writeError;
          written = true;
          return { data: { chatSetting: { isRemind: payload.chatSetting.isRemind } } };
        }
        throw new Error(`unexpected server command ${command}`);
      },
    },
  };
  class Socket {
    constructor() { peers.push(this); }
    send(raw) { this.reply = JSON.parse(raw); }
    close() {}
  }
  vm.runInNewContext(bridgeSource, {
    window: { LarkAPI: api }, LarkAPI: api, WebSocket: Socket, Date, console,
    __LARK_READ_OPERATIONS__: readOperations,
    __LARK_MUTATION_OPERATIONS__: mutationOperations,
    setTimeout(callback, delay) { timers.push({ callback, delay }); if (delay === 250) queueMicrotask(callback); return timers.length; },
  });
  async function invoke(input) {
    const peer = peers[0];
    await peer.onmessage({ data: JSON.stringify({ id: 'mute', operation: 'inbox.set-self-chat-mute', params: { expectedUserId: 'user-1', apply: true, input } }) });
    return peer.reply;
  }
  return { sdk, server, timers, invoke };
}

const input = (overrides = {}) => ({ chatId: '11', muted: 1, expectedMuted: 0, ...overrides });

test('self mute reads the exact self chat, writes once through server 60, then verifies server 64', async () => {
  const fixture = start();
  const reply = await fixture.invoke(input());
  assert.equal(reply.ok, true);
  assert.equal(reply.data.verified, true);
  assert.deepEqual(fixture.sdk.map(x => x.command), [messaging['messaging.p2p-chat'].command]);
  assert.deepEqual(fixture.server, [
    { command: inbox['inbox.chat-mute-state'].command, payload: { chatIds: ['11'] } },
    { command: inboxMutations['inbox.set-self-chat-mute'].command, payload: { chatId: '11', chatSetting: { isRemind: false }, updateChatSettingField: [1] } },
    { command: inbox['inbox.chat-mute-state'].command, payload: { chatIds: ['11'] } },
  ]);
  assert.deepEqual(fixture.timers.filter(x => x.delay === 250), []);
});

test('wrong self chat and stale mute snapshot fail before any server write', async () => {
  const wrongSelf = start({ selfChatId: '12' });
  const wrongReply = await wrongSelf.invoke(input());
  assert.equal(wrongReply.ok, false);
  assert.match(wrongReply.error, /self-chat/);
  assert.deepEqual(wrongSelf.server, []);

  const stale = start({ initialMuted: 1 });
  const staleReply = await stale.invoke(input());
  assert.equal(staleReply.ok, false);
  assert.match(staleReply.error, /state changed/);
  assert.deepEqual(stale.server.map(x => x.command), [inbox['inbox.chat-mute-state'].command]);
});

test('mute readback mismatch fails after four server reads and never retries the write', async () => {
  const fixture = start({ afterWriteMuted: 0 });
  const reply = await fixture.invoke(input());
  assert.equal(reply.ok, false);
  assert.match(reply.error, /Mute readback differs/);
  assert.equal(fixture.server.filter(x => x.command === inboxMutations['inbox.set-self-chat-mute'].command).length, 1);
  assert.equal(fixture.server.filter(x => x.command === inbox['inbox.chat-mute-state'].command).length, 5, 'preflight plus four readback attempts');
  assert.equal(fixture.timers.filter(x => x.delay === 250).length, 3);
});

test('native write failure cannot be treated as a verified mute', async () => {
  const fixture = start({ writeError: new Error('patch failed') });
  const reply = await fixture.invoke(input());
  assert.equal(reply.ok, false);
  assert.match(reply.error, /patch failed/);
  assert.equal(fixture.server.filter(x => x.command === inboxMutations['inbox.set-self-chat-mute'].command).length, 1);
  assert.equal(fixture.server.filter(x => x.command === inbox['inbox.chat-mute-state'].command).length, 1);
});
