package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

// buddyCommand creates the /buddy command group
func buddyCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "buddy",
			Description: "Manage your accountability buddies",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "request",
					Description: "Send a buddy request to another user",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The user to send a request to",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
					},
				},
				{
					Name:        "accept",
					Description: "Accept a buddy request",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The user whose request to accept",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
					},
				},
				{
					Name:        "decline",
					Description: "Decline a buddy request",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The user whose request to decline",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
					},
				},
				{
					Name:        "status",
					Description: "View a buddy's focus period progress",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The buddy to check (leave empty to see all buddies)",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    false,
						},
					},
				},
				{
					Name:        "list",
					Description: "List your buddies and pending requests",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "remove",
					Description: "Remove a buddy",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "user",
							Description: "The buddy to remove",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
					},
				},
			},
		},
		Handler: handleBuddyCommand,
	}
}

func handleBuddyCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	if guildID == "DM" {
		respondWithError(s, i, "Buddy commands can only be used in a server.")
		return
	}

	// Get or create user in database
	user, err := database.GetOrCreateUser(userID, guildID, username)
	if err != nil {
		log.Printf("Error getting/creating user: %v", err)
		respondWithError(s, i, "Failed to process your request. Please try again.")
		return
	}

	switch subCommand {
	case "request":
		targetUser := options[0].Options[0].UserValue(s)
		handleBuddyRequest(s, i, user, guildID, targetUser)
	case "accept":
		targetUser := options[0].Options[0].UserValue(s)
		handleBuddyAccept(s, i, user, guildID, targetUser)
	case "decline":
		targetUser := options[0].Options[0].UserValue(s)
		handleBuddyDecline(s, i, user, guildID, targetUser)
	case "status":
		var targetUser *discordgo.User
		if len(options[0].Options) > 0 {
			targetUser = options[0].Options[0].UserValue(s)
		}
		handleBuddyStatus(s, i, user, guildID, targetUser)
	case "list":
		handleBuddyList(s, i, user, guildID)
	case "remove":
		targetUser := options[0].Options[0].UserValue(s)
		handleBuddyRemove(s, i, user, guildID, targetUser)
	default:
		respondWithError(s, i, "Unknown subcommand")
	}
}

func handleBuddyRequest(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, targetUser *discordgo.User) {
	if targetUser.ID == user.DiscordID {
		respondWithError(s, i, "You can't be your own accountability buddy!")
		return
	}

	if targetUser.Bot {
		respondWithError(s, i, "You can't add bots as buddies.")
		return
	}

	// Get or create target user
	target, err := database.GetOrCreateUser(targetUser.ID, guildID, targetUser.Username)
	if err != nil {
		log.Printf("Error getting target user: %v", err)
		respondWithError(s, i, "Failed to find that user.")
		return
	}

	request, err := database.CreateBuddyRequest(user.ID, target.ID, guildID)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Buddy Request Sent!",
		Description: fmt.Sprintf("You've sent a buddy request to **%s**.\n\nThey have 7 days to accept.", targetUser.Username),
		Color:       0x00FF00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "What's Next",
				Value:  fmt.Sprintf("<@%s> can use `/buddy accept @%s` to accept your request.", targetUser.ID, user.Username),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Request expires: %s", request.ExpiresAt.Format("Jan 2, 2006")),
		},
	}

	respondWithEmbed(s, i, embed)

	// Try to DM the target user
	channel, err := s.UserChannelCreate(targetUser.ID)
	if err == nil {
		dmEmbed := &discordgo.MessageEmbed{
			Title:       "New Buddy Request!",
			Description: fmt.Sprintf("**%s** wants to be your accountability buddy!\n\nAccountability buddies get notified when you complete tasks and can track each other's progress.", user.Username),
			Color:       0x5865F2,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "To Accept",
					Value:  fmt.Sprintf("Use `/buddy accept @%s` in the server.", user.Username),
					Inline: false,
				},
			},
		}
		s.ChannelMessageSendEmbed(channel.ID, dmEmbed)
	}
}

