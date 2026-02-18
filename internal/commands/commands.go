package commands

import (
	"fmt"
	"log"
	"time"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/openai"
	"github.com/bwmarrin/discordgo"
)

// Command represents a slash command with its definition and handler
type Command struct {
	Definition *discordgo.ApplicationCommand
	Handler    func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// GetAllCommands returns all available bot commands
func GetAllCommands(openaiClient *openai.Client) []*Command {
	return []*Command{
		pingCommand(),
		helpCommand(),
		focusCommand(openaiClient),
		resourceCommand(),
		leaderboardCommand(),
		configCommand(),
		// New features
		standupCommand(),
		winCommand(),
		buddyCommand(),
		challengeCommand(),
		mrrCommand(),
		projectCommand(),
	}
}

// GetCommandDefinitions returns just the command definitions for registration
func GetCommandDefinitions() []*discordgo.ApplicationCommand {
	commands := GetAllCommands(nil)
	definitions := make([]*discordgo.ApplicationCommand, len(commands))
	for i, cmd := range commands {
		definitions[i] = cmd.Definition
	}
	return definitions
}

// GetHandlers returns a map of command names to their handlers
func GetHandlers(openaiClient *openai.Client) map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	commands := GetAllCommands(openaiClient)
	handlers := make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))
	for _, cmd := range commands {
		handlers[cmd.Definition.Name] = cmd.Handler
	}
	return handlers
}

// pingCommand creates the ping/pong test command
func pingCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "ping",
			Description: "Test if the Bootstrap Hub Bot is responsive - responds with Pong!",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			start := time.Now()

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("🏓 Pong! Latency: %dms\n\n*Bootstrap Hub Bot is here to help founders on their journey!*", time.Since(start).Milliseconds()),
				},
			})
			if err != nil {
				log.Printf("Error responding to ping command: %v", err)
			}
		},
	}
}

// helpCommand is defined in help.go
