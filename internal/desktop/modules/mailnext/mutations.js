'use strict';

function object(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new TypeError('input must be an object');
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

function createProject(response) {
  const value = object(response);
  const draft = value.draft && typeof value.draft === 'object' ? value.draft : {};
  return {
    draftId: typeof draft.id === 'string' ? draft.id : '',
    threadId: typeof draft.threadId === 'string' ? draft.threadId : '',
    replyMessageId: typeof draft.replyMessageId === 'string' ? draft.replyMessageId : '',
    attachmentCount: Array.isArray(draft.attachments) ? draft.attachments.length : 0,
    acknowledged: true,
    verified: false,
  };
}

function optionalString(value, name, limit) {
  if (typeof value !== 'string' || value.length > limit) throw new TypeError(`${name} must be a string up to ${limit} characters`);
  return value;
}

function html(value) {
  const result = optionalString(value, 'bodyHtml', 16384);
  // This fixture-only editor permits a small inert HTML subset. No attributes,
  // links, images, styles, scripts, or embedded resources are accepted.
  if (!/^(?:[^<>]|<(?:p|br\s*\/?|strong|em|u)>|<\/(?:p|strong|em|u)>)*$/.test(result)) {
    throw new TypeError('bodyHtml permits only p, br, strong, em, and u tags without attributes');
  }
  return result;
}

function normalizedAddress(value) {
  const source = value && typeof value === 'object' ? value : {};
  const operator = source.operator && typeof source.operator === 'object' ? source.operator : {};
  return {
    name: typeof source.name === 'string' ? source.name : '',
    address: typeof source.address === 'string' ? source.address : '',
    larkEntityType: Number.isInteger(source.larkEntityType) ? source.larkEntityType : 0,
    larkEntityIdString: typeof source.larkEntityIdString === 'string' ? source.larkEntityIdString : '',
    operator: {
      userId: typeof operator.userId === 'string' ? operator.userId : '',
      name: typeof operator.name === 'string' ? operator.name : '',
      address: typeof operator.address === 'string' ? operator.address : '',
      displayName: typeof operator.displayName === 'string' ? operator.displayName : '',
    },
    mailGroupType: Number.isInteger(source.mailGroupType) ? source.mailGroupType : 0,
    tenantId: typeof source.tenantId === 'string' ? source.tenantId : '',
    displayName: typeof source.displayName === 'string' ? source.displayName : '',
  };
}

function rawFixture(raw, input) {
  const response = object(raw);
  const draft = response.draft && typeof response.draft === 'object' ? response.draft : null;
  if (!draft) throw new TypeError('snapshot must contain a native draft');
  const value = exactKeys(input, ['draftId', 'threadId', 'replyMessageId', 'subject', 'bodyHtml', 'expectedSubject', 'expectedBodyHtml']);
  const draftId = id(value.draftId, 'draftId');
  const threadId = id(value.threadId, 'threadId');
  const replyMessageId = id(value.replyMessageId, 'replyMessageId', true);
  if (draft.id !== draftId || draft.threadId !== threadId || (draft.replyMessageId ?? '') !== replyMessageId) {
    throw new TypeError('snapshot identity does not match the fixture identity');
  }
  const expectedSubject = optionalString(value.expectedSubject, 'expectedSubject', 512);
  const expectedBodyHtml = html(value.expectedBodyHtml);
  if ((draft.subject ?? '') !== expectedSubject || (draft.bodyHtml ?? '') !== expectedBodyHtml) {
    throw new TypeError('snapshot subject or body does not match the expected fixture state');
  }
  const arrays = ['to', 'cc', 'bcc', 'images', 'attachments', 'docsPermissions'];
  for (const key of arrays) if (Array.isArray(draft[key]) && draft[key].length !== 0) throw new TypeError(`fixture snapshot must have no ${key}`);
  if (draft.calendarEvent !== undefined && draft.calendarEvent !== null) throw new TypeError('fixture snapshot must have no calendarEvent');
  if (draft.isSendSeparately || draft.needReadReceipt || draft.coverInfo || draft.bodySummary || draft.isFromLarkPlaintext) {
    throw new TypeError('fixture snapshot contains unsupported draft state');
  }
  const priorityType = draft.priorityType === undefined ? 3 : draft.priorityType;
  if (priorityType !== 3) throw new TypeError('fixture snapshot priority must be normal');
  const from = normalizedAddress(draft.from);
  if (!from.address || from.address.length > 512) throw new TypeError('fixture snapshot must have a bounded sender address');
  return { value, draft, draftId, threadId, from };
}

function deleteFixturePayload(raw, input) {
  const value = exactKeys(input, ['threadId', 'messageId', 'expectedSubject', 'expectedBodyHtml']);
  const threadId = id(value.threadId, 'threadId');
  const messageId = id(value.messageId, 'messageId');
  const response = object(raw);
  const draft = response.draft && typeof response.draft === 'object' ? response.draft : null;
  if (!draft) throw new TypeError('snapshot must contain a native draft');
  // MAIL_DELETE_DRAFT deletes a draft by the current draft id. The UI close
  // path passes draft.id, while replyMessageId only identifies a reply source.
  rawFixture(raw, {
    draftId: messageId,
    threadId,
    replyMessageId: typeof draft.replyMessageId === 'string' ? draft.replyMessageId : '',
    subject: typeof draft.subject === 'string' ? draft.subject : '',
    bodyHtml: typeof draft.bodyHtml === 'string' ? draft.bodyHtml : '',
    expectedSubject: value.expectedSubject,
    expectedBodyHtml: value.expectedBodyHtml,
  });
  return { threadId, messageId };
}

function h5FixturePayload(raw, input) {
  const { value, draftId, threadId, from } = rawFixture(raw, input);
  const subject = optionalString(value.subject, 'subject', 512);
  const bodyHtml = html(value.bodyHtml);
  return {
    draftId,
    payload: {
      from,
      to: [],
      cc: [],
      bcc: [],
      bodyHtml,
      subject,
      images: [],
      attachments: [],
      docsPermissions: [],
      threadId,
      isSendSeparately: false,
      priorityType: 3,
      coverInfo: '',
      calendarEvent: undefined,
      needReadReceipt: false,
      bodySummary: '',
      isFromLarkPlaintext: false,
    },
    isDelay: false,
    onlySaveLocal: false,
  };
}

function escapedSignatureHtml(value) {
  const text = optionalString(value, 'text', 20000);
  if (!text.length) throw new TypeError('text must be non-empty');
  return `<div>${text.replace(/[&<>"']/g, (character) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  })[character])}</div>`;
}

function signatureFixtureUpdatePayload(raw, input) {
  const snapshot = object(raw);
  const value = exactKeys(input, ['accountId', 'signatureId', 'text', 'expectedTemplateHtml']);
  const accountId = id(value.accountId, 'accountId');
  const signatureId = id(value.signatureId, 'signatureId');
  const expectedTemplateHtml = optionalString(value.expectedTemplateHtml, 'expectedTemplateHtml', 120000);
  const signatures = Array.isArray(snapshot.signatures) ? snapshot.signatures : [];
  const matches = signatures.filter((signature) => signature && typeof signature === 'object' && signature.id === signatureId);
  if (matches.length !== 1) throw new TypeError('snapshot must contain exactly one matching native signature');
  const signature = matches[0];
  // The desktop update wrapper forwards the complete signature object.  Keep
  // every raw field (including type/device/template JSON and image records)
  // byte-for-byte and replace only the HTML generated from the approved text.
  if (typeof signature.name !== 'string' || typeof signature.templateHtml !== 'string' ||
      typeof signature.templateValueJson !== 'string' || !Array.isArray(signature.images) ||
      !Object.prototype.hasOwnProperty.call(signature, 'signatureType') ||
      !Object.prototype.hasOwnProperty.call(signature, 'signatureDevice')) {
    throw new TypeError('snapshot signature is missing required native fields');
  }
  if (signature.templateHtml.length > 120000 || signature.templateHtml !== expectedTemplateHtml) {
    throw new TypeError('snapshot template HTML does not match the expected fixture state');
  }
  return {accountId, signature: {...signature, templateHtml: escapedSignatureHtml(value.text)}};
}

module.exports = {
  operations: {
    'mailnext.draft-fixture-create': {
      command: '3606|email.client.v1.MailCreateDraftRequest|email.client.v1.MailCreateDraftResponse|1|MAIL_CREATE_DRAFT',
      request(input) {
        exactKeys(input, []);
        // Active compose starts exactly with empty IDs and the compose action; the
        // wrapper appends isFromLarkPlaintext:false. A fixture deliberately
        // opts out of applying a user signature and sends no recipient/body.
        return { threadId: '', originMessageId: '', action: 0, needSignature: false, timeText: '', isFromLarkPlaintext: false };
      },
      project: createProject,
    },
    'mailnext.draft-fixture-delete': {
      command: '3601|email.v1.DeleteMailDraftRequest|email.v1.DeleteMailDraftResponse|1|DELETE_MAIL_DRAFT',
      request() {
        throw new TypeError('mailnext.draft-fixture-delete requires requestFromSnapshot(rawSnapshot, input)');
      },
      requestFromSnapshot(rawSnapshot, input) {
        return deleteFixturePayload(rawSnapshot, input);
      },
      project(response, input) {
        object(response);
        const value = exactKeys(input, ['threadId', 'messageId', 'expectedSubject', 'expectedBodyHtml']);
        return { threadId: id(value.threadId, 'threadId'), messageId: id(value.messageId, 'messageId'), acknowledged: true, verified: false };
      },
    },
    'mailnext.draft-fixture-update': {
      command: '3607|email.client.v1.MailUpdateDraftRequest|email.client.v1.MailUpdateDraftResponse|1|MAIL_UPDATE_DRAFT',
      request() {
        throw new TypeError('mailnext.draft-fixture-update requires requestFromSnapshot(rawSnapshot, input)');
      },
      requestFromSnapshot(rawSnapshot, input) {
        return h5FixturePayload(rawSnapshot, input);
      },
      project(response, input) {
        const value = object(response);
        const draft = value.draft && typeof value.draft === 'object' ? value.draft : {};
        const request = exactKeys(input, ['draftId', 'threadId', 'replyMessageId', 'subject', 'bodyHtml', 'expectedSubject', 'expectedBodyHtml']);
        return {
          draftId: typeof draft.id === 'string' ? draft.id : id(request.draftId, 'draftId'),
          threadId: typeof draft.threadId === 'string' ? draft.threadId : id(request.threadId, 'threadId'),
          subject: typeof draft.subject === 'string' ? draft.subject.slice(0, 512) : optionalString(request.subject, 'subject', 512),
          bodyHtmlLength: typeof draft.bodyHtml === 'string' ? draft.bodyHtml.length : html(request.bodyHtml).length,
          acknowledged: true,
          verified: false,
        };
      },
    },
    'mailnext.signature-fixture-update-text': {
      command: '3725|email.client.v1.MailUpdateSignatureRequest|email.client.v1.MailUpdateSignatureResponse|1|MAIL_UPDATE_SIGNATURE',
      request() {
        throw new TypeError('mailnext.signature-fixture-update-text requires requestFromSnapshot(rawSnapshot, input)');
      },
      requestFromSnapshot(rawSnapshot, input) {
        return signatureFixtureUpdatePayload(rawSnapshot, input);
      },
      project(response, input) {
        object(response);
        const value = exactKeys(input, ['accountId', 'signatureId', 'text', 'expectedTemplateHtml']);
        return {
          signatureId: id(value.signatureId, 'signatureId'),
          templateHtmlLength: escapedSignatureHtml(value.text).length,
          acknowledged: true,
          verified: false,
        };
      },
    },
  },
};
