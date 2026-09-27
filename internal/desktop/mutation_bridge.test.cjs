const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const template=fs.readFileSync(__dirname+'/mail_bridge.js','utf8');
async function execute(params,{failure=false}={}) {
 let socket,reply,calls=0;
 class Socket{constructor(){socket=this}send(raw){reply=JSON.parse(raw)}close(){}}
 const api={passport:{getUserId:()=> 'user1'},transport:{callSdkApi:async()=>{calls++;if(failure)throw Error('failed');return{data:{}}}}};
 const mutation=`({'test.change':{command:'known-command',request(p){if(!p||typeof p.value!=='boolean')throw Error('invalid value');return {value:p.value}},project(){return {acknowledged:true,verified:false}}}})`;
 const source=template.replace('__LARK_READ_OPERATIONS__','{}').replace('__LARK_MUTATION_OPERATIONS__',mutation).replace('__LARK_LOCAL_SESSION__',JSON.stringify({port:9330,token:'test',expiresAt:Date.now()+60000}));
 vm.runInNewContext(source,{window:{LarkAPI:api},LarkAPI:api,WebSocket:Socket,setTimeout:()=>0,Date});
 await socket.onmessage({data:JSON.stringify({id:'1',operation:'test.change',params})});
 return{calls,reply};
}
test('mutation preview validates request without a native write',async()=>{const r=await execute({expectedUserId:'user1',input:{value:true}});assert.equal(r.calls,0);assert.equal(r.reply.data.apply,false);assert.deepEqual(r.reply.data.request,{value:true})});
test('mutations fail closed for wrong identity and malformed request',async()=>{for(const params of [{expectedUserId:'other',apply:true,input:{value:true}},{expectedUserId:'user1',apply:true,input:{value:'true'}}]){const r=await execute(params);assert.equal(r.calls,0);assert.equal(r.reply.ok,false)}});
test('apply requires literal true and executes only once',async()=>{const p={expectedUserId:'user1',input:{value:true}};assert.equal((await execute({...p,apply:'true'})).calls,0);const r=await execute({...p,apply:true});assert.equal(r.calls,1);assert.equal(r.reply.data.verified,false)});
test('failed native writes are never retried',async()=>{const r=await execute({expectedUserId:'user1',apply:true,input:{value:true}},{failure:true});assert.equal(r.calls,1);assert.equal(r.reply.ok,false)});

async function guardedMail(op,input,{assigned=false,mismatch=false}={}){
 let socket,reply,writes=0;
 let signatures=[{id:'sig1',name:'existing'}];
 const usages=assigned?[{address:'a@example.com',newMailSignatureId:'sig1'}]:[];
 const rules=[{ruleIdString:'r1',name:'one',isEnable:false}];
 class Socket{constructor(){socket=this}send(raw){reply=JSON.parse(raw)}close(){}}
 const api={passport:{getUserId:()=> 'user1'},transport:{callSdkApi:async(command,params)=>{
  if(command==='signatures')return{data:{signatures,signatureUsages:usages}};
  if(command.startsWith('3693|'))return{data:{rules:structuredClone(rules)}};
  writes++;
  if(op==='mail.signature-delete'&&!mismatch)signatures=[];
  if(op==='mail.rule-enable'&&!mismatch)rules[0].isEnable=params.isEnable;
  return{data:{}};
 }}};
 const reads=`({'mail.signatures':{command:'signatures',request(p){return p}}})`;
 const muts=`({${JSON.stringify(op)}:{command:'write',request(p){return p},project(){return {acknowledged:true}}}})`;
 const source=template.replace('__LARK_READ_OPERATIONS__',reads).replace('__LARK_MUTATION_OPERATIONS__',muts).replace('__LARK_LOCAL_SESSION__',JSON.stringify({port:9330,token:'test',expiresAt:Date.now()+60000}));
 vm.runInNewContext(source,{window:{LarkAPI:api},LarkAPI:api,WebSocket:Socket,setTimeout:()=>0,Date});
 await socket.onmessage({data:JSON.stringify({id:'1',operation:op,params:{expectedUserId:'user1',apply:true,input}})});
 return {reply,writes};
}
test('signature deletion rejects assigned signatures and verifies absence',async()=>{
 const input={accountId:'a',signatureId:'sig1'};
 const blocked=await guardedMail('mail.signature-delete',input,{assigned:true});assert.equal(blocked.writes,0);assert.equal(blocked.reply.ok,false);
 const passed=await guardedMail('mail.signature-delete',input);assert.equal(passed.writes,1);assert.equal(passed.reply.data.verified,true);
 const mismatch=await guardedMail('mail.signature-delete',input,{mismatch:true});assert.equal(mismatch.writes,1);assert.equal(mismatch.reply.ok,false);
});
test('rule enable requires existing rule and verifies saved state',async()=>{
 const missing=await guardedMail('mail.rule-enable',{ruleIdString:'missing',isEnable:true});assert.equal(missing.writes,0);
 const changed=await guardedMail('mail.rule-enable',{ruleIdString:'r1',isEnable:true});assert.equal(changed.reply.data.verified,true);
 const mismatch=await guardedMail('mail.rule-enable',{ruleIdString:'r1',isEnable:true},{mismatch:true});assert.equal(mismatch.reply.ok,false);assert.equal(mismatch.writes,1);
});
test('rule reorder rejects partial or foreign rule sets before writing',async()=>{
 const result=await guardedMail('mail.rule-reorder',{ruleOrder:[{ruleIdString:'foreign',order:'0'}]});assert.equal(result.writes,0);assert.equal(result.reply.ok,false);
});

