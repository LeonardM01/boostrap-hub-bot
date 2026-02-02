package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

// standupCommand creates the /standup command group
func standupCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "standup",
			Description: "Daily standup check-ins with streak tracking",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "post",
					Description: "Post your daily standup",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "working-on",
							Description: "What are you working on today?",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "accomplished",
							Description: "What did you accomplish since your last standup?",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
						{
							Name:        "blockers",
							Description: "Any blockers or challenges?",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
					},
				},
				{
					Name:        "streak",
					Description: "View your standup streak stats",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "leaderboard",
					Description: "View the streak leaderboard",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "history",
					Description: "View your recent standups",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "days",
							Description: "Number of days to look back (default: 7)",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    false,
							MinValue:    floatPtr(1),
							MaxValue:    30,
						},
					},
				},
			},
		},
		Handler: handleStandupCommand,
	}
}

func handleStandupCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		respondWithError(s, i, "Invalid command usage")
		return
	}

	subCommand := options[0].Name

	// Get user info
	var userID, username, guildID string
	if i.Member != nil {
		userID = i.Member.User.ID
		username = i.Member.User.Username
		guildID = i.GuildID
	} else if i.User != nil {
		userID = i.User.ID
		username = i.User.Username
		guildID = "DM"
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(userID, guildID, username)
	if err != nil {
		log.Printf("Error getting/creating user: %v", err)
		respondWithError(s, i, "Failed to process your request. Please try again.")
		return
	}

	switch subCommand {
	case "post":
		handleStandupPost(s, i, user, guildID, options[0].Options)
	case "streak":
		handleStandupStreak(s, i, user, guildID)
	case "leaderboard":
		handleStandupLeaderboard(s, i, guildID)
	case "history":
		days := 7
		if len(options[0].Options) > 0 {
			days = int(options[0].Options[0].IntValue())
		}
		handleStandupHistory(s, i, user, guildID, days)
	default:
		respondWithError(s, i, "Unknown subcommand")
	}
}

func handleStandupPost(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var workingOn, accomplished, blockers string

	for _, opt := range options {
		switch opt.Name {
		case "working-on":
			workingOn = opt.StringValue()
		case "accomplished":
			accomplished = opt.StringValue()
		case "blockers":
			blockers = opt.StringValue()
		}
	}

	standup, streak, bonusPoints, err := database.CreateStandup(user.ID, guildID, workingOn, accomplished, blockers)
	if err != nil {
		if strings.Contains(err.Error(), "already posted") {
			embed := &discordgo.MessageEmbed{
				Title:       "Already Posted Today",
				Description: "You've already posted a standup today. Come back tomorrow to keep your streak going!",
				Color:       0xFFA500, // Orange
			}
			respondWithEmbed(s, i, embed)
			return
		}
		log.Printf("Error creating standup: %v", err)
		respondWithError(s, i, "Failed to post your standup.")
		return
	}

	// Build the standup display
	var description strings.Builder
	description.WriteString(fmt.Sprintf("**Working On:**\n%s\n\n", standup.WorkingOn))
	if standup.Accomplished != "" {
		description.WriteString(fmt.Sprintf("**Accomplished:**\n%s\n\n", standup.Accomplished))
	}
	if standup.Blockers != "" {
		description.WriteString(fmt.Sprintf("**Blockers:**\n%s\n\n", standup.Blockers))
	}

	// Streak display
	streakEmoji := getStreakEmoji(streak.CurrentStreak)
	streakText := fmt.Sprintf("%s %d day streak", streakEmoji, streak.CurrentStreak)

	embed := &discordgo.MessageEmbed{
		Title:       "Daily Standup Posted!",
		Description: description.String(),
		Color:       0x00FF00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Streak",
				Value:  streakText,
				Inline: true,
			},
			{
				Name:   "Points Earned",
				Value:  fmt.Sprintf("+1 base point"),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Total Standups: %d | Longest Streak: %d days", streak.TotalStandups, streak.LongestStreak),
		},
	}

	// Add bonus points field if earned
	if bonusPoints > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Streak Bonus!",
			Value:  fmt.Sprintf("+%d bonus points for %d day streak!", bonusPoints, streak.CurrentStreak),
			Inline: false,
		})
		embed.Color = 0xFFD700 // Gold for milestone
	}

	respondWithEmbed(s, i, embed)
}

