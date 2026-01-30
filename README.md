# Bootstrap Hub Bot

A Discord bot built for the Bootstrap Hub community - a server dedicated to solo founders and entrepreneurs building their businesses.

## Features

### Focus Periods (2-Week Goal Tracking)
Track your goals in 2-week "Focus Periods" - like sprints, but for founders!

- `/focus start` - Start a new 2-week Focus Period
- `/focus add <goal>` - Add a goal to your current period
- `/focus complete <number>` - Mark a goal as completed
- `/focus list` - View all your goals and their status
- `/focus status` - Get an overview of your progress

### Automatic Reminders
- **Progress reminders** on days 3, 7, 10, 12, and 13 of your Focus Period
- **Goal-setting reminders** if you have fewer than 3 goals set

### General Commands
- `/ping` - Test command to check if the bot is responsive
- `/help` - Get information about the bot and available commands

## Prerequisites

- Go 1.21 or higher
- A Discord account
- A Discord server where you have admin permissions

## Setup

### 1. Create a Discord Application

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications)
2. Click "New Application" and give it a name (e.g., "Bootstrap Hub Bot")
3. Go to the "Bot" section in the left sidebar
4. Click "Add Bot" and confirm
5. Under "Privileged Gateway Intents", you don't need any special intents for basic slash commands
6. Click "Reset Token" to get your bot token (save this securely!)

### 2. Get Your Credentials

From the Discord Developer Portal:
- **Bot Token**: Found in the "Bot" section (click "Reset Token" if needed)
- **Application ID**: Found in the "General Information" section (also called "Client ID")

### 3. Configure Environment Variables

```bash
# Copy the example environment file
cp .env.example .env

# Edit .env with your credentials
nano .env  # or use your preferred editor
```

Fill in your `.env` file:
```
DISCORD_BOT_TOKEN=your_bot_token_here
DISCORD_APPLICATION_ID=your_application_id_here
DISCORD_GUILD_ID=your_server_id_here  # Optional, for faster command registration
DISCORD_REMINDER_CHANNEL_ID=your_channel_id_here  # Optional, for reminder messages
DATABASE_PATH=data/bootstrap_hub.db  # Optional, default path shown
```

**Notes**:
- Setting `DISCORD_GUILD_ID` registers commands only to that server (instant). Leaving it empty registers commands globally (can take up to 1 hour).
- Setting `DISCORD_REMINDER_CHANNEL_ID` enables automatic reminder messages. To get a channel ID, enable Developer Mode in Discord settings, then right-click the channel and select "Copy Channel ID".

### 4. Install Dependencies

```bash
make deps
```

### 5. Register Slash Commands

**Important**: You must register commands before they appear in Discord!

```bash
make register
```

### 6. Invite the Bot to Your Server

Get the invite URL:
```bash
make invite
```

Click the URL and select your server to add the bot.

### 7. Run the Bot

```bash
make run
```

## Docker Deployment

### Quick Start with Docker Compose

For local development and testing:

```bash
# 1. Create your .env file with credentials
cp .env.example .env
# Edit .env with your Discord credentials

# 2. Build and start the bot
docker-compose up -d

# 3. Register Discord commands
docker exec bootstrap-hub-bot ./bootstrap-hub-bot -register

# 4. View logs
docker-compose logs -f
```

### Production Deployment with GitHub Actions

The bot includes automated Docker deployment via GitHub Actions that triggers on every push to `main`.

**One-Time Setup:**

1. **Configure GitHub Secrets** (Repository → Settings → Secrets → Actions):
   - `DISCORD_BOT_TOKEN` (required)
   - `DISCORD_APPLICATION_ID` (required)
   - `DISCORD_GUILD_ID` (optional)
   - `DISCORD_REMINDER_CHANNEL_ID` (optional)

2. **Set up a self-hosted runner** on your deployment server

3. **Create backups directory** on the runner:
   ```bash
   mkdir -p backups
   ```

**Automated Deployment Features:**
- Builds Docker image locally on push to `main`
- Creates automatic database backup before deployment
- Zero-downtime deployment with graceful shutdown
- Auto-registration of Discord slash commands
- Keeps last 5 Docker images for rollback
- Keeps last 10 database backups
- Persistent database storage via Docker volumes

