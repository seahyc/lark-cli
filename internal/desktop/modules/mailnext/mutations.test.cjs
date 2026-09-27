'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const { operations } = require('./mutations.js');

test('blank fixture create exactly matches the active normal-compose defaults', () => {
  assert.deepEqual(operations['mailnext.draft-fixture-create'].request({}), { threadId: '', originMessageId: '', action: 0, needSignature: false, timeText: '', isFromLarkPlaintext: false });
  assert.throws(() => operations['mailnext.draft-fixture-create'].request({ subject: 'not allowed' }), /unsupported/);
  assert.deepEqual(operations['mailnext.draft-fixture-create'].project({ draft: { id: 'draft-1', threadId: 'thread-1', replyMessageId: '', attachments: [] } }), { draftId: 'draft-1', threadId: 'thread-1', replyMessageId: '', attachmentCount: 0, acknowledged: true, verified: false });
});

test('fixture deletion derives the draft-id wire request from a fresh empty fixture snapshot', () => {
  const raw = { draft: { id: 'draft-1', threadId: 'thread-1', replyMessageId: '', subject: '', bodyHtml: '', from: { address: 'me@example.test' }, to: [], cc: [], bcc: [], images: [], attachments: [], docsPermissions: [], calendarEvent: null, priorityType: 3, coverInfo: '', bodySummary: '' } };
  const remove = operations['mailnext.draft-fixture-delete'];
  const input = { threadId: 'thread-1', messageId: 'draft-1', expectedSubject: '', expectedBodyHtml: '' };
  assert.deepEqual(remove.requestFromSnapshot(raw, input), { threadId: 'thread-1', messageId: 'draft-1' });
  assert.throws(() => remove.request({}), /requestFromSnapshot/);
  assert.throws(() => remove.requestFromSnapshot(raw, { ...input, expectedSubject: 'stale' }), /does not match/);
  assert.throws(() => remove.requestFromSnapshot(raw, { ...input, messageId: 'foreign' }), /identity/);
  assert.throws(() => remove.requestFromSnapshot({ draft: { ...raw.draft, to: [{ address: 'recipient@example.test' }] } }, input), /no to/);
  assert.throws(() => remove.requestFromSnapshot({ draft: { ...raw.draft, attachments: [{ fileToken: 'token' }] } }, input), /attachments/);
  assert.deepEqual(remove.project({}, input), { threadId: 'thread-1', messageId: 'draft-1', acknowledged: true, verified: false });
});

test('fixture update builds H5-compatible payload from a fresh blank native snapshot', () => {
  const raw = {
    draft: {
      id: 'draft-1', threadId: 'thread-1', replyMessageId: '', subject: '', bodyHtml: '',
      from: { name: 'Me', address: 'me@example.test', larkEntityType: 1, larkEntityIdString: 'user-1', operator: { userId: 'user-1', name: 'Me', address: 'me@example.test', displayName: 'Me' }, mailGroupType: 0, tenantId: 'tenant-1', displayName: 'Me' },
      to: [], cc: [], bcc: [], images: [], attachments: [], docsPermissions: [], priorityType: 3,
      coverInfo: '', calendarEvent: null, isSendSeparately: false, needReadReceipt: false, bodySummary: '', isFromLarkPlaintext: false,
    },
  };
  assert.deepEqual(operations['mailnext.draft-fixture-update'].requestFromSnapshot(raw, {
    draftId: 'draft-1', threadId: 'thread-1', replyMessageId: '', subject: 'Fixture', bodyHtml: '<p>Hello <strong>world</strong><br></p>', expectedSubject: '', expectedBodyHtml: '',
  }), {
    draftId: 'draft-1',
    payload: {
      from: { name: 'Me', address: 'me@example.test', larkEntityType: 1, larkEntityIdString: 'user-1', operator: { userId: 'user-1', name: 'Me', address: 'me@example.test', displayName: 'Me' }, mailGroupType: 0, tenantId: 'tenant-1', displayName: 'Me' },
      to: [], cc: [], bcc: [], bodyHtml: '<p>Hello <strong>world</strong><br></p>', subject: 'Fixture', images: [], attachments: [], docsPermissions: [], threadId: 'thread-1', isSendSeparately: false, priorityType: 3, coverInfo: '', calendarEvent: undefined, needReadReceipt: false, bodySummary: '', isFromLarkPlaintext: false,
    },
    isDelay: false,
    onlySaveLocal: false,
  });
  assert.deepEqual(operations['mailnext.draft-fixture-update'].project({ draft: { id: 'draft-1', threadId: 'thread-1', subject: 'Fixture', bodyHtml: '<p>Hello</p>' } }, { draftId: 'draft-1', threadId: 'thread-1', replyMessageId: '', subject: 'Fixture', bodyHtml: '<p>Hello</p>', expectedSubject: '', expectedBodyHtml: '' }), { draftId: 'draft-1', threadId: 'thread-1', subject: 'Fixture', bodyHtmlLength: 12, acknowledged: true, verified: false });
});

