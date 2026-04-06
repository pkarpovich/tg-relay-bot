# tg-relay-bot

## Introduction

`tg-relay-bot` is a universal messaging bot that bridges communication between HTTP requests or SMTP emails and a
Telegram bot channel. This service is particularly useful for integrating Telegram notifications into various web
services, alert systems, or any application requiring immediate message forwarding to a Telegram channel.

## Features

- **HTTP and SMTP Integration**: Accepts incoming messages from both HTTP requests and SMTP emails.
- **Telegram Forwarding**: Automatically forwards messages to a designated Telegram bot channel.
- **Formatted Messages**: Supports Telegram `MarkdownV2` and `HTML` formatting via the optional `parse_mode` field on the `/send` endpoint.

## Configuration

You could configure the bot by setting the following environment variables:

- `TELEGRAM_TOKEN`: The Telegram bot token.
- `TELEGRAM_SUPER_USERS`: A comma-separated list of Telegram user IDs that are allowed to interact with the bot.
- `HTTP_SECRET`: The secret key for authenticating incoming HTTP requests.
- `SMTP_ALLOWED_HOSTS`: A comma-separated list of allowed email domains.

## Usage

### Sending a Message via HTTP

```bash
# Plain text message
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -H "X-Secret: your-secret" \
  -d '{"message": "Your message here"}'

# Formatted message with MarkdownV2
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -H "X-Secret: your-secret" \
  -d '{"message": "*bold* _italic_", "parse_mode": "MarkdownV2"}'

# Formatted message with HTML
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -H "X-Secret: your-secret" \
  -d '{"message": "<b>bold</b> <i>italic</i>", "parse_mode": "HTML"}'
```

### Sending a Message via SMTP

Send an email to the configured SMTP server and it will be automatically forwarded to the Telegram channel.

## Contributing

Contributions are welcome! Feel free to open an issue or submit a pull request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
