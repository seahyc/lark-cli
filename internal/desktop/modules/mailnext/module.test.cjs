'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const { operations } = require('./module.js');

test('fixture metadata read preserves its identity and rejects arbitrary selectors', () => {
  assert.deepEqual(operations['mailnext.draft-fixture-metadata'].request({ draftId: 'draft-1' }), { draftId: 'draft-1' });
  assert.throws(() => operations['mailnext.draft-fixture-metadata'].request({ draftId: '' }), /draftId/);
  assert.throws(() => operations['mailnext.draft-fixture-metadata'].request({ threadId: 'thread-1', messageId: '', raw: true }), /unsupported/);
});

test('fixture metadata excludes body, recipients, and attachment file tokens', () => {
  const output = operations['mailnext.draft-fixture-metadata'].project({
    draft: { id: 'draft-1', threadId: 'thread-1', replyMessageId: '', bodyHtml: '<private>', to: [{ address: 'private@example.test' }], attachments: [{ fileName: 'x.pdf', fileSize: '10', fileKey: 'secret-token', type: 2 }], images: [{ fileToken: 'image-token' }] },
  });
  assert.deepEqual(output, { draftId: 'draft-1', threadId: 'thread-1', replyMessageId: '', subject: '', subjectTruncated: false, bodyHtml: '<private>', bodyTruncated: false, attachmentCount: 1, attachmentsTruncated: false, attachments: [{ fileName: 'x.pdf', fileSize: 10, type: 2, needConvertToLarge: false }], imageCount: 1, imagesTruncated: false });
  assert.equal(JSON.stringify(output).includes('secret-token'), false);
  assert.equal(JSON.stringify(output).includes('private@example.test'), false);
});


test('fixture metadata does not return partial subject or body snapshots', () => {
  const output = operations['mailnext.draft-fixture-metadata'].project({ draft: { subject: 's'.repeat(513), bodyHtml: '<p>' + 'x'.repeat(16384) + '</p>' } });
  assert.deepEqual(output, { draftId: '', threadId: '', replyMessageId: '', subject: '', subjectTruncated: true, bodyHtml: '', bodyTruncated: true, attachmentCount: 0, attachmentsTruncated: false, attachments: [], imageCount: 0, imagesTruncated: false });
});
