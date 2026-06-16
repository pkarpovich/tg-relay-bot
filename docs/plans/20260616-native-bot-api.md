# Replace the Telegram bot library with a native Bot API HTTP client

## Overview
`tg-relay-bot` depends on `github.com/OvyFlash/telegram-bot-api` for both receiving
updates (long-poll) and sending messages/reactions. The library lags behind new Bot
API features and releases slowly. The Telegram Bot API is a plain HTTP/JSON API, and
this bot uses only a tiny slice of it, so we replace the library with a small
in-repo native client (stdlib `net/http` + `encoding/json`), defining only the
fields we actually use.

Behavior is preserved: long-poll for updates (no webhook), super-user filtering,
`/ping`, forward-origin handling, message reactions, and HTTP/SMTP -> Telegram
sending with `parse_mode`. The external contract is unchanged (the `POST /send`
endpoint and SMTP path keep working identically).

Benefits: no third-party Bot API dependency (no release lag), always-current (any
method is one HTTP call), fewer deps, and minimal types we own. Cost: we hand-roll
the long-poll loop, the response envelope, and 429 handling that the library
provided.

## Context (from discovery)
- **Prerequisite / base branch**: the completed `parse-mode-support` branch is 10
  commits ahead of `master` and NOT merged (no PR). This rewrite builds on it
  (`SendMessage` must keep `parse_mode`). Land `parse-mode-support` on `master`
  first, then run this plan off `master`. This plan is written against the
  parse-mode code.
- Library surface actually used (the whole replacement target):
  - `GetUpdatesChan(UpdateConfig{offset, timeout=60})` long-poll
  - `Send(NewMessage(chatID, text))` with `.ParseMode` -> `sendMessage`
  - `Request(SetMessageReactionConfig{...})` -> `setMessageReaction`
  - `Update.Message` fields: `From.ID`, `Chat.ID`, `Chat.UserName`, `MessageID`,
    `Text`, `Caption`, `Time()`, `Command()` (parses `/ping`), `ForwardOrigin`
    (`Type`, `Chat.UserName`, `MessageID`, `SenderUser.{First,Last,User}Name`,
    `SenderUserName`) with `MessageOriginChannel/User/HiddenUser`
  - `ReactionType{Type:"emoji", Emoji:"👍"}`
- Files involved:
  - `app/events/events.go` - heavy lib consumer (`TbAPI` interface + `Do` loop +
    `processEvent`/`transform`/`handlePingCommand`/`SendMessagesForAdmins`/
    `reactToMessage`); already has a consumer-side interface, but typed against the lib
  - `app/main.go` - wiring (`tbapi.NewBotAPI`)
  - `app/bot/bot.go` - `bot.Message`/`OnMessage` (domain type; unaffected by the lib)
  - `app/telegram/` (new) - the native client
  - `go.mod` - drop `github.com/OvyFlash/telegram-bot-api`
  - `app/smtp_server/` - does NOT import the lib (sends via the channel); untouched
- Go 1.26, `testify` present. Conventions (per Go standards): stdlib HTTP, minimal
  types, consumer-side interfaces, table-driven tests with `testify`, `httptest` for
  the client, `moq` mocks under `mocks/`, target 80%+.

## Development Approach
- **Testing approach: regular** (code, then tests in the same task). Every task ships
  tests; all green before the next.
- Native client uses stdlib `net/http`+`encoding/json`; no new dependencies (this
  removes one). Define only the JSON fields the bot uses.
- Consumer-side interface in `events` (small), native client implements it; mock with
  `moq` for listener tests. Accept interfaces, return concrete types.
- `context.Context` first param on blocking calls (GetUpdates, the loop). Wrap errors
  with `%w`. Early-return style.
- Preserve behavior exactly; the rewrite is internal.

## Testing Strategy
- **Unit tests** (`testify`, table-driven, one `_test.go` per source file):
  - `telegram` client: `httptest.Server` for `do`/`SendMessage`/`GetUpdates`/
    `SetMessageReaction` - ok envelope, error envelope (`ok:false`+description),
    429 `retry_after`, JSON (un)marshal of the minimal types, `Command()` parsing,
    forward-origin variants.
  - `events`: `moq`-generated mock of the new client interface - super-user filter,
    `/ping`, transform (plain/caption/forward-origin), `OnMessage` error path,
    reaction on save, `SendMessagesForAdmins` with `parse_mode`.
- No e2e; live send/ping/reaction check is manual (Post-Completion).