func handleStandupStreak(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string) {
	streak, err := database.GetUserStreak(user.ID, guildID)
	if err != nil {
		log.Printf("Error getting streak: %v", err)
		respondWithError(s, i, "Failed to get your streak info.")
		return
	}

	streakEmoji := getStreakEmoji(streak.CurrentStreak)

	// Find next milestone
	nextMilestone := 0
	nextBonus := 0
	for days, bonus := range database.StreakMilestones {
		if days > streak.CurrentStreak && (nextMilestone == 0 || days < nextMilestone) {
			nextMilestone = days
			nextBonus = bonus
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: "Your Standup Streak",
		Color: 0x5865F2, // Discord blurple
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Current Streak",
				Value:  fmt.Sprintf("%s **%d days**", streakEmoji, streak.CurrentStreak),
				Inline: true,
			},
			{
				Name:   "Longest Streak",
				Value:  fmt.Sprintf("**%d days**", streak.LongestStreak),
				Inline: true,
			},
			{
				Name:   "Total Standups",
				Value:  fmt.Sprintf("**%d**", streak.TotalStandups),
				Inline: true,
			},
		},
	}

	if nextMilestone > 0 {
		daysToGo := nextMilestone - streak.CurrentStreak
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Next Milestone",
			Value:  fmt.Sprintf("%d days (%d to go) for +%d bonus points", nextMilestone, daysToGo, nextBonus),
			Inline: false,
		})
	}

	// Add milestone info
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Streak Milestones",
		Value:  "7 days: +10 pts | 14 days: +25 pts | 30 days: +50 pts\n60 days: +100 pts | 90 days: +200 pts",
		Inline: false,
	})

	respondWithEmbedEphemeral(s, i, embed, true)
}

func handleStandupLeaderboard(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) {
	entries, err := database.GetStreakLeaderboard(guildID, 10)
	if err != nil {
		log.Printf("Error getting streak leaderboard: %v", err)
		respondWithError(s, i, "Failed to get the leaderboard.")
		return
	}

	if len(entries) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "Streak Leaderboard",
			Description: "No standups posted yet! Be the first to start a streak with `/standup post`.",
			Color:       0x5865F2,
		}
		respondWithEmbed(s, i, embed)
		return
	}

	var description strings.Builder
	for _, entry := range entries {
		medal := ""
		switch entry.Rank {
		case 1:
			medal = "🥇"
		case 2:
			medal = "🥈"
		case 3:
			medal = "🥉"
		default:
			medal = fmt.Sprintf("`#%d`", entry.Rank)
		}

		streakEmoji := getStreakEmoji(entry.CurrentStreak)
		description.WriteString(fmt.Sprintf("%s **%s** - %s %d day streak (%d total)\n",
			medal, entry.Username, streakEmoji, entry.CurrentStreak, entry.TotalStandups))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Streak Leaderboard",
		Description: description.String(),
		Color:       0xFFD700, // Gold
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Post daily standups to climb the leaderboard!",
		},
	}

	respondWithEmbed(s, i, embed)
}

func handleStandupHistory(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, days int) {
	standups, err := database.GetUserStandups(user.ID, guildID, days)
	if err != nil {
		log.Printf("Error getting standup history: %v", err)
		respondWithError(s, i, "Failed to get your standup history.")
		return
	}

	if len(standups) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "Standup History",
			Description: fmt.Sprintf("No standups in the last %d days.\n\nStart your streak with `/standup post`!", days),
			Color:       0xFFA500, // Orange
		}
		respondWithEmbedEphemeral(s, i, embed, true)
		return
	}

	var description strings.Builder
	for _, standup := range standups {
		dateStr := standup.Date.Format("Jan 2, 2006")
		description.WriteString(fmt.Sprintf("**%s**\n", dateStr))
		description.WriteString(fmt.Sprintf("Working on: %s\n", truncateString(standup.WorkingOn, 100)))
		if standup.Accomplished != "" {
			description.WriteString(fmt.Sprintf("Accomplished: %s\n", truncateString(standup.Accomplished, 100)))
		}
		description.WriteString("\n")
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Your Standups (Last %d Days)", days),
		Description: description.String(),
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Showing %d standups", len(standups)),
		},
	}

	respondWithEmbedEphemeral(s, i, embed, true)
}

// getStreakEmoji returns an appropriate emoji based on streak length
func getStreakEmoji(streak int) string {
	switch {
	case streak >= 90:
		return "🔥🔥🔥"
	case streak >= 60:
		return "🔥🔥"
	case streak >= 30:
		return "🔥"
	case streak >= 14:
		return "⚡"
	case streak >= 7:
		return "✨"
	case streak >= 3:
		return "🌟"
	default:
		return "💪"
	}
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