test('fixture update rejects stale identity, rich-resource HTML, and any foreign draft state', () => {
  const input = { draftId: 'draft-1', threadId: 'thread-1', replyMessageId: '', subject: '', bodyHtml: '', expectedSubject: '', expectedBodyHtml: '' };
  const base = { draft: { id: 'draft-1', threadId: 'thread-1', replyMessageId: '', subject: '', bodyHtml: '', from: { address: 'me@example.test' }, to: [], cc: [], bcc: [], images: [], attachments: [], docsPermissions: [], calendarEvent: null, priorityType: 3, coverInfo: '', bodySummary: '' } };
  assert.throws(() => operations['mailnext.draft-fixture-update'].requestFromSnapshot(base, { ...input, bodyHtml: '<img src=x>' }), /permits only/);
  assert.throws(() => operations['mailnext.draft-fixture-update'].requestFromSnapshot({ draft: { ...base.draft, attachments: [{ fileToken: 'secret' }] } }, input), /attachments/);
  assert.throws(() => operations['mailnext.draft-fixture-update'].requestFromSnapshot(base, { ...input, expectedSubject: 'stale' }), /does not match/);
  assert.throws(() => operations['mailnext.draft-fixture-update'].requestFromSnapshot(base, { ...input, draftId: 'other' }), /identity/);
  assert.throws(() => operations['mailnext.draft-fixture-update'].request({}), /requestFromSnapshot/);
});

test('signature fixture text update changes only escaped HTML from one fresh complete signature snapshot', () => {
  const raw = {
    signatures: [{
      id: 'signature-1', name: 'Fixture', signatureType: 2, signatureDevice: 1,
      templateHtml: '<div>Before</div>', templateValueJson: '{"B-NAME":"Me"}',
      images: [{fileToken: 'preserved-native-token', imageName: 'logo.png', cid: 'logo'}],
      nativeExtra: {keep: 'exactly'},
    }],
  };
  const input = {accountId: 'account-1', signatureId: 'signature-1', text: 'After <script>', expectedTemplateHtml: '<div>Before</div>'};
  assert.deepEqual(operations['mailnext.signature-fixture-update-text'].requestFromSnapshot(raw, input), {
    accountId: 'account-1',
    signature: {
      id: 'signature-1', name: 'Fixture', signatureType: 2, signatureDevice: 1,
      templateHtml: '<div>After &lt;script&gt;</div>', templateValueJson: '{"B-NAME":"Me"}',
      images: [{fileToken: 'preserved-native-token', imageName: 'logo.png', cid: 'logo'}],
      nativeExtra: {keep: 'exactly'},
    },
  });
  assert.deepEqual(operations['mailnext.signature-fixture-update-text'].project({}, input), {
    signatureId: 'signature-1', templateHtmlLength: 31, acknowledged: true, verified: false,
  });
});

test('signature fixture update rejects missing, duplicate, stale, and untrusted snapshots', () => {
  const base = {
    id: 'signature-1', name: 'Fixture', signatureType: 2, signatureDevice: 1,
    templateHtml: '<div>Before</div>', templateValueJson: '{}', images: [],
  };
  const input = {accountId: 'account-1', signatureId: 'signature-1', text: 'After', expectedTemplateHtml: '<div>Before</div>'};
  const update = operations['mailnext.signature-fixture-update-text'];
  assert.throws(() => update.request({}), /requestFromSnapshot/);
  assert.throws(() => update.requestFromSnapshot({signatures: []}, input), /exactly one/);
  assert.throws(() => update.requestFromSnapshot({signatures: [base, {...base}]}, input), /exactly one/);
  assert.throws(() => update.requestFromSnapshot({signatures: [{...base, templateHtml: '<div>Changed</div>'}]}, input), /does not match/);
  assert.throws(() => update.requestFromSnapshot({signatures: [{...base, images: undefined}]}, input), /required native fields/);
  assert.throws(() => update.requestFromSnapshot({signatures: [base]}, {...input, text: ''}), /non-empty/);
});
