# Native recall protocol

The desktop's ordinary message-recall UI calls `recallMessage({id})` for the
selected non-local message. Its SDK adapter sends the exact `{id}` payload to
`RECALL_MESSAGE`; the worklet then uses `e.id`, marks the local record recalled,
and returns a response with `message: "ok"`.

The descriptor is deliberately limited to a parent-preflighted self-chat TEXT
fixture. Before apply, the parent must verify the current signed-in user owns the
message, it is in the user's self P2P chat, it remains plain TEXT, and its
decoded text equals `expectedText`. After apply, read back the exact message and
require recalled state. A transport response alone does not prove recall.

No group-owner recall or broad delete command is exposed.
