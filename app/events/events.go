package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/pkarpovich/tg-relay-bot/app/bot"
	"github.com/pkarpovich/tg-relay-bot/app/telegram"
)

const (
	PingCommand   = "ping"
	updateTimeout = 60
	errorBackoff  = 5 * time.Second
)

type MessagePayload struct {
	Text      string
	ParseMode string
}

type Bot interface {
	OnMessage(msg bot.Message) (bool, error)
}

//go:generate moq -out mocks/telegram_api.go -pkg mocks -skip-ensure . TelegramAPI

type TelegramAPI interface {
	GetUpdates(ctx context.Context, offset, timeoutSec int) ([]telegram.Update, error)
	SendMessage(ctx context.Context, chatID int64, text, parseMode string) error
	SetMessageReaction(ctx context.Context, chatID int64, messageID int, emoji string) error
}

type TelegramListener struct {
	SuperUsers      []int64
	TbAPI           TelegramAPI
	Bot             Bot
	MessagesForSend chan MessagePayload
}

func (tl *TelegramListener) Do(ctx context.Context) error {
	go tl.SendMessagesForAdmins(ctx)

	offset := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("telegram listener context done: %w", ctx.Err())
		default:
		}

		updates, err := tl.TbAPI.GetUpdates(ctx, offset, updateTimeout)
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("telegram listener context done: %w", ctx.Err())
			}

			log.Printf("[ERROR] failed to get updates: %v", err)
			select {
			case <-ctx.Done():
				return fmt.Errorf("telegram listener context done: %w", ctx.Err())
			case <-time.After(errorBackoff):
			}

			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1

			if update.Message == nil {
				continue
			}

			if err := tl.processEvent(ctx, update); err != nil {
				log.Printf("[ERROR] %v", err)
			}
		}
	}
}

func (tl *TelegramListener) Shutdown(_ context.Context) error {
	return nil
}

func (tl *TelegramListener) processEvent(ctx context.Context, update telegram.Update) error {
	msgJSON, errJSON := json.Marshal(update.Message)
	if errJSON != nil {
		return fmt.Errorf("failed to marshal update.Message to json: %w", errJSON)
	}
	log.Printf("[DEBUG] %s", string(msgJSON))

	if update.Message.From == nil {
		return nil
	}

	if !tl.isSuperUser(update.Message.From.ID) {
		log.Printf("[DEBUG] user %d is not super user", update.Message.From.ID)

		if err := tl.TbAPI.SendMessage(ctx, update.Message.Chat.ID, "I don't know you 🤷‍", ""); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		return nil
	}

	if update.Message.Command() == PingCommand {
		tl.handlePingCommand(ctx, update)
		return nil
	}

	msg := tl.transform(update.Message)
	saved, err := tl.Bot.OnMessage(msg)
	if err != nil {
		errText := "💥 Error: " + err.Error()
		if sendErr := tl.TbAPI.SendMessage(ctx, update.Message.Chat.ID, errText, ""); sendErr != nil {
			return fmt.Errorf("failed to send error message: %w", sendErr)
		}

		return errors.New(errText)
	}

	if !saved {
		return nil
	}

	if err := tl.TbAPI.SetMessageReaction(ctx, update.Message.Chat.ID, update.Message.MessageID, "👍"); err != nil {
		return fmt.Errorf("failed to react to message: %w", err)
	}

	return nil
}

func (tl *TelegramListener) transform(message *telegram.Message) bot.Message {
	msg := bot.Message{
		ID:     message.MessageID,
		From:   bot.User{},
		ChatID: message.Chat.ID,
		HTML:   message.Text,
		Text:   message.Text,
		Sent:   message.Time(),
	}

	if message.Caption != "" {
		msg.Text = message.Caption
	}

	if message.ForwardOrigin != nil {
		origin := message.ForwardOrigin

		switch origin.Type {
		case telegram.MessageOriginChannel:
			msg.Url = fmt.Sprintf("https://t.me/%s/%d", origin.Chat.UserName, origin.MessageID)
		case telegram.MessageOriginUser:
			msg.Text = fmt.Sprintf(
				"%s %s (%s):\n%s",
				origin.SenderUser.FirstName,
				origin.SenderUser.LastName,
				origin.SenderUser.UserName,
				message.Text,
			)
		case telegram.MessageOriginHiddenUser:
			msg.Text = fmt.Sprintf("%s:\n%s", origin.SenderUserName, message.Text)
		}
	}

	return msg
}

func (tl *TelegramListener) handlePingCommand(ctx context.Context, update telegram.Update) {
	if err := tl.TbAPI.SendMessage(ctx, update.Message.Chat.ID, "🏓 Pong!", ""); err != nil {
		log.Printf("[ERROR] failed to send message: %v", err)
	}
}

func (tl *TelegramListener) SendMessagesForAdmins(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case payload := <-tl.MessagesForSend:
			for _, adminID := range tl.SuperUsers {
				if err := tl.TbAPI.SendMessage(ctx, adminID, payload.Text, payload.ParseMode); err != nil {
					log.Printf("[ERROR] failed to send message: %v", err)
				}
			}
		}
	}
}

func (tl *TelegramListener) isSuperUser(userID int64) bool {
	return slices.Contains(tl.SuperUsers, userID)
}