async function guardedDraft(op,{existing=false,mismatch=false}={}){
 let socket,reply,writes=0;
 let drafts=existing?{d1:{chatId:'chat1',content:JSON.stringify({innerText:'existing'})}}:{};
 const reads=`({'messaging.chat-draft':{command:'fetch',request(p){return p}},'messaging.drafts':{command:'all',request(){return {}}}})`;
 const mutations=`({${JSON.stringify(op)}:{command:'write',request(p){return p},project(){return {acknowledged:true}}}})`;
 const api={passport:{getUserId:()=> 'user1'},transport:{callSdkApi:async(command,params)=>{
  if(command==='fetch'||command==='all')return{data:{entity:{drafts}}};
  writes++;
  if(!mismatch){if(op==='messaging.create-text-draft')drafts={d2:params.draft};else drafts={}}
  return{data:{draftId:'d2'}};
 }}};
 class Socket{constructor(){socket=this}send(raw){reply=JSON.parse(raw)}close(){}}
 const source=template.replace('__LARK_READ_OPERATIONS__',reads).replace('__LARK_MUTATION_OPERATIONS__',mutations).replace('__LARK_LOCAL_SESSION__',JSON.stringify({port:9330,token:'test',expiresAt:Date.now()+60000}));
 vm.runInNewContext(source,{window:{LarkAPI:api},LarkAPI:api,WebSocket:Socket,setTimeout:()=>0,Date});
 const input=op==='messaging.create-text-draft'?{draft:{chatId:'chat1',content:JSON.stringify({innerText:'fixture'})}}:{draftId:'d1'};
 await socket.onmessage({data:JSON.stringify({id:'1',operation:op,params:{expectedUserId:'user1',apply:true,input}})});
 return {reply,writes};
}
test('draft create refuses existing content and verifies map-key identity',async()=>{
 const blocked=await guardedDraft('messaging.create-text-draft',{existing:true});assert.equal(blocked.writes,0);assert.equal(blocked.reply.ok,false);
 const created=await guardedDraft('messaging.create-text-draft');assert.equal(created.reply.data.verified,true);assert.equal(created.reply.data.data.draftId,'d2');
 const missing=await guardedDraft('messaging.create-text-draft',{mismatch:true});assert.equal(missing.writes,1);assert.equal(missing.reply.ok,false);
});
test('draft deletion requires a real current draft and verifies absence',async()=>{
 const absent=await guardedDraft('messaging.delete-draft');assert.equal(absent.writes,0);
 const deleted=await guardedDraft('messaging.delete-draft',{existing:true});assert.equal(deleted.reply.data.verified,true);
 const retained=await guardedDraft('messaging.delete-draft',{existing:true,mismatch:true});assert.equal(retained.writes,1);assert.equal(retained.reply.ok,false);
});
