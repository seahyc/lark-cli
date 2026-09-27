'use strict';

function object(value) { return value && typeof value === 'object' && !Array.isArray(value) ? value : {}; }
function identifier(value, name) { if (typeof value !== 'string' || !value.length || value.length > 512) throw new TypeError(`${name} must be a native identifier up to 512 characters`); return value; }
function plainText(richText) {
  const source = object(richText);
  if (typeof source.innerText === 'string' && source.innerText.length > 0) return source.innerText.slice(0, 16384);
  const elements = object(source.elements);
  const ids = Array.isArray(source.elementIds) ? source.elementIds : [];
  if (!ids.length) return Object.keys(elements).length ? undefined : '';
  const values = ids.map(id => object(elements[id])).map(element => element.tag === 1 ? object(object(element.property).text).content : undefined);
  return values.every(value => typeof value === 'string') ? values.join('').slice(0, 16384) : undefined;
}
function request(input) {
  const source = object(input);
  if (Object.keys(source).some(key => key !== 'chatId')) throw new TypeError('unsupported scheduled-items parameter');
  return {chatId: identifier(source.chatId, 'chatId'), syncDataStrategy: 3, scene: 1};
}
function project(response) {
  const source = object(response), entity = object(source.entity), scheduled = object(entity.scheduleMessage);
  const ids = Array.isArray(source.messageItems) ? source.messageItems.map(item => object(item).itemId).filter(id => typeof id === 'string') : Object.keys(scheduled);
  const items = ids.slice(0, 100).flatMap(id => {
    const entry = object(scheduled[id]), message = object(object(entity.messages)[entry.messageId || id] || entry.message || entry.scheduleMessage);
    if (!Object.keys(entry).length) return [];
    const value = {messageId: typeof entry.id === 'string' ? entry.id : id, chatId: typeof entry.chatId === 'string' ? entry.chatId : message.chatId, scheduleTime: typeof entry.scheduleTime === 'string' ? entry.scheduleTime : undefined, status: typeof entry.status === 'number' ? entry.status : undefined, senderId: typeof message.fromId === 'string' ? message.fromId : undefined, type: typeof message.type === 'number' ? message.type : undefined, text: plainText(object(message.content).richText)};
    return [Object.fromEntries(Object.entries(value).filter(([, field]) => field !== undefined))];
  });
  return {items, total: ids.length, truncated: ids.length > items.length, ...(items.some(x=>!x.chatId) ? {schema:{entityFields:Object.keys(entity).slice(0,60),scheduleFields:Object.keys(object(scheduled[ids[0]])).slice(0,30),messageCount:Object.keys(object(entity.messages)).length}} : {})};
}
module.exports = {operations: {'triage.scheduled-items': {command: '39007|im.v1.GetScheduleMessagesRequest|im.v1.GetScheduleMessagesResponse|1|GET_SCHEDULE_MESSAGES', request, project}}};
