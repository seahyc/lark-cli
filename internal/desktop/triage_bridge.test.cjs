'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const messaging = require('./modules/messaging/module.js').operations;
const outbound = require('./modules/outbound/module.js').operations;
const triage = require('./modules/triage/mutations.js').operations;

const bridgeSource = fs.readFileSync(__dirname + '/mail_bridge.js', 'utf8')
  .replace('__LARK_LOCAL_SESSION__', JSON.stringify({port: 9330, token: 'test', expiresAt: Date.now() + 60000}));
const readOperations = {...messaging, ...outbound};

function favorite(id, messageId) {
  return {id, content: {messageId}};
}

function favoriteList(ids, entries, hasMore = false) {
  return {favoritesIds: ids, entity: {favorites: entries}, hasMore};
}

function start({before, after, message, selfChatId = '100'} = {}) {
  const peers = [];
  const timers = [];
  const writes = [];
  let favoriteCalls = 0;
  const ownMessage = message === undefined
    ? {fromId: 'user-1', chatId: '100', type: 4, content: {richText: {innerText: 'fixture'}}}
    : message;
  const api = {
    passport: {getUserId: () => 'user-1'},
    transport: {
      callSdkApi: async (command, payload) => {
        if (command === messaging['messaging.favorites'].command) {
          favoriteCalls += 1;
          return {data: writes.length ? after() : before()};
        }
        if (command === outbound['messaging.text-message'].command) {
          return {data: {entity: {messages: {'200': ownMessage}}}};
        }
        if (command === messaging['messaging.p2p-chat'].command) {
          return {data: {chatterId2chat: {'user-1': {id: selfChatId}}}};
        }
        if (command === triage['triage.favorite-add'].command) {
          writes.push({command, payload});
          return {data: {}};
        }
        if (command === triage['triage.favorite-delete'].command) {
          writes.push({command, payload});
          return {data: {}};
        }
        throw new Error(`unexpected command ${command}`);
      },
    },
  };
  class Socket {
    constructor() { peers.push(this); }
    send(raw) { this.reply = JSON.parse(raw); }
    close() {}
  }
  const context = {
    window: {LarkAPI: api}, LarkAPI: api, WebSocket: Socket, Date, console,
    __LARK_READ_OPERATIONS__: readOperations,
    __LARK_MUTATION_OPERATIONS__: triage,
    setTimeout(callback, delay) {
      timers.push({callback, delay});
      if (delay === 250) queueMicrotask(callback);
      return timers.length;
    },
  };
  vm.runInNewContext(bridgeSource, context);
  async function invoke(operation, input) {
    const peer = peers[0];
    await peer.onmessage({data: JSON.stringify({id: operation, operation, params: {expectedUserId: 'user-1', apply: true, input}})});
    return peer.reply;
  }
  return {invoke, writes, timers, get favoriteCalls() { return favoriteCalls; }};
}

const existing = () => ({'old-1': favorite('old-1', '199')});
const addInput = {messageId: '200', chatId: '100'};

test('favorite add accepts only an own self-chat TEXT and verifies exactly one new favorite for that message', async () => {
  const fixture = start({
    before: () => favoriteList(['old-1'], existing()),
    after: () => favoriteList(['old-1', 'new-1'], {...existing(), 'new-1': favorite('new-1', '200')}),
  });
  const reply = await fixture.invoke('triage.favorite-add', addInput);
  assert.equal(reply.ok, true);
  assert.deepEqual(reply.data, {userId: 'user-1', acknowledged: true, verified: true, data: {favoriteId: 'new-1'}});
  assert.deepEqual(fixture.writes, [{command: triage['triage.favorite-add'].command, payload: {favs: [{id: '200', type: 1, chatId: '100', originMergeForwardId: ''}]}}]);
  assert.equal(fixture.favoriteCalls, 2);
  assert.deepEqual(fixture.timers.filter(timer => timer.delay === 250), []);
});

