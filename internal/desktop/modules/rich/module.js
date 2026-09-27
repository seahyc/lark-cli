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
// Generated from `basic.v1.Content` in the shipped `pb.desc`, which is the
// native MGET content schema (not the similarly named server-side PostContent).
// The compact codes preserve protobuf field defaults exactly.  In particular,
// generated JS represents int64/uint64 zero as "0", and FileType / File.Source
// have non-zero enum defaults (1).  Unknown fields are never defaulted.
const generatedContent=Object.freeze({
title:'s',text:'s',image:'m',isOriginSource:'b0',originSize:'l',imageUploadId:'s',key:'s',name:'s',size:'l',mime:'s',fileUploadId:'s',template:'s',values:'o',duration:'n0',deprecated:'e1',systemType:'e0',event:'m',contentValues:'o',systemTypeValue:'n0',i18nTitleKey:'s',systemContentValues:'o',itemActions:'o',systemMessageVersion:'n0',voice2text:'s',hideVoice2text:'b1',originSenderId:'l',localUploadId:'s',originTosKey:'s',originSenderName:'s',isFriend:'b0',originSenderIdStr:'s',isAudioRecognizeFinish:'b0',audio2textStartTime:'n0',isAudioWithText:'b0',urlPreviewHangPointMap:'o',isGroupAnnouncement:'b0',docEntity:'m',filePath:'s',fileSource:'e1',namespace:'s',isInMyNutStore:'b0',cacheFilePath:'s',lanTransStatus:'e0',transIdentity:'e0',lanTransStatusText:'s',lanSenderDeviceId:'s',lanReceiverDeviceId:'s',isTryUseLanTrans:'b0',isAbleLanTrans:'b0',fileUri:'s',isEncrypted:'b0',riskObjectKeys:'a',fileAbility:'e0',filePermission:'e0',fileLastUpdateUserId:'l',fileLastUpdateTimeMs:'l',localStatus:'m',filePreviewStage:'e0',attachments:'a',fileAttachments:'a',shareChatId:'s',joinToken:'s',expireTime:'l',shareChatLastState:'n0',width:'n0',height:'n0',cryptoToken:'s',mergeForwardContent:'m',transmitInfo:'m',previewUrls:'a',richText:'m',calendarContent:'m',shareCalendarEventContent:'m',generalCalendarContent:'m',textUrlContents:'a',cardContent:'m',locationContent:'m',e2eeFromId:'s',e2eeToId:'s',triggerId:'s',vcFromId:'s',vcToId:'s',vcDurationTime:'s',vcMeetingId:'s',vcPasscode:'s',vcFeedbackStatus:'s',vcFromFeedbackStatus:'s',vcToFeedbackStatus:'s',isVoiceCall:'b0',mediaContent:'m',hongbaoContent:'m',hongbaoSystemContent:'m',systemExtraContent:'m',videochatContent:'m',resourceUrls:'a',fsUnit:'s',stickerInfo:'m',stickerId:'s',stickerSetId:'s',shareUserCardInfo:'m',shareChatterId:'s',abbreviation:'m',typedElementRefs:'o',todoOperationContent:'m',lingoOption:'m',voteContent:'m',folderArchiveKey:'s',contentReferences:'a',includeCleanedResources:'b0',cleanResourceKeyScene:'o',doubaoContent:'m',isUnknownDoubaoContent:'b0'
});
function generatedDefault(value,kind){
 // A nested message's protobuf default is absent.  Some native decode paths
 // materialize that as {}, but a populated object is meaningful and rejected.
 switch(kind){
 case 's':return value===''; case 'b0':return value===false; case 'b1':return value===true;
 case 'n0':case 'e0':return value===0; case 'e1':return value===1;
 case 'l':return value==='0'; case 'a':return Array.isArray(value)&&value.length===0;
 case 'o':case 'm':return !!value&&typeof value==='object'&&!Array.isArray(value)&&Object.keys(value).length===0;
 default:return false;
 }
}
function postPlainText(post){return post.paragraphs.map(line=>line.map(run=>run.text).join('')).join('\n')}
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
 const normalized=postJSON(JSON.stringify({title:content.title,paragraphs})),plainText=postPlainText(normalized);
 for(const [key,value] of Object.entries(content)){
  if(key==='title'||key==='richText')continue;
  if(key==='text'){if(value==='')continue;if(typeof value!=='string'||value!==plainText)throw Error('POST plain text differs from rich text');continue}
  if(!generatedDefault(value,generatedContent[key]))throw Error('unsupported meaningful POST content field: '+key+' ('+typeof value+')');
 }
 return normalized;
}
function randomCID(){return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g,c=>{const n=Math.floor(Math.random()*16);return(c==='x'?n:(n&3)|8).toString(16)})}
function boundedKeys(value,limit=16){const all=Object.keys(obj(value));return {keys:all.slice(0,limit),truncated:all.length>limit}}
function knownStyleValues(style){
 const out={};for(const key of ['fontWeight','fontStyle','-lark-textDecoration']){
  const value=obj(style)[key];if(typeof value==='string'&&['bold','normal','italic','underline','none','line-through'].includes(value))out[key]=value;
 }return out;
}
function postStructureDiagnostic(message){
 const content=obj(message?.content),rich=obj(content.richText),elements=obj(rich.elements),ids=Object.keys(elements),schemas=[],seen=new Set();
 for(const id of ids.slice(0,100)){
  const element=obj(elements[id]),property=obj(element.property),style=obj(element.style);
  const schema={tag:typeof element.tag==='number'?element.tag:null,elementFields:boundedKeys(element),styleFields:boundedKeys(style),styleValues:knownStyleValues(style),propertyFields:boundedKeys(property),textFields:boundedKeys(property.text)};
  const fingerprint=JSON.stringify(schema);if(!seen.has(fingerprint)){seen.add(fingerprint);schemas.push(schema);if(schemas.length===16)break}
 }
 return {contentFields:boundedKeys(content),richFields:boundedKeys(rich),elementCount:ids.length,elementSchemas:schemas,sampledElements:Math.min(ids.length,100),elementSchemasTruncated:ids.length>100||schemas.length===16};
}

module.exports={operations:{'messaging.post-message':{
 command:'1|im.v1.MGetMessagesRequest|im.v1.MGetMessagesResponse|1|MGET_MESSAGES',
 request(p){keys(p,['messageId']);return {messageIds:[nativeId(p.messageId)]}},
 project(r,p){const m=obj(obj(r).entity).messages?.[p.messageId];if(!m)return {found:false};const result={found:true,chatId:m.chatId,senderId:m.fromId,type:m.type};try{result.post=JSON.stringify(parseNativePost(m));result.editable=true}catch(error){result.editable=false;result.diagnostic={reason:error instanceof Error?error.message:'post parse failed',structure:postStructureDiagnostic(m)}}return result},
 normalize:postJSON
}}};
