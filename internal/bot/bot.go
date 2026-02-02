package bot

import (
	"fmt"
	"log"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/commands"
	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/config"
	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/openai"
	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/scheduler"
	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/voter"
	"github.com/bwmarrin/discordgo"
)

// Bot represents the Discord bot instance
type Bot struct {
	Session      *discordgo.Session
	Config       *config.Config
	Scheduler    *scheduler.Scheduler
	Voter        *voter.Voter
	OpenAIClient *openai.Client
}

// New creates a new Bot instance
func New(cfg *config.Config) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Initialize OpenAI client (may be nil if no API key)
	openaiClient := openai.New(cfg.OpenAIAPIKey)

	bot := &Bot{
		Session:      session,
		Config:       cfg,
		OpenAIClient: openaiClient,
	}

	// Register the interaction handler
	session.AddHandler(bot.handleInteraction)

	// Register the ready handler
	session.AddHandler(bot.handleReady)

	// Register reaction handlers for resource voting
	session.AddHandler(bot.handleMessageReactionAdd)
	session.AddHandler(bot.handleMessageReactionRemove)

	return bot, nil
}

// Start opens the Discord connection and starts listening
func (b *Bot) Start() error {
	// Set intents - we need guilds for slash commands and reactions for resource voting
	b.Session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessageReactions

	err := b.Session.Open()
	if err != nil {
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	// Start the reminder scheduler if a channel is configured
	if b.Config.ReminderChannelID != "" {
		b.Scheduler = scheduler.New(b.Session, b.Config.ReminderChannelID)
		b.Scheduler.Start()
	} else {
		log.Println("No reminder channel configured, scheduler not started")
	}

	// Start the vote processor
	b.Voter = voter.New(b.Session)
	b.Voter.Start()

	log.Println("Bootstrap Hub Bot is now running!")
	return nil
}

// Stop gracefully closes the Discord connection
func (b *Bot) Stop() error {
	log.Println("Shutting down Bootstrap Hub Bot...")

	// Stop the scheduler if running
	if b.Scheduler != nil {
		b.Scheduler.Stop()
	}

	// Stop the voter if running
	if b.Voter != nil {
		b.Voter.Stop()
	}

	return b.Session.Close()
}

// RegisterCommands registers all slash commands with Discord
func (b *Bot) RegisterCommands() error {
	log.Println("Registering slash commands...")

	cmdDefs := commands.GetCommandDefinitions()

	for _, cmd := range cmdDefs {
		_, err := b.Session.ApplicationCommandCreate(b.Config.ApplicationID, b.Config.GuildID, cmd)
		if err != nil {
			return fmt.Errorf("failed to register command '%s': %w", cmd.Name, err)
		}
		log.Printf("Registered command: /%s", cmd.Name)
	}

	log.Printf("Successfully registered %d commands", len(cmdDefs))
	return nil
}

// RemoveCommands removes all registered slash commands
func (b *Bot) RemoveCommands() error {
	log.Println("Removing slash commands...")

	registeredCmds, err := b.Session.ApplicationCommands(b.Config.ApplicationID, b.Config.GuildID)
	if err != nil {
		return fmt.Errorf("failed to fetch registered commands: %w", err)
	}

	for _, cmd := range registeredCmds {
		err := b.Session.ApplicationCommandDelete(b.Config.ApplicationID, b.Config.GuildID, cmd.ID)
		if err != nil {
			log.Printf("Failed to delete command '%s': %v", cmd.Name, err)
		} else {
			log.Printf("Removed command: /%s", cmd.Name)
		}
	}

	log.Println("All commands removed")
	return nil
}

// handleReady is called when the bot is ready
func (b *Bot) handleReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as: %s#%s", r.User.Username, r.User.Discriminator)
	log.Printf("Bot is in %d guilds", len(r.Guilds))

	// Set the bot's status
	err := s.UpdateGameStatus(0, "Helping founders bootstrap! | /help")
	if err != nil {
		log.Printf("Failed to set status: %v", err)
	}
}

// handleInteraction handles all incoming interactions (slash commands and components)
func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handlers := commands.GetHandlers(b.OpenAIClient)
		cmdName := i.ApplicationCommandData().Name

		if handler, ok := handlers[cmdName]; ok {
			handler(s, i)
		} else {
			log.Printf("Unknown command: %s", cmdName)
		}

	case discordgo.InteractionMessageComponent:
		// Handle button/select menu interactions
		commands.HandleHelpComponent(s, i)
	}
}

// GetInviteURL generates the bot invite URL with necessary permissions
func (b *Bot) GetInviteURL() string {
	// Permissions:
	// - Send Messages (2048)
	// - Use Slash Commands (2147483648)
	// - Embed Links (16384)
	// - Read Message History (65536)
	// - Add Reactions (64)
	// Combined: 2147567680
	permissions := "2147567680"
	return fmt.Sprintf(
		"https://discord.com/api/oauth2/authorize?client_id=%s&permissions=%s&scope=bot%%20applications.commands",
		b.Config.ApplicationID,
		permissions,
	)
}

// handleMessageReactionAdd handles reaction additions for resource voting
func (b *Bot) handleMessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Ignore bot's own reactions
	if r.UserID == s.State.User.ID {
		return
	}

	// Look up resource by message ID
	resource, err := database.GetPublicResourceByVoteMessageID(r.MessageID)
	if err != nil {
		log.Printf("Error fetching resource by message ID: %v", err)
		return
	}

	// If no resource found or not pending, ignore
	if resource == nil || resource.Status != database.ResourceStatusPending {
		return
	}

	// Only count valid emoji reactions
	if r.Emoji.Name == "👍" {
		if err := database.IncrementUsefulVotes(resource.ID); err != nil {
			log.Printf("Error incrementing useful votes: %v", err)
		}
	} else if r.Emoji.Name == "👎" {
		if err := database.IncrementNotUsefulVotes(resource.ID); err != nil {
			log.Printf("Error incrementing not useful votes: %v", err)
		}
	}
}

// handleMessageReactionRemove handles reaction removals for resource voting
func (b *Bot) handleMessageReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	// Ignore bot's own reactions
	if r.UserID == s.State.User.ID {
		return
	}

	// Look up resource by message ID
	resource, err := database.GetPublicResourceByVoteMessageID(r.MessageID)
	if err != nil {
		log.Printf("Error fetching resource by message ID: %v", err)
		return
	}

	// If no resource found or not pending, ignore
	if resource == nil || resource.Status != database.ResourceStatusPending {
		return
	}

	// Only count valid emoji reactions
	if r.Emoji.Name == "👍" {
		if err := database.DecrementUsefulVotes(resource.ID); err != nil {
			log.Printf("Error decrementing useful votes: %v", err)
		}
	} else if r.Emoji.Name == "👎" {
		if err := database.DecrementNotUsefulVotes(resource.ID); err != nil {
			log.Printf("Error decrementing not useful votes: %v", err)
		}
	}
}
