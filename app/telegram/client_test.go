package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errRoundTripper struct{}

func (errRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("transport boom")
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return NewClient(Config{
		Token:      "test-token",
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
	})
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient(Config{Token: "abc"})

	assert.Equal(t, "abc", c.token)
	assert.Equal(t, defaultBaseURL, c.baseURL)
	assert.NotNil(t, c.httpClient)
}

func TestDoSuccess(t *testing.T) {
	var gotMethod, gotPath, gotContentType string
	var gotBody json.RawMessage

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)

		_, _ = w.Write([]byte(`{"ok":true,"result":{"foo":"bar"}}`))
	})

	result, err := c.do(t.Context(), "someMethod", map[string]string{"hello": "world"})
	require.NoError(t, err)

	assert.JSONEq(t, `{"foo":"bar"}`, string(result))
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/bottest-token/someMethod", gotPath)
	assert.Equal(t, "application/json", gotContentType)
	assert.JSONEq(t, `{"hello":"world"}`, string(gotBody))
}

func TestDoAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: chat not found"}`))
	})

	_, err := c.do(t.Context(), "someMethod", nil)
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	assert.Equal(t, "Bad Request: chat not found", apiErr.Description)
	assert.Contains(t, apiErr.Error(), "chat not found")
}

func TestDoTransportError(t *testing.T) {
	c := NewClient(Config{
		Token:      "test-token",
		HTTPClient: &http.Client{Transport: errRoundTripper{}},
		BaseURL:    "http://example.invalid",
	})

	_, err := c.do(t.Context(), "someMethod", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send someMethod request")
}

func TestDoDecodeError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	})

	_, err := c.do(t.Context(), "someMethod", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode someMethod response")
}

func TestDoRetriesOn429(t *testing.T) {
	var calls atomic.Int32

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"description":"Too Many Requests","parameters":{"retry_after":1}}`))
			return
		}

		_, _ = w.Write([]byte(`{"ok":true,"result":{"retried":true}}`))
	})

	result, err := c.do(t.Context(), "someMethod", nil)
	require.NoError(t, err)

	assert.JSONEq(t, `{"retried":true}`, string(result))
	assert.Equal(t, int32(2), calls.Load())
}

func TestDoRetryReturnsSecondError(t *testing.T) {
	var calls atomic.Int32

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"description":"Too Many Requests","parameters":{"retry_after":1}}`))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: second failure"}`))
	})

	_, err := c.do(t.Context(), "someMethod", nil)
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Code)
	assert.Equal(t, "Bad Request: second failure", apiErr.Description)
	assert.Equal(t, int32(2), calls.Load())
}

func TestDoNoRetryWithoutRetryAfter(t *testing.T) {
	var calls atomic.Int32

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Too Many Requests"}`))
	})

	_, err := c.do(t.Context(), "someMethod", nil)
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusTooManyRequests, apiErr.Code)
	assert.Equal(t, int32(1), calls.Load())
}

func TestDoRetryStopsOnCancelledContext(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Too Many Requests","parameters":{"retry_after":60}}`))
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := c.do(ctx, "someMethod", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
