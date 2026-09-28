# Lark Mail auto-filter / auto-forward protocol (ASAR 8.0.3)

## Implementation status (2026-09-22)

The temporary mail-resource bridge is live-verified on Lark 8.0.3 / Chromium
147.0.7727.149. It returned the signed-in native identity and all 75 mail rules.
The M1 rule matched the public API and both forwarding actions had authStatus 2
and enableAutoTransfer true. Applying the already-correct recipient list returned
`changed: false`; no live recipient change was necessary. The mutation path is
covered by simulated native-bridge tests, but an actual changed write has not
been exercised live.

The patch modifies only the English mail HTML inside `mail.asar`, keeps a
hash-checked original backup, and expires after 30 minutes. It does not patch or
re-sign executables. It fails closed on an unrecognized archive hash after an
app update. App Management permission for the host app is required.

Commands:

```sh
lark desktop start
# Restart Lark and open Email. You may need to sign in again.
lark desktop identity
lark desktop list
lark desktop verified-emails
lark desktop set-forwarding RULE_ID --to first@example.com,second@example.com
# Review the preview, then use its userId:
lark desktop set-forwarding RULE_ID --to first@example.com,second@example.com --apply --expect-user USER_ID
lark desktop restore
# Restart Lark to unload immediately, or let the 30-minute in-memory session expire.
```

The listener exists only during a command and binds to 127.0.0.1. A random token,
origin check, four-operation allowlist, and 30-second timeout constrain access.
There is no arbitrary JavaScript execution operation. Native credentials remain
inside Lark. `restore` restores the archive and removes the local session token;
it does not reload the already-open mail view. Restarts unload the injected code.

Writes require a matching signed-in identity and fresh rule snapshot. They
preserve conditions and non-forward actions, reject unverified recipients, and
check the result with a native readback. Same-recipient changes are no-ops.
There are no automatic mutation retries. New-address verification and whole-mailbox
forwarding removal are not exposed by this experimental command.

Go tests, bridge race tests, and six JavaScript mutation-boundary tests pass.

The installed macOS build (Lark 8.0.3, Chromium 147.0.7727.149) did not open a
listener with either of these launch configurations:

* `--remote-debugging-port=9330 --remote-debugging-address=127.0.0.1`
* the same flags with `--from-starter --enable-devtools-mode`

Process arguments confirmed the flags arrived; both localhost discovery and
`lsof` confirmed no listener. This does not establish why the build ignores
them. Those initial attempts also returned to the sign-in screen. Subsequent
resource-patch verification succeeded as described above. Lark was restored to a normal launch without debug
arguments, and port 9330 was confirmed closed.

Before enabling writes: establish an authenticated native transport, verify the
selected mailbox identity, compare native list results with the public API,
preserve all conditions and non-forward actions, preview recipient changes,
and verify the exact rule after a single update. A timed-out update must never
be retried automatically. Verification emails and delivery are separate from
configuration readback. Do not export desktop credentials to implement this.

Scope: read-only inspection of `/tmp/lark-private-api-research/mail`; no network, profile, credential, or mutation access.

## Loading chain

`AutoFilterDialog/en-US.html` includes `AutoFilterDialog/64e9a6aa1e.js` (plus shared chunks). The dialog chunk imports `../chunk-common/835b3b6848.js`, `../71559/b4a48f3d52.js`, and dynamically loads chunk IDs including `60034`, `...`; the auto-filter manager logic is in `common/2de8247292.js` module `883136`, and its API module is module `6630` in `common/48be8628db.js`.

Verification command/output:

```text
$ python3 ... AutoFilterDialog/en-US.html
...<script src=.../AutoFilterDialog/64e9a6aa1e.js type=module></script>

$ rg -n -o 'h_:\(\)=>...|w2:\(\)=>...|xw:\(\)=>...|gu:\(\)=>...|cz:\(\)=>...|LW:\(\)=>...' ...
common/48be8628db.js:1:LW:()=>_,kc:()=>V,h_:()=>h,...,cz:()=>g,gu:()=>A,...,w2:()=>b,xw:()=>G,...
```

## Wire command strings

The command constants are in `18918/b2ed03e9bb.js` module `236171` (offsets approximate byte offsets shown by bounded Python extraction):

