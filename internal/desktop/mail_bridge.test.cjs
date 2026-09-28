const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const source=fs.readFileSync(__dirname+'/mail_bridge.js','utf8').replace('__LARK_READ_OPERATIONS__','{}').replace('__LARK_MUTATION_OPERATIONS__','{}').replace('__LARK_LOCAL_SESSION__',JSON.stringify({port:9330,token:'test',expiresAt:Date.now()+60000}));
const base={ruleIdString:'1',name:'test',isEnable:true,condition:{matchType:1,items:[{type:1,operator:5,input:'sender@example.com'}]},action:{items:[{type:3},{type:1},{type:12,input:'old@example.com',authStatus:2,enableAutoTransfer:true}]}};
async function execute(params, {readbackMismatch=false,operation='updateRule'}={}) {
 let peer, current=structuredClone(base),writes=0,reply;
 class Socket {constructor(){peer=this;}send(raw){reply=JSON.parse(raw);}close(){}}
 const api={passport:{getUserId:()=> 'user1'},transport:{callSdkApi:async(cmd,p)=>{
   if(cmd.startsWith('3693|'))return {data:{rules:[current]}};
   if(cmd.startsWith('3766|'))return {data:{emails:['old@example.com','new@example.com']}};
   if(cmd.startsWith('3691|')){writes++;current=structuredClone(p.rule);if(readbackMismatch)current.name='unexpected';return{data:{rule:current}};}
   throw Error('unexpected command');
 }}};
 const context={window:{LarkAPI:api},LarkAPI:api,WebSocket:Socket,setTimeout:()=>0,Date,console};
 vm.runInNewContext(source,context);
 await peer.onmessage({data:JSON.stringify({id:'1',operation,params})});
 return{reply,writes,current};
}
const params=()=>({expectedUserId:'user1',expectedRule:structuredClone(base),recipients:['new@example.com']});
test('forwarding write preserves conditions and non-forward actions',async()=>{const r=await execute(params());assert.equal(r.reply.ok,true);assert.equal(r.writes,1);assert.deepEqual(r.current.condition,base.condition);assert.deepEqual(r.current.action.items.slice(0,2),base.action.items.slice(0,2));assert.equal(r.reply.data.verified,true);});
test('wrong identity does not write',async()=>{const p=params();p.expectedUserId='other';const r=await execute(p);assert.equal(r.reply.ok,false);assert.equal(r.writes,0);});
test('unverified recipient does not write',async()=>{const p=params();p.recipients=['unknown@example.com'];const r=await execute(p);assert.equal(r.reply.ok,false);assert.equal(r.writes,0);});
test('stale preview does not write',async()=>{const p=params();p.expectedRule.name='stale';const r=await execute(p);assert.equal(r.reply.ok,false);assert.equal(r.writes,0);});
test('same recipients are a no-op',async()=>{const p=params();p.recipients=['old@example.com'];const r=await execute(p);assert.equal(r.reply.ok,true);assert.equal(r.writes,0);assert.equal(r.reply.data.changed,false);});
test('same recipients with different case are a no-op',async()=>{const p=params();p.recipients=['OLD@Example.COM'];const r=await execute(p);assert.equal(r.reply.ok,true);assert.equal(r.writes,0);assert.equal(r.reply.data.changed,false);});
test('verified recipients are case-insensitive',async()=>{const p=params();p.recipients=['NEW@Example.COM'];const r=await execute(p);assert.equal(r.reply.ok,true);assert.equal(r.writes,1);});
test('duplicate recipients are rejected',async()=>{const p=params();p.recipients=['new@example.com','NEW@example.com'];const r=await execute(p);assert.equal(r.reply.ok,false);assert.equal(r.writes,0);});
test('existing item is reused with new case',async()=>{const p=params();p.recipients=['OLD@Example.COM'];const r=await execute(p);assert.equal(r.reply.ok,true);assert.equal(r.writes,0);assert.equal(r.reply.data.changed,false);});
test('mismatching readback is failure without retry',async()=>{const r=await execute(params(),{readbackMismatch:true});assert.equal(r.reply.ok,false);assert.equal(r.writes,1);});

test('rule field patch preserves conditions and all forwarding settings',async()=>{const r=await execute({expectedUserId:'user1',expectedRule:structuredClone(base),patch:{name:'renamed',ignoreTheRestOfRules:true}},{operation:'updateRuleFields'});assert.equal(r.writes,1);assert.equal(r.reply.data.verified,true);assert.deepEqual(r.current.condition,base.condition);assert.deepEqual(r.current.action,base.action)});
test('rule patch rejects arbitrary fields and stale snapshots',async()=>{for(const p of [{expectedUserId:'user1',expectedRule:structuredClone(base),patch:{action:{items:[]}}},{expectedUserId:'user1',expectedRule:{...base,name:'stale'},patch:{name:'new'}}]){const r=await execute(p,{operation:'updateRuleFields'});assert.equal(r.writes,0);assert.equal(r.reply.ok,false)}});
