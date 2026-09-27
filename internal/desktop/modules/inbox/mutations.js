'use strict';

const MAX_INT64 = '9223372036854775807';

function object(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new TypeError('input must be an object');
  return value;
}

function chatId(value, name) {
  if (typeof value !== 'string' || !/^(?:0|[1-9][0-9]{0,18})$/.test(value) ||
      (value.length === 19 && value > MAX_INT64)) {
    throw new TypeError(`${name} must be a canonical non-negative native chat ID`);
  }
  return value;
}

function mute(value, name) {
  if (value !== 0 && value !== 1) throw new TypeError(`${name} must be 0 (unmuted) or 1 (muted)`);
  return value;
}

function setChatMuteRequest(input) {
  const source = object(input);
  for (const key of Object.keys(source)) {
    if (key !== 'chatId' && key !== 'muted' && key !== 'expectedMuted') {
      throw new TypeError(`unsupported inbox.set-self-chat-mute parameter: ${key}`);
    }
  }
  const id = chatId(source.chatId, 'chatId');
  const target = mute(source.muted, 'muted');
  mute(source.expectedMuted, 'expectedMuted');
  // The shipped setter creates chats.PatchChatSettingRequest and explicitly
  // marks only entities.ChatSetting.Field.IS_REMIND (1) as changed. `isRemind`
  // is the inverse of mute: false mutes the chat and true restores reminders.
  return {
    chatId: id,
    chatSetting: { isRemind: target === 0 },
    updateChatSettingField: [1],
  };
}

function setChatMuteProject(response) {
  const setting = object(object(response).chatSetting);
  const result = { acknowledged: true, verified: false };
  if (typeof setting.isRemind === 'boolean') result.muted = setting.isRemind ? 0 : 1;
  return result;
}

module.exports = {
  operations: {
    'inbox.set-self-chat-mute': {
      transport: 'server',
      command: '60|chats.PatchChatSettingRequest|chats.PatchChatSettingResponse',
      request: setChatMuteRequest,
      project: setChatMuteProject,
    },
  },
};
