# tg-relay-bot

## Introduction

`tg-relay-bot` is a universal messaging bot that bridges communication between HTTP requests or SMTP emails and a
Telegram bot channel. This service is particularly useful for integrating Telegram notifications into various web
services, alert systems, or any application requiring immediate message forwarding to a Telegram channel.

## Features

- **HTTP and SMTP Integration**: Accepts incoming messages from both HTTP requests and SMTP emails.
- **Telegram Forwarding**: Automatically forwards messages to a designated Telegram bot channel.

## Configuration

You could configure the bot by setting the following environment variables:

- `TELEGRAM_TOKEN`: The Telegram bot token.
- `TELEGRAM_SUPER_USERS`: A comma-separated list of Telegram user IDs that are allowed to interact with the bot.
- `HTTP_SECRET`: The secret key for authenticating incoming HTTP requests.
- `SMTP_ALLOWED_HOSTS`: A comma-separated list of allowed email domains.

## Usage

### Sending a Message via HTTP

```bash
curl -X POST http://localhost:8080/send -d '{"message": "Your message here"}'
```

### Sending a Message via SMTP

Send an email to the configured SMTP server and it will be automatically forwarded to the Telegram channel.

## Contributing

Contributions are welcome! Feel free to open an issue or submit a pull request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
