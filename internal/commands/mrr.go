package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

// mrrCommand creates the /mrr command group
func mrrCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "mrr",
			Description: "Track and share your Monthly Recurring Revenue",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "update",
					Description: "Log your current MRR",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "amount",
							Description: "Your current MRR (e.g., 1000 for $1,000)",
							Type:        discordgo.ApplicationCommandOptionNumber,
							Required:    true,
							MinValue:    floatPtr(0),
						},
						{
							Name:        "currency",
							Description: "Currency code (default: USD)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "USD ($)", Value: "USD"},
								{Name: "EUR (€)", Value: "EUR"},
								{Name: "GBP (£)", Value: "GBP"},
								{Name: "CAD (C$)", Value: "CAD"},
								{Name: "AUD (A$)", Value: "AUD"},
							},
						},
						{
							Name:        "note",
							Description: "Optional note about this update",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
					},
				},
				{
					Name:        "public",
					Description: "Make your MRR visible on the leaderboard",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "private",
					Description: "Hide your MRR from the leaderboard",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "history",
					Description: "View your MRR history",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "months",
							Description: "Number of months to look back (default: 6)",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    false,
							MinValue:    floatPtr(1),
							MaxValue:    24,
						},
					},
				},
				{
					Name:        "leaderboard",
					Description: "View the public MRR leaderboard",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "stats",
					Description: "View your MRR statistics",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: handleMRRCommand,
	}
}

func handleMRRCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	case "update":
		handleMRRUpdate(s, i, user, guildID, options[0].Options)
	case "public":
		handleMRRPublic(s, i, user, guildID)
	case "private":
		handleMRRPrivate(s, i, user, guildID)
	case "history":
		months := 6
		if len(options[0].Options) > 0 {
			months = int(options[0].Options[0].IntValue())
		}
		handleMRRHistory(s, i, user, guildID, months)
	case "leaderboard":
		handleMRRLeaderboard(s, i, guildID)
	case "stats":
		handleMRRStats(s, i, user, guildID)
	default:
		respondWithError(s, i, "Unknown subcommand")
	}
}

func handleMRRUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var amount float64
	var currency, note string

	for _, opt := range options {
		switch opt.Name {
		case "amount":
			amount = opt.FloatValue()
		case "currency":
			currency = opt.StringValue()
		case "note":
			note = opt.StringValue()
		}
	}

	if currency == "" {
		currency = "USD"
	}

	// Get previous MRR for comparison
	previousEntry, _ := database.GetLatestMRR(user.ID, guildID)
	var growth float64
	if previousEntry != nil {
		growth = database.GetMRRGrowth(amount, previousEntry.Amount)
	}

	entry, milestone, err := database.CreateMRREntry(user.ID, guildID, amount, currency, note)
	if err != nil {
		log.Printf("Error creating MRR entry: %v", err)
		respondWithError(s, i, "Failed to update your MRR.")
		return
	}

	currencySymbol := getCurrencySymbol(currency)

	embed := &discordgo.MessageEmbed{
		Title:       "MRR Updated!",
		Description: fmt.Sprintf("Your current MRR: **%s%.2f**", currencySymbol, amount),
		Color:       0x00FF00, // Green
		Fields:      []*discordgo.MessageEmbedField{},
	}

	if previousEntry != nil && previousEntry.Amount > 0 {
		growthEmoji := "📈"
		if growth < 0 {
			growthEmoji = "📉"
		} else if growth == 0 {
			growthEmoji = "➡️"
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Change",
			Value:  fmt.Sprintf("%s %+.1f%% from last update", growthEmoji, growth),
			Inline: true,
		})
	}

	if note != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Note",
			Value:  note,
			Inline: false,
		})
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Logged at %s | Use /mrr public to share on leaderboard", entry.Date.Format("Jan 2, 2006")),
	}

	respondWithEmbed(s, i, embed)

	// Handle milestone celebration
	if milestone > 0 {
		milestoneStr := database.FormatMRRMilestone(milestone)
		celebrationEmbed := &discordgo.MessageEmbed{
			Title:       "🎉 MRR Milestone Reached!",
			Description: fmt.Sprintf("Congratulations! You've reached **%s MRR**!", milestoneStr),
			Color:       0xFFD700, // Gold
		}

		// Post to MRR channel if configured and user is public
		settings, _ := database.GetMRRSettings(user.ID, guildID)
		if settings != nil && settings.IsPublic {
			mrrChannel, _ := database.GetMRRChannel(guildID)
			if mrrChannel != "" {
				channelEmbed := &discordgo.MessageEmbed{
					Title:       "🎉 Milestone Reached!",
					Description: fmt.Sprintf("**%s** just hit **%s MRR**!\n\nCongratulations!", user.Username, milestoneStr),
					Color:       0xFFD700,
					Footer: &discordgo.MessageEmbedFooter{
						Text: "Track your MRR with /mrr update",
					},
				}
				msg, err := s.ChannelMessageSendEmbed(mrrChannel, channelEmbed)
				if err == nil {
					s.MessageReactionAdd(mrrChannel, msg.ID, "🎉")
					s.MessageReactionAdd(mrrChannel, msg.ID, "🚀")
				}
			}
		}

		// Follow up with milestone message to user
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{celebrationEmbed},
		})
	}
}

func handleMRRPublic(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string) {
	err := database.UpdateMRRVisibility(user.ID, guildID, true)
	if err != nil {
		log.Printf("Error updating MRR visibility: %v", err)
		respondWithError(s, i, "Failed to update visibility.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "MRR Now Public",
		Description: "Your MRR is now visible on the leaderboard.\n\nOther founders can see your progress and celebrate your milestones!",
		Color:       0x00FF00, // Green
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /mrr private to hide it again",
		},
	}

	respondWithEmbed(s, i, embed)
}

