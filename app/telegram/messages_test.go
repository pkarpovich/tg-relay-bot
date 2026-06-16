package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMessage(t *testing.T) {
	tests := []struct {
		name      string
		chatID    int64
		text      string
		parseMode string
		want      map[string]any
	}{
		{
			name:   "plain text omits parse_mode",
			chatID: 111,
			text:   "hello",
			want:   map[string]any{"chat_id": float64(111), "text": "hello"},
		},
		{
			name:      "MarkdownV2 passes parse_mode",
			chatID:    222,
			text:      "*bold*",
			parseMode: "MarkdownV2",
			want:      map[string]any{"chat_id": float64(222), "text": "*bold*", "parse_mode": "MarkdownV2"},
		},
		{
			name:      "HTML passes parse_mode",
			chatID:    333,
			text:      "<b>bold</b>",
			parseMode: "HTML",
			want:      map[string]any{"chat_id": float64(333), "text": "<b>bold</b>", "parse_mode": "HTML"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			var gotBody map[string]any

			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				raw, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(raw, &gotBody)

				_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
			})

			err := c.SendMessage(t.Context(), tt.chatID, tt.text, tt.parseMode)
			require.NoError(t, err)

			assert.Equal(t, "/bottest-token/sendMessage", gotPath)
			assert.Equal(t, tt.want, gotBody)
		})
	}
}

func TestSendMessageError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: chat not found"}`))
	})

	err := c.SendMessage(t.Context(), 111, "hello", "")
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
}
