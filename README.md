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

This bot can be deployed to a VPS (Virtual Private Server) or a cloud platform like Render.

### Prerequisites

- **Go**: Version 1.24+ (if running/building locally).
- **MongoDB**: A running instance (local or MongoDB Atlas).
- **Docker**: (Optional) if deploying via containers.

### Environment Variables

Create a `.env` file in the root directory (you can copy `.env.example` as a template) and configure the following variables:

```env
# Bot configuration
BOT_TOKEN=your_telegram_bot_token
ADMINS=123456789,987654321
MONGODB_URI=mongodb+srv://...
DATABASE_NAME=movie_bot
COLLECTION_NAME=files
LOG_LEVEL=info
LOG_CHANNEL=-100xxxxxxxxxx

# Telegram API credentials (for indexing/sessions)
APP_ID=12345
APP_HASH=your_telegram_app_hash
```

---

### VPS Deployment Methods

#### Method A: Docker Compose (Recommended)

1. Ensure `docker` and `docker-compose` (or `docker compose`) are installed on your VPS.
2. Configure your `.env` file in the project folder.
3. Build and start the container in background mode:
   ```bash
   docker compose up --build -d
   ```
4. Check the logs to ensure the bot started successfully:
   ```bash
   docker compose logs -f
   ```

#### Method B: Systemd Service (Process Manager)

If you want to run the compiled Go binary directly on your VPS as a background service:

1. Build the binary on your server:
   ```bash
   go build -o bot main.go
   ```
2. Create a systemd service file:
   ```bash
   sudo nano /etc/systemd/system/moviebot.service
   ```
3. Paste the following configuration (replace `/path/to/bot` with the actual path to your bot folder):
   ```ini
   [Unit]
   Description=Movie Filter Bot Service
   After=network.target

   [Service]
   Type=simple
   User=root
   WorkingDirectory=/path/to/bot
   ExecStart=/path/to/bot/bot
   Restart=always
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   ```
4. Start and enable the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl start moviebot
   sudo systemctl enable moviebot
   ```
5. Check status and logs:
   ```bash
   sudo systemctl status moviebot
   journalctl -u moviebot -f
   ```

#### Method C: PM2

If you prefer PM2 for process management:

1. Build the binary: `go build -o bot main.go`.
2. Start the binary with PM2:
   ```bash
   pm2 start ./bot --name "moviebot"
   ```
3. Save the process list: `pm2 save`.

---

### Render Deployment

This repository includes a `render.yaml` blueprint file for easy deployment as a Web Service.

#### Web Service setup (Render Free Tier)
The bot includes a built-in health check HTTP server that listens on the `PORT` environment variable (default `10001` or `10000`). This ensures Render's health check passes and the container stays active.

1. Push your code to GitHub.
2. Go to the **Render Dashboard** -> **Blueprints** -> **New Blueprint Instance**.
3. Connect your repository.
4. Render will automatically detect the `render.yaml` configuration.
5. Provide the required environment variables (`BOT_TOKEN`, `MONGODB_URI`, `ADMINS`, `APP_ID`, `APP_HASH`) in the Render Dashboard when prompted.
6. Click **Deploy**.

#### Memory Optimizations for Render Free Tier
The bot includes a memory recycler routine (`debug.FreeOSMemory()` every 5 minutes and `debug.SetGCPercent(50)`) designed specifically to keep the RAM footprint under Render's **512MB free-tier limit**.

## License

All data handled by the bot is the responsibility of the user. This project is provided as-is.
