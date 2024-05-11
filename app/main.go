package main

import (
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkarpovich/tg-relay-bot/app/bot"
	"github.com/pkarpovich/tg-relay-bot/app/config"
	"github.com/pkarpovich/tg-relay-bot/app/events"
	"github.com/pkarpovich/tg-relay-bot/app/http"
	"log"
	"sync"
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
	var wg sync.WaitGroup

	wg.Add(2)
	go startTelegramListener(cfg, &wg)
	go startHttpServer(cfg, &wg)
	wg.Wait()

	return nil
}

func startTelegramListener(cfg *config.Config, wg *sync.WaitGroup) {
	defer wg.Done()

	messagesForSend := make(chan string)
	botClient := bot.NewClient()

	tbAPI, err := tbapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		log.Fatalf("[ERROR] failed to create Telegram events: %s", err)
	}

	tgListener := &events.TelegramListener{
		SuperUsers:      cfg.Telegram.SuperUsers,
		TbAPI:           tbAPI,
		Bot:             botClient,
		MessagesForSend: messagesForSend,
	}

	if err := tgListener.Do(); err != nil {
		log.Fatalf("[ERROR] failed to start Telegram listener: %s", err)
	}
}

func startHttpServer(cfg *config.Config, wg *sync.WaitGroup) {
	defer wg.Done()

	httpClient := http.CreateClient(cfg)
	httpClient.Start()
}
