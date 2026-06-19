package telegram

import (
	"context"
	"fmt"
)

type inputRichMessage struct {
	Markdown string `json:"markdown"`
}

type sendRichMessageRequest struct {
	ChatID      int64            `json:"chat_id"`
	RichMessage inputRichMessage `json:"rich_message"`
}

func (c *Client) SendRichMessage(ctx context.Context, chatID int64, markdown string) error {
	payload := sendRichMessageRequest{
		ChatID:      chatID,
		RichMessage: inputRichMessage{Markdown: markdown},
	}

	if _, err := c.do(ctx, "sendRichMessage", payload); err != nil {
		return fmt.Errorf("send rich message: %w", err)
	}

	return nil
}
