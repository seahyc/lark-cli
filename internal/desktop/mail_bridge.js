(() => {
  'use strict';
  const cfg = __LARK_LOCAL_SESSION__;
  const readOperations = __LARK_READ_OPERATIONS__;
  const mutationOperations = __LARK_MUTATION_OPERATIONS__;
  const mailFixtures = new Map();
  const signatureFixtures = new Map();
  const scheduledFixtures = new Map();
  if (window.__larkLocalMailBridge) return;
  window.__larkLocalMailBridge = true;
  const commands = {
    listRules: '3693|email.client.v1.MailGetRulesRequest|email.client.v1.MailGetRulesResponse|1|MAIL_GET_RULES',
    verifiedEmails: '3766|email.client.v1.MailGetAutoTransferEmailRequest|email.client.v1.MailGetAutoTransferEmailResponse|1|MAIL_GET_AUTO_TRANSFER_EMAIL_REQUEST',
    updateRule: '3691|email.client.v1.MailUpdateRuleRequest|email.client.v1.MailUpdateRuleResponse|1|MAIL_UPDATE_RULE'
  };
  async function call(op, params) {
    const r = await LarkAPI.transport.callSdkApi(commands[op], params, {extendParams: {}, useFast: false});
    return r.data;
  }
  async function invoke(descriptor, params) {
    if (descriptor.transport === 'shell-navigation') {
      if (descriptor.command !== 'ShellAPI.app.navigation.getNavigationInfo') throw new Error('Unsupported shell operation');
      const navigation = globalThis.ShellAPI?.app?.navigation;
      if (typeof navigation?.getNavigationInfo !== 'function') throw new Error('Shell navigation unavailable in this desktop surface');
      return {data: await navigation.getNavigationInfo()};
    }
    if (descriptor.transport === 'server') {
      if (!LarkAPI.transport.callServerApi) throw new Error('Native server transport unavailable');
      if(['900034|messages.PatchScheduleMessageRequest|messages.PatchScheduleMessageResponse','64|chats.PullChatsByIdsRequest|chats.PullChatsByIdsResponse','60|chats.PatchChatSettingRequest|chats.PatchChatSettingResponse'].includes(descriptor.command)) return LarkAPI.transport.callServerApi(descriptor.command,params,{parentContextId:'',collectTrace:false});
      const metadata = readOperations['mail.account-metadata'];
      if (!metadata) throw new Error('Mailbox identity guard unavailable');
      const current = await LarkAPI.transport.callSdkApi(metadata.command,metadata.request({}),{extendParams:{},useFast:false});
      const selected = metadata.project(current.data).accounts.filter(a=>a.isSelected);
      const userId = String(await LarkAPI.passport.getUserId());
      if (selected.length!==1 || selected[0].mailAccountId!==userId || selected[0].isShared) throw new Error('Server mail operations require the primary mailbox selected');
      return LarkAPI.transport.callServerApi(descriptor.command,params,{parentContextId:'',collectTrace:false});
    }
    if (descriptor.transport && descriptor.transport !== 'sdk') throw new Error('Unsupported transport');
    return LarkAPI.transport.callSdkApi(descriptor.command,params,{extendParams:{},useFast:false});
  }
  async function run(op, p) {
    const userId = String(await LarkAPI.passport.getUserId());
    if (!userId || userId === 'undefined') throw new Error('Not signed in');
    if (op === 'identity') return {userId};
    if (Object.prototype.hasOwnProperty.call(readOperations, op)) {
      const descriptor = readOperations[op];
      const params = descriptor.request(p);
      const response = await invoke(descriptor, params);
      return {userId, data:descriptor.project(response.data,p)};
    }
    if (Object.prototype.hasOwnProperty.call(mutationOperations, op)) {
      if (p.expectedUserId !== userId) throw new Error('Signed-in identity changed');
      const descriptor = mutationOperations[op];
      let params;
      let before;
      if (op === 'workspacenext.modify-navigation-order') throw new Error('Navigation reorder requires a verified ShellAPI-to-native identity mapping; currently unavailable');
      if (op.startsWith('mailnext.draft-fixture-')) {
        const a = readOperations['mail.account-metadata'];
        const r = await invoke(a,a.request({}));
        const selected = a.project(r.data).accounts.filter(x=>x.isSelected);
        if (selected.length!==1 || selected[0].mailAccountId!==userId || selected[0].isShared) throw new Error('Draft fixtures require the selected primary mailbox');
      }
      if (op === 'mailnext.draft-fixture-delete') {
        const d = readOperations['mailnext.draft-fixture-metadata'];
        const r = await invoke(d,d.request({draftId:p.input.messageId}));
        params = descriptor.requestFromSnapshot(r.data,p.input);
      } else if (op === 'mailnext.draft-fixture-update') {
        const owned = mailFixtures.get(p.input.draftId);
        if (!owned || owned.userId!==userId || owned.threadId!==p.input.threadId || owned.replyMessageId!==p.input.replyMessageId) throw new Error('Only this session newly-created mail fixture may be updated');
        const d = readOperations['mailnext.draft-fixture-metadata'];
        const r = await invoke(d,d.request({draftId:owned.draftId}));
        before = r.data;
        params = descriptor.requestFromSnapshot(before,p.input);
      } else if (op === 'mailnext.signature-fixture-update-text') {
        const owned = signatureFixtures.get(p.input.signatureId);
        if (!owned || owned.userId!==userId || owned.accountId!==p.input.accountId) throw new Error('Only this session created signature fixture may be updated');
        const d = readOperations['mail.signatures'];
        const r = await invoke(d,d.request({accountId:p.input.accountId}));
        before=r.data;
        params=descriptor.requestFromSnapshot(before,p.input);
      } else params = descriptor.request(p.input);
      if (op === 'triage.favorite-add' || op === 'triage.favorite-delete') {
        const f = readOperations['messaging.favorites'];
        const r = await invoke(f,f.request({count:100,time:0}));
        before = r.data;
        if (before?.hasMore) throw new Error('Favorite fixture guard requires complete bounded list');
        const favorites = before?.entity?.favorites || {};
        if (op === 'triage.favorite-delete' && !favorites[p.input.favoriteId]) throw new Error('Favorite not found in current account');
        if (op === 'triage.favorite-add') {
          if (Object.values(favorites).some(x=>x.content?.messageId===p.input.messageId)) throw new Error('Message already saved; refusing duplicate fixture');
          const d = readOperations['messaging.text-message'];
          const r = await invoke(d,d.request({messageId:p.input.messageId}));
          const m = r.data?.entity?.messages?.[p.input.messageId];
          if (!m || m.chatId!==p.input.chatId || String(m.fromId)!==userId) throw new Error('Favorite fixture must be your own message in the exact chat');
          const self = readOperations['messaging.p2p-chat'];
          const c = await invoke(self,self.request({userId}));
          if(c.data?.chatterId2chat?.[userId]?.id!==p.input.chatId) throw new Error('Favorite fixture requires self chat');
        }
      }
      if (op === 'messaging.edit-post') {
        const d = readOperations['messaging.post-message'];
        const r = await invoke(d,d.request({messageId:params.msgId}));
        const post = d.project(r.data,{messageId:params.msgId});
        if (!post.editable || post.senderId!==userId || post.chatId!==p.input.chatId || post.post!==JSON.stringify(d.normalize(p.input.expectedPost))) throw new Error('Own supported POST and exact current post snapshot required');
      }
      if (op === 'messaging.send-self-text' || op === 'messaging.send-self-post' || op === 'triage.schedule-self-text' || op === 'triage.schedule-cancel-text' || op === 'messaging.recall-self-text' || op === 'inbox.set-self-chat-mute') {
        const d = readOperations['messaging.p2p-chat'];
        if (!d) throw new Error('Self-chat lookup unavailable');
        const r = await invoke(d,d.request({userId}));
        const selfChatId = r.data?.chatterId2chat?.[userId]?.id;
        if (!selfChatId || selfChatId !== p.input.chatId) throw new Error('Send requires the signed-in user self-chat; no fallback destination');
      }
      if (op === 'inbox.set-self-chat-mute') {
        const d=readOperations['inbox.chat-mute-state'];
        const r=await invoke(d,d.request({chatId:p.input.chatId}));
        before=d.project(r.data);
        if(before.chatId!==p.input.chatId || before.muted!==p.input.expectedMuted) throw new Error('Self chat mute state changed; refresh snapshot');
      }
      if (op === 'triage.schedule-self-text') {
        const when=Number(params.scheduleTime), now=Math.floor(Date.now()/1000);
        if(!Number.isSafeInteger(when)||when<now+3600||when>now+366*86400) throw new Error('Scheduled fixture must be 1 hour to 366 days in the future (Unix seconds)');
      }
      if (op === 'triage.schedule-cancel-text') {
        const d=readOperations['triage.scheduled-items'];
        const r=await invoke(d,d.request({chatId:p.input.chatId}));
        const items=d.project(r.data).items;
        const found=items.find(x=>x.messageId===p.input.messageId);
        if(!found || found.chatId!==p.input.chatId || found.status!==1 || found.type!==4 || found.scheduleTime!==p.input.scheduleTime || (found.text ?? '')!==p.input.expectedText || found.senderId!==userId) throw new Error('Pending self TEXT schedule does not match expected snapshot');
      }
      if (op === 'messaging.recall-self-text') {
        const d = readOperations['messaging.text-message'];
        if (!d) throw new Error('Text-message lookup unavailable');
        const r = await invoke(d,d.request({messageId:p.input.messageId}));
        const message = r.data?.entity?.messages?.[p.input.messageId];
        const richText = message?.content?.richText;
        const value = d.project(r.data,{messageId:p.input.messageId});
        if (!message || message.isRecalled || String(message.fromId)!==userId || message.chatId!==p.input.chatId || message.type!==4 || value.text!==p.input.expectedText) throw new Error('Recall requires the current own self-chat plain-text snapshot');
        if (!richText || ['imageIds','atIds','anchorIds','mediaIds','docsIds'].some(key=>richText[key]?.length) || Object.values(richText.elements || {}).some(element=>element.tag!==1)) throw new Error('Recall requires plain text without mentions, links, or media');
      }
      if (op === 'messaging.edit-text') {
        const d = readOperations['messaging.text-message'];
        const r = await invoke(d,d.request({messageId:params.msgId}));
        const message = r.data?.entity?.messages?.[params.msgId];
        if (!message || String(message.fromId)!==userId || message.chatId!==p.input.chatId || message.type!==4) throw new Error('Edit requires your own plain-text message in the exact chat');
        const richText = message.content?.richText;
        if (!richText || d.project({entity:{messages:{[params.msgId]:message}}}).text!==p.input.expectedText) throw new Error('Original text changed or is unavailable; refresh before editing');
        if ((richText.innerText?.length || 0)>16384 || Object.values(richText.elements || {}).reduce((n,e)=>n+(e.property?.text?.content?.length || 0),0)>16384) throw new Error('Original text exceeds edit limit');
        if (['imageIds','atIds','anchorIds','mediaIds','docsIds'].some(key=>richText[key]?.length) || Object.values(richText.elements || {}).some(element=>element.tag!==1)) throw new Error('Only plain text without mentions, links or media can be edited');
      }
      if (op === 'messaging.create-text-draft') {
        const d = readOperations['messaging.drafts'];
        const r = await LarkAPI.transport.callSdkApi(d.command,d.request({}),{extendParams:{},useFast:false});
        before = r.data;
        if (Object.values(before?.entity?.drafts || {}).some(v=>v.chatId===params.draft.chatId)) throw new Error('Chat already has a draft; refusing to overwrite it');
      }
      if (op === 'messaging.delete-draft') {
        const d = readOperations['messaging.drafts'];
        const r = await LarkAPI.transport.callSdkApi(d.command,d.request({}),{extendParams:{},useFast:false});
        if (!r.data?.entity?.drafts?.[params.draftId]) throw new Error('Draft not found in current account');
      }
      if (op === 'mail.rule-enable' || op === 'mail.rule-reorder') {
        before = await call('listRules',{version:'3'});
        const ids = (before.rules || []).map(r=>r.ruleIdString);
        if (op === 'mail.rule-enable' && !ids.includes(params.ruleIdString)) throw new Error('Rule not found in current mailbox');
        if (op === 'mail.rule-reorder' && JSON.stringify([...ids].sort()) !== JSON.stringify(params.ruleOrder.map(r=>r.ruleIdString).sort())) throw new Error('Reorder must include every current rule exactly once');
      }
      if (op === 'mail.signature-usage-update' || op === 'mail.signature-delete' || op === 'mail.signature-create-text') {
        const descriptor = readOperations['mail.signatures'];
        const r = await LarkAPI.transport.callSdkApi(descriptor.command,descriptor.request({accountId:params.accountId}),{extendParams:{},useFast:false});
        before = r.data;
        const ids = (before.signatures || []).map(s=>s.id);
        if (op === 'mail.signature-usage-update') {
          const usage = params.signatureUsage;
          if (!(before.signatureUsages || []).some(u=>u.address===usage.address)) throw new Error('Signature address not found in account');
          if (![usage.newMailSignatureId,usage.replyMailSignatureId].every(id=>ids.includes(id))) throw new Error('Signature not found in account');
        }
        if (op === 'mail.signature-delete') {
          if (!ids.includes(params.signatureId)) throw new Error('Signature not found in account');
          if ((before.signatureUsages || []).some(u=>u.newMailSignatureId===params.signatureId || u.replyMailSignatureId===params.signatureId)) throw new Error('Signature is assigned to an address; reassign it before deleting');
        }
      }
      if (p.apply !== true) return {userId,apply:false,operation:op,request:params};
      let response;
      if (op === 'messaging.send-self-text' || op === 'messaging.send-self-post') {
        const prepared = await LarkAPI.transport.callSdkApi('1001|im.v1.CreateQuasiMessageRequest|im.v1.CreateQuasiMessageResponse|1|CREATE_QUASI_MESSAGE',params,{extendParams:{},useFast:false});
        if (!prepared.data?.cid || prepared.data.cid !== params.cid) throw new Error('Prepared message identity differs; inspect self-chat before retrying');
        response = await invoke(descriptor,{cid:prepared.data.cid});
      } else if(op === 'triage.schedule-self-text') {
        if(!LarkAPI.transport.callServerApi) throw new Error('Native scheduled-message persistence transport unavailable');
        const staged=await invoke(descriptor,params);
        response=await LarkAPI.transport.callServerApi(descriptor.persistCommand,descriptor.persistRequest(params,staged.data),{parentContextId:'',collectTrace:false});
        const m=response.data?.message;
        if(!m?.messageId) throw new Error('Schedule persistence returned no identity; inspect scheduled messages before retrying');
        scheduledFixtures.set(m.messageId,{userId,chatId:p.input.chatId,text:p.input.text,scheduleTime:p.input.scheduleTime});
      } else response = await invoke(descriptor,params);
      let verified = false;
      if (op === 'mail.rule-enable' || op === 'mail.rule-reorder') {
        const after = await call('listRules',{version:'3'});
        if (op === 'mail.rule-enable') {
          const saved = (after.rules || []).find(r=>r.ruleIdString===params.ruleIdString);
          if (!saved || !!saved.isEnable !== params.isEnable) throw new Error('Rule enable readback differs; inspect before retrying');
          const old = before.rules.find(r=>r.ruleIdString===params.ruleIdString);
          if (JSON.stringify({...old,isEnable:params.isEnable}) !== JSON.stringify(saved)) throw new Error('Other rule fields changed; inspect before retrying');
        } else if (JSON.stringify((after.rules || []).map(r=>r.ruleIdString)) !== JSON.stringify(params.ruleOrder.map(r=>r.ruleIdString))) throw new Error('Rule order readback differs; inspect before retrying');
        verified = true;
      }
      if (op === 'mail.signature-usage-update') {
        const d = readOperations['mail.signatures'];
        const r = await LarkAPI.transport.callSdkApi(d.command,d.request({accountId:params.accountId}),{extendParams:{},useFast:false});
        const usage = (r.data.signatureUsages || []).find(u=>u.address===params.signatureUsage.address);
        if (!usage || usage.newMailSignatureId!==params.signatureUsage.newMailSignatureId || usage.replyMailSignatureId!==params.signatureUsage.replyMailSignatureId) throw new Error('Signature usage readback differs; inspect before retrying');
        verified = true;
      }
      if (op === 'mail.signature-create-text' || op === 'mail.signature-delete') {
        const d = readOperations['mail.signatures'];
        const r = await LarkAPI.transport.callSdkApi(d.command,d.request({accountId:params.accountId}),{extendParams:{},useFast:false});
        const saved = r.data.signatures || [];
        if (op === 'mail.signature-create-text') {
          const id = response.data?.signature?.id;
          const found = saved.find(s=>s.id===id);
          if (!id || !found || found.name!==params.signature.name || found.templateHtml!==params.signature.templateHtml) throw new Error('Created signature readback differs; inspect signatures before retrying');
          signatureFixtures.set(id,{userId,accountId:params.accountId});
        } else if (saved.some(s=>s.id===params.signatureId)) throw new Error('Deleted signature is still present; inspect before retrying');
        verified = true;
      }
      if (op === 'messaging.create-text-draft') {
        const d = readOperations['messaging.drafts'];
        const r = await LarkAPI.transport.callSdkApi(d.command,d.request({}),{extendParams:{},useFast:false});
        const drafts = [r.data?.draft,...Object.entries(r.data?.entity?.drafts || {}).map(([id,value])=>({...value,id:value.id || id}))].filter(Boolean);
        const expectedText = JSON.parse(params.draft.content).innerText;
        const saved = drafts.find(v=>{try{return v.chatId===params.draft.chatId && JSON.parse(v.content).innerText===expectedText}catch{return false}});
        if (!saved?.id) throw new Error('Draft readback differs; inspect current draft before retrying');
        return {userId,acknowledged:true,verified:true,data:{draftId:saved.id,chatId:saved.chatId}};
      }
      if (op === 'messaging.delete-draft') {
        const d = readOperations['messaging.drafts'];
        const r = await LarkAPI.transport.callSdkApi(d.command,d.request({}),{extendParams:{},useFast:false});
        if (r.data?.entity?.drafts?.[params.draftId]) throw new Error('Draft remains after deletion; inspect before retrying');
        verified = true;
      }
      if (op === 'messaging.send-self-text' || op === 'messaging.send-self-post') {
        const messageId = descriptor.project(response.data).messageId;
        if (!messageId) throw new Error('Send returned no message identity; inspect self-chat before retrying');
        const isPost = op === 'messaging.send-self-post';
        const d = readOperations[isPost?'messaging.post-message':'messaging.text-message'];
        for (let attempt=0;attempt<4;attempt++) {
          if (attempt) await new Promise(resolve=>setTimeout(resolve,250));
          const r = await invoke(d,d.request({messageId}));
          const saved = r.data?.entity?.messages?.[messageId];
          const value = d.project(r.data,{messageId});
          const matches = isPost ? value.post===JSON.stringify(d.normalize(p.input.post)) : value.text===p.input.text;
          if (saved && String(saved.fromId)===userId && saved.chatId===p.input.chatId && saved.type===(isPost?2:4) && matches) {verified=true;break;}
        }
        if (!verified) throw new Error('Sent text readback differs; inspect self-chat before retrying');
      }
      if (op === 'messaging.recall-self-text') {
        const d = readOperations['messaging.text-message'];
        for (let attempt=0;attempt<4;attempt++) {
          if (attempt) await new Promise(resolve=>setTimeout(resolve,250));
          const r = await invoke(d,d.request({messageId:p.input.messageId}));
          const saved = r.data?.entity?.messages?.[p.input.messageId];
          // isRecalled is the shipped Message record's recall-state field and
          // drives the recalled-message renderer; absence is not success.
          if (saved && saved.isRecalled===true) { verified=true; break; }
        }
        if (!verified) throw new Error('Recall readback does not confirm recalled state; inspect message before retrying');
        return {userId,acknowledged:true,verified:true,data:{messageId:p.input.messageId,chatId:p.input.chatId}};
      }
      if (op === 'messaging.edit-text') {
        const d = readOperations['messaging.text-message'];
        for (let attempt=0;attempt<4;attempt++) {
          if (attempt) await new Promise(resolve=>setTimeout(resolve,250));
          const r = await invoke(d,d.request({messageId:params.msgId}));
          const saved = r.data?.entity?.messages?.[params.msgId];
          if (saved && String(saved.fromId)===userId && saved.chatId===p.input.chatId && d.project({entity:{messages:{[params.msgId]:saved}}}).text===p.input.text) {verified=true;break;}
        }
        if (!verified) throw new Error('Edited text readback differs; inspect message before retrying');
      }
      if (op === 'messaging.edit-post') {
        const d = readOperations['messaging.post-message'];
        for (let attempt=0;attempt<4;attempt++) {
          if(attempt) await new Promise(resolve=>setTimeout(resolve,250));
          const r = await invoke(d,d.request({messageId:params.msgId}));
          const post = d.project(r.data,{messageId:params.msgId});
          if(post.editable && post.senderId===userId && post.chatId===p.input.chatId && post.post===JSON.stringify(d.normalize(p.input.post))) {verified=true;break;}
        }
        if(!verified) throw new Error('Edited POST readback differs; inspect before retrying');
      }
      if (op === 'mailnext.draft-fixture-create') {
        const created = descriptor.project(response.data);
        if(!created.draftId || !created.threadId) throw new Error('Created fixture has no identity; inspect drafts before retrying');
        mailFixtures.set(created.draftId,{...created,userId});
        const d = readOperations['mailnext.draft-fixture-metadata'];
        const r = await invoke(d,d.request({draftId:created.draftId}));
        if(r.data?.draft?.id!==created.draftId) throw new Error('Draft identity readback differs; inspect before retrying');
        verified=true;
      }
      if (op === 'mailnext.draft-fixture-update') {
        const d = readOperations['mailnext.draft-fixture-metadata'];
        for(let attempt=0;attempt<4;attempt++){
          if(attempt) await new Promise(resolve=>setTimeout(resolve,250));
          const r = await invoke(d,d.request({draftId:p.input.draftId}));
          if(r.data?.draft?.id===p.input.draftId && r.data.draft.subject===p.input.subject && r.data.draft.bodyHtml===p.input.bodyHtml) {verified=true;break;}
        }
        if(!verified) throw new Error('Mail fixture update readback differs; inspect before retrying');
      }
      if (op === 'triage.favorite-add' || op === 'triage.favorite-delete') {
        const d = readOperations['messaging.favorites'];
        let favoriteId;
        const oldIds = before.favoritesIds || [];
        for(let attempt=0;attempt<4;attempt++){
          if(attempt) await new Promise(resolve=>setTimeout(resolve,250));
          const r = await invoke(d,d.request({count:100,time:0}));
          const ids = r.data?.favoritesIds || [], entries=r.data?.entity?.favorites || {};
          if(r.data?.hasMore) continue;
          if(op==='triage.favorite-add'){
            const added=ids.filter(id=>!oldIds.includes(id));
            if(added.length===1 && entries[added[0]]?.content?.messageId===p.input.messageId && oldIds.every(id=>ids.includes(id))) {verified=true;favoriteId=added[0];break;}
          } else if(!ids.includes(p.input.favoriteId) && oldIds.filter(id=>id!==p.input.favoriteId).every(id=>ids.includes(id))) {verified=true;favoriteId=p.input.favoriteId;break;}
        }
        if(!verified) throw new Error('Favorite readback differs; inspect saved items before retrying');
        return {userId,acknowledged:true,verified:true,data:{favoriteId}};
      }
      if(op === 'triage.schedule-self-text' || op === 'triage.schedule-cancel-text') {
        const messageId=op==='triage.schedule-self-text'?response.data.message.messageId:p.input.messageId;
        const d=readOperations['triage.scheduled-items'];
        for(let attempt=0;attempt<8;attempt++) {
          if(attempt) await new Promise(resolve=>setTimeout(resolve,250));
          const r=await invoke(d,d.request({chatId:p.input.chatId}));
          const state=d.project(r.data), found=state.items.find(x=>x.messageId===messageId);
          if(op==='triage.schedule-self-text' ? found && found.chatId===p.input.chatId && found.scheduleTime===p.input.scheduleTime && found.status===1 && found.type===4 && found.text===p.input.text : !state.truncated && (!found || found.status===5)) {verified=true;break;}
        }
        if(!verified) throw new Error('Schedule readback differs; inspect pending schedules before retrying');
        if(op==='triage.schedule-cancel-text') scheduledFixtures.delete(messageId);
        return {userId,acknowledged:true,verified:true,data:{messageId,chatId:p.input.chatId,scheduleTime:p.input.scheduleTime}};
      }
      if (op === 'mailnext.draft-fixture-delete') {
        const d=readOperations['mailnext.draft-fixture-metadata'];
        for(let attempt=0;attempt<4;attempt++) {
          if(attempt) await new Promise(resolve=>setTimeout(resolve,250));
          try {
            const r=await invoke(d,d.request({draftId:p.input.messageId}));
            if(!r.data?.draft?.id){verified=true;break;}
          } catch(error) {
            if((String(error).includes('NotFound') || String(error).includes('Normal(404)'))){verified=true;break;}
            throw error;
          }
        }
        if(!verified) throw new Error('Deleted draft still exists; inspect before retrying');
        mailFixtures.delete(p.input.messageId);
      }
      if (op === 'mailnext.signature-fixture-update-text') {
        const d=readOperations['mail.signatures'];
        for(let attempt=0;attempt<4;attempt++) {
          if(attempt) await new Promise(resolve=>setTimeout(resolve,250));
          const r=await invoke(d,d.request({accountId:params.accountId}));
          const found=(r.data?.signatures || []).find(x=>x.id===params.signature.id);
          if(found && JSON.stringify(found)===JSON.stringify(params.signature)){verified=true;break;}
        }
        if(!verified) throw new Error('Signature update readback differs; inspect before retrying');
      }
      if(op==='inbox.set-self-chat-mute') {
        const d=readOperations['inbox.chat-mute-state'];
        for(let attempt=0;attempt<4;attempt++) {
          if(attempt) await new Promise(resolve=>setTimeout(resolve,250));
          const r=await invoke(d,d.request({chatId:p.input.chatId}));
          const state=d.project(r.data);
          if(state.chatId===p.input.chatId && state.muted===p.input.muted){verified=true;break;}
        }
        if(!verified) throw new Error('Mute readback differs; inspect before retrying');
        return {userId,acknowledged:true,verified:true,data:{chatId:p.input.chatId,muted:p.input.muted}};
      }
      const {acknowledged: _ack, verified: _verified, ...data} = descriptor.project(response.data,p.input);
      return {userId,acknowledged:true,verified,data};
    }
    if (op === 'listRules') return {userId, ...await call(op, {version: '3'})};
    if (op === 'verifiedEmails') return {userId, ...await call(op, {})};
    if (op !== 'updateRule' && op !== 'updateRuleFields') throw new Error('Unsupported operation');
    if (p.expectedUserId !== userId) throw new Error('Signed-in identity changed');
    const before = await call('listRules', {version: '3'});
    const rule = before.rules?.find(r => r.ruleIdString === p.expectedRule?.ruleIdString);
    if (!rule || JSON.stringify(rule) !== JSON.stringify(p.expectedRule)) throw new Error('Rule changed; refresh preview');
    if (op === 'updateRuleFields') {
      const patch = p.patch;
      if (!patch || typeof patch !== 'object' || Array.isArray(patch) || !Object.keys(patch).length) throw new Error('Empty rule patch');
      for (const key of Object.keys(patch)) {
        if (!['name','isEnable','ignoreTheRestOfRules'].includes(key)) throw new Error('Unsupported rule field');
        if (key === 'name' ? typeof patch[key] !== 'string' || !patch[key].trim() || patch[key].length>256 : typeof patch[key] !== 'boolean') throw new Error('Invalid rule field');
      }
      if (Object.entries(patch).every(([key,value])=>rule[key]===value)) return {changed:false,rule};
      const wanted = {...rule,...patch};
      const result = await call('updateRule',{rule:wanted});
      if (Object.values(result?.emailErrors || {}).some(v=>v!==0)) throw new Error('Rule update returned email errors; inspect before retrying');
      const after = await call('listRules',{version:'3'});
      const saved = after.rules?.find(r=>r.ruleIdString===rule.ruleIdString);
      const fingerprint = r => JSON.stringify({name:r.name,isEnable:r.isEnable,ignoreTheRestOfRules:r.ignoreTheRestOfRules,isInvisible:r.isInvisible,condition:r.condition,action:r.action});
      if (!saved || fingerprint(saved)!==fingerprint(wanted)) throw new Error('Rule patch readback differs; inspect before retrying');
      return {changed:true,verified:true,rule:saved};
    }
    if (!Array.isArray(p.recipients) || p.recipients.length < 1 || p.recipients.length > 20 || p.recipients.some(v => typeof v !== 'string' || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v))) throw new Error('Invalid recipients');
    const normalize = v => v.trim().toLowerCase();
    const normalizedRecipients = p.recipients.map(normalize);
    if (new Set(normalizedRecipients).size !== normalizedRecipients.length) throw new Error('Duplicate recipients');
    const verified = await call('verifiedEmails', {});
    const normalizedVerified = (verified.emails || []).map(normalize);
    if (normalizedRecipients.some(v => !normalizedVerified.includes(v))) throw new Error('Recipient not verified; verify in Lark first');
    const current = (rule.action?.items || []).filter(a => a.type === 12).map(a => normalize(a.input)).sort();
    if (JSON.stringify(current) === JSON.stringify([...normalizedRecipients].sort())) return {changed:false,rule};
    const items = (rule.action?.items || []).filter(a => a.type !== 12);
    for (const input of p.recipients) {
      const existing = (rule.action?.items || []).find(a => a.type === 12 && normalize(a.input) === normalize(input));
      items.push(existing ? {...existing, input} : {type:12,input,authStatus:2,enableAutoTransfer:true});
    }
    const wanted = {...rule, action:{...rule.action, items}};
    const result = await call('updateRule', {rule:wanted});
    if (Object.values(result?.emailErrors || {}).some(v => v !== 0)) throw new Error('Native update returned email errors; read current state before retrying');
    const after = await call('listRules', {version:'3'});
    const saved = after.rules?.find(r => r.ruleIdString === rule.ruleIdString);
    const unchanged = r => JSON.stringify({name:r.name,isEnable:r.isEnable,ignoreTheRestOfRules:r.ignoreTheRestOfRules,isInvisible:r.isInvisible,condition:r.condition,action:(r.action?.items || []).filter(a => a.type !== 12)});
    const savedRecipients = (saved?.action?.items || []).filter(a => a.type === 12).map(a => normalize(a.input)).sort();
    if (!saved || unchanged(saved) !== unchanged(rule) || JSON.stringify(savedRecipients) !== JSON.stringify([...normalizedRecipients].sort())) throw new Error('Readback differs from intended change; inspect rule before retrying');
    return {changed:true,verified:true,rule:saved};
  }
  function connect() {
    if (Date.now() >= cfg.expiresAt) return;
    if (!window.LarkAPI?.transport?.callSdkApi || !window.LarkAPI?.passport?.getUserId) {setTimeout(connect,1000); return;}
    const ws = new WebSocket(`ws://127.0.0.1:${cfg.port}/bridge?token=${cfg.token}`);
    let busy = false;
    ws.onmessage = async event => {
      if (busy || Date.now() >= cfg.expiresAt) return;
      busy = true;
      let req;
      try {
        req = JSON.parse(event.data);
        const data = await run(req.operation,req.params || {});
        ws.send(JSON.stringify({id:req.id,ok:true,data}));
      } catch (e) {
        ws.send(JSON.stringify({id:req?.id,ok:false,error:String(e?.message || 'Native bridge failed').slice(0,240)}));
      }
    };
    ws.onerror = () => {};
    ws.onclose = () => setTimeout(connect,1000);
    setTimeout(() => ws.close(),Math.max(0,cfg.expiresAt-Date.now()));
  }
  connect();
})();
