'use strict';

const MAX_INT64 = '9223372036854775807';

function inputObject(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new TypeError('input must be an object');
  return value;
}

function int64(value, name) {
  if (typeof value !== 'string' || !/^(?:0|[1-9][0-9]{0,18})$/.test(value) ||
      (value.length === 19 && value > MAX_INT64)) throw new TypeError(`${name} must be a canonical non-negative int64 string`);
  return value;
}

function request(input) {
  const source = inputObject(input);
  for (const key of Object.keys(source)) if (!['messageId', 'chatId', 'expectedText'].includes(key)) throw new TypeError(`unsupported recall parameter: ${key}`);
  const messageId = int64(source.messageId, 'messageId');
  int64(source.chatId, 'chatId');
  if (typeof source.expectedText !== 'string' || !source.expectedText.length || source.expectedText.length > 16384) throw new TypeError('expectedText must be non-empty plain text up to 16 KiB');
  // The desktop's ordinary recall action passes only this message identity.
  return {id: messageId};
}

module.exports = {operations: {
  'messaging.recall-self-text': {
    command: '2079|im.v1.RecallMessageRequest|im.v1.RecallMessageResponse|1|RECALL_MESSAGE',
    request,
    project(response) {
      inputObject(response || {});
      return {acknowledged: true, verified: false};
    }
  }
}};
