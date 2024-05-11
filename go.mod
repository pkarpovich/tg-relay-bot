module github.com/pkarpovich/tg-relay-bot

go 1.22

require (
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/ilyakaznacheev/cleanenv v1.5.0
	github.com/joho/godotenv v1.5.1
)

replace github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1 => github.com/OvyFlash/telegram-bot-api/v5 v5.0.0-20240511102618-f8ef3a5696e6

require (
	github.com/BurntSushi/toml v1.3.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	olympos.io/encoding/edn v0.0.0-20201019073823-d3554ca0b0a3 // indirect
)
