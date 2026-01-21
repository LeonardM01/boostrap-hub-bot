package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the bot
type Config struct {
	// BotToken is the Discord bot token
	BotToken string
	// ApplicationID is the Discord application ID (client ID)
	ApplicationID string
	// GuildID is optional - if set, commands are registered to this guild only (faster for development)
	GuildID string
	// DatabasePath is the path to the SQLite database file
	DatabasePath string
	// ReminderChannelID is the Discord channel ID where reminders will be sent
	ReminderChannelID string
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if it doesn't)
	_ = godotenv.Load()

	// Default database path
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "data/bootstrap_hub.db"
	}

	config := &Config{
		BotToken:          os.Getenv("DISCORD_BOT_TOKEN"),
		ApplicationID:     os.Getenv("DISCORD_APPLICATION_ID"),
		GuildID:           os.Getenv("DISCORD_GUILD_ID"),
		DatabasePath:      dbPath,
		ReminderChannelID: os.Getenv("DISCORD_REMINDER_CHANNEL_ID"),
	}

	if config.BotToken == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN environment variable is required")
	}

	if config.ApplicationID == "" {
		return nil, fmt.Errorf("DISCORD_APPLICATION_ID environment variable is required")
	}

	return config, nil
}
