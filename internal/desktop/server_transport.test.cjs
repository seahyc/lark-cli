const {test}=require('node:test');const assert=require('node:assert/strict');const fs=require('node:fs');const vm=require('node:vm');
test('curated server read uses native server transport without credential extraction',async()=>{
 let peer,reply;const calls=[];
 const api={passport:{getUserId:()=> 'u'},transport:{callSdkApi:async()=>({data:{}}),callServerApi:async(...args)=>{calls.push(args);return{data:{count:3}}}}};
 class Socket{constructor(){peer=this}send(s){reply=JSON.parse(s)}close(){}}
 const ops=`({'mail.account-metadata':{command:'metadata',request(){return {}},project(){return {accounts:[{mailAccountId:'u',isSelected:true,isShared:false}]}}},'mail.server-read':{transport:'server',command:'123|TestRequest|TestResponse',request(){return {size:3}},project(r){return {count:r.count}}}})`;
 const source=fs.readFileSync(__dirname+'/mail_bridge.js','utf8').replace('__LARK_READ_OPERATIONS__',ops).replace('__LARK_MUTATION_OPERATIONS__','{}').replace('__LARK_LOCAL_SESSION__',JSON.stringify({port:9330,token:'test',expiresAt:Date.now()+60000}));
 vm.runInNewContext(source,{window:{LarkAPI:api},LarkAPI:api,WebSocket:Socket,setTimeout:()=>0,Date});
 await peer.onmessage({data:JSON.stringify({id:'1',operation:'mail.server-read',params:{}})});
 assert.equal(reply.ok,true);assert.equal(reply.data.data.count,3);assert.deepEqual(JSON.parse(JSON.stringify(calls)),[['123|TestRequest|TestResponse',{size:3},{parentContextId:'',collectTrace:false}]]);
});
