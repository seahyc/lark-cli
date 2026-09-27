'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const mail = require('./modules/mail/module.js').operations;
const mailMutations = require('./modules/mail/mutations.js').operations;
const mailnextMutations = require('./modules/mailnext/mutations.js').operations;

const bridgeSource = fs.readFileSync(__dirname + '/mail_bridge.js', 'utf8')
  .replace('__LARK_LOCAL_SESSION__', JSON.stringify({port: 9330, token: 'test', expiresAt: Date.now() + 60000}));
const readOperations = {...mail};
const mutationOperations = {...mailMutations, ...mailnextMutations};

function start({applyUpdate = true} = {}) {
  const peers = [];
  const timers = [];
  const writes = [];
  let created = false;
  let signature = {
    id: 'signature-1', name: 'Fixture', signatureType: 2, signatureDevice: 1,
    templateHtml: '<div>Before</div>', templateValueJson: '{"B-NAME":"Me"}',
    images: [], nativeExtra: {preserve: true},
  };
  const signatureUsages = [{address: 'me@example.test', newMailSignatureId: 'other-1', replyMailSignatureId: 'other-1'}];
  const api = {
    passport: {getUserId: () => 'user-1'},
    transport: {
      callSdkApi: async (command, payload) => {
        if (command === mail['mail.signatures'].command) {
          return {data: {signatures: created ? [signature] : [], signatureUsages}};
        }
        if (command === mailMutations['mail.signature-create-text'].command) {
          writes.push({command, payload});
          created = true;
          signature = {...signature, name: payload.signature.name, templateHtml: payload.signature.templateHtml, images: payload.signature.images};
          return {data: {signature}};
        }
        if (command === mailnextMutations['mailnext.signature-fixture-update-text'].command) {
          writes.push({command, payload});
          if (applyUpdate) signature = payload.signature;
          return {data: {signature: applyUpdate ? signature : {...signature, templateHtml: '<div>Before</div>'}}};
        }
        if (command === mailMutations['mail.signature-usage-update'].command) {
          throw new Error('signature text fixture update must not alter signature usage');
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
    __LARK_MUTATION_OPERATIONS__: mutationOperations,
    setTimeout(callback, delay) {
      timers.push({callback, delay});
      if (delay === 250) queueMicrotask(callback);
      return timers.length;
    },
  };
  vm.runInNewContext(bridgeSource, context);
  async function invoke(peer, operation, input) {
    await peer.onmessage({data: JSON.stringify({id: `${peers.indexOf(peer)}-${operation}`, operation, params: {expectedUserId: 'user-1', apply: true, input}})});
    return peer.reply;
  }
  function reconnect() {
    const old = peers.at(-1);
    old.onclose();
    const timer = timers.find((entry) => entry.delay === 1000);
    assert.ok(timer, 'bridge should reconnect while retaining session-owned fixture IDs');
    timer.callback();
    return peers.at(-1);
  }
  return {peers, timers, writes, invoke, reconnect, get signature() { return signature; }, signatureUsages};
}

const createInput = {accountId: 'account-1', name: 'Fixture', text: 'Before'};
const updateInput = (overrides = {}) => ({
  accountId: 'account-1', signatureId: 'signature-1', text: 'After <tag>', expectedTemplateHtml: '<div>Before</div>',
  ...overrides,
});

test('session-created signature fixture updates from a fresh raw snapshot and preserves all non-HTML state', async () => {
  const fixture = start();
  const created = await fixture.invoke(fixture.peers[0], 'mail.signature-create-text', createInput);
  assert.equal(created.ok, true);
  assert.equal(created.data.verified, true);

  const updated = await fixture.invoke(fixture.reconnect(), 'mailnext.signature-fixture-update-text', updateInput());
  assert.equal(updated.ok, true);
  assert.equal(updated.data.verified, true);
  assert.equal(fixture.signature.templateHtml, '<div>After &lt;tag&gt;</div>');
  assert.deepEqual(fixture.signature, {
    id: 'signature-1', name: 'Fixture', signatureType: 2, signatureDevice: 1,
    templateHtml: '<div>After &lt;tag&gt;</div>', templateValueJson: '{"B-NAME":"Me"}',
    images: [], nativeExtra: {preserve: true},
  });
  assert.deepEqual(fixture.signatureUsages, [{address: 'me@example.test', newMailSignatureId: 'other-1', replyMailSignatureId: 'other-1'}]);
  assert.deepEqual(fixture.writes.map((call) => call.command), [
    mailMutations['mail.signature-create-text'].command,
    mailnextMutations['mailnext.signature-fixture-update-text'].command,
  ]);
});

test('foreign fixture ID and stale raw HTML reject before an update write', async () => {
  const fixture = start();
  await fixture.invoke(fixture.peers[0], 'mail.signature-create-text', createInput);
  const foreign = await fixture.invoke(fixture.reconnect(), 'mailnext.signature-fixture-update-text', updateInput({signatureId: 'foreign'}));
  assert.equal(foreign.ok, false);
  assert.match(foreign.error, /session.*fixture|Only this session/i);
  assert.equal(fixture.writes.length, 1);

  const stale = await fixture.invoke(fixture.reconnect(), 'mailnext.signature-fixture-update-text', updateInput({expectedTemplateHtml: '<div>stale</div>'}));
  assert.equal(stale.ok, false);
  assert.match(stale.error, /does not match/);
  assert.equal(fixture.writes.length, 1);
});

test('failed signature readback reports an error after one update write without retrying it', async () => {
  const fixture = start({applyUpdate: false});
  await fixture.invoke(fixture.peers[0], 'mail.signature-create-text', createInput);
  const reply = await fixture.invoke(fixture.reconnect(), 'mailnext.signature-fixture-update-text', updateInput());
  assert.equal(reply.ok, false);
  assert.match(reply.error, /signature.*readback|readback.*signature/i);
  assert.deepEqual(fixture.writes.map((call) => call.command), [
    mailMutations['mail.signature-create-text'].command,
    mailnextMutations['mailnext.signature-fixture-update-text'].command,
  ]);
});
