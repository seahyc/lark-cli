'use strict';

// Every descriptor below is tied to a shipped mail-webview wrapper.  This
// module deliberately does not offer an arbitrary native-command escape hatch.

function object(input) {
  if (input === undefined) return {};
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    throw new TypeError('input must be an object');
  }
  return input;
}

function exactKeys(input, keys) {
  const value = object(input);
  for (const key of Object.keys(value)) {
    if (!keys.includes(key)) throw new TypeError(`unsupported input field: ${key}`);
  }
  return value;
}

function requiredString(value, name) {
  if (typeof value !== 'string' || value.length === 0) {
    throw new TypeError(`${name} must be a non-empty string`);
  }
  return value;
}

function signatures(response) {
  const value = object(response);
  const sourceSignatures = Array.isArray(value.signatures) ? value.signatures : [];
  const sourceUsages = Array.isArray(value.signatureUsages) ? value.signatureUsages : [];
  return {
    signatures: sourceSignatures.slice(0, 100).map((signature) => ({
      id: typeof signature?.id === 'string' ? signature.id : '',
      name: typeof signature?.name === 'string' ? signature.name : '',
      signatureType: signature?.signatureType,
      signatureDevice: signature?.signatureDevice,
    })),
    signaturesTruncated: sourceSignatures.length > 100,
    signatureUsages: sourceUsages.slice(0, 100).map((usage) => ({
      address: typeof usage?.address === 'string' ? usage.address : '',
      newMailSignatureId: typeof usage?.newMailSignatureId === 'string' ? usage.newMailSignatureId : '',
      replyMailSignatureId: typeof usage?.replyMailSignatureId === 'string' ? usage.replyMailSignatureId : '',
    })),
    signatureUsagesTruncated: sourceUsages.length > 100,
    optionalSignatureMap: value.optionalSignatureMap && typeof value.optionalSignatureMap === 'object'
      ? Object.fromEntries(Object.entries(value.optionalSignatureMap).map(([address, setting]) => [address, {
        signatureIds: Array.isArray(setting?.signatureIds) ? setting.signatureIds.filter((id) => typeof id === 'string') : [],
        isForceApply: !!setting?.isForceApply,
      }]))
      : {},
  };
}

function accountMetadata(response) {
  const value = object(response);
  const source = Array.isArray(value.account) ? value.account : value.account ? [value.account] : [];
  const flatten = (account, output = []) => {
    if (!account || typeof account !== 'object' || output.length >= 51) return output;
    output.push(account);
    if (Array.isArray(account.sharedAccounts)) account.sharedAccounts.forEach((shared) => flatten(shared, output));
    return output;
  };
  const rawAccounts = [];
  source.forEach((account) => flatten(account, rawAccounts));
  return {
    accounts: rawAccounts.slice(0, 50).map((account) => {
    const setting = account.mailSetting || {};
    return {
      accountAddress: typeof account.accountAddress === 'string' ? account.accountAddress : '',
      accountName: typeof account.accountName === 'string' ? account.accountName : '',
      mailAccountId: typeof account.mailAccountId === 'string' ? account.mailAccountId : '',
      larkUserId: typeof account.larkUserId === 'string' ? account.larkUserId : '',
      isSelected: !!account.accountSelected?.isSelected,
      isShared: !!account.isShared,
      isPersonal: !!account.isPersonal,
      provider: account.provider,
      protocol: account.protocol,
      settings: {
        enableConversationMode: !!setting.enableConversationMode,
        undoSendEnable: !!setting.undoSendEnable,
        undoTime: typeof setting.undoTime === 'string' ? setting.undoTime : '',
        replyLanguage: setting.replyLanguage,
        enableStranger: !!setting.enableStranger,
        webImageDisplay: !!setting.webImageDisplay,
        adminEnableAutoTransfer: !!setting.adminEnableAutoTransfer,
      },
    };
    }),
    accountsTruncated: rawAccounts.length > 50,
  };
}

function globalForwarding(response) {
  const value = object(response);
  return {
    adminEnableAutoTransfer: !!value.adminEnableAutoTransfer,
    rules: Array.isArray(value.rules) ? value.rules.map((rule) => ({
      ruleIdString: typeof rule?.ruleIdString === 'string' ? rule.ruleIdString : '',
      isEnable: !!rule?.isEnable,
      actions: Array.isArray(rule?.action?.items) ? rule.action.items
        .filter((action) => action?.type === 12)
        .map((action) => ({
          type: action.type,
          input: typeof action.input === 'string' ? action.input : '',
          authStatus: action.authStatus,
          enableAutoTransfer: !!action.enableAutoTransfer,
        })) : [],
    })) : [],
  };
}

