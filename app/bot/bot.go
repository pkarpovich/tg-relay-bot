package bot

import (
	"time"
)

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"user_name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

type Message struct {
	ID     int
	From   User
	ChatID int64
	Sent   time.Time
	HTML   string `json:",omitempty"`
	Text   string `json:",omitempty"`
	Url    string
}

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) OnMessage(msg Message) (bool, error) {
	return true, nil
}
