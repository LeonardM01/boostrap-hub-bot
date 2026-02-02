# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bootstrap Hub Bot is a Discord bot built with Go and the discordgo library, designed for solo founders and entrepreneurs. The bot helps users track their business goals using 2-week "Focus Periods" (similar to agile sprints) with automated reminders and progress tracking.

## Available Resources

### API Documentation
When working with the discordgo library, use the **context7 MCP server** to retrieve up-to-date documentation and code examples. This provides accurate API references for:
- Discord API methods and types
- discordgo library usage patterns
- Best practices and examples

To access discordgo documentation, ask Claude to query the context7 server for specific API information.

## Development Commands

### Build and Run
```bash
make build          # Compile the bot binary to bin/bootstrap-hub-bot
make run            # Build and start the bot
make deps           # Download and tidy Go dependencies
make test           # Run all tests
make clean          # Remove build artifacts
```

### Discord Command Management
```bash
make register       # Register slash commands with Discord (REQUIRED before first use)
make remove-commands # Remove all registered slash commands
make invite         # Display the bot invite URL with correct permissions
```

**Important**: After adding or modifying slash commands, you must run `make register` for changes to take effect in Discord.

### Direct Binary Execution
The binary accepts command-line flags for special operations:
```bash
./bin/bootstrap-hub-bot -register          # Register commands
./bin/bootstrap-hub-bot -remove-commands   # Remove commands
./bin/bootstrap-hub-bot -invite            # Show invite URL
./bin/bootstrap-hub-bot                    # Normal operation (listen for commands)
```

## Architecture

### Entry Point and Control Flow
- **cmd/bot/main.go**: Application entry point that handles command-line flags, initializes the database, creates the bot instance, and manages graceful shutdown
- The bot operates in three modes based on flags: command registration/removal, invite URL display, or normal listening mode

### Core Components

#### Bot Management (internal/bot/bot.go)
- `Bot` struct holds the Discord session, configuration, and scheduler
- Handles Discord connection lifecycle and registers event handlers:
  - `handleInteraction`: Routes slash command interactions to appropriate handlers
  - `handleReady`: Sets bot status when connected
- Command registration/removal happens through the Discord API using application command endpoints

#### Configuration (internal/config/config.go)
Environment variables loaded from `.env` file:
- `DISCORD_BOT_TOKEN` (required): Bot authentication token
- `DISCORD_APPLICATION_ID` (required): Application ID for command registration
- `DISCORD_GUILD_ID` (optional): Guild-specific registration for faster updates during development
- `DISCORD_REMINDER_CHANNEL_ID` (optional): Channel for automated reminder messages
- `DATABASE_PATH` (optional): SQLite database location (defaults to `data/bootstrap_hub.db`)

#### Command System (internal/commands/)
Commands use a unified structure with definition and handler:
- **commands.go**: Defines the `Command` type and provides aggregation functions (`GetAllCommands`, `GetCommandDefinitions`, `GetHandlers`)
- **focus.go**: Implements the `/focus` command group with subcommands (start, add, complete, list, status)
  - Each subcommand handler interacts with the database layer to manage focus periods and tasks
  - Uses Discord embeds for rich, formatted responses

To add a new command:
1. Create a command function returning `*Command` with definition and handler
2. Add it to the slice returned by `GetAllCommands()` in commands.go
3. Run `make register` to register it with Discord

#### Database Layer (internal/database/)
Uses GORM with SQLite for persistence:
- **models.go**: Defines three core models:
  - `User`: Discord users tracked per guild
  - `FocusPeriod`: 2-week goal tracking periods with start/end dates
  - `Task`: Individual goals within a focus period, with position and completion tracking

- **database.go**: Provides database operations:
  - `Initialize()`: Sets up connection and runs auto-migrations
  - User management: `GetOrCreateUser()`
  - Focus period operations: `GetCurrentFocusPeriod()`, `CreateFocusPeriod()`
  - Task operations: `AddTask()`, `CompleteTask()`, `GetTasksByFocusPeriod()`
  - Reminder queries: `GetUsersWithActiveFocusPeriods()`, `GetFocusPeriodsForReminder()`, `GetUsersWithInsufficientTasks()`

Key constants:
- `FocusPeriodDuration`: 14 days (2 weeks)
- `MinimumTasksRequired`: 3 goals recommended per period
- `ReminderDays`: [3, 7, 10, 12, 13] - days to send progress reminders

#### Scheduler (internal/scheduler/scheduler.go)
Automated reminder system running on an hourly tick:
- **Daily reminders**: Sent on days 3, 7, 10, 12, and 13 of active focus periods (only between 9-10 AM)
- **Goal-setting reminders**: Sent on days 2-3 if user has fewer than 3 tasks
- Reminders are posted as embeds to the configured reminder channel
- Scheduler starts automatically when `DISCORD_REMINDER_CHANNEL_ID` is configured

### Data Flow Example (Focus Period Creation)
1. User invokes `/focus start` in Discord
2. Discord sends interaction to bot's `handleInteraction()`
3. Routed to `handleFocusCommand()` based on command name
4. `handleFocusStart()` called with user info from interaction
5. `GetOrCreateUser()` ensures user exists in database
6. `GetCurrentFocusPeriod()` checks for existing active period
7. If none exists, `CreateFocusPeriod()` creates new 14-day period
8. Response embed sent back to Discord with period details

## Key Implementation Details

### Discord Interaction Pattern
All slash commands use the Discord interaction model:
- Commands must respond within 3 seconds using `InteractionRespond()`
- The bot uses `InteractionResponseChannelMessageWithSource` type for visible responses
- Embeds are the primary UI, providing rich formatting with fields, colors, and footers

### Guild-Specific Data
Users and focus periods are scoped per guild (Discord server), allowing:
- The same user to have different focus periods in different servers
- Server-specific reminder channels
- Isolated data per community

### Command Registration Modes
- **Guild-specific** (with `DISCORD_GUILD_ID`): Commands appear instantly in that server (best for development)
- **Global** (without `DISCORD_GUILD_ID`): Commands available in all servers, but can take up to 1 hour to propagate

### Bot Permissions
The invite URL requests these Discord permissions (value: 2147567616):
- Send Messages (2048)
- Use Slash Commands (2147483648)
- Embed Links (16384)
- Read Message History (65536)

## Testing Strategy

Currently the codebase has minimal test coverage. When adding tests:
- Use `make test` to run tests
- Database tests should use in-memory SQLite (`:memory:`) or temporary files
- Command handler tests can mock `*discordgo.Session` and `*discordgo.InteractionCreate`
