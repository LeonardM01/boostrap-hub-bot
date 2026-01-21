package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/bot"
	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/config"
	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
)

func main() {
	// Parse command line flags
	registerCmds := flag.Bool("register", false, "Register slash commands with Discord and exit")
	removeCmds := flag.Bool("remove-commands", false, "Remove all slash commands from Discord and exit")
	showInvite := flag.Bool("invite", false, "Show the bot invite URL and exit")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Ensure data directory exists
	dbDir := filepath.Dir(cfg.DatabasePath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database
	if err := database.Initialize(cfg.DatabasePath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create bot instance
	b, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Handle invite URL request
	if *showInvite {
		log.Println("Bot Invite URL:")
		log.Println(b.GetInviteURL())
		return
	}

	// Start the bot to establish connection
	if err := b.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}
	defer b.Stop()

	// Handle command registration/removal
	if *registerCmds {
		if err := b.RegisterCommands(); err != nil {
			log.Fatalf("Failed to register commands: %v", err)
		}
		log.Println("Commands registered successfully!")
		return
	}

	if *removeCmds {
		if err := b.RemoveCommands(); err != nil {
			log.Fatalf("Failed to remove commands: %v", err)
		}
		log.Println("Commands removed successfully!")
		return
	}

	// Print invite URL on startup
	log.Println("")
	log.Println("===========================================")
	log.Println("    Bootstrap Hub Bot - For Solo Founders")
	log.Println("===========================================")
	log.Println("")
	log.Println("Invite the bot to your server:")
	log.Println(b.GetInviteURL())
	log.Println("")
	log.Println("Press Ctrl+C to exit")
	log.Println("===========================================")

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	log.Println("Gracefully shutting down...")
}
