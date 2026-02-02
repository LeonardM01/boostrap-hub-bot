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

// helpCommand creates the help command
func helpCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "help",
			Description: "Get information about Bootstrap Hub Bot and available commands",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			embed := &discordgo.MessageEmbed{
				Title:       "🚀 Bootstrap Hub Bot",
				Description: "Welcome to Bootstrap Hub! I'm your AI-powered assistant designed to help solo founders and entrepreneurs on their business journey.",
				Color:       0x5865F2, // Discord blurple
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "🎯 Focus Period Commands",
						Value:  "`/focus start` - Start a new 2-week Focus Period\n`/focus add <goal>` - Add a goal to your period (AI calculates points 1-10)\n`/focus complete <#>` - Mark a goal as done and earn points\n`/focus list` - View your goals\n`/focus status` - See your progress",
						Inline: false,
					},
					{
						Name:   "🏆 Leaderboard Commands",
						Value:  "`/leaderboard alltime` - View all-time rankings\n`/leaderboard sprint` - View current sprint rankings\n\nEarn points by completing tasks. Harder tasks = more points!",
						Inline: false,
					},
					{
						Name:   "⚙️ Admin Commands",
						Value:  "`/config leaderboard-channel <channel>` - Set channel for bi-weekly leaderboard posts",
						Inline: false,
					},
					{
						Name:   "📋 General Commands",
						Value:  "`/ping` - Check if the bot is responsive\n`/help` - Show this help message",
						Inline: false,
					},
					{
						Name:   "🚀 About Focus Periods",
						Value:  "Set goals for 2-week sprints and track your progress. Stay accountable with reminders on days 3, 7, 10, 12, and 13! Compete on the leaderboard by completing tasks and earning points.",
						Inline: false,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Bootstrap Hub Bot • Built for Founders, by Founders",
				},
			}

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			})
			if err != nil {
				log.Printf("Error responding to help command: %v", err)
			}
		},
	}
}
