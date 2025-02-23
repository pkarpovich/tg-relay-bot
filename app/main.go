package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	tbapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/pkarpovich/tg-relay-bot/app/bot"
	"github.com/pkarpovich/tg-relay-bot/app/config"
	"github.com/pkarpovich/tg-relay-bot/app/events"
	"github.com/pkarpovich/tg-relay-bot/app/http"
	"github.com/pkarpovich/tg-relay-bot/app/smtp_server"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	messagesForSend := make(chan string, 100)
	defer close(messagesForSend)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	httpServer := startHttpServer(ctx, &wg, cfg, messagesForSend)
	smtpServer := startMailServer(ctx, &wg, cfg, messagesForSend)
	tgListener := startTelegramListener(ctx, &wg, cfg, messagesForSend)

	sig := <-sigChan
	log.Printf("[INFO] Received shutdown signal: %v", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ERROR] HTTP server shutdown error: %s", err)
	}

	if err := smtpServer.Shutdown(); err != nil {
		log.Printf("[ERROR] SMTP server shutdown error: %s", err)
	}

	if err := tgListener.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ERROR] Telegram listener shutdown error: %s", err)
	}

	wg.Wait()
	log.Printf("[INFO] Application shutdown complete")
	return nil
}

func startHttpServer(ctx context.Context, wg *sync.WaitGroup, cfg *config.Config, messagesForSend chan string) *http.Server {
	wg.Add(1)
	httpServer := http.CreateServer(cfg, messagesForSend)
	go func() {
		defer wg.Done()
		if err := httpServer.Start(ctx); err != nil {
			log.Printf("[ERROR] HTTP server error: %s", err)
		}
	}()
	return httpServer
}

func startMailServer(ctx context.Context, wg *sync.WaitGroup, cfg *config.Config, messagesForSend chan string) *smtp_server.Server {
	wg.Add(1)
	mailServer := smtp_server.NewServer(cfg, messagesForSend)
	go func() {
		defer wg.Done()
		if err := mailServer.Start(ctx); err != nil {
			log.Printf("[ERROR] Mail server error: %s", err)
		}
	}()
	return mailServer
}

func startTelegramListener(ctx context.Context, wg *sync.WaitGroup, cfg *config.Config, messagesForSend chan string) *events.TelegramListener {
	wg.Add(1)
	botClient := bot.NewClient()

	tbAPI, err := tbapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		log.Fatalf("[ERROR] Failed to create Telegram bot: %s", err)
	}

	tgListener := &events.TelegramListener{
		SuperUsers:      cfg.Telegram.SuperUsers,
		TbAPI:           tbAPI,
		Bot:             botClient,
		MessagesForSend: messagesForSend,
	}

	go func() {
		defer wg.Done()
		if err := tgListener.Do(ctx); err != nil {
			log.Printf("[ERROR] Telegram listener error: %s", err)
		}
	}()

	return tgListener
}
