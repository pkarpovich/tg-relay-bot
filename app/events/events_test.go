package events

import (
	"fmt"
	"sync"
	"testing"
	"time"

	tbapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTbAPI struct {
	mu       sync.Mutex
	messages []tbapi.MessageConfig
}

func (m *mockTbAPI) GetUpdatesChan(_ tbapi.UpdateConfig) tbapi.UpdatesChannel {
	return make(chan tbapi.Update)
}

func (m *mockTbAPI) Send(c tbapi.Chattable) (tbapi.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := c.(tbapi.MessageConfig)
	if !ok {
		return tbapi.Message{}, fmt.Errorf("unexpected Chattable type: %T", c)
	}
	m.messages = append(m.messages, msg)
	return tbapi.Message{}, nil
}

func (m *mockTbAPI) Request(_ tbapi.Chattable) (*tbapi.APIResponse, error) {
	return &tbapi.APIResponse{Ok: true}, nil
}

func (m *mockTbAPI) getMessages() []tbapi.MessageConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]tbapi.MessageConfig, len(m.messages))
	copy(result, m.messages)
	return result
}

func TestSendMessagesForAdmins(t *testing.T) {
	tests := []struct {
		name          string
		payload       MessagePayload
		superUsers    []int64
		wantParseMode string
	}{
		{
			name:          "plain text sends without parse mode",
			payload:       MessagePayload{Text: "hello"},
			superUsers:    []int64{111, 222},
			wantParseMode: "",
		},
		{
			name:          "MarkdownV2 sets parse mode",
			payload:       MessagePayload{Text: "*bold*", ParseMode: "MarkdownV2"},
			superUsers:    []int64{111},
			wantParseMode: tbapi.ModeMarkdownV2,
		},
		{
			name:          "HTML sets parse mode",
			payload:       MessagePayload{Text: "<b>bold</b>", ParseMode: "HTML"},
			superUsers:    []int64{111},
			wantParseMode: tbapi.ModeHTML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTbAPI{}
			ch := make(chan MessagePayload, 1)

			tl := &TelegramListener{
				SuperUsers:      tt.superUsers,
				TbAPI:           mock,
				MessagesForSend: ch,
			}

			go tl.SendMessagesForAdmins(t.Context())

			ch <- tt.payload

			require.Eventually(t, func() bool {
				return len(mock.getMessages()) == len(tt.superUsers)
			}, time.Second, 10*time.Millisecond)

			msgs := mock.getMessages()
			require.Len(t, msgs, len(tt.superUsers))

			for i, msg := range msgs {
				assert.Equal(t, tt.superUsers[i], msg.ChatID)
				assert.Equal(t, tt.payload.Text, msg.Text)
				assert.Equal(t, tt.wantParseMode, msg.ParseMode)
			}
		})
	}
}
