'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const { operations } = require('./module.js');

test('requests match shipped wrapper transforms and reject unrecognized input', () => {
  assert.deepEqual(operations['mail.signatures'].request({ accountId: 'a' }), { accountId: 'a', fromSetting: false });
  assert.deepEqual(operations['mail.blocked-sender'].request({ address: 'x@example.test' }), { addresses: ['x@example.test'] });
  assert.deepEqual(operations['mail.schedule-status'].request({}), {});
  assert.deepEqual(operations['mail.account-metadata'].request({}), { fetchDb: false, fetchCurrentAccount: false });
  assert.deepEqual(operations['mail.verified-auto-transfer-emails'].request({ accountId: 'a' }), { accountId: 'a' });
  assert.deepEqual(operations['mail.global-forwarding'].request({ accountId: 'a' }), { version: '2', isGlobalTransfer: true, accountId: 'a' });
  assert.deepEqual(operations['mail.draft-item'].request({ threadId: 't', messageId: 'm' }), { threadId: 't', messageId: 'm' });
  assert.throws(() => operations['mail.signatures'].request({ accountId: '' }), /accountId/);
  assert.throws(() => operations['mail.schedule-status'].request({ accountId: 'a' }), /unsupported/);
  assert.throws(() => operations['mail.blocked-sender'].request({ address: 1 }), /address/);
});

test('projections retain bounded relevant mail data', () => {
  assert.deepEqual(operations['mail.blocked-sender'].project({ blockedAddresses: [] }, { address: 'x@example.test' }), { blocked: false });
  assert.deepEqual(operations['mail.blocked-sender'].project({ blockedAddresses: ['other@example.test'] }, { address: 'x@example.test' }), { blocked: false });
  assert.deepEqual(operations['mail.blocked-sender'].project({ blockedAddresses: ['x@example.test'] }, { address: 'x@example.test' }), { blocked: true });
  assert.deepEqual(operations['mail.schedule-status'].project({ scheduleSendMessageCount: '3' }), { count: 3 });
  assert.deepEqual(operations['mail.signatures'].project({ signatures: [{ id: 's', name: 'Work', templateHtml: '<secret>' }], signatureUsages: [] }), { signatures: [{ id: 's', name: 'Work', signatureType: undefined, signatureDevice: undefined }], signaturesTruncated: false, signatureUsages: [], signatureUsagesTruncated: false, optionalSignatureMap: {} });
  assert.deepEqual(operations['mail.global-forwarding'].project({ adminEnableAutoTransfer: true, rules: [{ ruleIdString: 'r', isEnable: true, action: { items: [{ type: 12, input: 'to@example.test', authStatus: 2, enableAutoTransfer: true }, { type: 1 }] } }] }), { adminEnableAutoTransfer: true, rules: [{ ruleIdString: 'r', isEnable: true, actions: [{ type: 12, input: 'to@example.test', authStatus: 2, enableAutoTransfer: true }] }] });
  const accounts = operations['mail.account-metadata'].project({ account: { accountAddress: 'a@example.test', accountToken: 'must-not-leak', mailAccountId: 'id', mailSetting: { vacationResponder: { autoReplyBody: 'must-not-leak' }, undoSendEnable: true } } });
  assert.equal(accounts.accounts[0].accountAddress, 'a@example.test');
  assert.equal(JSON.stringify(accounts).includes('must-not-leak'), false);
  assert.deepEqual(operations['mail.draft-item'].project({ draft: { id: 'd', threadId: 't', subject: 'private subject', bodyHtml: '<p>body</p>', images: [{}], attachments: [{}], lastUpdatedTimestamp: '9' } }), { id: 'd', threadId: 't', replyMessageId: '', subject: 'private subject', lastUpdatedTimestamp: 9, createdTimestamp: 0, attachmentCount: 1, imageCount: 1, hasBody: true });
});
