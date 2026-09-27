'use strict';
const test=require('node:test'), assert=require('node:assert/strict');
const op=require('./module.js').operations['triage.scheduled-items'];
test('scheduled reader projects only cancellation preflight fields',()=>{assert.deepEqual(op.request({chatId:'7'}),{chatId:'7',syncDataStrategy:3,scene:1});assert.deepEqual(op.project({messageItems:[{itemId:'9'}],entity:{scheduleMessage:{'9':{id:'9',chatId:'7',scheduleTime:'1790000000',status:1,message:{fromId:'u',type:4,content:{richText:{innerText:'fixture'}}}}}}}),{items:[{messageId:'9',chatId:'7',scheduleTime:'1790000000',status:1,senderId:'u',type:4,text:'fixture'}],total:1,truncated:false});});

test('scheduled metadata joins separate entity message records',()=>{const r=op.project({messageItems:[{itemId:'9'}],entity:{scheduleMessage:{'9':{messageId:'9',scheduleTime:'1790000000',status:1}},messages:{'9':{chatId:'7',fromId:'u',type:4,content:{richText:{innerText:'fixture'}}}}}});assert.equal(r.items[0].text,'fixture');assert.equal(r.items[0].chatId,'7');});
