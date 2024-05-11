package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"log"
)

type TelegramConfig struct {
	Token      string  `env:"TELEGRAM_TOKEN"`
	SuperUsers []int64 `env:"TELEGRAM_SUPER_USERS" env-separator:","`
}

type Config struct {
	Telegram TelegramConfig
}

func Init() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Printf("[WARN] error while loading .env file: %v", err)
	}

	var cfg Config
	err = cleanenv.ReadEnv(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
