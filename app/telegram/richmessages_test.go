package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendRichMessage(t *testing.T) {
	tests := []struct {
		name     string
		chatID   int64
		markdown string
		want     map[string]any
	}{
		{
			name:     "plain markdown",
			chatID:   111,
			markdown: "# Title\n**bold**",
			want: map[string]any{
				"chat_id":      float64(111),
				"rich_message": map[string]any{"markdown": "# Title\n**bold**"},
			},
		},
		{
			name:     "empty markdown still nests rich_message",
			chatID:   222,
			markdown: "",
			want: map[string]any{
				"chat_id":      float64(222),
				"rich_message": map[string]any{"markdown": ""},
			},
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

			err := c.SendRichMessage(t.Context(), tt.chatID, tt.markdown)
			require.NoError(t, err)

			assert.Equal(t, "/bottest-token/sendRichMessage", gotPath)
			assert.Equal(t, tt.want, gotBody)
		})
	}
}

func TestSendRichMessageError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: chat not found"}`))
	})

	err := c.SendRichMessage(t.Context(), 111, "# Title")
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
}
