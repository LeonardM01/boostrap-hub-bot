package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

// winCommand creates the /win command group
func winCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "win",
			Description: "Share and celebrate your wins with the community",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "share",
					Description: "Share a win with the community",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "message",
							Description: "Describe your win",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "category",
							Description: "Category of your win",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "Revenue", Value: database.WinCategoryRevenue},
								{Name: "Product", Value: database.WinCategoryProduct},
								{Name: "Marketing", Value: database.WinCategoryMarketing},
								{Name: "Customer", Value: database.WinCategoryCustomer},
								{Name: "Other", Value: database.WinCategoryOther},
							},
						},
					},
				},
				{
					Name:        "recent",
					Description: "View recent community wins",
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
				{
					Name:        "stats",
					Description: "View win statistics for this server",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: handleWinCommand,
	}
}

func handleWinCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	case "share":
		handleWinShare(s, i, user, guildID, options[0].Options)
	case "recent":
		days := 7
		if len(options[0].Options) > 0 {
			days = int(options[0].Options[0].IntValue())
		}
		handleWinRecent(s, i, guildID, days)
	case "stats":
		handleWinStats(s, i, guildID)
	default:
		respondWithError(s, i, "Unknown subcommand")
	}
}

func handleWinShare(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var message, category string

	for _, opt := range options {
		switch opt.Name {
		case "message":
			message = opt.StringValue()
		case "category":
			category = opt.StringValue()
		}
	}

	win, err := database.CreateWin(user.ID, guildID, message, category)
	if err != nil {
		log.Printf("Error creating win: %v", err)
		respondWithError(s, i, "Failed to share your win.")
		return
	}

	// Get win count for user
	winCount, _ := database.GetUserWinCount(user.ID, guildID)

	categoryEmoji := getCategoryEmoji(win.Category)
	categoryDisplay := getCategoryDisplay(win.Category)

	embed := &discordgo.MessageEmbed{
		Title:       "Win Shared!",
		Description: fmt.Sprintf("%s **%s**\n\n%s", categoryEmoji, categoryDisplay, message),
		Color:       0x00FF00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Points Earned",
				Value:  "+2 points",
				Inline: true,
			},
			{
				Name:   "Your Total Wins",
				Value:  fmt.Sprintf("%d", winCount),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Celebrating wins together builds momentum!",
		},
	}

	respondWithEmbed(s, i, embed)

	// Post to wins channel if configured
	winsChannel, _ := database.GetWinsChannel(guildID)
	if winsChannel != "" {
		celebrationEmbed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s New Win from @%s!", categoryEmoji, user.Username),
			Description: message,
			Color:       0xFFD700, // Gold
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Category",
					Value:  categoryDisplay,
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Celebrate wins with /win share",
			},
		}

		msg, err := s.ChannelMessageSendEmbed(winsChannel, celebrationEmbed)
		if err != nil {
			log.Printf("Error posting win to channel: %v", err)
		} else {
			// Update win with message ID
			database.UpdateWinMessageID(win.ID, msg.ID)
			// Add celebration reactions
			s.MessageReactionAdd(winsChannel, msg.ID, "🎉")
			s.MessageReactionAdd(winsChannel, msg.ID, "🔥")
		}
	}
}

func handleWinRecent(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, days int) {
	wins, err := database.GetRecentWins(guildID, days)
	if err != nil {
		log.Printf("Error getting recent wins: %v", err)
		respondWithError(s, i, "Failed to get recent wins.")
		return
	}

	if len(wins) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "Recent Wins",
			Description: fmt.Sprintf("No wins shared in the last %d days.\n\nBe the first! Use `/win share` to celebrate a win.", days),
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	var description strings.Builder
	for idx, win := range wins {
		if idx >= 10 {
			description.WriteString(fmt.Sprintf("\n*...and %d more*", len(wins)-10))
			break
		}
		categoryEmoji := getCategoryEmoji(win.Category)
		dateStr := win.CreatedAt.Format("Jan 2")
		description.WriteString(fmt.Sprintf("%s **%s** (%s)\n%s\n\n",
			categoryEmoji, win.User.Username, dateStr, truncateString(win.Message, 100)))
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Recent Wins (Last %d Days)", days),
		Description: description.String(),
		Color:       0xFFD700, // Gold
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Total: %d wins | Share yours with /win share", len(wins)),
		},
	}

	respondWithEmbed(s, i, embed)
}

func handleWinStats(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) {
	stats, err := database.GetWinStats(guildID)
	if err != nil {
		log.Printf("Error getting win stats: %v", err)
		respondWithError(s, i, "Failed to get win statistics.")
		return
	}

	if stats.TotalWins == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "Win Statistics",
			Description: "No wins have been shared yet!\n\nStart the celebration with `/win share`.",
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	// Build category breakdown
	var categoryBreakdown strings.Builder
	categories := []string{database.WinCategoryRevenue, database.WinCategoryProduct, database.WinCategoryMarketing, database.WinCategoryCustomer, database.WinCategoryOther}
	for _, cat := range categories {
		count := stats.WinsByCategory[cat]
		if count > 0 {
			categoryBreakdown.WriteString(fmt.Sprintf("%s %s: %d\n", getCategoryEmoji(cat), getCategoryDisplay(cat), count))
		}
	}

	// Build top sharers
	var topSharers strings.Builder
	for idx, sharer := range stats.TopSharers {
		medal := ""
		switch idx {
		case 0:
			medal = "🥇"
		case 1:
			medal = "🥈"
		case 2:
			medal = "🥉"
		default:
			medal = fmt.Sprintf("`#%d`", idx+1)
		}
		topSharers.WriteString(fmt.Sprintf("%s **%s** - %d wins\n", medal, sharer.Username, sharer.WinCount))
	}

	embed := &discordgo.MessageEmbed{
		Title: "Win Statistics",
		Color: 0xFFD700, // Gold
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Total Wins",
				Value:  fmt.Sprintf("**%d**", stats.TotalWins),
				Inline: true,
			},
			{
				Name:   "This Week",
				Value:  fmt.Sprintf("**%d**", stats.RecentWinsCount),
				Inline: true,
			},
			{
				Name:   "By Category",
				Value:  categoryBreakdown.String(),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Share your wins with /win share",
		},
	}

	if len(stats.TopSharers) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Top Win Sharers",
			Value:  topSharers.String(),
			Inline: false,
		})
	}

	respondWithEmbed(s, i, embed)
}

func getCategoryEmoji(category string) string {
	switch category {
	case database.WinCategoryRevenue:
		return "💰"
	case database.WinCategoryProduct:
		return "🚀"
	case database.WinCategoryMarketing:
		return "📣"
	case database.WinCategoryCustomer:
		return "🤝"
	default:
		return "🎉"
	}
}

func getCategoryDisplay(category string) string {
	switch category {
	case database.WinCategoryRevenue:
		return "Revenue"
	case database.WinCategoryProduct:
		return "Product"
	case database.WinCategoryMarketing:
		return "Marketing"
	case database.WinCategoryCustomer:
		return "Customer"
	default:
		return "Other"
	}
}
