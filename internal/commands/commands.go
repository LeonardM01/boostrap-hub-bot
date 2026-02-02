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
						Value:  "`/focus start` - Start a new 2-week Focus Period\n`/focus add <goal>` - Add a goal (AI calculates points)\n`/focus complete <#>` - Mark a goal as done\n`/focus list` - View your goals\n`/focus status` - See your progress",
						Inline: false,
					},
					{
						Name:   "📝 Daily Standup Commands",
						Value:  "`/standup post` - Post your daily standup\n`/standup streak` - View your streak stats\n`/standup leaderboard` - View streak rankings\n`/standup history` - View recent standups",
						Inline: false,
					},
					{
						Name:   "🎉 Win Sharing Commands",
						Value:  "`/win share <message>` - Share a win\n`/win recent` - View recent community wins\n`/win stats` - View win statistics",
						Inline: false,
					},
					{
						Name:   "🤝 Accountability Buddy Commands",
						Value:  "`/buddy request @user` - Send buddy request\n`/buddy accept @user` - Accept request\n`/buddy status` - View buddy progress\n`/buddy list` - List your buddies\n`/buddy remove @user` - Remove a buddy",
						Inline: false,
					},
					{
						Name:   "⚔️ Challenge Commands",
						Value:  "`/challenge create` - Create a challenge with buddies\n`/challenge progress` - Log progress\n`/challenge complete` - Submit with proof\n`/challenge validate` - Validate buddy completion\n`/challenge list` - View your challenges",
						Inline: false,
					},
					{
						Name:   "💰 MRR Tracking Commands",
						Value:  "`/mrr update <amount>` - Log your MRR\n`/mrr public` / `/mrr private` - Toggle visibility\n`/mrr history` - View MRR trend\n`/mrr leaderboard` - View public rankings\n`/mrr stats` - View your statistics",
						Inline: false,
					},
					{
						Name:   "🏆 Leaderboard Commands",
						Value:  "`/leaderboard alltime` - All-time rankings\n`/leaderboard sprint` - Current sprint rankings",
						Inline: false,
					},
					{
						Name:   "⚙️ Admin Commands",
						Value:  "`/config leaderboard-channel` - Set leaderboard channel\n`/config wins-channel` - Set wins channel\n`/config mrr-channel` - Set MRR milestone channel",
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
