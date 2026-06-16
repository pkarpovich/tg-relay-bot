package telegram

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMe(t *testing.T) {
	var gotPath string

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"ok":true,"result":{"id":42,"username":"relay_bot"}}`))
	})

	user, err := c.GetMe(t.Context())
	require.NoError(t, err)

	assert.Equal(t, "/bottest-token/getMe", gotPath)
	assert.Equal(t, int64(42), user.ID)
	assert.Equal(t, "relay_bot", user.UserName)
}

func TestGetMeError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Unauthorized"}`))
	})

	_, err := c.GetMe(t.Context())
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Code)
}
