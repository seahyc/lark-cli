'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const mail = require('./modules/mail/module.js').operations;
const mailnext = require('./modules/mailnext/module.js').operations;
const mutations = require('./modules/mailnext/mutations.js').operations;

const bridgeSource = fs.readFileSync(__dirname + '/mail_bridge.js', 'utf8')
  .replace('__LARK_LOCAL_SESSION__', JSON.stringify({ port: 9330, token: 'test', expiresAt: Date.now() + 60000 }));
const readOperations = { ...mail, ...mailnext };
const mutationOperations = { ...mutations };

function blankDraft(input = {}) {
  const { id = 'draft-1', threadId = 'thread-1', replyMessageId = '', subject = '', bodyHtml = '<br/>' } = input;
  return {
    id, threadId, replyMessageId, subject, bodyHtml,
    from: { name: 'Me', address: 'me@example.test', larkEntityType: 1, larkEntityIdString: 'user-1', operator: { userId: 'user-1', name: 'Me', address: 'me@example.test', displayName: 'Me' }, mailGroupType: 0, tenantId: 'tenant-1', displayName: 'Me' },
    to: [], cc: [], bcc: [], images: [], attachments: [], docsPermissions: [],
    priorityType: 3, coverInfo: '', calendarEvent: null, isSendSeparately: false,
    needReadReceipt: false, bodySummary: '', isFromLarkPlaintext: false,
    ...input,
  };
}