```text
3694|email.client.v1.MailCreateRuleRequest|email.client.v1.MailCreateRuleResponse|1|MAIL_CREATE_RULE
3691|email.client.v1.MailUpdateRuleRequest|email.client.v1.MailUpdateRuleResponse|1|MAIL_UPDATE_RULE
3695|email.client.v1.MailDeleteRuleRequest|email.client.v1.MailDeleteRuleResponse|1|MAIL_DELETE_RULE
3693|email.client.v1.MailGetRulesRequest|email.client.v1.MailGetRulesResponse|1|MAIL_GET_RULES
3690|email.client.v1.MailEnableRuleRequest|email.client.v1.MailEnableRuleResponse|1|MAIL_ENABLE_RULE
3697|email.client.v1.MailAdjustRuleOrderRequest|email.client.v1.MailAdjustRuleOrderResponse|1|MAIL_ADJUST_RULE_ORDER
3767|email.client.v1.MailSendAutoTransferAuthMailRequest|email.client.v1.MailSendAutoTransferAuthMailResponse|1|MAIL_SEND_AUTO_TRANSFER_AUTH_MAIL
3766|email.client.v1.MailGetAutoTransferEmailRequest|email.client.v1.MailGetAutoTransferEmailResponse|1|MAIL_GET_AUTO_TRANSFER_EMAIL_REQUEST
3692|email.client.v1.MailPreviewRuleRequest|email.client.v1.MailPreviewRuleResponse|1|MAIL_PREVIEW_RULE
3696|email.client.v1.MailApplyRuleRequest|email.client.v1.MailApplyRuleResponse|1|MAIL_APPLY_RULE
```

In module `236171` aliases used by module `6630` map as follows: `oiO` → create, `p6L` → update, `t1q` → delete, `jdy` → get rules, `$uT` → enable, `Iz_` → adjust order, `M6d` → send auth mail, `Suv` → get verified auto-transfer emails, `GSW` → preview, `uUU` → apply.

## Rule object schema and operations

Known from `common/48be8628db.js`, module `6630` (single-line module; offsets are byte offsets):

* `O(rule)` serializes a UI rule to wire shape:
  `{ruleIdString, name, isEnable, ignoreTheRestOfRules, isInvisible, condition:{matchType,items}, action:{items}}`.
* `N(condition)` returns `{type, operator, input}`.
* `R(action)` returns `{type, authStatus, enableAutoTransfer, input}`. UI-only `frontendId`, `frontendStatus`, and `errorCode` are omitted.
* `m(wireRule,emailErrors)` parses wire data. It takes `wireRule.condition.matchType`, maps `condition.items`, maps `action.items`, and computes forwarding status from `authStatus`.

Operation request transforms:

* Create (`h_`, `n.oiO`, `MAIL_CREATE_RULE`): `{...O(rule), accountId}`. Response is `{rule: m(response.rule,response.emailErrors), emailErrors: response.emailErrors ?? {}}`.
* Update (`w2`, `n.p6L`, `MAIL_UPDATE_RULE`): `{rule: O(rule), ...accountContext}`. The wrapper receives `(rule, accountContext)`; the UI call passes only the rule and the transport layer supplies the account context. The logged schema names `rule`, `checkPwd`, and `accountId`, so preserve any caller-specific `checkPwd`/`accountId` context if using the underlying transport directly. Response has the same `{rule,emailErrors}` shape.
* Delete (`xw`, `n.t1q`, `MAIL_DELETE_RULE`): `{ruleIdString, accountId}` when called as `(ruleIdString, accountId)`; UI delete calls `xw(ruleIdString)` and relies on transport account context.
* List (`gu`, `n.jdy`, `MAIL_GET_RULES`): input defaults to `{version: 3, ...optional}` and serializes `version` as a string. Response is `{adminEnableAutoTransfer: boolean, rules: [...]}`. The UI calls `gu({version: 4})` only when the `filterRuleSenderInChatGroup` feature is enabled; otherwise it calls `gu()` (default version 3).
* Enable/disable (`cz`, `n.$uT`, `MAIL_ENABLE_RULE`): `{ruleIdString, isEnable}`; response is boolean `isEnable`. UI call is `cz(ruleIdString, status)`.
* Reorder (`y`, `n.Iz_`, `MAIL_ADJUST_RULE_ORDER`): `{ruleOrder: [{ruleId, order: String(order), ruleIdString?}, ...]}`. UI builds entries from each rule's `ruleIdString` as `{ruleIdString, order}`; wrapper maps the property to `ruleId` while retaining `ruleIdString` in logging.

Direct bounded extraction verification:

