package smtp_server

import (
	"context"
	"fmt"
	"log"

	"github.com/flashmob/go-guerrilla"
	"github.com/flashmob/go-guerrilla/backends"
	"github.com/flashmob/go-guerrilla/mail"
	"github.com/jhillyerd/enmime"
	"github.com/pkarpovich/tg-relay-bot/app/config"
	"github.com/pkarpovich/tg-relay-bot/app/events"
)

type FormattedEmail struct {
	Subject string
	Text    string
}

type Server struct {
	messagesForSend chan events.MessagePayload
	daemon          guerrilla.Daemon
	quit            chan struct{}
}

func NewServer(cfg *config.Config, messagesForSend chan events.MessagePayload) *Server {
	appCfg := guerrilla.AppConfig{
		AllowedHosts: cfg.Smtp.AllowedHosts,
	}
	sc := guerrilla.ServerConfig{
		ListenInterface: cfg.Smtp.ListenAddr,
		IsEnabled:       true,
	}
	bc := backends.BackendConfig{
		"save_process": "HeadersParser|Header|Hasher|TelegramBot",
	}
	appCfg.Servers = append(appCfg.Servers, sc)
	appCfg.BackendConfig = bc

	d := guerrilla.Daemon{
		Config: &appCfg,
	}

	return &Server{
		messagesForSend: messagesForSend,
		daemon:          d,
		quit:            make(chan struct{}),
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.daemon.AddProcessor("TelegramBot", s.telegramBotProcessorFactory())

	if err := s.daemon.Start(); err != nil {
		return fmt.Errorf("start smtp daemon: %w", err)
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("smtp server context done: %w", ctx.Err())
	case <-s.quit:
		return nil
	}
}

func (s *Server) Shutdown() error {
	close(s.quit)
	s.daemon.Shutdown()

	return nil
}

func (s *Server) telegramBotProcessorFactory() func() backends.Decorator {
	return func() backends.Decorator {
		return func(p backends.Processor) backends.Processor {
			return backends.ProcessWith(
				func(e *mail.Envelope, task backends.SelectTask) (backends.Result, error) {
					if task == backends.TaskSaveMail {
						err := s.sendEmailToTelegram(e)
						if err != nil {
							return backends.NewResult(fmt.Sprintf("554 Error: %s", err)), err
						}
						return p.Process(e, task)
					}
					return p.Process(e, task)
				},
			)
		}
	}
}

func (s *Server) sendEmailToTelegram(e *mail.Envelope) error {
	formattedEmail, err := processEnvelope(e)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received email with subject: %s", formattedEmail.Subject)
	s.messagesForSend <- events.MessagePayload{Text: formattedEmail.Text}

	return nil
}

func processEnvelope(e *mail.Envelope) (*FormattedEmail, error) {
	reader := e.NewReader()
	env, err := enmime.ReadEnvelope(reader)
	if err != nil {
		return nil, fmt.Errorf("%s\n\nError occurred during email parsing: %w", e, err)
	}

	return &FormattedEmail{
		Subject: e.Subject,
		Text:    env.Text,
	}, nil
}
