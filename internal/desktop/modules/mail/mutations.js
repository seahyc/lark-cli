'use strict';

// These descriptors are intentionally unregistered. The parent integrates
// preview, explicit apply, selected-identity checks, and readback.
function object(input) {
  if (!input || typeof input !== 'object' || Array.isArray(input)) throw new TypeError('input must be an object');
  return input;
}

function exactKeys(input, keys) {
  const value = object(input);
  for (const key of Object.keys(value)) if (!keys.includes(key)) throw new TypeError(`unsupported input field: ${key}`);
  return value;
}

function string(value, name) {
  if (typeof value !== 'string' || value.length === 0) throw new TypeError(`${name} must be a non-empty string`);
  if (value.length > 512) throw new TypeError(`${name} exceeds 512 characters`);
  return value;
}

function emailAddress(value, name) {
  const result = string(value, name);
  if (result.length > 254 || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(result)) throw new TypeError(`${name} must be a valid email address`);
  return result;
}

function boolean(value, name) {
  if (typeof value !== 'boolean') throw new TypeError(`${name} must be a boolean`);
  return value;
}

function stringArray(value, name) {
  if (!Array.isArray(value) || value.length === 0 || value.length > 100) throw new TypeError(`${name} must contain 1-100 strings`);
  const result = value.map((item) => string(item, name));
  if (new Set(result).size !== result.length) throw new TypeError(`${name} must not contain duplicates`);
  return result;
}

function text(value, name, limit) {
  if (typeof value !== 'string' || value.length === 0) throw new TypeError(`${name} must be a non-empty string`);
  const result = value;
  if (result.length > limit) throw new TypeError(`${name} exceeds ${limit} characters`);
  return result;
}

function integer(value, name) {
  if (!Number.isInteger(value) || value < 0) throw new TypeError(`${name} must be a non-negative integer`);
  return value;
}

function escapeHtml(textValue) {
  return textValue.replace(/[&<>"']/g, (character) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  })[character]);
}

module.exports = {
  operations: {
    'mail.rule-enable': {
      command: '3690|email.client.v1.MailEnableRuleRequest|email.client.v1.MailEnableRuleResponse|1|MAIL_ENABLE_RULE',
      request(input) {
        const value = exactKeys(input, ['ruleIdString', 'isEnable']);
        return { ruleIdString: string(value.ruleIdString, 'ruleIdString'), isEnable: boolean(value.isEnable, 'isEnable') };
      },
      project(response, input) {
        const value = object(response);
        return { ruleIdString: string(exactKeys(input, ['ruleIdString', 'isEnable']).ruleIdString, 'ruleIdString'), isEnable: !!value.isEnable, acknowledged: true, verified: false };
      },
    },
    'mail.rule-reorder': {
      command: '3697|email.client.v1.MailAdjustRuleOrderRequest|email.client.v1.MailAdjustRuleOrderResponse|1|MAIL_ADJUST_RULE_ORDER',
      request(input) {
        const value = exactKeys(input, ['ruleIds']);
        return { ruleOrder: stringArray(value.ruleIds, 'ruleIds').map((ruleIdString, order) => ({ ruleIdString, order: String(order) })) };
      },
      project(response, input) {
        object(response);
        const value = exactKeys(input, ['ruleIds']);
        return { ruleIds: stringArray(value.ruleIds, 'ruleIds'), acknowledged: true, verified: false };
      },
    },
    'mail.signature-usage-update': {
      command: '3727|email.client.v1.MailUpdateSignatureUsageRequest|email.client.v1.MailUpdateSignatureUsageResponse|1|MAIL_UPDATE_SIGNATURE_USAGE',
      request(input) {
        const value = exactKeys(input, ['accountId', 'address', 'newMailSignatureId', 'replyMailSignatureId']);
        return {
          accountId: string(value.accountId, 'accountId'),
          signatureUsage: {
            address: string(value.address, 'address'),
            newMailSignatureId: string(value.newMailSignatureId, 'newMailSignatureId'),
            replyMailSignatureId: string(value.replyMailSignatureId, 'replyMailSignatureId'),
          },
        };
      },
      project(response, input) {
        object(response);
        const request = exactKeys(input, ['accountId', 'address', 'newMailSignatureId', 'replyMailSignatureId']);
        return {
          address: string(request.address, 'address'),
          newMailSignatureId: string(request.newMailSignatureId, 'newMailSignatureId'),
          replyMailSignatureId: string(request.replyMailSignatureId, 'replyMailSignatureId'),
          acknowledged: true,
          verified: false,
        };
      },
    },
    'mail.signature-create-text': {
      command: '3724|email.client.v1.MailAddSignatureRequest|email.client.v1.MailAddSignatureResponse|1|MAIL_ADD_SIGNATURE',
      request(input) {
        const value = exactKeys(input, ['accountId', 'name', 'text']);
        const content = text(value.text, 'text', 20000);
        return {
          accountId: string(value.accountId, 'accountId'),
          // Static UI callsite constructs {name, images, templateHtml,
          // id:void 0}. Empty images makes this disposable fixture text-only.
          signature: { name: text(value.name, 'name', 256), images: [], templateHtml: `<div>${escapeHtml(content)}</div>`, id: undefined },
        };
      },
      project(response) {
        const value = object(response);
        const signature = value.signature && typeof value.signature === 'object' ? value.signature : {};
        return { id: typeof signature.id === 'string' ? signature.id : '', acknowledged: true, verified: false };
      },
    },
    'mail.signature-delete': {
      command: '3726|email.client.v1.MailDeleteSignatureRequest|email.client.v1.MailDeleteSignatureResponse|1|MAIL_DELETE_SIGNATURE',
      request(input) {
        const value = exactKeys(input, ['accountId', 'signatureId', 'signatureUsageCount']);
        if (integer(value.signatureUsageCount, 'signatureUsageCount') !== 0) {
          throw new TypeError('signature must be unassigned before deletion');
        }
        return { accountId: string(value.accountId, 'accountId'), signatureId: string(value.signatureId, 'signatureId') };
      },
      project(response, input) {
        object(response);
        const value = exactKeys(input, ['accountId', 'signatureId', 'signatureUsageCount']);
        return { signatureId: string(value.signatureId, 'signatureId'), acknowledged: true, verified: false };
      },
    },
    'mail.sender-block': {
      command: '4021|email.client.v1.MailAddUserAllowBlockRequest|email.client.v1.MailAddUserAllowBlockResponse|1|MAIL_ADD_USER_ALLOW_BLOCK',
      request(input) {
        const value = exactKeys(input, ['address']);
        const address = emailAddress(value.address, 'address');
        return { multiFrom: [{ address }], isAllow: false, scene: 1 };
      },
      project(response, input) {
        object(response);
        const value = exactKeys(input, ['address']);
        return { address: emailAddress(value.address, 'address'), blocked: true, acknowledged: true, verified: false };
      },
    },
    'mail.sender-unblock': {
      command: '4021|email.client.v1.MailAddUserAllowBlockRequest|email.client.v1.MailAddUserAllowBlockResponse|1|MAIL_ADD_USER_ALLOW_BLOCK',
      request(input) {
        const value = exactKeys(input, ['address']);
        const address = emailAddress(value.address, 'address');
        return { multiFrom: [{ address }], isAllow: true, scene: 1 };
      },
      project(response, input) {
        object(response);
        const value = exactKeys(input, ['address']);
        return { address: emailAddress(value.address, 'address'), allowListed: true, acknowledged: true, verified: false };
      },
    },
  },
};
