# Add an `md` message type backed by Telegram sendRichMessage

## Overview
Telegram Bot API 10.1 (June 2026) added `sendRichMessage`, whose `InputRichMessage`
takes a plain **Markdown string** (`rich_message.markdown`) and renders it fully -
no `parse_mode`, no fragile MarkdownV2 escaping. The old library never supported
this method; the recent native-client rewrite is what makes it reachable.

Add a new `md` message type to tg-relay: a `POST /send` with `parse_mode: "md"`
routes the body through `sendRichMessage` (Markdown string) instead of `sendMessage`.
Callers send clean Markdown with zero escaping; the existing `MarkdownV2` / `HTML` /
plain modes are unchanged (backward compatible). This lets consumers (e.g. a future
playlist-synchronizer notifier) author plain Markdown without dragging MarkdownV2.

Scope: tg-relay only - the native client method, the send routing, and the `/send`
validation. Consumers adopting `md` are out of scope.

## Context (from discovery)
- Bot API 10.1 spec (verified from the machine-readable schema):
  - `sendRichMessage(chat_id, rich_message: InputRichMessage, ...)` -> `Message`.
  - `InputRichMessage`: "Exactly one of `html` or `markdown` must be used";
    `markdown: String`, `html: String`, `is_rtl: Boolean`,
    `skip_entity_detection: Boolean`. No `parse_mode`.
- Files involved:
  - `app/telegram/` (native client): `client.go` (`do`/`doOnce`/`APIError`),
    `messages.go` (`SendMessage`), `reactions.go`, `updates.go`. Add a
    `SendRichMessage` method here (new `richmessages.go`).
  - `app/events/events.go`: `TelegramAPI` interface + `SendMessagesForAdmins` routes
    on `payload.ParseMode`. Add `SendRichMessage` to the interface and an `md` branch.
    `MessagePayload{Text, ParseMode}` is unchanged.
  - `app/events/mocks/`: `moq` mock of `TelegramAPI` - regenerate.
  - `app/http/client.go`: `/send` handler validates `parse_mode` in a `switch`
    (unknown -> 400). Add `"md"` to the allowlist.
- Conventions (Go standards): stdlib `net/http`+`encoding/json`, no new deps,
  consumer-side interface, table-driven `testify` tests, `httptest` for the client,
  `moq` mocks under `mocks/`, 80%+.

## Development Approach
- **Testing approach: regular** (code, then tests in the same task). Every task ships
  tests; all green before the next.
- No new dependencies. Define a shared constant for the `"md"` value (in `events`,
  referenced by `app/http`) to avoid a magic string in two places.
- `skip_entity_detection`/`is_rtl` are NOT exposed for now (YAGNI); default behavior
  (auto entity detection, LTR) is fine for notifications.
- Preserve backward compatibility: existing `MarkdownV2`/`HTML`/plain paths unchanged.

## Testing Strategy
- **Unit tests** (`testify`, table-driven, one `_test.go` per source file):
  - `app/telegram`: `httptest` for `SendRichMessage` - asserts the request hits
    `sendRichMessage` with body `{chat_id, rich_message:{markdown}}`; ok envelope +
    `APIError` on `ok:false`.
  - `app/events`: `moq` mock - `payload.ParseMode == "md"` calls `SendRichMessage`
    (not `SendMessage`); other modes still call `SendMessage` with the parse_mode.
  - `app/http`: `/send` accepts `parse_mode:"md"` and enqueues; unknown parse_mode
    still 400; existing modes still accepted.
- Live smoke is manual (Post-Completion).

## Progress Tracking
- Mark `[x]` when done; `➕` discovered; `⚠️` blockers. Keep in sync.

## What Goes Where
- **Implementation Steps**: client method, interface/routing, handler validation,
  docs, tests.
- **Post-Completion**: live smoke against a test bot, deploy.

## Implementation Steps

### Task 1: Native `SendRichMessage` client method
- [x] add `app/telegram/richmessages.go`: `SendRichMessage(ctx, chatID int64,
      markdown string) error` calling `do(ctx, "sendRichMessage", payload)` with
      `{chat_id, rich_message:{markdown}}` (small `inputRichMessage` request struct)
- [x] write `httptest` tests: correct method + JSON body shape, success, and
      `APIError` on an `ok:false` response
- [x] run `go test ./...` - must pass before next task

### Task 2: Route the `md` type in events
- [x] add `SendRichMessage(ctx, chatID int64, markdown string) error` to the
      `TelegramAPI` interface in `app/events/events.go`
- [x] add a shared const (e.g. `ParseModeMarkdown = "md"`) and route
      `SendMessagesForAdmins`: when `payload.ParseMode == ParseModeMarkdown` call
      `SendRichMessage(adminID, payload.Text)`, else `SendMessage(...)` as today
- [x] regenerate the `moq` mock for `TelegramAPI` under `app/events/mocks/`
- [x] write tests: `md` payload -> `SendRichMessage` called (not `SendMessage`);
      `MarkdownV2`/`HTML`/plain -> `SendMessage` with the parse_mode
- [x] run tests - must pass before next task

### Task 3: Accept `parse_mode: "md"` in the /send handler
- [x] add `"md"` (via the shared const) to the `parse_mode` validation `switch` in
      `app/http/client.go` so it is enqueued instead of rejected
- [x] write tests: `/send` with `parse_mode:"md"` -> 200 + enqueued payload; unknown
      parse_mode -> 400; existing `MarkdownV2`/`HTML`/empty still accepted
- [x] run tests - must pass before next task

### Task 4: Document the `md` type
- [ ] update `README.md`: add `md` to the `/send` `parse_mode` options (sends via
      `sendRichMessage` with a plain Markdown string; no escaping needed), with a
      `curl` example

### Task 5: Verify acceptance criteria
- [ ] `go build ./...`, `go test ./...`, `golangci-lint run` all clean
- [ ] backward compatibility: existing modes unchanged; only `md` is new
- [ ] coverage of the changed packages meets the project standard (80%+)

## Technical Details
- Request payload: `{"chat_id": <id>, "rich_message": {"markdown": "<text>"}}` to
  the `sendRichMessage` method; returns a `Message` (we only need success/err).
- `parse_mode` values after this change: `""` (plain), `MarkdownV2`, `HTML`
  (all via `sendMessage`), and `md` (via `sendRichMessage`).
- Errors surface through the existing `do`/`APIError` path (non-`ok` -> typed error).

## Post-Completion
*Manual / external - no checkboxes.*

- Live smoke with a test bot: `POST /send` with
  `{"message":"# Title\n**bold** _italic_ `code`\n- a\n- b","parse_mode":"md"}` and
  confirm the rich-formatted message renders in Telegram; confirm existing modes
  still work.
- Build/publish the image and redeploy wherever tg-relay-bot runs.
- Follow-up (separate): wire the playlist-synchronizer notifier to send `parse_mode:"md"`.
