const {test}=require('node:test');const assert=require('node:assert/strict');
const read=require('./module.js').operations['messaging.post-message'];const ops=require('./mutations.js').operations;
const post=JSON.stringify({title:'Fixture',paragraphs:[[{text:'Bold',bold:true},{text:' and italic',italic:true}],[{text:'underlined',underline:true}]]});
test('styled POST uses paragraph graph, native style names, and POST envelope',()=>{
 const p=ops['messaging.send-self-post'].request({chatId:'11',post});assert.equal(p.type,2);assert.equal(p.channel.id,'11');const r=p.content.richText;
 assert.equal(r.elements[r.elementIds[0]].tag,3);assert.equal(r.elements.t2.style.fontWeight,'bold');assert.equal(r.elements.t3.style.fontStyle,'italic');assert.equal(r.elements.t5.style['-lark-textDecoration'],'underline');
 const got=read.project({entity:{messages:{'22':{id:'22',chatId:'11',fromId:'99',type:2,content:p.content}}}},{messageId:'22'});assert.equal(got.editable,true);assert.equal(got.post,post);
});
test('POST parser accepts only default protobuf union fields and matching derived text',()=>{
 const p=ops['messaging.send-self-post'].request({chatId:'11',post});
 Object.assign(p.content,{text:'Bold and italic\nunderlined',isOriginSource:false,originSize:'0',imageUploadId:'',key:'',name:'',size:'0',mime:'',fileUploadId:'',template:'',values:{},duration:0,deprecated:1,systemType:0,contentValues:{},attachments:[],docEntity:{},typedElementRefs:{},hideVoice2text:true,fileSource:1});
 const parsed=read.project({entity:{messages:{'22':{type:2,content:p.content}}}},{messageId:'22'});
 assert.equal(parsed.editable,true);assert.equal(parsed.post,post);
 // Numeric protobuf zero is permitted only for generated int64 fields, never strings/tokens.
 const tokenZero={...p.content,fileUploadId:'0'};
 const tokenRejected=read.project({entity:{messages:{'22':{type:2,content:tokenZero}}}},{messageId:'22'});
 assert.equal(tokenRejected.editable,false);assert.equal(tokenRejected.diagnostic.reason,'unsupported meaningful POST content field: fileUploadId (string)');
 const enumNonDefault={...p.content,systemType:1};
 const enumRejected=read.project({entity:{messages:{'22':{type:2,content:enumNonDefault}}}},{messageId:'22'});
 assert.equal(enumRejected.editable,false);assert.equal(enumRejected.diagnostic.reason,'unsupported meaningful POST content field: systemType (number)');
 const fileEnumNonDefault={...p.content,fileSource:0};
 const fileEnumRejected=read.project({entity:{messages:{'22':{type:2,content:fileEnumNonDefault}}}},{messageId:'22'});
 assert.equal(fileEnumRejected.editable,false);assert.equal(fileEnumRejected.diagnostic.reason,'unsupported meaningful POST content field: fileSource (number)');
 const meaningful={...p.content,fileUploadId:'non-default'};
 const rejected=read.project({entity:{messages:{'22':{type:2,content:meaningful}}}},{messageId:'22'});
 assert.equal(rejected.editable,false);assert.equal(rejected.diagnostic.reason,'unsupported meaningful POST content field: fileUploadId (string)');
 const mismatch={...p.content,text:'different'};
 const textRejected=read.project({entity:{messages:{'22':{type:2,content:mismatch}}}},{messageId:'22'});
 assert.equal(textRejected.editable,false);assert.equal(textRejected.diagnostic.reason,'POST plain text differs from rich text');
});
test('unsupported rich content cannot be edited or silently removed',()=>{
 const p=ops['messaging.send-self-post'].request({chatId:'11',post});p.content.richText.elements.t2.tag=5;
 const got=read.project({entity:{messages:{'22':{type:2,content:p.content}}}},{messageId:'22'});assert.equal(got.editable,false);assert.equal(got.post,undefined);assert.equal(got.diagnostic.reason,'unsupported rich element');assert.deepEqual(got.diagnostic.structure.contentFields.keys,['title','richText']);assert.ok(got.diagnostic.structure.elementSchemas.some(schema=>schema.tag===5));assert.ok(got.diagnostic.structure.elementSchemas.every(schema=>!Object.values(schema).includes('Bold')));
 for(const bad of [{title:'',paragraphs:[[{text:'x',image:'key'}]]},{title:'',paragraphs:[[{text:'x',bold:'true'}]]},{title:'',paragraphs:[]}])assert.throws(()=>ops['messaging.send-self-post'].request({chatId:'11',post:JSON.stringify(bad)}));
});
test('edit carries exact message identity but expected snapshot stays local',()=>{
 const r=ops['messaging.edit-post'].request({chatId:'11',messageId:'22',post,expectedPost:post});assert.equal(r.msgId,'22');assert.equal(r.type,2);assert.equal(r.expectedPost,undefined);
 assert.throws(()=>ops['messaging.edit-post'].request({chatId:'11',messageId:'22',post}));
});