test('favorite add rejects wrong sender, wrong chat, non-self mapping, and preexisting favorite before write', async () => {
  const cases = [
    {name: 'wrong sender', message: {fromId: 'other', chatId: '100', type: 4}, selfChatId: '100'},
    {name: 'wrong chat', message: {fromId: 'user-1', chatId: '101', type: 4}, selfChatId: '100'},
    {name: 'non-self mapping', message: {fromId: 'user-1', chatId: '100', type: 4}, selfChatId: '101'},
    {name: 'preexisting favorite', preexisting: true},
  ];
  for (const item of cases) {
    const entries = item.preexisting ? {...existing(), 'old-2': favorite('old-2', '200')} : existing();
    const fixture = start({
      before: () => favoriteList(Object.keys(entries), entries),
      after: () => favoriteList(Object.keys(entries), entries),
      message: item.message,
      selfChatId: item.selfChatId,
    });
    const reply = await fixture.invoke('triage.favorite-add', addInput);
    assert.equal(reply.ok, false, item.name);
    assert.deepEqual(fixture.writes, [], item.name);
    assert.deepEqual(fixture.timers.filter(timer => timer.delay === 250), [], item.name);
  }
});

test('favorite mutations reject a partial favorite list before write', async () => {
  const entries = existing();
  const fixture = start({
    before: () => favoriteList(['old-1'], entries, true),
    after: () => favoriteList(['old-1'], entries, true),
  });
  const reply = await fixture.invoke('triage.favorite-add', addInput);
  assert.equal(reply.ok, false);
  assert.match(reply.error, /complete bounded list/);
  assert.deepEqual(fixture.writes, []);
});

test('favorite delete removes the exact ID and verifies all other favorites remain', async () => {
  const beforeEntries = {...existing(), 'fixture-1': favorite('fixture-1', '200')};
  const fixture = start({
    before: () => favoriteList(['old-1', 'fixture-1'], beforeEntries),
    after: () => favoriteList(['old-1'], existing()),
  });
  const reply = await fixture.invoke('triage.favorite-delete', {favoriteId: 'fixture-1'});
  assert.equal(reply.ok, true);
  assert.deepEqual(reply.data, {userId: 'user-1', acknowledged: true, verified: true, data: {favoriteId: 'fixture-1'}});
  assert.deepEqual(fixture.writes, [{command: triage['triage.favorite-delete'].command, payload: {ids: ['fixture-1']}}]);
  assert.equal(fixture.favoriteCalls, 2);
});

test('changed or unmatched favorite readback errors after one write and never retries the write', async () => {
  const addFixture = start({
    before: () => favoriteList(['old-1'], existing()),
    after: () => favoriteList(['old-1', 'new-1', 'other-1'], {...existing(), 'new-1': favorite('new-1', '200'), 'other-1': favorite('other-1', '201')}),
  });
  const addReply = await addFixture.invoke('triage.favorite-add', addInput);
  assert.equal(addReply.ok, false);
  assert.match(addReply.error, /Favorite readback differs/);
  assert.equal(addFixture.writes.length, 1);
  assert.deepEqual(addFixture.timers.filter(timer => timer.delay === 250).map(timer => timer.delay), [250, 250, 250]);

  const beforeEntries = {...existing(), 'fixture-1': favorite('fixture-1', '200')};
  const deleteFixture = start({
    before: () => favoriteList(['old-1', 'fixture-1'], beforeEntries),
    after: () => favoriteList(['fixture-1'], {'fixture-1': favorite('fixture-1', '200')}),
  });
  const deleteReply = await deleteFixture.invoke('triage.favorite-delete', {favoriteId: 'fixture-1'});
  assert.equal(deleteReply.ok, false);
  assert.match(deleteReply.error, /Favorite readback differs/);
  assert.equal(deleteFixture.writes.length, 1);
  assert.deepEqual(deleteFixture.timers.filter(timer => timer.delay === 250).map(timer => timer.delay), [250, 250, 250]);
});
