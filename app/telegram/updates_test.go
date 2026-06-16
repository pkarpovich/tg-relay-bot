package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUpdates(t *testing.T) {
	var gotPath string
	var gotBody map[string]any

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)

		_, _ = w.Write([]byte(`{"ok":true,"result":[
			{"update_id":10,"message":{"message_id":1,"date":1700000000,"text":"hi","from":{"id":42,"username":"alice"},"chat":{"id":99,"username":"alicechat"}}},
			{"update_id":11,"message":{"message_id":2,"date":1700000100,"caption":"a photo"}}
		]}`))
	})

	updates, err := c.GetUpdates(t.Context(), 5, 60)
	require.NoError(t, err)

	assert.Equal(t, "/bottest-token/getUpdates", gotPath)
	assert.Equal(t, map[string]any{"offset": float64(5), "timeout": float64(60)}, gotBody)

	require.Len(t, updates, 2)

	assert.Equal(t, 10, updates[0].UpdateID)
	require.NotNil(t, updates[0].Message)
	assert.Equal(t, 1, updates[0].Message.MessageID)
	assert.Equal(t, "hi", updates[0].Message.Text)
	require.NotNil(t, updates[0].Message.From)
	assert.Equal(t, int64(42), updates[0].Message.From.ID)
	assert.Equal(t, "alice", updates[0].Message.From.UserName)
	assert.Equal(t, int64(99), updates[0].Message.Chat.ID)
	assert.Equal(t, "alicechat", updates[0].Message.Chat.UserName)
	assert.Equal(t, time.Unix(1700000000, 0), updates[0].Message.Time())

	assert.Equal(t, 11, updates[1].UpdateID)
	assert.Equal(t, "a photo", updates[1].Message.Caption)
	assert.Nil(t, updates[1].Message.From)
}

func TestGetUpdatesEmptyResult(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
	})

	updates, err := c.GetUpdates(t.Context(), 0, 60)
	require.NoError(t, err)
	assert.Empty(t, updates)
}

func TestGetUpdatesError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Unauthorized"}`))
	})

	_, err := c.GetUpdates(t.Context(), 0, 60)
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Code)
}

func TestMessageCommand(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		entities []MessageEntity
		want     string
	}{
		{
			name:     "ping command",
			text:     "/ping",
			entities: []MessageEntity{{Type: "bot_command", Offset: 0, Length: 5}},
			want:     "ping",
		},
		{
			name:     "command with at mention stripped",
			text:     "/ping@my_bot",
			entities: []MessageEntity{{Type: "bot_command", Offset: 0, Length: 12}},
			want:     "ping",
		},
		{
			name:     "no entities",
			text:     "/ping",
			entities: nil,
			want:     "",
		},
		{
			name:     "non-command entity",
			text:     "hello",
			entities: []MessageEntity{{Type: "bold", Offset: 0, Length: 5}},
			want:     "",
		},
		{
			name:     "command not at start",
			text:     "hey /ping",
			entities: []MessageEntity{{Type: "bot_command", Offset: 4, Length: 5}},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Message{Text: tt.text, Entities: tt.entities}
			assert.Equal(t, tt.want, m.Command())
		})
	}
}

func TestForwardOriginUnmarshal(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		assert func(t *testing.T, o *ForwardOrigin)
	}{
		{
			name: "channel",
			raw:  `{"type":"channel","chat":{"id":-100,"username":"newschannel"},"message_id":777}`,
			assert: func(t *testing.T, o *ForwardOrigin) {
				assert.Equal(t, MessageOriginChannel, o.Type)
				require.NotNil(t, o.Chat)
				assert.Equal(t, "newschannel", o.Chat.UserName)
				assert.Equal(t, 777, o.MessageID)
			},
		},
		{
			name: "user",
			raw:  `{"type":"user","sender_user":{"id":7,"first_name":"John","last_name":"Doe","username":"jdoe"}}`,
			assert: func(t *testing.T, o *ForwardOrigin) {
				assert.Equal(t, MessageOriginUser, o.Type)
				require.NotNil(t, o.SenderUser)
				assert.Equal(t, "John", o.SenderUser.FirstName)
				assert.Equal(t, "Doe", o.SenderUser.LastName)
				assert.Equal(t, "jdoe", o.SenderUser.UserName)
			},
		},
		{
			name: "hidden user",
			raw:  `{"type":"hidden_user","sender_user_name":"Anonymous"}`,
			assert: func(t *testing.T, o *ForwardOrigin) {
				assert.Equal(t, MessageOriginHiddenUser, o.Type)
				assert.Equal(t, "Anonymous", o.SenderUserName)
				assert.Nil(t, o.SenderUser)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var o ForwardOrigin
			require.NoError(t, json.Unmarshal([]byte(tt.raw), &o))
			tt.assert(t, &o)
		})
	}
}
