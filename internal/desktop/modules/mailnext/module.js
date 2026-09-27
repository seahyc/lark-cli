'use strict';

function object(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new TypeError('value must be an object');
  return value;
}

function exactKeys(input, keys) {
  const value = object(input);
  for (const key of Object.keys(value)) if (!keys.includes(key)) throw new TypeError(`unsupported input field: ${key}`);
  return value;
}

function id(value, name, allowEmpty) {
  if (typeof value !== 'string' || (!allowEmpty && value.length === 0) || value.length > 512) throw new TypeError(`${name} must be ${allowEmpty ? 'a' : 'a non-empty'} string up to 512 characters`);
  return value;
}

function integer(value) {
  const number = Number(value);
  return Number.isFinite(number) && number >= 0 ? number : 0;
}

function draftFixtureMetadata(response) {
  const value = object(response);
  const draft = value.draft && typeof value.draft === 'object' ? value.draft : {};
  const attachments = Array.isArray(draft.attachments) ? draft.attachments : [];
  const images = Array.isArray(draft.images) ? draft.images : [];
  const rawSubject = typeof draft.subject === 'string' ? draft.subject : '';
  const rawBodyHtml = typeof draft.bodyHtml === 'string' ? draft.bodyHtml : '';
  return {
    draftId: typeof draft.id === 'string' ? draft.id : '',
    threadId: typeof draft.threadId === 'string' ? draft.threadId : '',
    replyMessageId: typeof draft.replyMessageId === 'string' ? draft.replyMessageId : '',
    // Do not return a partial expected-value snapshot: callers either receive
    // the complete bounded value or an explicit truncation indicator.
    subject: rawSubject.length <= 512 ? rawSubject : '',
    subjectTruncated: rawSubject.length > 512,
    bodyHtml: rawBodyHtml.length <= 16384 ? rawBodyHtml : '',
    bodyTruncated: rawBodyHtml.length > 16384,
    attachmentCount: attachments.length,
    attachmentsTruncated: attachments.length > 20,
    attachments: attachments.slice(0, 20).map((attachment) => ({
      fileName: typeof attachment?.fileName === 'string' ? attachment.fileName.slice(0, 256) : '',
      fileSize: integer(attachment?.fileSize),
      type: attachment?.type,
      needConvertToLarge: !!attachment?.needConvertToLarge,
    })),
    imageCount: images.length,
    imagesTruncated: images.length > 20,
  };
}

module.exports = {
  operations: {
    'mailnext.draft-fixture-metadata': {
      command: '3612|email.client.v1.MailGetDraftItemRequest|email.client.v1.MailGetDraftItemResponse|1|MAIL_GET_DRAFT_ITEM',
      request(input) {
        const value = exactKeys(input, ['draftId']);
        return { draftId: id(value.draftId, 'draftId') };
      },
      project: draftFixtureMetadata,
    },
  },
};
