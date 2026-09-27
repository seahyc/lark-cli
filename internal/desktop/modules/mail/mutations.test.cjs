'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const { operations } = require('./mutations.js');

test('rule mutations preserve identity-sensitive IDs and reject arbitrary objects', () => {
  assert.deepEqual(operations['mail.rule-enable'].request({ ruleIdString: 'rule-1', isEnable: false }), { ruleIdString: 'rule-1', isEnable: false });
  assert.deepEqual(operations['mail.rule-reorder'].request({ ruleIds: ['rule-b', 'rule-a'] }), { ruleOrder: [{ ruleIdString: 'rule-b', order: '0' }, { ruleIdString: 'rule-a', order: '1' }] });
  assert.throws(() => operations['mail.rule-enable'].request({ ruleIdString: 'rule-1', isEnable: 'false' }), /isEnable/);
  assert.throws(() => operations['mail.rule-reorder'].request({ ruleIds: ['rule-1', 'rule-1'] }), /duplicates/);
  assert.throws(() => operations['mail.rule-enable'].request({ ruleIdString: 'rule-1', isEnable: true, accountId: 'wrong-context' }), /unsupported/);
});

test('signature usage requires explicit account, address, and existing signature identities', () => {
  assert.deepEqual(operations['mail.signature-usage-update'].request({ accountId: 'account-1', address: 'me@example.test', newMailSignatureId: 'new-1', replyMailSignatureId: 'reply-1' }), { accountId: 'account-1', signatureUsage: { address: 'me@example.test', newMailSignatureId: 'new-1', replyMailSignatureId: 'reply-1' } });
  assert.throws(() => operations['mail.signature-usage-update'].request({ accountId: 'account-1', address: 'me@example.test', newMailSignatureId: '', replyMailSignatureId: 'reply-1' }), /newMailSignatureId/);
  assert.deepEqual(operations['mail.rule-enable'].project({ isEnable: true }, { ruleIdString: 'rule-1', isEnable: true }), { ruleIdString: 'rule-1', isEnable: true, acknowledged: true, verified: false });
});

test('text-only signature fixture has no image fields and delete preserves the target identity', () => {
  assert.deepEqual(operations['mail.signature-create-text'].request({ accountId: 'account-1', name: 'fixture', text: '<script src="x">&</script>' }), { accountId: 'account-1', signature: { name: 'fixture', images: [], templateHtml: '<div>&lt;script src=&quot;x&quot;&gt;&amp;&lt;/script&gt;</div>', id: undefined } });
  assert.deepEqual(operations['mail.signature-create-text'].project({ signature: { id: 'sig-1' } }), { id: 'sig-1', acknowledged: true, verified: false });
  assert.deepEqual(operations['mail.signature-delete'].request({ accountId: 'account-1', signatureId: 'sig-1', signatureUsageCount: 0 }), { accountId: 'account-1', signatureId: 'sig-1' });
  assert.throws(() => operations['mail.signature-delete'].request({ accountId: 'account-1', signatureId: 'sig-1', signatureUsageCount: 1 }), /unassigned/);
  assert.throws(() => operations['mail.signature-delete'].request({ accountId: 'account-1', signatureId: 'sig-1', name: 'wrong' }), /unsupported/);
});

test('sender block supports bounded valid email addresses; isAllow writes an allow-list entry', () => {
  assert.deepEqual(operations['mail.sender-block'].request({ address: 'person@example.test' }), { multiFrom: [{ address: 'person@example.test' }], isAllow: false, scene: 1 });
  assert.deepEqual(operations['mail.sender-unblock'].request({ address: 'person@example.test' }), { multiFrom: [{ address: 'person@example.test' }], isAllow: true, scene: 1 });
  assert.throws(() => operations['mail.sender-block'].request({ address: 'not-an-address' }), /valid email/);
  assert.throws(() => operations['mail.sender-block'].request({ address: `x@${'a'.repeat(251)}.test` }), /valid email/);
  assert.deepEqual(operations['mail.sender-block'].project({}, { address: 'person@example.test' }), { address: 'person@example.test', blocked: true, acknowledged: true, verified: false });
  assert.deepEqual(operations['mail.sender-unblock'].project({}, { address: 'person@example.test' }), { address: 'person@example.test', allowListed: true, acknowledged: true, verified: false });
});
