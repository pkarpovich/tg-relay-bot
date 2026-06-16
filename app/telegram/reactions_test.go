package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetMessageReaction(t *testing.T) {
	var gotPath string
	var gotBody map[string]any

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)

		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	})

	err := c.SetMessageReaction(t.Context(), 111, 222, "👍")
	require.NoError(t, err)

	assert.Equal(t, "/bottest-token/setMessageReaction", gotPath)
	assert.Equal(t, map[string]any{
		"chat_id":    float64(111),
		"message_id": float64(222),
		"reaction":   []any{map[string]any{"type": "emoji", "emoji": "👍"}},
	}, gotBody)
}

func TestSetMessageReactionError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: message not found"}`))
	})

	err := c.SetMessageReaction(t.Context(), 111, 222, "👍")
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
}