function draftMetadata(response) {
  const value = object(response);
  const draft = value.draft && typeof value.draft === 'object' ? value.draft : {};
  return {
    id: typeof draft.id === 'string' ? draft.id : '',
    threadId: typeof draft.threadId === 'string' ? draft.threadId : '',
    replyMessageId: typeof draft.replyMessageId === 'string' ? draft.replyMessageId : '',
    subject: typeof draft.subject === 'string' ? draft.subject : '',
    lastUpdatedTimestamp: Number.isFinite(Number(draft.lastUpdatedTimestamp)) ? Number(draft.lastUpdatedTimestamp) : 0,
    createdTimestamp: Number.isFinite(Number(draft.createdTimestamp)) ? Number(draft.createdTimestamp) : 0,
    attachmentCount: Array.isArray(draft.attachments) ? draft.attachments.length : 0,
    imageCount: Array.isArray(draft.images) ? draft.images.length : 0,
    hasBody: typeof draft.bodyHtml === 'string' && draft.bodyHtml.length > 0,
  };
}

module.exports = {
  operations: {
    'mail.signatures': {
      command: '3728|email.client.v1.MailGetSignaturesRequest|email.client.v1.MailGetSignatureResponse|1|MAIL_GET_SIGNATURE',
      request(input) {
        const value = exactKeys(input, ['accountId']);
        return {
          accountId: requiredString(value.accountId, 'accountId'),
          // The shipped wrapper accepts a boolean second argument. The shared
          // manifest permits only string/integer parameters, so this read-only
          // operation fixes it to the wrapper's safe false value.
          fromSetting: false,
        };
      },
      project: signatures,
    },
    'mail.blocked-sender': {
      command: '4088|email.client.v1.MailGetBlockedAddressesRequest|email.client.v1.MailGetBlockedAddressesResponse|1|MAIL_GET_BLOCKED_ADDRESSES',
      request(input) {
        const value = exactKeys(input, ['address']);
        return { addresses: [requiredString(value.address, 'address')] };
      },
      project(response, input) {
        const value = object(response);
        const request = exactKeys(input, ['address']);
        const address = requiredString(request.address, 'address');
        return { blocked: Array.isArray(value.blockedAddresses) && value.blockedAddresses.includes(address) };
      },
    },
    'mail.schedule-status': {
      command: '3639|email.client.v1.MailGetScheduleMessageCountRequest|email.client.v1.MailGetScheduleMessageCountResponse|1|MAIL_GET_SCHEDULE_SEND_MESSAGE_COUNT',
      request(input) {
        exactKeys(input, []);
        return {};
      },
      project(response) {
        const value = object(response);
        const count = Number(value.scheduleSendMessageCount ?? '-1');
        return { count: Number.isFinite(count) ? count : -1 };
      },
    },
    'mail.account-metadata': {
      command: '3745|email.client.v1.MailGetAccountRequest|email.client.v1.MailGetAccountResponse|1|MAIL_GET_ACCOUNT',
      request(input) {
        exactKeys(input, []);
        return { fetchDb: false, fetchCurrentAccount: false };
      },
      project: accountMetadata,
    },
    'mail.verified-auto-transfer-emails': {
      command: '3766|email.client.v1.MailGetAutoTransferEmailRequest|email.client.v1.MailGetAutoTransferEmailResponse|1|MAIL_GET_AUTO_TRANSFER_EMAIL_REQUEST',
      request(input) {
        const value = exactKeys(input, ['accountId']);
        return { accountId: requiredString(value.accountId, 'accountId') };
      },
      project(response) {
        const value = object(response);
        return { emails: Array.isArray(value.emails) ? value.emails.filter((email) => typeof email === 'string') : [] };
      },
    },
    'mail.global-forwarding': {
      command: '3693|email.client.v1.MailGetRulesRequest|email.client.v1.MailGetRulesResponse|1|MAIL_GET_RULES',
      request(input) {
        const value = exactKeys(input, ['accountId']);
        return { version: '2', isGlobalTransfer: true, accountId: requiredString(value.accountId, 'accountId') };
      },
      project: globalForwarding,
    },
    'mail.draft-item': {
      command: '3612|email.client.v1.MailGetDraftItemRequest|email.client.v1.MailGetDraftItemResponse|1|MAIL_GET_DRAFT_ITEM',
      request(input) {
        const value = exactKeys(input, ['threadId', 'messageId']);
        return {
          threadId: requiredString(value.threadId, 'threadId'),
          messageId: requiredString(value.messageId, 'messageId'),
        };
      },
      project: draftMetadata,
    },
  },
};