function start({ selected = true, shared = false, createIdentity = true, deleteReadback = 'absent', deleteWriteError = null, draftOverrides = {} } = {}) {
  const peers = [];
  const timers = [];
  const writes = [];
  const reads = [];
  let draft = blankDraft(draftOverrides);
  let deleteWritten = false;
  const api = {
    passport: { getUserId: () => 'user-1' },
    transport: {
      callSdkApi: async (command, payload) => {
        if (command === mail['mail.account-metadata'].command) {
          return { data: { account: { mailAccountId: 'user-1', accountSelected: { isSelected: selected }, isShared: shared } } };
        }
        if (command === mutations['mailnext.draft-fixture-create'].command) {
          writes.push({ command, payload });
          draft = blankDraft(createIdentity ? draftOverrides : { ...draftOverrides, id: '', threadId: '' });
          return { data: { draft } };
        }
        if (command === mailnext['mailnext.draft-fixture-metadata'].command) {
          reads.push(payload);
          if (deleteWritten && deleteReadback === 'NotFound') throw new Error('NotFound');
          if (deleteWritten && deleteReadback === 'Normal(404)') throw new Error('Normal(404)');
          if (deleteWritten && deleteReadback === 'absent') return { data: { draft: undefined } };
          return { data: { draft } };
        }
        if (command === mutations['mailnext.draft-fixture-update'].command) {
          writes.push({ command, payload });
          draft = { ...draft, ...payload.payload, id: draft.id, replyMessageId: draft.replyMessageId };
          return { data: { draft } };
        }
        if (command === mutations['mailnext.draft-fixture-delete'].command) {
          writes.push({ command, payload });
          if (deleteWriteError) throw deleteWriteError;
          deleteWritten = true;
          return { data: {} };
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
    window: { LarkAPI: api }, LarkAPI: api, WebSocket: Socket, Date, console,
    __LARK_READ_OPERATIONS__: readOperations,
    __LARK_MUTATION_OPERATIONS__: mutationOperations,
    setTimeout(callback, delay) {
      timers.push({ callback, delay });
      if (delay === 250) queueMicrotask(callback);
      return timers.length;
    },
  };
  vm.runInNewContext(bridgeSource, context);
  async function invoke(peer, operation, input) {
    await peer.onmessage({ data: JSON.stringify({ id: `${peers.indexOf(peer)}-${operation}`, operation, params: { expectedUserId: 'user-1', apply: true, input } }) });
    return peer.reply;
  }
  function reconnect() {
    const old = peers.at(-1);
    old.onclose();
    const reconnectTimer = timers.find(timer => timer.delay === 1000);
    assert.ok(reconnectTimer, 'bridge should schedule reconnect');
    reconnectTimer.callback();
    return peers.at(-1);
  }
  return { peers, timers, writes, reads, get draft() { return draft; }, invoke, reconnect };
}

const updateInput = (overrides = {}) => ({
  draftId: 'draft-1', threadId: 'thread-1', replyMessageId: '',
  expectedSubject: '', expectedBodyHtml: '<br/>', subject: 'Fixture subject', bodyHtml: '<p>Fixture <strong>body</strong></p>',
  ...overrides,
});

test('selected mailbox mismatch rejects mailnext fixture before any native write', async () => {
  const fixture = start({ selected: false });
  const reply = await fixture.invoke(fixture.peers[0], 'mailnext.draft-fixture-create', {});
  assert.equal(reply.ok, false);
  assert.match(reply.error, /selected primary mailbox/);
  assert.deepEqual(fixture.writes, []);
});

test('same-session create is read back, then a fresh fixture update is verified without write retry', async () => {
  const fixture = start();
  const created = await fixture.invoke(fixture.peers[0], 'mailnext.draft-fixture-create', {});
  assert.equal(created.ok, true);
  assert.equal(created.data.verified, true);
  assert.deepEqual(fixture.writes.map(call => call.command), [mutations['mailnext.draft-fixture-create'].command]);
  assert.deepEqual(fixture.reads.map(value => JSON.parse(JSON.stringify(value))), [{ draftId: 'draft-1' }]);

  const peer = fixture.reconnect();
  const updated = await fixture.invoke(peer, 'mailnext.draft-fixture-update', updateInput());
  assert.equal(updated.ok, true);
  assert.equal(updated.data.verified, true);
  assert.equal(fixture.draft.subject, 'Fixture subject');
  assert.equal(fixture.draft.bodyHtml, '<p>Fixture <strong>body</strong></p>');
  assert.deepEqual(fixture.writes.map(call => call.command), [
    mutations['mailnext.draft-fixture-create'].command,
    mutations['mailnext.draft-fixture-update'].command,
  ]);
  assert.deepEqual(fixture.reads.map(value => JSON.parse(JSON.stringify(value))), [
    { draftId: 'draft-1' }, { draftId: 'draft-1' }, { draftId: 'draft-1' },
  ]);
  assert.deepEqual(fixture.timers.filter(timer => timer.delay === 250), []);
});

test('stale expected state fails before update and does not queue a retry', async () => {
  const fixture = start();
  await fixture.invoke(fixture.peers[0], 'mailnext.draft-fixture-create', {});
  const reply = await fixture.invoke(fixture.reconnect(), 'mailnext.draft-fixture-update', updateInput({ expectedSubject: 'stale' }));
  assert.equal(reply.ok, false);
  assert.match(reply.error, /does not match/);
  assert.deepEqual(fixture.writes.map(call => call.command), [mutations['mailnext.draft-fixture-create'].command]);
  assert.deepEqual(fixture.timers.filter(timer => timer.delay === 250), []);
});

test('foreign draft ID and missing create identity cannot reach update', async () => {
  const owned = start();
  await owned.invoke(owned.peers[0], 'mailnext.draft-fixture-create', {});
  const foreign = await owned.invoke(owned.reconnect(), 'mailnext.draft-fixture-update', updateInput({ draftId: 'foreign' }));
  assert.equal(foreign.ok, false);
  assert.match(foreign.error, /Only this session/);
  assert.deepEqual(owned.writes.map(call => call.command), [mutations['mailnext.draft-fixture-create'].command]);

  const noIdentity = start({ createIdentity: false });
  const created = await noIdentity.invoke(noIdentity.peers[0], 'mailnext.draft-fixture-create', {});
  assert.equal(created.ok, false);
  assert.match(created.error, /no identity/);
  const update = await noIdentity.invoke(noIdentity.reconnect(), 'mailnext.draft-fixture-update', updateInput());
  assert.equal(update.ok, false);
  assert.match(update.error, /Only this session/);
  assert.deepEqual(noIdentity.writes.map(call => call.command), [mutations['mailnext.draft-fixture-create'].command]);
});


const deleteInput = (overrides = {}) => ({
  threadId: 'thread-1', messageId: 'draft-1', expectedSubject: '', expectedBodyHtml: '<br/>',
  ...overrides,
});

test('restart-safe draft deletion uses a fresh 3612 snapshot and the native draft ID', async () => {
  const fixture = start();
  const deleted = await fixture.invoke(fixture.peers[0], 'mailnext.draft-fixture-delete', deleteInput());
  assert.equal(deleted.ok, true);
  assert.equal(deleted.data.verified, true);
  assert.deepEqual(fixture.writes.map(call => ({ command: call.command, payload: JSON.parse(JSON.stringify(call.payload)) })), [{
    command: mutations['mailnext.draft-fixture-delete'].command,
    payload: { threadId: 'thread-1', messageId: 'draft-1' },
  }]);
  assert.deepEqual(fixture.reads.map(value => JSON.parse(JSON.stringify(value))), [{ draftId: 'draft-1' }, { draftId: 'draft-1' }]);
});

test('draft deletion verifies documented NotFound variants after the acknowledged write', async () => {
  for (const deleteReadback of ['NotFound', 'Normal(404)']) {
    const fixture = start({ deleteReadback });
    const deleted = await fixture.invoke(fixture.peers[0], 'mailnext.draft-fixture-delete', deleteInput());
    assert.equal(deleted.ok, true, deleteReadback);
    assert.equal(deleted.data.verified, true, deleteReadback);
    assert.equal(fixture.writes.length, 1, deleteReadback);
  }
});

test('delete write failure is never accepted as absence', async () => {
  const fixture = start({ deleteWriteError: new Error('native delete failed') });
  const reply = await fixture.invoke(fixture.peers[0], 'mailnext.draft-fixture-delete', deleteInput());
  assert.equal(reply.ok, false);
  assert.match(reply.error, /native delete failed/);
  assert.equal(fixture.writes.length, 1);
  assert.equal(fixture.reads.length, 1, 'only the required before snapshot is read');
});

test('delete rejects stale or recipient-bearing fresh snapshots before the native write', async () => {
  const stale = start();
  const staleReply = await stale.invoke(stale.peers[0], 'mailnext.draft-fixture-delete', deleteInput({ expectedSubject: 'stale' }));
  assert.equal(staleReply.ok, false);
  assert.match(staleReply.error, /snapshot subject or body does not match/);
  assert.deepEqual(stale.writes, []);

  const recipient = start({ draftOverrides: { to: [{ address: 'other@example.test' }] } });
  const recipientReply = await recipient.invoke(recipient.peers[0], 'mailnext.draft-fixture-delete', deleteInput());
  assert.equal(recipientReply.ok, false);
  assert.match(recipientReply.error, /no to/);
  assert.deepEqual(recipient.writes, []);
});

test('present empty fixture after all four delete polls fails closed without retrying the write', async () => {
  const fixture = start({ deleteReadback: 'present' });
  const reply = await fixture.invoke(fixture.peers[0], 'mailnext.draft-fixture-delete', deleteInput());
  assert.equal(reply.ok, false);
  assert.match(reply.error, /Deleted draft still exists/);
  assert.equal(fixture.writes.length, 1);
  assert.equal(fixture.reads.length, 5, 'one fresh snapshot plus four readback polls');
  assert.equal(fixture.timers.filter(timer => timer.delay === 250).length, 3);
});
