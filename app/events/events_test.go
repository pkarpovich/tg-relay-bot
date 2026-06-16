package events

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pkarpovich/tg-relay-bot/app/bot"
	"github.com/pkarpovich/tg-relay-bot/app/events/mocks"
	"github.com/pkarpovich/tg-relay-bot/app/telegram"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type botStub struct {
	saved  bool
	err    error
	called bool
	gotMsg bot.Message
}

func (b *botStub) OnMessage(msg bot.Message) (bool, error) {
	b.called = true
	b.gotMsg = msg
	return b.saved, b.err
}

const testChatID = 555

func userMessage(fromID int64) *telegram.Message {
	return &telegram.Message{
		MessageID: 100,
		From:      &telegram.User{ID: fromID},
		Chat:      telegram.Chat{ID: testChatID},
		Text:      "hello",
	}
}

func pingMessage() *telegram.Message {
	return &telegram.Message{
		MessageID: 100,
		From:      &telegram.User{ID: 111},
		Chat:      telegram.Chat{ID: testChatID},
		Text:      "/ping",
		Entities:  []telegram.MessageEntity{{Type: "bot_command", Offset: 0, Length: 5}},
	}
}

func TestTelegramListener_processEvent(t *testing.T) {
	tests := []struct {
		name          string
		message       *telegram.Message
		superUsers    []int64
		botSaved      bool
		botErr        error
		sendErr       error
		reactionErr   error
		wantSendTexts []string
		wantReaction  bool
		wantBotCalled bool
		wantErr       bool
	}{
		{
			name:          "non super user gets rejection",
			message:       userMessage(999),
			superUsers:    []int64{111},
			wantSendTexts: []string{"I don't know you 🤷‍"},
		},
		{
			name:          "ping command answers pong",
			message:       pingMessage(),
			superUsers:    []int64{111},
			wantSendTexts: []string{"🏓 Pong!"},
		},
		{
			name:          "saved message gets reaction",
			message:       userMessage(111),
			superUsers:    []int64{111},
			botSaved:      true,
			wantReaction:  true,
			wantBotCalled: true,
		},
		{
			name:          "unsaved message gets no reaction",
			message:       userMessage(111),
			superUsers:    []int64{111},
			botSaved:      false,
			wantBotCalled: true,
		},
		{
			name:          "onmessage error sends error message",
			message:       userMessage(111),
			superUsers:    []int64{111},
			botErr:        errors.New("boom"),
			wantSendTexts: []string{"💥 Error: boom"},
			wantBotCalled: true,
			wantErr:       true,
		},
		{
			name:          "rejection send failure returns error",
			message:       userMessage(999),
			superUsers:    []int64{111},
			sendErr:       errors.New("send failed"),
			wantSendTexts: []string{"I don't know you 🤷‍"},
			wantErr:       true,
		},
		{
			name:          "error message send failure returns error",
			message:       userMessage(111),
			superUsers:    []int64{111},
			botErr:        errors.New("boom"),
			sendErr:       errors.New("send failed"),
			wantSendTexts: []string{"💥 Error: boom"},
			wantBotCalled: true,
			wantErr:       true,
		},
		{
			name:          "reaction failure returns error",
			message:       userMessage(111),
			superUsers:    []int64{111},
			botSaved:      true,
			reactionErr:   errors.New("reaction failed"),
			wantReaction:  true,
			wantBotCalled: true,
			wantErr:       true,
		},
		{
			name:       "nil sender is ignored",
			message:    &telegram.Message{MessageID: 100, Chat: telegram.Chat{ID: testChatID}, Text: "hello"},
			superUsers: []int64{111},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgMock := &mocks.TelegramAPIMock{
				SendMessageFunc: func(_ context.Context, _ int64, _, _ string) error {
					return tt.sendErr
				},
				SetMessageReactionFunc: func(_ context.Context, _ int64, _ int, _ string) error {
					return tt.reactionErr
				},
			}
			botMock := &botStub{saved: tt.botSaved, err: tt.botErr}

			tl := &TelegramListener{
				SuperUsers: tt.superUsers,
				TbAPI:      tgMock,
				Bot:        botMock,
			}

			err := tl.processEvent(t.Context(), telegram.Update{Message: tt.message})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			sendCalls := tgMock.SendMessageCalls()
			require.Len(t, sendCalls, len(tt.wantSendTexts))
			for i, want := range tt.wantSendTexts {
				assert.Equal(t, want, sendCalls[i].Text)
				assert.Equal(t, tt.message.Chat.ID, sendCalls[i].ChatID)
				assert.Empty(t, sendCalls[i].ParseMode)
			}

			reactionCalls := tgMock.SetMessageReactionCalls()
			if tt.wantReaction {
				require.Len(t, reactionCalls, 1)
				assert.Equal(t, tt.message.Chat.ID, reactionCalls[0].ChatID)
				assert.Equal(t, tt.message.MessageID, reactionCalls[0].MessageID)
				assert.Equal(t, "👍", reactionCalls[0].Emoji)
			} else {
				assert.Empty(t, reactionCalls)
			}

			assert.Equal(t, tt.wantBotCalled, botMock.called)
			if tt.wantBotCalled {
				assert.Equal(t, tt.message.Chat.ID, botMock.gotMsg.ChatID)
				assert.Equal(t, tt.message.Text, botMock.gotMsg.Text)
			}
		})
	}
}

