'use strict';

// Narrow conversion of a staged CLI TEXT rich-text value into the *server*
// PutMessageRequest.Content value used by the installed scheduler worklet.
// The outer protobuf packet is encoded by LarkAPI.transport.callServerApi.
var MAX_UTF8_BYTES = 16384;

function utf8(value) {
  if (typeof value !== 'string') throw new TypeError('text must be a string');
  var out = [];
  for (var i = 0; i < value.length; i += 1) {
    var code = value.charCodeAt(i);
    if (code >= 0xd800 && code <= 0xdbff) {
      if (i + 1 >= value.length) throw new TypeError('text contains an unpaired surrogate');
      var low = value.charCodeAt(i + 1);
      if (low < 0xdc00 || low > 0xdfff) throw new TypeError('text contains an unpaired surrogate');
      code = 0x10000 + ((code - 0xd800) * 0x400) + (low - 0xdc00);
      i += 1;
    } else if (code >= 0xdc00 && code <= 0xdfff) throw new TypeError('text contains an unpaired surrogate');
    if (code < 0x80) out.push(code);
    else if (code < 0x800) out.push(0xc0 | (code >> 6), 0x80 | (code & 0x3f));
    else if (code < 0x10000) out.push(0xe0 | ((code >> 12) & 0x0f), 0x80 | ((code >> 6) & 0x3f), 0x80 | (code & 0x3f));
    else out.push(0xf0 | (code >> 18), 0x80 | ((code >> 12) & 0x3f), 0x80 | ((code >> 6) & 0x3f), 0x80 | (code & 0x3f));
  }
  return out;
}

function varint(value) {
  var out = [];
  do { var next = value % 128; value = Math.floor(value / 128); out.push(next | (value ? 0x80 : 0)); } while (value);
  return out;
}

function bytesField(number, bytes) { return varint(number * 8 + 2).concat(varint(bytes.length), bytes); }

function textProperty(text) {
  var content = utf8(text);
  if (!content.length) throw new TypeError('text must not be empty');
  if (content.length > MAX_UTF8_BYTES) throw new RangeError('text exceeds 16384 UTF-8 bytes');
  // entities.RichTextElement.TextProperty { required string content = 1; }
  return new Uint8Array(bytesField(1, content));
}

function scheduledTextContent(richText) {
  if (!richText || typeof richText !== 'object' || Array.isArray(richText)) throw new TypeError('richText must be an object');
  var ids = richText.elementIds;
  if (!Array.isArray(ids) || ids.length !== 1 || typeof ids[0] !== 'string' || !ids[0] || ids[0].length > 512) throw new TypeError('richText must contain one native TEXT element id');
  var id = ids[0];
  if (!richText.elements || typeof richText.elements !== 'object' || Array.isArray(richText.elements) || Object.keys(richText.elements).length !== 1) throw new TypeError('richText must contain one TEXT element');
  var prop = textProperty(richText.innerText);
  var element = richText.elements[id];
  if (!element || element.tag !== 1 || !element.property || !element.property.text || element.property.text.content !== richText.innerText) throw new TypeError('richText must be one consistent TEXT element');
  // This is module 13947's r.ak() result for constrained TEXT input. Its
  // Uint8Array property is accepted by descriptor-aware server transport.
  return {richText: {
    elementIds: [id], innerText: richText.innerText,
    imageIds: [], atIds: [], anchorIds: [], mediaIds: [], atUserGroupIds: [], markdownIds: [],
    elements: {dictionary: {[id]: {tag: 1, property: prop}}}
  }};
}

module.exports = {MAX_UTF8_BYTES: MAX_UTF8_BYTES, textProperty: textProperty, scheduledTextContent: scheduledTextContent};
