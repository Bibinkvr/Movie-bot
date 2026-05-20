# Autofilter Bot

A powerful and customizable Telegram bot written in Go for automatically filtering and indexing files.

## Commands

- `start` - Check if the bot is alive.
- `help` - How to use the bot.
- `about` - About the bot.
- `settings` - Bot configuration (Admin only).
- `index` - Import files from channels (Admin only).
- `broadcast` - Send messages to all users (Admin only).

## Features

- **Automatic File Filtering:** Automatically saves and filters files from configured channels.
- **Large Volume Indexing:** Efficiently index large amounts of files.
- **Customizable:** Extensive configuration options via environment variables.
- **Multi-Database Support:** Support for multiple MongoDB databases.
- **Force Subscribe:** Optional requirement for users to subscribe to channels before using the bot.

## Deployment

### Prerequisites

- [Go](https://go.dev/dl) (latest version)
- [MongoDB](https://www.mongodb.com/)

### Environment Variables

Configure the following variables in a `.env` file:

- `BOT_TOKEN`: Your Telegram bot token.
- `ADMINS`: List of admin user IDs.
- `MONGODB_URI`: Primary MongoDB connection URI.
- `FILE_CHANNELS`: List of channel IDs to index.
- `APP_ID`: Telegram App ID.
- `APP_HASH`: Telegram App Hash.

### Running Locally

```bash
go build -o bot .
./bot
```

## License

All data handled by the bot is the responsibility of the user. This project is provided as-is.
