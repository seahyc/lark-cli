# Experimental desktop bridge

The desktop bridge runs narrow typed operations through the signed-in Lark
application. It supplements the normal CLI; it does not establish parity with
all desktop features. See the [coverage map](research/desktop-cli-gap.md) and
[live verification ledger](research/desktop-verification.md) for exact limits.

The bridge requires the Lark process because `LarkAPI.transport`, its worklet
server transport, and `ShellAPI` are supplied by the authenticated desktop
runtime. After the Email surface loads the bridge, Lark can remain in the
background; the Email tab does not need to stay selected. Quitting Lark ends
native access. The CLI deliberately does not export or independently replay the
desktop account's private credentials.

Use public API commands whenever they provide the same behavior. For example,
`lark find` searches messages, documents, people, and chats without Lark
Desktop or this bridge.

## Public-first replacements

Keep the bridge for native IDs and desktop state. Prefer these desktop-free
commands where the object exists in the public API:

| Behavior | Desktop-free command | Native bridge still needed for |
|---|---|---|
| Search messages/docs/people/chats | `lark find QUERY` | Local search cache/history and native recommendations |
| Read ordinary messages | `lark msg get om_xxx`, `lark msg history` | Native numeric personal-self-chat messages |
| Edit/recall own ordinary messages | `lark msg edit --message-id om_xxx`, `lark msg recall om_xxx` | Native numeric personal-self-chat messages |
| Send ordinary text or styled posts | `lark msg send --as user ...` | The actual personal self-chat; sending to the user's public ID reaches the app conversation |
| Mail read/search/drafts/send | `lark mail ...` | Desktop signatures, rule ordering, block/allow state, and native recipient-free fixtures |
| Scheduled mail | `lark mail schedule ...` | Native scheduled-chat messages |
| Tasks and calendar fundamentals | `lark task ...`, `lark cal ...` | Chat-linked task provenance and desktop-only settings/state |

Do not translate identifiers by prefix substitution. Public `om_`/`oc_` IDs
and native decimal message/chat IDs belong to different identity spaces.

## Start and restore

### macOS

```sh
lark desktop status
lark desktop start
# Restart Lark when convenient, then open its Email tab.
lark desktop status --probe
lark desktop operations
lark desktop mutations
```

Start the Lark application with the debugging port:

```sh
# Option 1: Using open with --args
open -a Lark --args --remote-debugging-port=9330

# Option 2: Direct launcher
/Applications/LarkSuite.app/Contents/MacOS/Lark --remote-debugging-port=9330
```

### Linux

```sh
lark desktop status
# Set the environment variable to allow Linux ASAR patching (until native protocol is revalidated)
export LARK_LINUX_ALLOW_UNTRUSTED_ASAR=1
# Run with sudo to patch system-owned files
sudo -E lark desktop start
# Restart Lark with debugging port, then open its Email tab.
lark desktop status --probe
lark desktop operations
lark desktop mutations
```

Start the Lark application with the debugging port:

```sh
# Option 1: Using the system launcher
/usr/bin/bytedance-lark-stable --remote-debugging-port=9330

# Option 2: Direct binary
/opt/bytedance/lark/lark --remote-debugging-port=9330
```

**Note for Linux**: The `/opt/bytedance/lark` directory is typically owned by root. The `lark desktop start` command requires elevated permissions to patch the mail.asar file. Use `sudo -E` to preserve the `LARK_LINUX_ALLOW_UNTRUSTED_ASAR` environment variable. Session files will be stored in your user's cache directory even when using sudo.

**Security warning**: Setting `LARK_LINUX_ALLOW_UNTRUSTED_ASAR=1` bypasses hash validation. The Linux Lark 7.72.23 native protocol has not been fully revalidated. Use at your own risk for development/testing only.

### General

The temporary session lasts 30 minutes and is pinned to the supported installed
archive hash. It uses authenticated localhost transport. It does not export the
account's credentials. Do not restart Lark during an active meeting or edit.
When finished:

```sh
lark desktop restore
# Restart Lark when convenient to unload the in-memory bridge.
```

## Personal self-chat

```sh
lark desktop send-self --text 'Example self note'
# Review the preview and use its native userId:
lark desktop send-self --text 'Example self note' --apply --expect-user NATIVE_USER_ID
```

This resolves the signed-in user's actual personal self-chat. Sending to your
public open_id through a bot API can instead reach the app conversation.

The scheduling helpers also resolve the destination automatically:

```sh
lark desktop schedule-self --text 'Example scheduled note' --at FUTURE_RFC3339_TIME
lark desktop scheduled-self
lark desktop cancel-scheduled-self NATIVE_MESSAGE_ID
```

Choose an actual time between one hour and 366 days ahead. Writes are previews
unless `--apply --expect-user NATIVE_USER_ID` is provided. The nonempty create/list/cancel roundtrip is live-verified for the personal
self-chat. Future delivery and other-recipient scheduling remain unverified.

## Other typed operations

`desktop operations` and `desktop mutations` list each operation's accepted inputs.
For example:

```sh
lark desktop run workspacenext.navigation-layout
lark desktop run workspacenext.time-format-preference
lark desktop run workspacenext.calendar-resource-equipments
lark desktop mutate OPERATION --input '{"documentedField":"value"}'
```

Generic mutations also preview by default and require `--apply --expect-user`
to write. Their readback checks reject stale snapshots, wrong identities, and
unsupported rich content. No write is automatically retried. If a command says
its outcome may be uncertain, inspect the relevant read operation before trying
again; a timeout or readback failure can occur after a successful write.

Styled posts support titles, paragraphs, and bold/italic/underline text. They do
not yet cover links, mentions, media, lists, code blocks, or arbitrary rich-text
editing. Mail draft tests are recipient-free with a restricted HTML subset.
Navigation layout reading does not imply navigation reorder support.
