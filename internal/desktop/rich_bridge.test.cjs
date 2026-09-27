const {test}=require('node:test'),assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm');
const rich=require('./modules/rich/mutations.js').operations;
const reads={...require('./modules/rich/module.js').operations,...require('./modules/messaging/module.js').operations};
const original=JSON.stringify({title:'Fixture',paragraphs:[[{text:'before',bold:true}]]}),replacement=JSON.stringify({title:'Edited',paragraphs:[[{text:'after',italic:true}]]});
async function run({sender='99',stale=false,readbackMismatch=false,send=false,foreignChat=false}={}){
 let ws,reply,writes=0,prepares=0;const content=rich['messaging.send-self-post'].request({chatId:'11',post:original}).content;
 let message={id:'22',chatId:'11',fromId:sender,type:2,content};
 class Socket{constructor(){ws=this}send(s){reply=JSON.parse(s)}close(){}}
 const api={passport:{getUserId:()=> '99'},transport:{callSdkApi:async(command,p)=>{
  if(command.startsWith('1046|'))return {data:{chatterId2chat:{'99':{id:foreignChat?'33':'11'}}}};
  if(command.startsWith('1|'))return {data:{entity:{messages:{'22':message}}}};
  if(command.startsWith('1001|')){prepares++;message={...message,content:p.content};return {data:{cid:p.cid}}}
  if(command.startsWith('2003|')){writes++;return {data:{messageId:'22'}}}
  if(command.startsWith('4520|')){writes++;if(!readbackMismatch)message={...message,content:p.content};return {data:{}}}
  throw Error(command);
 }}};
 const source=fs.readFileSync(__dirname+'/mail_bridge.js','utf8').replace('__LARK_READ_OPERATIONS__','globalThis.reads').replace('__LARK_MUTATION_OPERATIONS__','globalThis.rich').replace('__LARK_LOCAL_SESSION__',JSON.stringify({port:9330,token:'test',expiresAt:Date.now()+60000}));
 vm.runInNewContext(source,{window:{LarkAPI:api},LarkAPI:api,WebSocket:Socket,reads,rich,Date,setTimeout:(f,ms)=>{if(ms===250)queueMicrotask(f);return 0},Map});
 await ws.onmessage({data:JSON.stringify({id:'1',operation:send?'messaging.send-self-post':'messaging.edit-post',params:{expectedUserId:'99',apply:true,input:send?{chatId:'11',post:replacement}:{chatId:'11',messageId:'22',post:replacement,expectedPost:stale?replacement:original}}})});return {reply,writes,prepares};
}
test('rich own edit verifies formatting and stale or foreign messages never write',async()=>{
 const success=await run();assert.equal(success.reply.data.verified,true);assert.equal(success.writes,1);
 for(const options of [{sender:'other'},{stale:true}]){const r=await run(options);assert.equal(r.reply.ok,false);assert.equal(r.writes,0)}
 const mismatch=await run({readbackMismatch:true});assert.equal(mismatch.reply.ok,false);assert.equal(mismatch.writes,1);
});
test('rich selfsend checks destination then prepares/sends once and verifies styled content',async()=>{
 const success=await run({send:true});assert.equal(success.reply.data.verified,true);assert.equal(success.prepares,1);assert.equal(success.writes,1);
 const rejected=await run({send:true,foreignChat:true});assert.equal(rejected.reply.ok,false);assert.equal(rejected.prepares,0);assert.equal(rejected.writes,0);
});
