package telegram

import (
	"context"
	"fmt"
)

type sendMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text, parseMode string) error {
	payload := sendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: parseMode,
	}

	if _, err := c.do(ctx, "sendMessage", payload); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}
