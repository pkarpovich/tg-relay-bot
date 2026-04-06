# Add parse_mode Support to /send API

## Overview

Add optional `parse_mode` field to the `POST /send` HTTP endpoint. When set to `MarkdownV2` (or `HTML`), messages are sent to Telegram with the corresponding formatting. Existing clients sending plain text without `parse_mode` continue to work unchanged.

The `NewMarkdownMessage` helper already exists in `events.go` but is unused. This wires it up.

## Context (from discovery)

**Files affected:**
- `app/http/client.go` — add `ParseMode` to send request struct, pass it through channel
- `app/events/events.go` — `SendMessagesForAdmins` uses `NewMessage` (plain text); switch to `NewMarkdownMessage` when parse_mode is set
- `app/main.go` — channel type changes from `chan string` to `chan MessagePayload`
- `app/smtp_server/server.go` — update channel writes to use new `MessagePayload` type (plain text, no parse_mode)

**Three producers write to the channel:**
1. `sendHandler` (`http/client.go:72`) — HTTP POST `/send` with `X-Secret`
2. `webhookHandler` (`http/client.go:99`) — HTTP POST `/webhook` (no auth)
3. `sendEmailToTelegram` (`smtp_server/server.go:92`) — SMTP email relay

**Consumer:**
- `SendMessagesForAdmins` (`events/events.go:166`) — reads channel, sends via `NewMessage` to all SuperUsers

**Existing helper (unused):**
```go
func NewMarkdownMessage(chatID int64, text string, replyMarkup *tbapi.InlineKeyboardMarkup) tbapi.MessageConfig
```

**No existing tests in the project.**

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Backward compatible: empty/missing `parse_mode` = plain text
- Minimal scope: only add parse_mode plumbing, no other changes
- Add tests for the new functionality (table-driven, testify)

## Testing Strategy

- **Unit tests**: table-driven tests for `sendHandler` (with and without parse_mode) and `SendMessagesForAdmins` (plain text vs MarkdownV2)
- Mock `TbAPI` interface for Telegram calls
- Use `httptest` for HTTP handler tests

## Progress Tracking

- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with ➕ prefix
- Document issues/blockers with ⚠️ prefix

## Implementation Steps

### Task 0: Upgrade Go to 1.26, update dependencies, add golangci-lint

- [x] Update `mise.toml`: `go = "1.26"`
- [x] Run `mise install` to install Go 1.26
- [x] Update `go.mod`: `go 1.26`
- [x] Run `go get -u ./...` to update all dependencies
- [x] Run `go mod tidy` to clean up
- [x] Copy `.golangci.yml` from `/Users/pavel.karpovich/Projects/tuclaw/.golangci.yml`
- [x] Run `golangci-lint run ./...` and fix all reported issues
- [x] Verify project compiles: `go build ./...`

### Task 1: Introduce `MessagePayload` type and update channel

- [ ] Define `MessagePayload` struct in `app/events/events.go` with `Text string` and `ParseMode string` fields
- [ ] Change channel type from `chan string` to `chan MessagePayload` in `TelegramListener`, `Server` (http), and `Server` (smtp)
- [ ] Update `main.go`: `messagesForSend := make(chan MessagePayload, 100)`
- [ ] Update `webhookHandler` to send `MessagePayload{Text: data.Content}`
- [ ] Update `sendEmailToTelegram` to send `MessagePayload{Text: formattedEmail.Text}`
- [ ] Update `sendHandler` to parse optional `parse_mode` from JSON body and send `MessagePayload{Text: data.Message, ParseMode: data.ParseMode}`
- [ ] Verify project compiles: `go build ./...`

### Task 2: Use parse_mode in `SendMessagesForAdmins`

- [ ] Update `SendMessagesForAdmins` to read `MessagePayload` from channel
- [ ] When `ParseMode` is `MarkdownV2` or `HTML`: create message with corresponding `tbapi` parse mode
- [ ] When `ParseMode` is empty: use `NewMessage` (plain text, existing behavior)
- [ ] Remove `replyMarkup` parameter from `NewMarkdownMessage` (unused, simplify to match `NewMessage` signature)
- [ ] Verify project compiles: `go build ./...`

### Task 3: Add tests

- [ ] Create `app/http/client_test.go` with table-driven tests for `sendHandler`: plain text request, request with `parse_mode: "MarkdownV2"`, missing message, wrong secret
- [ ] Create `app/events/events_test.go` with tests for `SendMessagesForAdmins`: plain text payload dispatches `NewMessage`, MarkdownV2 payload dispatches with parse mode set
- [ ] Run tests: `go test ./...` — all pass

### Task 4: Verify acceptance criteria

- [ ] `POST /send {"message": "hello"}` sends plain text (backward compat)
- [ ] `POST /send {"message": "*bold*", "parse_mode": "MarkdownV2"}` sends with MarkdownV2
- [ ] `POST /send {"message": "<b>bold</b>", "parse_mode": "HTML"}` sends with HTML
- [ ] Webhook and SMTP continue to send plain text
- [ ] All tests pass
- [ ] Run linter if configured

## Technical Details

**New struct:**
```go
type MessagePayload struct {
    Text      string
    ParseMode string
}
```

**Updated send request body:**
```go
var data struct {
    Message   string `json:"message"`
    ParseMode string `json:"parse_mode"`
}
```

**Updated SendMessagesForAdmins logic:**
```go
for _, adminID := range adminIds {
    var msg tbapi.MessageConfig
    switch payload.ParseMode {
    case "MarkdownV2":
        msg = NewMarkdownMessage(adminID, payload.Text)
    case "HTML":
        msg = NewHTMLMessage(adminID, payload.Text)
    default:
        msg = NewMessage(adminID, payload.Text)
    }
    tl.TbAPI.Send(msg)
}
```

**API contract (unchanged for existing clients):**
```
POST /send
Headers: X-Secret: <secret>
Body: {"message": "plain text"}
→ sends plain text (backward compatible)

POST /send
Headers: X-Secret: <secret>
Body: {"message": "*bold* text", "parse_mode": "MarkdownV2"}
→ sends with Telegram MarkdownV2 formatting
```

## Post-Completion

**Manual verification:**
- `curl -X POST -H "X-Secret: ..." -H "Content-Type: application/json" -d '{"message": "*bold* _italic_", "parse_mode": "MarkdownV2"}' https://relay.pkarpovich.space/send`
- Verify bold/italic renders in Telegram
- Send without `parse_mode` to confirm backward compat

**Downstream consumers:**
- `podcast-transcriber` — will use `parse_mode: "MarkdownV2"` for rich notifications (separate plan)
- `radio-t-monitor` — existing plain text notifications continue working unchanged
