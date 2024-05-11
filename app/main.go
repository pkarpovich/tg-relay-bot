package main

import (
	"fmt"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkarpovich/tg-relay-bot/app/bot"
	"github.com/pkarpovich/tg-relay-bot/app/config"
	"github.com/pkarpovich/tg-relay-bot/app/events"
	"log"
)

func main() {
	log.Printf("[INFO] Starting application...")

	cfg, err := config.Init()
	if err != nil {
		log.Fatalf("[ERROR] Error reading config: %s", err)
	}

	if err = run(cfg); err != nil {
		log.Fatalf("[ERROR] Application error: %s", err)
	}
}

func run(cfg *config.Config) error {
	messagesForSend := make(chan string)
	botClient := bot.NewClient()

	tbAPI, err := tbapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return fmt.Errorf("failed to create Telegram events: %w", err)
	}

	tgListener := &events.TelegramListener{
		SuperUsers:      cfg.Telegram.SuperUsers,
		TbAPI:           tbAPI,
		Bot:             botClient,
		MessagesForSend: messagesForSend,
	}

	if err := tgListener.Do(); err != nil {
		return fmt.Errorf("failed to start Telegram listener: %w", err)
	}

	return nil
}