func handleBuddyAccept(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, targetUser *discordgo.User) {
	// Get requester
	requester, err := database.GetOrCreateUser(targetUser.ID, guildID, targetUser.Username)
	if err != nil {
		log.Printf("Error getting requester: %v", err)
		respondWithError(s, i, "Failed to find that user.")
		return
	}

	_, err = database.AcceptBuddyRequest(requester.ID, user.ID, guildID)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Buddy Request Accepted!",
		Description: fmt.Sprintf("You and **%s** are now accountability buddies!", targetUser.Username),
		Color:       0x00FF00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Benefits",
				Value:  "- Get notified when your buddy completes tasks\n- Track each other's progress with `/buddy status`\n- Stay accountable together!",
				Inline: false,
			},
		},
	}

	respondWithEmbed(s, i, embed)

	// Try to DM the requester
	channel, err := s.UserChannelCreate(targetUser.ID)
	if err == nil {
		dmEmbed := &discordgo.MessageEmbed{
			Title:       "Buddy Request Accepted!",
			Description: fmt.Sprintf("**%s** accepted your buddy request! You're now accountability buddies.", user.Username),
			Color:       0x00FF00,
		}
		s.ChannelMessageSendEmbed(channel.ID, dmEmbed)
	}
}

func handleBuddyDecline(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, targetUser *discordgo.User) {
	// Get requester
	requester, err := database.GetOrCreateUser(targetUser.ID, guildID, targetUser.Username)
	if err != nil {
		log.Printf("Error getting requester: %v", err)
		respondWithError(s, i, "Failed to find that user.")
		return
	}

	err = database.DeclineBuddyRequest(requester.ID, user.ID, guildID)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Buddy Request Declined",
		Description: fmt.Sprintf("You've declined the buddy request from **%s**.", targetUser.Username),
		Color:       0xFFA500, // Orange
	}

	respondWithEmbed(s, i, embed)
}

func handleBuddyStatus(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, targetUser *discordgo.User) {
	if targetUser != nil {
		// Check specific buddy's status
		target, err := database.GetOrCreateUser(targetUser.ID, guildID, targetUser.Username)
		if err != nil {
			log.Printf("Error getting target user: %v", err)
			respondWithError(s, i, "Failed to find that user.")
			return
		}

		isBuddy, err := database.AreBuddies(user.ID, target.ID, guildID)
		if err != nil || !isBuddy {
			respondWithError(s, i, "You're not buddies with this user.")
			return
		}

		// Get their focus period
		period, err := database.GetCurrentFocusPeriod(target.ID)
		if err != nil {
			log.Printf("Error getting focus period: %v", err)
			respondWithError(s, i, "Failed to get buddy's status.")
			return
		}

		if period == nil {
			embed := &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("%s's Focus Period", targetUser.Username),
				Description: "This buddy doesn't have an active Focus Period right now.",
				Color:       0xFFA500, // Orange
			}
			respondWithEmbed(s, i, embed)
			return
		}

		completedCount := period.CompletedTaskCount()
		totalCount := len(period.Tasks)
		progressPercent := 0
		if totalCount > 0 {
			progressPercent = (completedCount * 100) / totalCount
		}
		progressBar := buildProgressBar(progressPercent)

		embed := &discordgo.MessageEmbed{
			Title: fmt.Sprintf("%s's Focus Period", targetUser.Username),
			Color: 0x5865F2,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Progress",
					Value:  fmt.Sprintf("%s %d%%", progressBar, progressPercent),
					Inline: false,
				},
				{
					Name:   "Tasks",
					Value:  fmt.Sprintf("✅ %d completed | ⏳ %d pending", completedCount, totalCount-completedCount),
					Inline: true,
				},
				{
					Name:   "Day",
					Value:  fmt.Sprintf("Day %d of 14 (%d days left)", period.DayNumber(), period.DaysRemaining()),
					Inline: true,
				},
			},
		}

		respondWithEmbed(s, i, embed)
		return
	}

	// Show all buddies' status
	buddies, err := database.GetUserBuddies(user.ID, guildID)
	if err != nil {
		log.Printf("Error getting buddies: %v", err)
		respondWithError(s, i, "Failed to get buddies.")
		return
	}

	if len(buddies) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "Buddy Status",
			Description: "You don't have any buddies yet!\n\nUse `/buddy request @user` to send a request.",
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	var description strings.Builder
	for _, buddy := range buddies {
		period, _ := database.GetCurrentFocusPeriod(buddy.ID)
		if period != nil {
			completedCount := period.CompletedTaskCount()
			totalCount := len(period.Tasks)
			progressPercent := 0
			if totalCount > 0 {
				progressPercent = (completedCount * 100) / totalCount
			}
			description.WriteString(fmt.Sprintf("**%s** - Day %d | %d%% (%d/%d tasks)\n",
				buddy.Username, period.DayNumber(), progressPercent, completedCount, totalCount))
		} else {
			description.WriteString(fmt.Sprintf("**%s** - No active Focus Period\n", buddy.Username))
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Your Buddies' Progress",
		Description: description.String(),
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /buddy status @user for detailed view",
		},
	}

	respondWithEmbed(s, i, embed)
}

