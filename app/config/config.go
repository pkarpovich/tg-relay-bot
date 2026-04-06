package config

import (
	"fmt"
	"log"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type TelegramConfig struct {
	Token      string  `env:"TELEGRAM_TOKEN"`
	SuperUsers []int64 `env:"TELEGRAM_SUPER_USERS" env-separator:","`
}

type HttpConfig struct {
	Port         int    `env:"HTTP_PORT" env-default:"8080"`
	SecretApiKey string `env:"HTTP_SECRET"`
}

type SmtpConfig struct {
	AllowedHosts []string `env:"SMTP_ALLOWED_HOSTS" env-separator:","`
	ListenAddr   string   `env:"SMTP_LISTEN_ADDR" env-default:"0.0.0.0:2525"`
}

type Config struct {
	Telegram TelegramConfig
	Http     HttpConfig
	Smtp     SmtpConfig
}

func Init() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Printf("[WARN] error while loading .env file: %v", err)
	}

	var cfg Config
	if err = cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read env config: %w", err)
	}

	return &cfg, nil
}