func handleMRRPrivate(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string) {
	err := database.UpdateMRRVisibility(user.ID, guildID, false)
	if err != nil {
		log.Printf("Error updating MRR visibility: %v", err)
		respondWithError(s, i, "Failed to update visibility.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "MRR Now Private",
		Description: "Your MRR is now hidden from the leaderboard.\n\nYou can still track your progress privately.",
		Color:       0xFFA500, // Orange
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /mrr public to share it again",
		},
	}

	respondWithEmbed(s, i, embed)
}

func handleMRRHistory(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, months int) {
	entries, err := database.GetMRRHistory(user.ID, guildID, months)
	if err != nil {
		log.Printf("Error getting MRR history: %v", err)
		respondWithError(s, i, "Failed to get your MRR history.")
		return
	}

	if len(entries) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "MRR History",
			Description: "No MRR entries found.\n\nStart tracking with `/mrr update`!",
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	var description strings.Builder
	var previousAmount float64

	for idx, entry := range entries {
		if idx >= 12 {
			description.WriteString(fmt.Sprintf("\n*...and %d more entries*", len(entries)-12))
			break
		}

		currencySymbol := getCurrencySymbol(entry.Currency)
		dateStr := entry.Date.Format("Jan 2, 2006")

		// Calculate growth from previous entry (which is actually newer due to DESC order)
		growthStr := ""
		if previousAmount > 0 {
			growth := database.GetMRRGrowth(previousAmount, entry.Amount)
			if growth > 0 {
				growthStr = fmt.Sprintf(" 📈 +%.1f%%", growth)
			} else if growth < 0 {
				growthStr = fmt.Sprintf(" 📉 %.1f%%", growth)
			}
		}
		previousAmount = entry.Amount

		description.WriteString(fmt.Sprintf("**%s** - %s%.2f%s\n", dateStr, currencySymbol, entry.Amount, growthStr))
		if entry.Note != "" {
			description.WriteString(fmt.Sprintf("  *%s*\n", truncateString(entry.Note, 50)))
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Your MRR History (Last %d Months)", months),
		Description: description.String(),
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%d entries | Use /mrr stats for more insights", len(entries)),
		},
	}

	respondWithEmbed(s, i, embed)
}

func handleMRRLeaderboard(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) {
	entries, err := database.GetMRRLeaderboard(guildID, 10)
	if err != nil {
		log.Printf("Error getting MRR leaderboard: %v", err)
		respondWithError(s, i, "Failed to get the leaderboard.")
		return
	}

	if len(entries) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "MRR Leaderboard",
			Description: "No public MRR entries yet!\n\nBe the first to share with `/mrr update` and `/mrr public`.",
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

		currencySymbol := getCurrencySymbol(entry.Currency)
		description.WriteString(fmt.Sprintf("%s **%s** - %s%.2f/mo\n",
			medal, entry.Username, currencySymbol, entry.Amount))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "MRR Leaderboard",
		Description: description.String(),
		Color:       0xFFD700, // Gold
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Share your MRR with /mrr update & /mrr public",
		},
	}

	respondWithEmbed(s, i, embed)
}

func handleMRRStats(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string) {
	stats, err := database.GetMRRStats(user.ID, guildID)
	if err != nil {
		log.Printf("Error getting MRR stats: %v", err)
		respondWithError(s, i, "Failed to get your MRR stats.")
		return
	}

	if stats.TotalEntries == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "MRR Statistics",
			Description: "No MRR data yet.\n\nStart tracking with `/mrr update`!",
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	currencySymbol := getCurrencySymbol(stats.Currency)

	embed := &discordgo.MessageEmbed{
		Title: "Your MRR Statistics",
		Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Current MRR",
				Value:  fmt.Sprintf("%s%.2f", currencySymbol, stats.CurrentMRR),
				Inline: true,
			},
			{
				Name:   "All-Time High",
				Value:  fmt.Sprintf("%s%.2f", currencySymbol, stats.AllTimeHigh),
				Inline: true,
			},
			{
				Name:   "Monthly Growth",
				Value:  fmt.Sprintf("%+.1f%%", stats.MonthlyGrowth),
				Inline: true,
			},
			{
				Name:   "Total Updates",
				Value:  fmt.Sprintf("%d", stats.TotalEntries),
				Inline: true,
			},
			{
				Name:   "Milestones Hit",
				Value:  fmt.Sprintf("%d/%d", stats.MilestonesHit, len(database.MRRMilestones)),
				Inline: true,
			},
			{
				Name:   "Visibility",
				Value:  getVisibilityStatus(stats.IsPublic),
				Inline: true,
			},
		},
	}

	if stats.NextMilestone > 0 {
		nextMilestoneStr := database.FormatMRRMilestone(stats.NextMilestone)
		toGo := float64(stats.NextMilestone)/100 - stats.CurrentMRR
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Next Milestone",
			Value:  fmt.Sprintf("%s (%s%.2f to go)", nextMilestoneStr, currencySymbol, toGo),
			Inline: false,
		})
	}

	if stats.FirstEntry != nil {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Tracking since %s", stats.FirstEntry.Format("Jan 2006")),
		}
	}

	respondWithEmbed(s, i, embed)
}

func getCurrencySymbol(currency string) string {
	switch currency {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	case "CAD":
		return "C$"
	case "AUD":
		return "A$"
	default:
		return "$"
	}
}

func getVisibilityStatus(isPublic bool) string {
	if isPublic {
		return "🌐 Public"
	}
	return "🔒 Private"
}
