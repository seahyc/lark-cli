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

const cardDownload = { cardId: 'card-1', ownerUserId: '7', messageId: 'm1', attachmentIndex: 0, fileName: 'list.xlsx', fileSize: 2048, destinationPath: '/tmp/dl/list.xlsx' };
const cardSnapshot = { messageItems: [{ message: { id: 'm1', attachments: [{ fileName: 'list.xlsx', fileSize: '2048', fileToken: 'file-token-1' }] } }] };

test('card attachment download validates its destination and names the attachment exactly', () => {
  const op = operations['mail.card-attachment-download'];
  assert.deepEqual(op.request(cardDownload), cardDownload);
  for (const destinationPath of ['relative/list.xlsx', '/tmp/dl/', '/tmp/../etc/passwd', '/tmp/./x', '/tmp/dl\0/x']) {
    assert.throws(() => op.request({ ...cardDownload, destinationPath }), /destinationPath/);
  }
  assert.throws(() => op.request({ ...cardDownload, fileToken: 'caller-supplied' }), /unsupported/);
  assert.throws(() => op.request({ ...cardDownload, attachmentIndex: -1 }), /attachmentIndex/);
  assert.throws(() => op.request({ ...cardDownload, fileSize: '2048' }), /fileSize/);
});

test('card attachment download resolves the token from a matching snapshot only', () => {
  const op = operations['mail.card-attachment-download'];
  assert.deepEqual(op.requestFromSnapshot(cardSnapshot, cardDownload, 'user-1'), {
    driveReq: { localPath: '/tmp/dl/list.xlsx', fileToken: 'file-token-1', mountNodePoint: 'user-1', mountPoint: 'email', apiType: 0, bizScene: 8, watermarkScene: 0 },
    messageId: 'm1', forceDownload: false, mailScene: 3,
  });
  assert.throws(() => op.requestFromSnapshot(cardSnapshot, { ...cardDownload, messageId: 'other' }, 'user-1'), /not found/);
  assert.throws(() => op.requestFromSnapshot(cardSnapshot, { ...cardDownload, fileName: 'other.xlsx' }, 'user-1'), /does not match/);
  assert.throws(() => op.requestFromSnapshot(cardSnapshot, { ...cardDownload, fileSize: 1 }, 'user-1'), /does not match/);
  assert.throws(() => op.requestFromSnapshot(cardSnapshot, { ...cardDownload, attachmentIndex: 3 }, 'user-1'), /does not match/);
  assert.throws(() => op.requestFromSnapshot({ messageItems: [{ message: { id: 'm1', attachments: [{ fileName: 'list.xlsx', fileSize: '2048' }] } }] }, cardDownload, 'user-1'), /file token/);
  assert.throws(() => op.requestFromSnapshot(cardSnapshot, cardDownload, ''), /identity/);
});

test('card attachment download projection never echoes a file token', () => {
  const projected = operations['mail.card-attachment-download'].project({ driveResp: { key: 'dl-key' }, path: '' }, cardDownload);
  assert.deepEqual(projected, { messageId: 'm1', attachmentIndex: 0, destinationPath: '/tmp/dl/list.xlsx', key: 'dl-key', path: '', acknowledged: true, verified: false });
  assert.equal(JSON.stringify(projected).includes('file-token-1'), false);
});