func handleBuddyList(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string) {
	buddies, err := database.GetUserBuddies(user.ID, guildID)
	if err != nil {
		log.Printf("Error getting buddies: %v", err)
		respondWithError(s, i, "Failed to get your buddy list.")
		return
	}

	pendingReceived, err := database.GetPendingBuddyRequests(user.ID, guildID)
	if err != nil {
		log.Printf("Error getting pending requests: %v", err)
	}

	pendingSent, err := database.GetSentBuddyRequests(user.ID, guildID)
	if err != nil {
		log.Printf("Error getting sent requests: %v", err)
	}

	var fields []*discordgo.MessageEmbedField

	// Current buddies
	if len(buddies) > 0 {
		var buddyList strings.Builder
		for _, buddy := range buddies {
			buddyList.WriteString(fmt.Sprintf("- **%s** (<@%s>)\n", buddy.Username, buddy.DiscordID))
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("Your Buddies (%d/%d)", len(buddies), database.MaxBuddiesPerUser),
			Value:  buddyList.String(),
			Inline: false,
		})
	} else {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Your Buddies (0/3)",
			Value:  "No buddies yet. Use `/buddy request @user` to add one!",
			Inline: false,
		})
	}

	// Pending received
	if len(pendingReceived) > 0 {
		var requestList strings.Builder
		for _, req := range pendingReceived {
			requestList.WriteString(fmt.Sprintf("- **%s** - use `/buddy accept @%s`\n",
				req.Requester.Username, req.Requester.Username))
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Pending Requests",
			Value:  requestList.String(),
			Inline: false,
		})
	}

	// Pending sent
	if len(pendingSent) > 0 {
		var sentList strings.Builder
		for _, req := range pendingSent {
			sentList.WriteString(fmt.Sprintf("- **%s** (expires %s)\n",
				req.Receiver.Username, req.ExpiresAt.Format("Jan 2")))
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Sent Requests",
			Value:  sentList.String(),
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Your Accountability Buddies",
		Color:  0x5865F2,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Buddies get notified when you complete tasks!",
		},
	}

	respondWithEmbed(s, i, embed)
}

func handleBuddyRemove(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, targetUser *discordgo.User) {
	target, err := database.GetOrCreateUser(targetUser.ID, guildID, targetUser.Username)
	if err != nil {
		log.Printf("Error getting target user: %v", err)
		respondWithError(s, i, "Failed to find that user.")
		return
	}

	err = database.RemoveBuddy(user.ID, target.ID, guildID)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Buddy Removed",
		Description: fmt.Sprintf("You and **%s** are no longer accountability buddies.", targetUser.Username),
		Color:       0xFFA500, // Orange
	}

	respondWithEmbed(s, i, embed)
}

// NotifyBuddiesOfCompletion sends DMs to buddies when a user completes a task
func NotifyBuddiesOfCompletion(s *discordgo.Session, user *database.User, guildID string, task *database.Task) {
	buddies, err := database.GetBuddiesWithNotifications(user.ID, guildID)
	if err != nil {
		log.Printf("Error getting buddies for notification: %v", err)
		return
	}

	for _, buddy := range buddies {
		channel, err := s.UserChannelCreate(buddy.DiscordID)
		if err != nil {
			continue
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Buddy Progress Update!",
			Description: fmt.Sprintf("**%s** just completed a task!", user.Username),
			Color:       0x00FF00, // Green
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Completed Task",
					Value:  task.Title,
					Inline: false,
				},
				{
					Name:   "Points Earned",
					Value:  fmt.Sprintf("+%d points", task.Points),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Keep each other accountable!",
			},
		}

		s.ChannelMessageSendEmbed(channel.ID, embed)
	}
}
