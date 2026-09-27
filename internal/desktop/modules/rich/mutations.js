'use strict';
function obj(v){return v && typeof v==='object'&&!Array.isArray(v)?v:{}}
function nativeId(v){if(typeof v!=='string'||!/^[1-9][0-9]{0,18}$/.test(v)||(v.length===19&&v>'9223372036854775807'))throw Error('canonical positive native ID required');return v}
function keys(o,allowed){if(Object.keys(o).some(k=>!allowed.includes(k)))throw Error('unsupported field')}
function postJSON(raw){
 if(typeof raw!=='string'||raw.length>32768)throw Error('post must be JSON text up to 32 KiB');
 const p=JSON.parse(raw); if(!p||Array.isArray(p)||typeof p!=='object')throw Error('post must be an object');keys(p,['title','paragraphs']);
 if(typeof p.title!=='string'||p.title.length>256)throw Error('title required, at most 256 characters');
 if(!Array.isArray(p.paragraphs)||!p.paragraphs.length||p.paragraphs.length>100)throw Error('1-100 paragraphs required');
 let chars=0,runs=0;
 const paragraphs=p.paragraphs.map(line=>{if(!Array.isArray(line)||!line.length)throw Error('each paragraph needs runs');return line.map(run=>{
  if(!run||Array.isArray(run)||typeof run!=='object')throw Error('run must be an object');keys(run,['text','bold','italic','underline']);
  if(typeof run.text!=='string'||!run.text.length)throw Error('run text required');chars+=run.text.length;runs++;
  const r={text:run.text};for(const k of ['bold','italic','underline']){if(run[k]!==undefined&&typeof run[k]!=='boolean')throw Error('styles must be boolean');if(run[k])r[k]=true}return r;
 })});
 if(chars>16384||runs>500||!paragraphs.flat().some(r=>r.text.trim()))throw Error('post exceeds limit or is empty');
 return {title:p.title,paragraphs};
}
function makeRich(post){
 let seq=0;const elements={},elementIds=[];
 for(const line of post.paragraphs){const pid='p'+(++seq);elementIds.push(pid);const p={tag:3,property:{paragraph:{}},childIds:[],style:{}};elements[pid]=p;
  for(const run of line){const tid='t'+(++seq),style={};if(run.bold)style['fontWeight']='bold';if(run.italic)style['fontStyle']='italic';if(run.underline)style['-lark-textDecoration']='underline';elements[tid]={tag:1,property:{text:{content:run.text}},childIds:[],style};p.childIds.push(tid)}
 }
 return {elementIds,innerText:post.paragraphs.map(p=>p.map(r=>r.text).join('')).join('\n'),elements,imageIds:[],atIds:[],anchorIds:[],mediaIds:[],docsIds:[]};
}
function parseNativePost(message){
 if(message?.type!==2)throw Error('not a POST message');const content=obj(message.content),rich=obj(content.richText),elements=obj(rich.elements);
 if(['imageIds','atIds','anchorIds','mediaIds','docsIds'].some(k=>rich[k]?.length)||Object.keys(elements).length>1000)throw Error('post contains unsupported embedded content');
 if(typeof content.title!=='string'||!Array.isArray(rich.elementIds)||rich.elementIds.length>100)throw Error('post structure unavailable');
 const visited=new Set();
 const paragraphs=rich.elementIds.map(pid=>{const p=elements[pid];if(!p||p.tag!==3||!Array.isArray(p.childIds))throw Error('unsupported paragraph');visited.add(pid);return p.childIds.map(tid=>{
  if(visited.has(tid))throw Error('duplicate/cyclic rich element');visited.add(tid);const t=elements[tid];if(!t||t.tag!==1||t.childIds?.length)throw Error('unsupported rich element');
  const style=obj(t.style);keys(style,['fontWeight','fontStyle','-lark-textDecoration']);
  if(style['fontWeight']!==undefined&&style['fontWeight']!=='bold')throw Error('unsupported weight');if(style['fontStyle']!==undefined&&style['fontStyle']!=='italic')throw Error('unsupported font style');if(style['-lark-textDecoration']!==undefined&&style['-lark-textDecoration']!=='underline')throw Error('unsupported decoration');
  const r={text:t.property?.text?.content};if(style['fontWeight'])r.bold=true;if(style['fontStyle'])r.italic=true;if(style['-lark-textDecoration'])r.underline=true;return r;
 })});
 if(visited.size!==Object.keys(elements).length)throw Error('unreferenced rich elements');
 return postJSON(JSON.stringify({title:content.title,paragraphs}));
}
function randomCID(){return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g,c=>{const n=Math.floor(Math.random()*16);return(c==='x'?n:(n&3)|8).toString(16)})}

function request(p,edit){keys(p,edit?['chatId','messageId','post','expectedPost']:['chatId','post']);const chatId=nativeId(p.chatId),post=postJSON(p.post),content={title:post.title,richText:makeRich(post)};
 if(edit){postJSON(p.expectedPost);return {msgId:nativeId(p.messageId),chatId,cid:randomCID(),type:2,content}}
 return {cid:randomCID(),channel:{id:chatId,type:1},type:2,content,shouldNotify:true};
}
module.exports={operations:{
 'messaging.send-self-post':{command:'2003|im.v1.SendMessageRequest|im.v1.SendMessageResponse|1|SEND_MESSAGE',request:p=>request(p,false),project(r){const result={};try{result.messageId=nativeId(r?.messageId)}catch{}return result}},
 'messaging.edit-post':{command:'4520|im.v1.EditMessageRequest|im.v1.EditMessageResponse|1|EDIT_MESSAGE',request:p=>request(p,true),project(){return {}}}
}};
