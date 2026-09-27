'use strict';
const assert = require('node:assert/strict');
const {execFileSync} = require('node:child_process');
const {mkdtempSync, readFileSync, rmSync, writeFileSync} = require('node:fs');
const {tmpdir} = require('node:os');
const {join} = require('node:path');
const test = require('node:test');
const {scheduledTextContent, textProperty, MAX_UTF8_BYTES} = require('./text_codec.js');

const proto = `syntax = "proto2";
package messages;
message Content { optional RichText rich_text = 14; }
message RichText { repeated string element_ids = 1; required string inner_text = 2; required RichTextElements elements = 3; repeated string image_ids = 5; repeated string at_ids = 6; repeated string anchor_ids = 7; repeated string media_ids = 9; repeated string at_user_group_ids = 14; repeated string markdown_ids = 15; }
message RichTextElements { map<string, RichTextElement> dictionary = 1; }
message RichTextElement { enum Tag { UNKNOWN_TAG = 0; TEXT = 1; } required Tag tag = 1; required bytes property = 3; }
message TextProperty { required string content = 1; }
`;

function stage(text) { return {elementIds: ['text'], innerText: text, elements: {text: {tag: 1, property: {text: {content: text}}}}}; }
function hex(bytes) { return Buffer.from(bytes).toString('hex'); }

test('converts staged TEXT into the worklet server Content object', () => {
  const actual = scheduledTextContent(stage('fixture'));
  assert.equal(MAX_UTF8_BYTES, 16384);
  assert.deepEqual(Object.keys(actual), ['richText']);
  assert.deepEqual(actual.richText.elementIds, ['text']);
  assert.equal(actual.richText.innerText, 'fixture');
  assert.deepEqual(Object.keys(actual.richText.elements), ['dictionary']);
  assert.equal(actual.richText.elements.dictionary.text.tag, 1);
  assert.ok(actual.richText.elements.dictionary.text.property instanceof Uint8Array);
  assert.equal(hex(actual.richText.elements.dictionary.text.property), '0a0766697874757265');
  assert.deepEqual(actual.richText.imageIds, []);
  assert.deepEqual(actual.richText.atUserGroupIds, []);
});

test('protoc validates the exact nested bytes at the server Content boundary', () => {
  const dir = mkdtempSync(join(tmpdir(), 'lark-text-codec-'));
  try {
    const schema = join(dir, 'messages.proto');
    const expected = join(dir, 'expected.bin');
    writeFileSync(schema, proto);
    const fixture = 'rich_text { element_ids: "text" inner_text: "fixture" elements { dictionary { key: "text" value { tag: TEXT property: "\\n\\007fixture" } } } }';
    const encoded = execFileSync('protoc', ['-I', dir, '--encode=messages.Content', 'messages.proto'], {input: fixture});
    writeFileSync(expected, encoded);
    const decoded = execFileSync('protoc', ['-I', dir, '--decode=messages.Content', 'messages.proto'], {input: readFileSync(expected)}).toString();
    assert.match(decoded, /inner_text: "fixture"/);
    assert.match(decoded, /property: "\\n\\007fixture"/);
    assert.equal(hex(textProperty('fixture')), '0a0766697874757265');
  } finally { rmSync(dir, {recursive: true, force: true}); }
});

test('rejects malformed, non-text, and oversized staged content', () => {
  assert.throws(() => scheduledTextContent(stage('')), /must not be empty/);
  assert.throws(() => scheduledTextContent(stage('\ud800')), /unpaired surrogate/);
  assert.throws(() => scheduledTextContent(stage('a'.repeat(MAX_UTF8_BYTES + 1))), /exceeds/);
  assert.throws(() => scheduledTextContent({elementIds: ['text'], innerText: 'x', elements: {text: {tag: 2, property: {text: {content: 'x'}}}}}), /consistent TEXT/);
  assert.throws(() => scheduledTextContent({elementIds: ['a', 'b'], innerText: 'x', elements: {a: {tag: 1, property: {text: {content: 'x'}}}}}), /one native TEXT/);
});