```text
common/48be8628db.js@~0-9728:
O(r){return{ruleIdString:r.ruleIdString,name:r.name,isEnable:r.isEnable,ignoreTheRestOfRules:r.ignoreTheRestOfRules,isInvisible:r.isInvisible,condition:{matchType:r.matchType,items:r.condition.map(N)},action:{items:r.action.map(R)}}}
R(r){return{type:r.type,authStatus:r.authStatus,enableAutoTransfer:r.enableAutoTransfer,input:r.input}}
... h_ ... transform:(r,t)=>({...O(r),accountId:t}) ...
... b ... transform:(r,t)=>({rule:O(r),...t}) ...
... G ... transform:(r,t)=>({ruleIdString:r,isEnable:t}) ...
... y ... transform:r=>({ruleOrder:r.map(r=>({...r,order:String(r.order)}))}) ...
```

## Auto-forward verification

The UI's `handleSendAutoTransferAuthMail` (`common/2de8247292.js` around offset 95044) extracts the current `ruleIdString` and calls `I4(ruleIdString)`. In module `6630`, `I4` is wrapper `Y` over `n.M6d` (`MAIL_SEND_AUTO_TRANSFER_AUTH_MAIL`) and serializes `{ruleId: ruleIdString, accountId}`. This is the explicit “send/re-send verification” request.

The UI calls `LW()` on action-editor mount (around offset 71948); module `6630` maps `LW` to `n.Suv` (`MAIL_GET_AUTO_TRANSFER_EMAIL_REQUEST`) with `{accountId}`, and response transform returns `response.emails ?? []`. These are the addresses already verified. A forwarding action is considered `WAIT_AUTH` when `authStatus` is unknown and its `input` is absent from this list; `SUCCESS` when present. This status computation is client-side; it does not itself verify an address.

The rule create/update response can contain per-action `emailErrors`; the UI treats any nonempty `emailErrors` as failure and maps each action's `errorCode`. Known error enum values in `chunk-common/835b3b6848.js` are `NO_ERR:0`, `ERR_SELF_ADDRESS:10000`, `ERR_GROUP_ADDRESS:10001`, `ERR_INVALID_ADDRESS:10002` (the UI's normalized `NOERROR` is 0).

## Enums used by forwarding rules

From `chunk-common/835b3b6848.js` around offset 18599:

* Match type: `MEET_ALL_CONDITIONS=0`, `MEET_ANY_CONDITIONS=1`.
* Condition types: `FROM=1`, `TO=2`, `CC=3`, `TO_OR_CC=4`, `REPLY_TO=5`, `SUBJECT=6`, `BODY=7`, `ATTACHMENT_NAME=8`, `ATTACHMENT_TYPE=9`, `ANY_ADDRESS=10`, `MESSAGE_SIZE=11`, `MATCH_ALL_MESSAGE=12`, `IS_EXTERNAL=13`, `IS_SPAM=14`, `IS_NOT_SPAM=15`, `HAS_ATTACHMENT=16`, `PRIORITY=17`.
* Operators: `CONTAINS=1`, `DOES_NOT_CONTAINS=2`, `STARTS_WITH=3`, `ENDS_WITH=4`, `IS=5`, `IS_NOT=6`, `INCLUDES_ME=7`, `GREATER_THAN=8`, `LESS_THAN=9`, `IS_EMPTY=10`, `BELONG_TO=12`.
* Action types: `ARCHIVE_MESSAGE=1`, `DELETE_MESSAGE=2`, `MARK_AS_READ=3`, `MARK_AS_SPAM=4`, `NEVER_MARK_AS_SPAM=5`, `MARK_AS_IMPORTANT=6`, `NEVER_MARK_AS_IMPORTANT=7`, `APPLY_LABEL=8`, `FLAG=9`, `NEVER_PUSH_NOTIFICATION=10`, `MOVE_TO=11`, `AUTO_TRANSFER=12`, `SEND_TO_CHAT=13`.
* Forward auth status: `UNKNOWN=0`, `WAIT_AUTH=1`, `SUCCESS=2`, `TIMEOUT=3`, `FAIL=4`, `NO_AUTH=5`.

The enums and fields above are known from shipped code. The recommendation to preserve the full list-returned rule object for update is an operational inference: update serialization intentionally drops UI-only fields but retains all server rule fields represented by `O`, so reconstructing a rule from only the forwarding action risks losing conditions/actions/options.
