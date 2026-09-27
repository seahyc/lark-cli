'use strict';

const {test} = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const readOperations = require('./modules/outbound/module.js').operations;
const mutationOperations = require('./modules/outbound/mutations.js').operations;
const source = fs.readFileSync(__dirname + '/mail_bridge.js', 'utf8')
  .replace('__LARK_READ_OPERATIONS__', 'globalThis.__readOperations')
  .replace('__LARK_MUTATION_OPERATIONS__', 'globalThis.__mutationOperations')
  .replace('__LARK_LOCAL_SESSION__', JSON.stringify({port: 9331, token: 'test', expiresAt: Date.now() + 60000}));

function message({sender = 'self', chatId = '22', type = 4, text = 'before', richText} = {}) {
  return {
    id: '11', fromId: sender, chatId, type,
    content: {richText: richText || {
      innerText: text, imageIds: [], atIds: [], anchorIds: [], mediaIds: [], docsIds: [],
      elements: {plain: {tag: 1}}
    }}
  };
}

async function execute({before = message(), after = before, readbacks = [after], apply = true} = {}) {
  let peer;
  let writes = 0;
  let reads = 0;
  let reply;
  const delays = [];
  class Socket {
    constructor() { peer = this; }
    send(raw) { reply = JSON.parse(raw); }
    close() {}
  }
  const api = {
    passport: {getUserId: async () => 'self'},
    transport: {
      callSdkApi: async (command, payload) => {
        if (command.startsWith('1|im.v1.MGetMessagesRequest')) {
          const index = reads++;
          const value = index === 0 ? before : readbacks[Math.min(index - 1, readbacks.length - 1)];
          return {data: {entity: {messages: {[payload.messageIds[0]]: value}}}};
        }
        if (command.startsWith('4520|im.v1.EditMessageRequest')) {
          writes += 1;
          return {data: {message: {content: 'not projected'}}};
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
    id: 'edit', operation: 'messaging.edit-text', params: {
      expectedUserId: 'self', apply,
      input: {messageId: '11', chatId: '22', text: 'after', expectedText: 'before'}
    }
  })});
  return {reply, writes, reads, delays};
}

test('edit rejects wrong sender, chat, and TEXT type before native write', async () => {
  for (const before of [
    message({sender: 'other'}),
    message({chatId: 'other-chat'}),
    message({type: 2})
  ]) {
    const result = await execute({before});
    assert.equal(result.reply.ok, false);
    assert.equal(result.writes, 0);
    assert.equal(result.reads, 1);
  }
});

test('edit rejects stale text and rich elements before native write', async () => {
  const stale = await execute({before: message({text: 'changed'})});
  assert.equal(stale.reply.ok, false);
  assert.equal(stale.writes, 0);
  const formatted = await execute({before: message({richText: {
    innerText: 'before', imageIds: [], atIds: [], anchorIds: [], mediaIds: [], docsIds: [],
    elements: {bold: {tag: 7}}
  }})});
  assert.equal(formatted.reply.ok, false);
  assert.equal(formatted.writes, 0);
});

test('edit succeeds only after own-text preflight and matching readback', async () => {
  const result = await execute({after: message({text: 'after'})});
  assert.equal(result.reply.ok, true);
  assert.equal(result.reply.data.verified, true);
  assert.equal(result.writes, 1);
  assert.equal(result.reads, 2);
  assert.deepEqual(result.delays, []);
});

test('edit accepts delayed native readback without a second write', async () => {
  const result = await execute({readbacks: [message({text: 'before'}), message({text: 'after'})]});
  assert.equal(result.reply.ok, true);
  assert.equal(result.reply.data.verified, true);
  assert.equal(result.writes, 1);
  assert.equal(result.reads, 3);
  assert.deepEqual(result.delays, [250]);
});

test('edit readback mismatch makes four verification reads without a retry', async () => {
  const result = await execute({after: message({text: 'different'})});
  assert.equal(result.reply.ok, false);
  assert.match(result.reply.error, /readback differs/);
  assert.equal(result.writes, 1);
  assert.equal(result.reads, 5);
  assert.deepEqual(result.delays, [250, 250, 250]);
});

test('edit reconstructs ordered plain elements when innerText is blank', async () => {
  const graph = ({first, second}) => ({
    innerText: '', elementIds: ['first', 'second'], imageIds: [], atIds: [], anchorIds: [], mediaIds: [], docsIds: [],
    elements: {
      first: {tag: 1, property: {text: {content: first}}},
      second: {tag: 1, property: {text: {content: second}}}
    }
  });
  const result = await execute({
    before: message({richText: graph({first: 'be', second: 'fore'})}),
    after: message({richText: graph({first: 'af', second: 'ter'})})
  });
  assert.equal(result.reply.ok, true);
  assert.equal(result.reply.data.verified, true);
  assert.equal(result.writes, 1);
  assert.equal(result.reads, 2);
});