## Progress Tracking
- Mark `[x]` when done; `➕` discovered; `⚠️` blockers. Keep in sync.

## What Goes Where
- **Implementation Steps**: the native client, the events port, wiring, dep removal,
  tests.
- **Post-Completion**: live verification against the real Bot API, deploy.

## Implementation Steps

### Task 1: Native client core + SendMessage
- [x] add `app/telegram` package: a `Client` (config struct: token, optional
      `*http.Client`, base URL override for tests) calling
      `https://api.telegram.org/bot<token>/<method>`
- [x] add `do(ctx, method, payload)` - POST JSON, decode the
      `{ok, result, description, parameters:{retry_after}}` envelope, return a typed
      error on `ok:false`, honor `429`/`retry_after` (single bounded retry)
- [x] add `SendMessage(ctx, chatID int64, text, parseMode string)`
- [x] write tests with `httptest`: success, `ok:false` error, 429 retry, parse_mode
      passthrough
- [x] run `go test ./...` - must pass before next task

### Task 2: Minimal update types + GetUpdates long-poll
- [x] define minimal types in `app/telegram`: `Update`, `Message`, `User`, `Chat`,
      `ForwardOrigin` (only used fields, Bot API JSON tags) + a `Command()` helper
      and a `Time()` from `date`
- [x] add `GetUpdates(ctx, offset int, timeoutSec int) ([]Update, error)` (long-poll)
- [x] write tests: unmarshal a sample `getUpdates` response, `Command()` parsing,
      forward-origin (channel/user/hidden) shapes, empty result
- [x] run tests - must pass before next task

### Task 3: SetMessageReaction
- [ ] add `SetMessageReaction(ctx, chatID int64, messageID int, emoji string)`
- [ ] write tests with `httptest` (correct payload + ok/error envelope)
- [ ] run tests - must pass before next task

### Task 4: Port `events` to the native client
- [ ] replace the lib-typed `TbAPI` interface with a native consumer interface
      (`GetUpdates`, `SendMessage`, `SetMessageReaction`) over `telegram` types
- [ ] rewrite `Do` to poll `GetUpdates` in a loop with offset management +
      `context` cancellation (replacing `GetUpdatesChan`); port `processEvent`,
      `transform`, `handlePingCommand`, `SendMessagesForAdmins` (keep `parse_mode`),
      `reactToMessage` to native types
- [ ] generate a `moq` mock of the new interface under `mocks/`
- [ ] write tests: super-user filter, `/ping`, transform (plain/caption/forward
      variants), `OnMessage` error -> error message sent, reaction on save
- [ ] run tests - must pass before next task

### Task 5: Wire main.go and drop the library
- [ ] replace `tbapi.NewBotAPI(token)` in `app/main.go` with the native
      `telegram.NewClient(...)`; inject into `TelegramListener`
- [ ] remove `github.com/OvyFlash/telegram-bot-api` from `go.mod`; run `go mod tidy`
- [ ] confirm `app/smtp_server` still compiles unchanged (it uses the channel, not
      the lib)
- [ ] run tests - must pass before next task

### Task 6: Verify acceptance criteria
- [ ] no remaining references to `OvyFlash/telegram-bot-api` / `tbapi` in the repo
- [ ] `go build ./...`, `go test ./...`, `golangci-lint run` all clean
- [ ] coverage of `telegram` + `events` meets the project standard (80%+)

## Technical Details
- Envelope: `type apiResponse[T any] struct { OK bool; Result T; Description string;
  Parameters *struct{ RetryAfter int } }` (or non-generic with `json.RawMessage`).
- Long-poll: `GetUpdates(offset, timeout=60)`; advance `offset = lastUpdateID + 1`;
  on context cancel, return; on transient error, log + brief backoff (no tight loop).
- `parse_mode` is passed through `SendMessage`; empty string omits it (plain text).
- Reactions: `setMessageReaction` with `reaction:[{type:"emoji", emoji:"👍"}]`.
- Webhook is explicitly NOT used - long-poll preserves current behavior with no
  public endpoint.

## Post-Completion
*Manual / external - no checkboxes.*

- Land `parse-mode-support` on `master` first (open its PR + merge); run this plan off
  `master`.
- Live-verify against the real Bot API with a test token: `/send` (plain + MarkdownV2
  + HTML), SMTP -> Telegram, an incoming `/ping` -> Pong, a forwarded message ->
  reaction, and the non-super-user rejection path.
- Build/publish the image and redeploy wherever tg-relay-bot runs.