func TestTelegramListener_transform(t *testing.T) {
	tests := []struct {
		name     string
		message  *telegram.Message
		wantText string
		wantHTML string
		wantURL  string
	}{
		{
			name:     "plain text",
			message:  &telegram.Message{MessageID: 1, Chat: telegram.Chat{ID: 5}, Text: "hello"},
			wantText: "hello",
			wantHTML: "hello",
		},
		{
			name:     "caption overrides text",
			message:  &telegram.Message{MessageID: 1, Chat: telegram.Chat{ID: 5}, Text: "body", Caption: "cap"},
			wantText: "cap",
			wantHTML: "body",
		},
		{
			name: "forward from channel sets url",
			message: &telegram.Message{
				MessageID: 1,
				Chat:      telegram.Chat{ID: 5},
				Text:      "channel post",
				ForwardOrigin: &telegram.ForwardOrigin{
					Type:      telegram.MessageOriginChannel,
					Chat:      &telegram.Chat{UserName: "mychan"},
					MessageID: 42,
				},
			},
			wantText: "channel post",
			wantHTML: "channel post",
			wantURL:  "https://t.me/mychan/42",
		},
		{
			name: "forward from user prefixes sender",
			message: &telegram.Message{
				MessageID: 1,
				Chat:      telegram.Chat{ID: 5},
				Text:      "hi",
				ForwardOrigin: &telegram.ForwardOrigin{
					Type:       telegram.MessageOriginUser,
					SenderUser: &telegram.User{FirstName: "John", LastName: "Doe", UserName: "jd"},
				},
			},
			wantText: "John Doe (jd):\nhi",
			wantHTML: "hi",
		},
		{
			name: "forward from hidden user prefixes name",
			message: &telegram.Message{
				MessageID: 1,
				Chat:      telegram.Chat{ID: 5},
				Text:      "hi",
				ForwardOrigin: &telegram.ForwardOrigin{
					Type:           telegram.MessageOriginHiddenUser,
					SenderUserName: "Anon",
				},
			},
			wantText: "Anon:\nhi",
			wantHTML: "hi",
		},
	}

	tl := &TelegramListener{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tl.transform(tt.message)
			assert.Equal(t, tt.message.MessageID, msg.ID)
			assert.Equal(t, tt.message.Chat.ID, msg.ChatID)
			assert.Equal(t, tt.wantText, msg.Text)
			assert.Equal(t, tt.wantHTML, msg.HTML)
			assert.Equal(t, tt.wantURL, msg.Url)
		})
	}
}

func TestTelegramListener_Do(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var mu sync.Mutex
	callCount := 0
	var offsets []int

	tgMock := &mocks.TelegramAPIMock{
		GetUpdatesFunc: func(_ context.Context, offset, _ int) ([]telegram.Update, error) {
			mu.Lock()
			callCount++
			c := callCount
			offsets = append(offsets, offset)
			mu.Unlock()

			if c == 1 {
				return []telegram.Update{{UpdateID: 5, Message: pingMessage()}}, nil
			}

			cancel()
			return nil, context.Canceled
		},
		SendMessageFunc: func(_ context.Context, _ int64, _, _ string) error {
			return nil
		},
	}

	tl := &TelegramListener{
		SuperUsers:      []int64{111},
		TbAPI:           tgMock,
		Bot:             &botStub{},
		MessagesForSend: make(chan MessagePayload),
	}

	err := tl.Do(ctx)
	require.ErrorIs(t, err, context.Canceled)

	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, len(offsets), 2)
	assert.Equal(t, 0, offsets[0])
	assert.Equal(t, 6, offsets[1])

	sendCalls := tgMock.SendMessageCalls()
	require.Len(t, sendCalls, 1)
	assert.Equal(t, "🏓 Pong!", sendCalls[0].Text)
}

func TestTelegramListener_DoTransientError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	tgMock := &mocks.TelegramAPIMock{
		GetUpdatesFunc: func(_ context.Context, _, _ int) ([]telegram.Update, error) {
			callCount++
			if callCount == 1 {
				go func() {
					time.Sleep(10 * time.Millisecond)
					cancel()
				}()
				return nil, errors.New("transient")
			}

			return nil, context.Canceled
		},
	}

	tl := &TelegramListener{
		TbAPI:           tgMock,
		Bot:             &botStub{},
		MessagesForSend: make(chan MessagePayload),
	}

	err := tl.Do(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.GreaterOrEqual(t, callCount, 1)
}

func TestSendMessagesForAdmins(t *testing.T) {
	tests := []struct {
		name       string
		payload    MessagePayload
		superUsers []int64
	}{
		{
			name:       "plain text sends without parse mode",
			payload:    MessagePayload{Text: "hello"},
			superUsers: []int64{111, 222},
		},
		{
			name:       "MarkdownV2 sets parse mode",
			payload:    MessagePayload{Text: "*bold*", ParseMode: "MarkdownV2"},
			superUsers: []int64{111},
		},
		{
			name:       "HTML sets parse mode",
			payload:    MessagePayload{Text: "<b>bold</b>", ParseMode: "HTML"},
			superUsers: []int64{111},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgMock := &mocks.TelegramAPIMock{
				SendMessageFunc: func(_ context.Context, _ int64, _, _ string) error {
					return nil
				},
			}
			ch := make(chan MessagePayload, 1)

			tl := &TelegramListener{
				SuperUsers:      tt.superUsers,
				TbAPI:           tgMock,
				MessagesForSend: ch,
			}

			go tl.SendMessagesForAdmins(t.Context())

			ch <- tt.payload

			require.Eventually(t, func() bool {
				return len(tgMock.SendMessageCalls()) == len(tt.superUsers)
			}, time.Second, 10*time.Millisecond)

			calls := tgMock.SendMessageCalls()
			require.Len(t, calls, len(tt.superUsers))

			for i, c := range calls {
				assert.Equal(t, tt.superUsers[i], c.ChatID)
				assert.Equal(t, tt.payload.Text, c.Text)
				assert.Equal(t, tt.payload.ParseMode, c.ParseMode)
			}
		})
	}
}
