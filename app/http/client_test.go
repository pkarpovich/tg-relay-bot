package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pkarpovich/tg-relay-bot/app/config"
	"github.com/pkarpovich/tg-relay-bot/app/events"
)

func TestSendHandler(t *testing.T) {
	tests := []struct {
		name           string
		secret         string
		body           any
		wantStatus     int
		wantPayload    *events.MessagePayload
		wantErrMessage string
	}{
		{
			name:       "plain text message",
			secret:     "test-secret",
			body:       map[string]string{"message": "hello"},
			wantStatus: http.StatusOK,
			wantPayload: &events.MessagePayload{
				Text: "hello",
			},
		},
		{
			name:       "message with MarkdownV2 parse mode",
			secret:     "test-secret",
			body:       map[string]string{"message": "*bold*", "parse_mode": "MarkdownV2"},
			wantStatus: http.StatusOK,
			wantPayload: &events.MessagePayload{
				Text:      "*bold*",
				ParseMode: "MarkdownV2",
			},
		},
		{
			name:       "message with HTML parse mode",
			secret:     "test-secret",
			body:       map[string]string{"message": "<b>bold</b>", "parse_mode": "HTML"},
			wantStatus: http.StatusOK,
			wantPayload: &events.MessagePayload{
				Text:      "<b>bold</b>",
				ParseMode: "HTML",
			},
		},
		{
			name:           "wrong secret",
			secret:         "wrong-secret",
			body:           map[string]string{"message": "hello"},
			wantStatus:     http.StatusUnauthorized,
			wantErrMessage: "unauthorized",
		},
		{
			name:           "invalid json body",
			secret:         "test-secret",
			body:           "not json",
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "invalid character",
		},
		{
			name:           "empty message",
			secret:         "test-secret",
			body:           map[string]string{"message": ""},
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "message is required",
		},
		{
			name:           "unsupported parse_mode",
			secret:         "test-secret",
			body:           map[string]string{"message": "hello", "parse_mode": "Markdown"},
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "unsupported parse_mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan events.MessagePayload, 1)
			srv := &Server{
				config: &config.Config{
					Http: config.HttpConfig{SecretApiKey: "test-secret"},
				},
				messagesForSend: ch,
			}

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				var err error
				bodyBytes, err = json.Marshal(v)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/send", bytes.NewReader(bodyBytes))
			req.Header.Set("X-Secret", tt.secret)
			rec := httptest.NewRecorder()

			srv.sendHandler(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantPayload != nil {
				require.Len(t, ch, 1)
				got := <-ch
				assert.Equal(t, *tt.wantPayload, got)
			}

			if tt.wantErrMessage != "" {
				var errResp ErrorResponse
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&errResp))
				assert.Contains(t, errResp.Error, tt.wantErrMessage)
			}
		})
	}
}
