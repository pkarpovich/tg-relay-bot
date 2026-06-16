package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	MessageOriginUser       = "user"
	MessageOriginHiddenUser = "hidden_user"
	MessageOriginChannel    = "channel"
)

type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	MessageID     int             `json:"message_id"`
	From          *User           `json:"from"`
	Chat          Chat            `json:"chat"`
	Date          int64           `json:"date"`
	Text          string          `json:"text"`
	Caption       string          `json:"caption"`
	Entities      []MessageEntity `json:"entities"`
	ForwardOrigin *ForwardOrigin  `json:"forward_origin"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	UserName  string `json:"username"`
}

type Chat struct {
	ID       int64  `json:"id"`
	UserName string `json:"username"`
}

type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

type ForwardOrigin struct {
	Type           string `json:"type"`
	Chat           *Chat  `json:"chat"`
	MessageID      int    `json:"message_id"`
	SenderUser     *User  `json:"sender_user"`
	SenderUserName string `json:"sender_user_name"`
}

func (m *Message) Time() time.Time {
	return time.Unix(m.Date, 0)
}

func (m *Message) Command() string {
	if len(m.Entities) == 0 {
		return ""
	}

	entity := m.Entities[0]
	if entity.Offset != 0 || entity.Type != "bot_command" {
		return ""
	}

	command := m.Text[1:entity.Length]
	if i := strings.Index(command, "@"); i != -1 {
		command = command[:i]
	}

	return command
}

type getUpdatesRequest struct {
	Offset  int `json:"offset"`
	Timeout int `json:"timeout"`
}

func (c *Client) GetUpdates(ctx context.Context, offset, timeoutSec int) ([]Update, error) {
	payload := getUpdatesRequest{
		Offset:  offset,
		Timeout: timeoutSec,
	}

	result, err := c.do(ctx, "getUpdates", payload)
	if err != nil {
		return nil, fmt.Errorf("get updates: %w", err)
	}

	var updates []Update
	if err := json.Unmarshal(result, &updates); err != nil {
		return nil, fmt.Errorf("unmarshal updates: %w", err)
	}

	return updates, nil
}
