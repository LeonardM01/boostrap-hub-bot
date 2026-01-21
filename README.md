# Bootstrap Hub Bot

A Discord bot built for the Bootstrap Hub community - a server dedicated to solo founders and entrepreneurs building their businesses.

## Features

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
DISCORD_GUILD_ID=your_server_id_here  # Optional, for faster command registration during development
```

**Note**: Setting `DISCORD_GUILD_ID` registers commands only to that server (instant). Leaving it empty registers commands globally (can take up to 1 hour).

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

## Available Commands

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
│   │   └── commands.go       # Slash command definitions
│   └── config/
│       └── config.go         # Configuration loading
├── .env.example              # Example environment variables
├── .gitignore                # Git ignore file
├── go.mod                    # Go module file
├── go.sum                    # Go dependencies checksum
├── Makefile                  # Build automation
└── README.md                 # This file
```

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

## Troubleshooting

### Commands not appearing
- Make sure you ran `make register`
- If using guild-specific registration, ensure the guild ID is correct
- Global commands can take up to 1 hour to propagate

### Bot not responding
- Check that the bot is online in your server
- Verify your bot token is correct
- Check the console for error messages

### Permission errors
- Ensure the bot has the necessary permissions in your server
- Re-invite the bot using `make invite` if permissions changed

## License

MIT License - Feel free to use and modify for your own projects!

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
