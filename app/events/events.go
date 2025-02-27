package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	tbapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/pkarpovich/tg-relay-bot/app/bot"
	"log"
)

const (
	PingCommand = "ping"
)

type Bot interface {
	OnMessage(msg bot.Message) (bool, error)
}

type TbAPI interface {
	GetUpdatesChan(config tbapi.UpdateConfig) tbapi.UpdatesChannel
	Send(c tbapi.Chattable) (tbapi.Message, error)
	Request(c tbapi.Chattable) (*tbapi.APIResponse, error)
}

type TelegramListener struct {
	SuperUsers      []int64
	TbAPI           TbAPI
	Bot             Bot
	MessagesForSend chan string
}

type RemoveTaskData struct {
	TaskID string `json:"taskId"`
	Type   string `json:"type"`
}

func (tl *TelegramListener) Do(ctx context.Context) error {
	u := tbapi.NewUpdate(0)
	u.Timeout = 60

	updates := tl.TbAPI.GetUpdatesChan(u)

	go tl.SendMessagesForAdmins(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				return fmt.Errorf("telegram update chan closed")
			}

			if update.Message == nil {
				continue
			}

			if err := tl.processEvent(update); err != nil {
				log.Printf("[ERROR] %v", err)
			}
		}
	}
}

func (tl *TelegramListener) Shutdown(_ context.Context) error {
	return nil
}

func (tl *TelegramListener) processEvent(update tbapi.Update) error {
	msgJSON, errJSON := json.Marshal(update.Message)
	if errJSON != nil {
		return fmt.Errorf("failed to marshal update.Message to json: %w", errJSON)
	}
	log.Printf("[DEBUG] %s", string(msgJSON))

	if !tl.isSuperUser(update.Message.From.ID) {
		log.Printf("[DEBUG] user %d is not super user", update.Message.From.ID)

		msg := tbapi.NewMessage(update.Message.Chat.ID, "I don't know you 🤷‍")
		_, err := tl.TbAPI.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		return nil
	}

	switch update.Message.Command() {
	case PingCommand:
		tl.handlePingCommand(update)
		return nil
	}

	msg := tl.transform(update.Message)
	saved, err := tl.Bot.OnMessage(msg)
	if err != nil {
		errMsg := tbapi.NewMessage(update.Message.Chat.ID, "💥 Error: "+err.Error())
		_, err := tl.TbAPI.Send(errMsg)
		if err != nil {
			return fmt.Errorf("failed to send error message: %w", err)
		}

		return errors.New(errMsg.Text)
	}

	if !saved {
		return nil
	}

	if err := tl.reactToMessage(update.Message.Chat.ID, update.Message.MessageID, tbapi.ReactionType{
		Type:  "emoji",
		Emoji: "👍",
	}); err != nil {
		return fmt.Errorf("failed to react to message: %w", err)
	}

	return nil
}

func (tl *TelegramListener) transform(message *tbapi.Message) bot.Message {
	msg := bot.Message{
		ID:     message.MessageID,
		From:   bot.User{},
		ChatID: message.Chat.ID,
		HTML:   message.Text,
		Text:   message.Text,
		Sent:   message.Time(),
	}

	if len(message.Caption) > 0 {
		msg.Text = message.Caption
	}

	if message.ForwardOrigin != nil {
		origin := message.ForwardOrigin

		switch message.ForwardOrigin.Type {
		case tbapi.MessageOriginChannel:
			msg.Url = fmt.Sprintf("https://t.me/%s/%d", origin.Chat.UserName, origin.MessageID)
		case tbapi.MessageOriginUser:
			msg.Text = fmt.Sprintf(
				"%s %s (%s):\n%s",
				origin.SenderUser.FirstName,
				origin.SenderUser.LastName,
				origin.SenderUser.UserName,
				message.Text,
			)
		case tbapi.MessageOriginHiddenUser:
			msg.Text = fmt.Sprintf("%s:\n%s", origin.SenderUserName, message.Text)
		}
	}

	return msg
}

func (tl *TelegramListener) handlePingCommand(update tbapi.Update) {
	msg := tbapi.NewMessage(update.Message.Chat.ID, "🏓 Pong!")
	_, err := tl.TbAPI.Send(msg)
	if err != nil {
		log.Printf("[ERROR] failed to send message: %v", err)
	}
}

func (tl *TelegramListener) SendMessagesForAdmins(ctx context.Context) {
	adminIds := tl.SuperUsers

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-tl.MessagesForSend:
			for _, adminID := range adminIds {
				_, err := tl.TbAPI.Send(NewMessage(adminID, msg))
				if err != nil {
					log.Printf("[ERROR] failed to send message: %v", err)
				}
			}
		}
	}
}

func (tl *TelegramListener) isSuperUser(userID int64) bool {
	for _, su := range tl.SuperUsers {
		if su == userID {
			return true
		}
	}

	return false
}

func (tl *TelegramListener) reactToMessage(chatID int64, messageID int, reaction tbapi.ReactionType) error {
	reactionMsg := tbapi.SetMessageReactionConfig{
		BaseChatMessage: tbapi.BaseChatMessage{
			ChatConfig: tbapi.ChatConfig{
				ChatID: chatID,
			},
			MessageID: messageID,
		},
		Reaction: []tbapi.ReactionType{reaction},
		IsBig:    false,
	}

	_, err := tl.TbAPI.Request(reactionMsg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func NewMarkdownMessage(chatID int64, text string, replyMarkup *tbapi.InlineKeyboardMarkup) tbapi.MessageConfig {
	return tbapi.MessageConfig{
		BaseChat: tbapi.BaseChat{
			ChatConfig: tbapi.ChatConfig{
				ChatID: chatID,
			},
			ReplyMarkup: replyMarkup,
		},
		LinkPreviewOptions: tbapi.LinkPreviewOptions{
			IsDisabled: false,
		},
		ParseMode: tbapi.ModeMarkdownV2,
		Text:      text,
	}
}

func NewMessage(chatID int64, text string) tbapi.MessageConfig {
	return tbapi.NewMessage(chatID, text)
}
