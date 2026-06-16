package telegram

import (
	"context"
	"fmt"
)

type ReactionType struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
}

type setMessageReactionRequest struct {
	ChatID    int64          `json:"chat_id"`
	MessageID int            `json:"message_id"`
	Reaction  []ReactionType `json:"reaction"`
}

func (c *Client) SetMessageReaction(ctx context.Context, chatID int64, messageID int, emoji string) error {
	payload := setMessageReactionRequest{
		ChatID:    chatID,
		MessageID: messageID,
		Reaction:  []ReactionType{{Type: "emoji", Emoji: emoji}},
	}

	if _, err := c.do(ctx, "setMessageReaction", payload); err != nil {
		return fmt.Errorf("set message reaction: %w", err)
	}

	return nil
}
