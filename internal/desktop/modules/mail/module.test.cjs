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

test('forward-card reads send the shipped request shape and reject unknown fields', () => {
  assert.deepEqual(operations['mail.forward-card-messages'].request({ cardId: 'c', ownerUserId: '7' }), { cardId: 'c', ownerUserId: '7', msgNeedClipped: false });
  assert.deepEqual(operations['mail.forward-card-messages'].request({ cardId: 'c', ownerUserId: '7', cardSenderUserId: '8', includeBody: true }), { cardId: 'c', ownerUserId: '7', msgNeedClipped: false, cardSenderUserId: '8' });
  // The network variant has no cardSenderUserId in the shipped wrapper.
  assert.deepEqual(operations['mail.forward-card-messages-net'].request({ cardId: 'c', ownerUserId: '7' }), { cardId: 'c', ownerUserId: '7', msgNeedClipped: false });
  assert.throws(() => operations['mail.forward-card-messages-net'].request({ cardId: 'c', ownerUserId: '7', cardSenderUserId: '8' }), /unsupported/);
  assert.throws(() => operations['mail.forward-card-messages'].request({ cardId: '', ownerUserId: '7' }), /cardId/);
  assert.throws(() => operations['mail.forward-card-messages'].request({ cardId: 'c', ownerUserId: '7', includeBody: 'yes' }), /includeBody/);
  assert.throws(() => operations['mail.forward-card-messages'].request({ cardId: 'c', ownerUserId: '7', threadId: 't' }), /unsupported/);
});

test('forward-card projection returns attachment metadata but never file tokens', () => {
  const response = {
    notInDatabase: false,
    isLastPage: true,
    subject: 'RE: Project X',
    messageItems: [{ message: {
      id: 'm1', subject: 'RE: Project X', createdTimestamp: '1790000000000',
      from: { name: 'Sender', address: 's@example.test' },
      to: [{ name: 'Rcpt', address: 'r@example.test' }], cc: [],
      bodySummary: 'summary', bodyHtml: '<p>full body</p>',
      attachments: [{ fileName: 'list.xlsx', fileSize: '2048', fileToken: 'secret-token', type: 1, expireTime: '0', fileUrl: 'https://example.test/secret-url' }],
      images: [{}],
    } }],
  };
  const summary = operations['mail.forward-card-messages'].project(response, {});
  assert.equal(summary.messages[0].from.address, 's@example.test');
  assert.deepEqual(summary.messages[0].attachments, [{ index: 0, fileName: 'list.xlsx', fileSize: 2048, type: 1, expireTime: '0', hasToken: true }]);
  assert.equal(summary.messages[0].bodyHtml, undefined);
  assert.equal(summary.messages[0].hasBodyHtml, true);
  assert.equal(summary.messages[0].imageCount, 1);
  assert.equal(JSON.stringify(summary).includes('secret-token'), false);
  assert.equal(JSON.stringify(summary).includes('secret-url'), false);
  const full = operations['mail.forward-card-messages'].project(response, { includeBody: true });
  assert.equal(full.messages[0].bodyHtml, '<p>full body</p>');
  assert.equal(full.messages[0].bodyHtmlTruncated, false);
});

test('forward-card projection is bounded and tolerates an empty or missing list', () => {
  assert.deepEqual(operations['mail.forward-card-messages'].project({ notInDatabase: true }, {}), { notInDatabase: true, isLastPage: null, subject: '', messageCount: 0, messagesTruncated: false, messages: [] });
  const many = { messageItems: Array.from({ length: 60 }, (_, i) => ({ message: { id: `m${i}`, attachments: Array.from({ length: 120 }, () => ({ fileName: 'a', fileSize: '1', fileToken: 't' })) } })) };
  const bounded = operations['mail.forward-card-messages'].project(many, {});
  assert.equal(bounded.messages.length, 50);
  assert.equal(bounded.messagesTruncated, true);
  assert.equal(bounded.messageCount, 60);
  assert.equal(bounded.messages[0].attachments.length, 100);
  assert.equal(bounded.messages[0].attachmentsTruncated, true);
  const huge = { messageItems: [{ message: { id: 'm', bodyHtml: 'x'.repeat(250000) } }] };
  const capped = operations['mail.forward-card-messages'].project(huge, { includeBody: true });
  assert.equal(capped.messages[0].bodyHtml.length, 200000);
  assert.equal(capped.messages[0].bodyHtmlTruncated, true);
});
