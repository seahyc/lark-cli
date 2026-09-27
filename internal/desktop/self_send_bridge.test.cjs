'use strict';

const {test} = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const readOperations = {
  ...require('./modules/messaging/module.js').operations,
  ...require('./modules/outbound/module.js').operations
};
const mutationOperations = require('./modules/outbound/mutations.js').operations;
const source = fs.readFileSync(__dirname + '/mail_bridge.js', 'utf8')
  .replace('__LARK_READ_OPERATIONS__', 'globalThis.__readOperations')
  .replace('__LARK_MUTATION_OPERATIONS__', 'globalThis.__mutationOperations')
  .replace('__LARK_LOCAL_SESSION__', JSON.stringify({port: 9332, token: 'test', expiresAt: Date.now() + 60000}));

function message({sender = 'self', chatId = '22', type = 4, text = 'self test'} = {}) {
  return {
    id: '11', fromId: sender, chatId, type,
    content: {richText: {
      innerText: text, imageIds: [], atIds: [], anchorIds: [], mediaIds: [], docsIds: [],
      elements: {plain: {tag: 1, property: {text: {content: text}}}}
    }}
  };
}

async function execute({
  selfChatId = '22', inputChatId = '22', apply = true, prepareFailure = false,
  preparedCid = 'same', sendMessageId = '11', readbacks = [message()]
} = {}) {
  let peer;
  let prepares = 0;
  let sends = 0;
  let reads = 0;
  let lookups = 0;
  const delays = [];
  let preparedPayload;
  let sentPayload;
  let reply;
  class Socket {
    constructor() { peer = this; }
    send(raw) { reply = JSON.parse(raw); }
    close() {}
  }
  const api = {
    passport: {getUserId: async () => 'self'},
    transport: {
      callSdkApi: async (command, payload) => {
        if (command.startsWith('1046|im.v1.GetP2PChatsByChatterIdsRequest')) {
          lookups += 1;
          assert.deepEqual(payload, {chatterIds: ['self']});
          return {data: {chatterId2chat: selfChatId === null ? {} : {self: {id: selfChatId}}}};
        }
        if (command.startsWith('1001|im.v1.CreateQuasiMessageRequest')) {
          prepares += 1;
          preparedPayload = payload;
          if (prepareFailure) throw new Error('local quasi create failed');
          return {data: {cid: preparedCid === 'same' ? payload.cid : preparedCid, entity: {quasiMessages: {[payload.cid]: {cid: payload.cid}}}}};
        }
        if (command.startsWith('2003|im.v1.SendMessageRequest')) {
          sends += 1;
          sentPayload = payload;
          return {data: sendMessageId === null ? {} : {messageId: sendMessageId}};
        }
        if (command.startsWith('1|im.v1.MGetMessagesRequest')) {
          const value = readbacks[Math.min(reads++, readbacks.length - 1)];
          return {data: {entity: {messages: {[payload.messageIds[0]]: value}}}};
        }
        throw new Error(`unexpected command: ${command}`);
      }
    }
  };
  const context = {
    window: {LarkAPI: api}, LarkAPI: api, WebSocket: Socket,
    setTimeout: (callback, delay) => {
      if (delay === 250) { delays.push(delay); queueMicrotask(callback); }
      return 0;
    },
    queueMicrotask, Date, console, __readOperations: readOperations, __mutationOperations: mutationOperations
  };
  vm.runInNewContext(source, context);
  await peer.onmessage({data: JSON.stringify({
    id: 'send', operation: 'messaging.send-self-text', params: {
      expectedUserId: 'self', apply, input: {chatId: inputChatId, text: 'self test'}
    }
  })});
  return {reply, prepares, sends, reads, lookups, delays, preparedPayload, sentPayload};
}

test('self send rejects a missing P2P mapping before native write', async () => {
  const result = await execute({selfChatId: null});
  assert.equal(result.reply.ok, false);
  assert.match(result.reply.error, /self-chat/);
  assert.equal(result.lookups, 1);
  assert.equal(result.prepares, 0);
  assert.equal(result.sends, 0);
  assert.equal(result.reads, 0);
});

test('self send rejects a P2P destination mismatch before native write', async () => {
  const result = await execute({selfChatId: '99', inputChatId: '22'});
  assert.equal(result.reply.ok, false);
  assert.match(result.reply.error, /self-chat/);
  assert.equal(result.lookups, 1);
  assert.equal(result.prepares, 0);
  assert.equal(result.sends, 0);
  assert.equal(result.reads, 0);
});

test('self send preview performs no quasi creation, send, or readback', async () => {
  const result = await execute({apply: false});
  assert.equal(result.reply.ok, true);
  assert.equal(result.reply.data.apply, false);
  assert.equal(result.reply.data.operation, 'messaging.send-self-text');
  assert.equal(result.lookups, 1);
  assert.equal(result.prepares, 0);
  assert.equal(result.sends, 0);
  assert.equal(result.reads, 0);
});

test('self send creates the quasi entity, sends only its cid, and verifies native text', async () => {
  const result = await execute();
  assert.equal(result.reply.ok, true);
  assert.equal(result.reply.data.acknowledged, true);
  assert.equal(result.reply.data.verified, true);
  assert.equal(result.reply.data.data.messageId, '11');
  assert.equal(result.lookups, 1);
  assert.equal(result.prepares, 1);
  assert.equal(result.sends, 1);
  assert.equal(result.reads, 1);
  assert.deepEqual(result.delays, []);
  assert.equal(result.preparedPayload.type, 4);
  assert.deepEqual(result.preparedPayload.channel, {id: '22', type: 1});
  assert.equal(result.preparedPayload.content.richText.innerText, 'self test');
  assert.match(result.preparedPayload.cid, /^[0-9a-f-]{36}$/);
  assert.deepEqual(Object.keys(result.sentPayload), ['cid']);
  assert.equal(result.sentPayload.cid, result.preparedPayload.cid);
});

test('self send stops when quasi creation fails and does not retry', async () => {
  const result = await execute({prepareFailure: true});
  assert.equal(result.reply.ok, false);
  assert.match(result.reply.error, /local quasi create failed/);
  assert.equal(result.prepares, 1);
  assert.equal(result.sends, 0);
  assert.equal(result.reads, 0);
});

test('self send stops when the prepared cid differs and does not retry', async () => {
  const result = await execute({preparedCid: 'other-cid'});
  assert.equal(result.reply.ok, false);
  assert.match(result.reply.error, /identity differs/);
  assert.equal(result.prepares, 1);
  assert.equal(result.sends, 0);
  assert.equal(result.reads, 0);
});

test('self send fails without a canonical response message ID and does not resend', async () => {
  const result = await execute({sendMessageId: null});
  assert.equal(result.reply.ok, false);
  assert.match(result.reply.error, /no message identity/);
  assert.equal(result.prepares, 1);
  assert.equal(result.sends, 1);
  assert.equal(result.reads, 0);
});

test('self send makes four bounded verification reads on a mismatch without resending', async () => {
  const result = await execute({readbacks: [message({text: 'different'})]});
  assert.equal(result.reply.ok, false);
  assert.match(result.reply.error, /readback differs/);
  assert.equal(result.prepares, 1);
  assert.equal(result.sends, 1);
  assert.equal(result.reads, 4);
  assert.deepEqual(result.delays, [250, 250, 250]);
});
