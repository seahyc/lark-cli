'use strict';

const MAX_INT64 = '9223372036854775807';

function object(value) {
  return value && typeof value === 'object' && !Array.isArray(value) ? value : {};
}

function nativeMessageId(value) {
  if (typeof value !== 'string' || !/^(?:0|[1-9][0-9]{0,18})$/.test(value) ||
      (value.length === 19 && value > MAX_INT64)) {
    throw new Error('messageId must be a canonical non-negative int64 string');
  }
  return value;
}

function text(value) {
  return typeof value === 'string' ? value.slice(0, 16384) : undefined;
}

function plainTextFromRichText(value) {
  const richText = object(value);
  const ids = Array.isArray(richText.elementIds) ? richText.elementIds : [];
  if (ids.length) {
    let result = '';
    for (const id of ids) {
      const element = object(object(richText.elements)[id]);
      const content = object(object(element.property).text).content;
      if (element.tag !== 1 || typeof content !== 'string') return undefined;
      result += content;
      if (result.length > 16384) return result.slice(0, 16384);
    }
    return result;
  }
  return text(richText.innerText);
}

function textMessageRequest(input) {
  const source = object(input);
  if (Object.keys(source).length !== 1 || !Object.prototype.hasOwnProperty.call(source, 'messageId')) {
    throw new Error('messageId is required');
  }
  return {messageIds: [nativeMessageId(source.messageId)]};
}

function textMessageProject(response) {
  const messages = object(object(response).entity).messages;
  const message = object(Object.values(object(messages))[0]);
  const result = {};
  if (typeof message.chatId === 'string') result.chatId = message.chatId;
  if (typeof message.fromId === 'string') result.senderId = message.fromId;
  if (typeof message.type === 'number') result.type = message.type;
  const plainText = plainTextFromRichText(object(message.content).richText);
  if (plainText !== undefined) result.text = plainText;
  return result;
}

// Outbound mutations are deliberately kept in mutations.js so the shared
// runtime can apply its preview, identity, and approval guards.
module.exports = {
  operations: {
    'messaging.text-message': {
      command: '1|im.v1.MGetMessagesRequest|im.v1.MGetMessagesResponse|1|MGET_MESSAGES',
      request: textMessageRequest,
      project: textMessageProject
    }
  }
};
