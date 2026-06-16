package telegram

import (
	"context"
	"encoding/json"
	"fmt"
)

func (c *Client) GetMe(ctx context.Context) (*User, error) {
	result, err := c.do(ctx, "getMe", struct{}{})
	if err != nil {
		return nil, fmt.Errorf("get me: %w", err)
	}

	var user User
	if err := json.Unmarshal(result, &user); err != nil {
		return nil, fmt.Errorf("unmarshal me: %w", err)
	}

	return &user, nil
}