**Manual Operations:**

```bash
# View running container
docker ps -f name=bootstrap-hub-bot

# View logs
docker logs -f bootstrap-hub-bot

# Manual backup
docker run --rm \
  -v bootstrap-hub-bot-data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/manual-backup-$(date +%Y%m%d-%H%M%S).tar.gz -C /data .

# Restore from backup
docker stop bootstrap-hub-bot
docker run --rm \
  -v bootstrap-hub-bot-data:/data \
  -v $(pwd)/backups:/backup \
  alpine sh -c "rm -rf /data/* && tar xzf /backup/db-backup-YYYYMMDD-HHMMSS.tar.gz -C /data"
docker start bootstrap-hub-bot
```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the bot binary |
| `make run` | Build and run the bot |
| `make register` | Register all slash commands with Discord |
| `make remove-commands` | Remove all slash commands from Discord |
| `make invite` | Display the bot invite URL |
| `make deps` | Download and tidy dependencies |
| `make test` | Run tests |
| `make clean` | Clean build artifacts |
| `make help` | Display help |

## Project Structure

```
bootstrap-hub-bot/
├── cmd/
│   └── bot/
│       └── main.go           # Entry point
├── internal/
│   ├── bot/
│   │   └── bot.go            # Bot initialization and handlers
│   ├── commands/
│   │   ├── commands.go       # Base command definitions
│   │   └── focus.go          # Focus Period commands
│   ├── config/
│   │   └── config.go         # Configuration loading
│   ├── database/
│   │   ├── database.go       # Database operations
│   │   └── models.go         # Data models
│   └── scheduler/
│       └── scheduler.go      # Reminder scheduler
├── data/
│   └── bootstrap_hub.db      # SQLite database (created on first run)
├── .env.example              # Example environment variables
├── .gitignore                # Git ignore file
├── go.mod                    # Go module file
├── go.sum                    # Go dependencies checksum
├── Makefile                  # Build automation
└── README.md                 # This file
```

## How Focus Periods Work

1. **Start a Focus Period**: Use `/focus start` to begin a new 2-week period
2. **Add Goals**: Use `/focus add <goal>` to add goals (aim for at least 3!)
3. **Track Progress**: Use `/focus list` to see your goals and `/focus status` for an overview
4. **Complete Goals**: Use `/focus complete <number>` to mark goals as done
5. **Stay Accountable**: Receive reminders throughout the period to keep you on track

### Reminder Schedule
- **Day 3**: Early check-in to build momentum
- **Day 7**: Halfway point review
- **Day 10**: Final stretch begins
- **Day 12**: Two days remaining
- **Day 13**: Last day reminder for final push

## Adding New Commands

1. Open `internal/commands/commands.go`
2. Create a new command function following the existing pattern:
   ```go
   func myNewCommand() *Command {
       return &Command{
           Definition: &discordgo.ApplicationCommand{
               Name:        "mycommand",
               Description: "Description of my command",
           },
           Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
               // Handle the command
           },
       }
   }
   ```
3. Add your command to the `GetAllCommands()` function
4. Run `make register` to register the new command with Discord

## Development Tips

- Use `DISCORD_GUILD_ID` during development for instant command updates
- Remove the guild ID for production to register commands globally
- Global commands can take up to 1 hour to appear in all servers
- The database is automatically created on first run

## Troubleshooting

### Commands not appearing
- Make sure you ran `make register`
- If using guild-specific registration, ensure the guild ID is correct
- Global commands can take up to 1 hour to propagate

### Bot not responding
- Check that the bot is online in your server
- Verify your bot token is correct
- Check the console for error messages

### Reminders not working
- Ensure `DISCORD_REMINDER_CHANNEL_ID` is set in your `.env` file
- Verify the bot has permission to send messages in that channel
- Reminders are checked hourly and sent around 9 AM server time

### Permission errors
- Ensure the bot has the necessary permissions in your server
- Re-invite the bot using `make invite` if permissions changed

## License

MIT License - Feel free to use and modify for your own projects!

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
