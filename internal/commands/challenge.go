package commands

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

// challengeCommand creates the /challenge command group
func challengeCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "challenge",
			Description: "Create and participate in time-boxed challenges with your buddies",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a new challenge",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "goal",
							Description: "The challenge goal",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "days",
							Description: "Number of days for the challenge",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
							MinValue:    floatPtr(1),
							MaxValue:    30,
						},
						{
							Name:        "buddy1",
							Description: "First buddy to challenge",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
						{
							Name:        "buddy2",
							Description: "Second buddy (optional)",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    false,
						},
						{
							Name:        "buddy3",
							Description: "Third buddy (optional)",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    false,
						},
						{
							Name:        "multiplier",
							Description: "Points multiplier (1.0-3.0, default: 1.5)",
							Type:        discordgo.ApplicationCommandOptionNumber,
							Required:    false,
							MinValue:    floatPtr(1.0),
							MaxValue:    3.0,
						},
					},
				},
				{
					Name:        "progress",
					Description: "Log progress on a challenge",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "id",
							Description: "The challenge ID",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
							MinValue:    floatPtr(1),
						},
						{
							Name:        "update",
							Description: "Your progress update",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "complete",
					Description: "Submit challenge completion with proof",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "id",
							Description: "The challenge ID",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
							MinValue:    floatPtr(1),
						},
						{
							Name:        "proof-url",
							Description: "URL to proof of completion (screenshot, link, etc.)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "validate",
					Description: "Validate a buddy's challenge completion",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "id",
							Description: "The challenge ID",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
							MinValue:    floatPtr(1),
						},
						{
							Name:        "user",
							Description: "The user to validate",
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    true,
						},
						{
							Name:        "approve",
							Description: "Approve or reject the completion",
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Required:    true,
						},
					},
				},
				{
					Name:        "list",
					Description: "List your challenges",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "status",
							Description: "Filter by status",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "Active", Value: database.ChallengeStatusActive},
								{Name: "Completed", Value: database.ChallengeStatusCompleted},
								{Name: "Failed", Value: database.ChallengeStatusFailed},
							},
						},
					},
				},
				{
					Name:        "view",
					Description: "View challenge details",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "id",
							Description: "The challenge ID",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
							MinValue:    floatPtr(1),
						},
					},
				},
			},
		},
		Handler: handleChallengeCommand,
	}
}

func handleChallengeCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
		respondWithError(s, i, "Challenge commands can only be used in a server.")
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
	case "create":
		handleChallengeCreate(s, i, user, guildID, options[0].Options)
	case "progress":
		handleChallengeProgress(s, i, user, options[0].Options)
	case "complete":
		handleChallengeComplete(s, i, user, options[0].Options)
	case "validate":
		handleChallengeValidate(s, i, user, guildID, options[0].Options)
	case "list":
		status := ""
		if len(options[0].Options) > 0 {
			status = options[0].Options[0].StringValue()
		}
		handleChallengeList(s, i, user, guildID, status)
	case "view":
		challengeID := uint(options[0].Options[0].IntValue())
		handleChallengeView(s, i, user, challengeID)
	default:
		respondWithError(s, i, "Unknown subcommand")
	}
}

func handleChallengeCreate(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var goal string
	var days int
	var multiplier float64 = 1.5
	var buddyUsers []*discordgo.User

	for _, opt := range options {
		switch opt.Name {
		case "goal":
			goal = opt.StringValue()
		case "days":
			days = int(opt.IntValue())
		case "multiplier":
			multiplier = opt.FloatValue()
		case "buddy1", "buddy2", "buddy3":
			buddyUser := opt.UserValue(s)
			if buddyUser != nil && buddyUser.ID != user.DiscordID && !buddyUser.Bot {
				buddyUsers = append(buddyUsers, buddyUser)
			}
		}
	}

	if len(buddyUsers) == 0 {
		respondWithError(s, i, "You need to add at least one buddy to create a challenge.")
		return
	}

	// Verify all buddies are actually buddies
	participantIDs := make([]uint, 0, len(buddyUsers))
	for _, buddyUser := range buddyUsers {
		buddy, err := database.GetOrCreateUser(buddyUser.ID, guildID, buddyUser.Username)
		if err != nil {
			continue
		}

		isBuddy, _ := database.AreBuddies(user.ID, buddy.ID, guildID)
		if !isBuddy {
			respondWithError(s, i, fmt.Sprintf("**%s** is not your buddy. Add them as a buddy first with `/buddy request`.", buddyUser.Username))
			return
		}
		participantIDs = append(participantIDs, buddy.ID)
	}

	challenge, err := database.CreateChallenge(user.ID, guildID, goal, "", days, participantIDs, multiplier)
	if err != nil {
		log.Printf("Error creating challenge: %v", err)
		respondWithError(s, i, "Failed to create challenge.")
		return
	}

	// Build participant list
	var participantList strings.Builder
	participantList.WriteString(fmt.Sprintf("- **%s** (creator)\n", user.Username))
	for _, buddyUser := range buddyUsers {
		participantList.WriteString(fmt.Sprintf("- **%s**\n", buddyUser.Username))
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Challenge #%d Created!", challenge.ID),
		Description: fmt.Sprintf("**Goal:** %s", goal),
		Color:       0x00FF00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Duration",
				Value:  fmt.Sprintf("%d days (ends %s)", days, challenge.EndDate.Format("Jan 2, 2006")),
				Inline: true,
			},
			{
				Name:   "Points Multiplier",
				Value:  fmt.Sprintf("%.1fx", multiplier),
				Inline: true,
			},
			{
				Name:   "Participants",
				Value:  participantList.String(),
				Inline: false,
			},
			{
				Name:   "How to Complete",
				Value:  fmt.Sprintf("1. Log progress: `/challenge progress %d <update>`\n2. Submit proof: `/challenge complete %d <proof-url>`\n3. Get validated by a buddy!", challenge.ID, challenge.ID),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Good luck! May the best founder win!",
		},
	}

	respondWithEmbed(s, i, embed)

	// Notify buddies
	for _, buddyUser := range buddyUsers {
		channel, err := s.UserChannelCreate(buddyUser.ID)
		if err != nil {
			continue
		}

		dmEmbed := &discordgo.MessageEmbed{
			Title:       "New Challenge!",
			Description: fmt.Sprintf("**%s** challenged you!\n\n**Goal:** %s", user.Username, goal),
			Color:       0x5865F2,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Duration",
					Value:  fmt.Sprintf("%d days", days),
					Inline: true,
				},
				{
					Name:   "Points Multiplier",
					Value:  fmt.Sprintf("%.1fx", multiplier),
					Inline: true,
				},
				{
					Name:   "Challenge ID",
					Value:  fmt.Sprintf("#%d", challenge.ID),
					Inline: true,
				},
			},
		}
		s.ChannelMessageSendEmbed(channel.ID, dmEmbed)
	}
}

func handleChallengeProgress(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var challengeID uint
	var update string

	for _, opt := range options {
		switch opt.Name {
		case "id":
			challengeID = uint(opt.IntValue())
		case "update":
			update = opt.StringValue()
		}
	}

	progress, err := database.AddChallengeProgress(challengeID, user.ID, update)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Progress Logged - Challenge #%d", challengeID),
		Description: update,
		Color:       0x00FF00, // Green
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Logged at %s", progress.CreatedAt.Format("Jan 2, 3:04 PM")),
		},
	}

	respondWithEmbed(s, i, embed)
}

func handleChallengeComplete(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var challengeID uint
	var proofURL string

	for _, opt := range options {
		switch opt.Name {
		case "id":
			challengeID = uint(opt.IntValue())
		case "proof-url":
			proofURL = opt.StringValue()
		}
	}

	_, err := database.SubmitChallengeCompletion(challengeID, user.ID, proofURL)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Completion Submitted!",
		Description: fmt.Sprintf("Your completion for Challenge #%d is pending validation.\n\n**Proof:** %s", challengeID, proofURL),
		Color:       0xFFA500, // Orange
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "What's Next",
				Value:  "A challenge buddy needs to validate your completion with `/challenge validate`",
				Inline: false,
			},
		},
	}

	respondWithEmbed(s, i, embed)

	// Notify other participants
	_, participants, _ := database.GetChallengeWithParticipants(challengeID)
	for _, p := range participants {
		if p.UserID == user.ID {
			continue
		}
		channel, err := s.UserChannelCreate(p.User.DiscordID)
		if err != nil {
			continue
		}

		dmEmbed := &discordgo.MessageEmbed{
			Title:       "Buddy Submitted Challenge Completion!",
			Description: fmt.Sprintf("**%s** submitted completion for Challenge #%d.\n\n**Proof:** %s", user.Username, challengeID, proofURL),
			Color:       0x5865F2,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "To Validate",
					Value:  fmt.Sprintf("`/challenge validate %d @%s true/false`", challengeID, user.Username),
					Inline: false,
				},
			},
		}
		s.ChannelMessageSendEmbed(channel.ID, dmEmbed)
	}
}

func handleChallengeValidate(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var challengeID uint
	var targetUser *discordgo.User
	var approve bool

	for _, opt := range options {
		switch opt.Name {
		case "id":
			challengeID = uint(opt.IntValue())
		case "user":
			targetUser = opt.UserValue(s)
		case "approve":
			approve = opt.BoolValue()
		}
	}

	target, err := database.GetOrCreateUser(targetUser.ID, guildID, targetUser.Username)
	if err != nil {
		respondWithError(s, i, "Failed to find that user.")
		return
	}

	err = database.ValidateChallengeCompletion(challengeID, user.ID, target.ID, approve)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	var embed *discordgo.MessageEmbed
	if approve {
		embed = &discordgo.MessageEmbed{
			Title:       "Completion Validated!",
			Description: fmt.Sprintf("You approved **%s**'s completion for Challenge #%d.\n\nThey've earned bonus points!", targetUser.Username, challengeID),
			Color:       0x00FF00, // Green
		}
	} else {
		embed = &discordgo.MessageEmbed{
			Title:       "Completion Rejected",
			Description: fmt.Sprintf("You rejected **%s**'s completion for Challenge #%d.\n\nThey can resubmit with better proof.", targetUser.Username, challengeID),
			Color:       0xFF0000, // Red
		}
	}

	respondWithEmbed(s, i, embed)

	// Notify the target user
	channel, err := s.UserChannelCreate(targetUser.ID)
	if err == nil {
		var dmEmbed *discordgo.MessageEmbed
		if approve {
			dmEmbed = &discordgo.MessageEmbed{
				Title:       "Challenge Completion Approved!",
				Description: fmt.Sprintf("**%s** validated your completion for Challenge #%d.\n\nCongratulations on earning bonus points!", user.Username, challengeID),
				Color:       0x00FF00,
			}
		} else {
			dmEmbed = &discordgo.MessageEmbed{
				Title:       "Challenge Completion Rejected",
				Description: fmt.Sprintf("**%s** rejected your completion for Challenge #%d.\n\nYou can resubmit with `/challenge complete`.", user.Username, challengeID),
				Color:       0xFF0000,
			}
		}
		s.ChannelMessageSendEmbed(channel.ID, dmEmbed)
	}
}

func handleChallengeList(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string, status string) {
	challenges, err := database.GetUserChallenges(user.ID, guildID, status)
	if err != nil {
		log.Printf("Error getting challenges: %v", err)
		respondWithError(s, i, "Failed to get your challenges.")
		return
	}

	if len(challenges) == 0 {
		title := "Your Challenges"
		if status != "" {
			title = fmt.Sprintf("Your %s Challenges", strings.Title(status))
		}
		embed := &discordgo.MessageEmbed{
			Title:       title,
			Description: "No challenges found.\n\nCreate one with `/challenge create`!",
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	var description strings.Builder
	for _, challenge := range challenges {
		statusEmoji := getChallengeStatusEmoji(challenge.Status)
		daysLeft := int(time.Until(challenge.EndDate).Hours() / 24)
		if daysLeft < 0 {
			daysLeft = 0
		}

		description.WriteString(fmt.Sprintf("%s **#%d** - %s\n", statusEmoji, challenge.ID, truncateString(challenge.Title, 40)))
		if challenge.Status == database.ChallengeStatusActive {
			description.WriteString(fmt.Sprintf("   %d days left | %.1fx multiplier\n", daysLeft, challenge.PointsMultiplier))
		}
		description.WriteString("\n")
	}

	title := "Your Challenges"
	if status != "" {
		title = fmt.Sprintf("Your %s Challenges", strings.Title(status))
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description.String(),
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /challenge view <id> for details",
		},
	}

	respondWithEmbed(s, i, embed)
}

func handleChallengeView(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, challengeID uint) {
	challenge, participants, err := database.GetChallengeWithParticipants(challengeID)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	// Verify user is a participant
	isParticipant, _ := database.IsUserInChallenge(challengeID, user.ID)
	if !isParticipant {
		respondWithError(s, i, "You're not a participant in this challenge.")
		return
	}

	// Build participant status
	var participantList strings.Builder
	for _, p := range participants {
		statusEmoji := getParticipantStatusEmoji(p.Status)
		participantList.WriteString(fmt.Sprintf("%s **%s** - %s\n", statusEmoji, p.User.Username, p.Status))
	}

	daysLeft := int(time.Until(challenge.EndDate).Hours() / 24)
	if daysLeft < 0 {
		daysLeft = 0
	}

	statusEmoji := getChallengeStatusEmoji(challenge.Status)

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Challenge #%d - %s", challenge.ID, challenge.Title),
		Description: challenge.Description,
		Color:       getChallengeStatusColor(challenge.Status),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  fmt.Sprintf("%s %s", statusEmoji, strings.Title(challenge.Status)),
				Inline: true,
			},
			{
				Name:   "Time Left",
				Value:  fmt.Sprintf("%d days", daysLeft),
				Inline: true,
			},
			{
				Name:   "Multiplier",
				Value:  fmt.Sprintf("%.1fx", challenge.PointsMultiplier),
				Inline: true,
			},
			{
				Name:   "Participants",
				Value:  participantList.String(),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Created by %s | Ends %s", challenge.Creator.Username, challenge.EndDate.Format("Jan 2, 2006")),
		},
	}

	// Get recent progress
	progress, _ := database.GetChallengeProgress(challengeID)
	if len(progress) > 0 {
		var progressList strings.Builder
		for idx, p := range progress {
			if idx >= 5 {
				progressList.WriteString(fmt.Sprintf("*...and %d more updates*", len(progress)-5))
				break
			}
			progressList.WriteString(fmt.Sprintf("**%s** (%s): %s\n",
				p.User.Username, p.CreatedAt.Format("Jan 2"), truncateString(p.Update, 50)))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Recent Progress",
			Value:  progressList.String(),
			Inline: false,
		})
	}

	respondWithEmbed(s, i, embed)
}

func getChallengeStatusEmoji(status string) string {
	switch status {
	case database.ChallengeStatusActive:
		return "🏃"
	case database.ChallengeStatusCompleted:
		return "✅"
	case database.ChallengeStatusFailed:
		return "❌"
	default:
		return "❓"
	}
}

func getChallengeStatusColor(status string) int {
	switch status {
	case database.ChallengeStatusActive:
		return 0x5865F2 // Blurple
	case database.ChallengeStatusCompleted:
		return 0x00FF00 // Green
	case database.ChallengeStatusFailed:
		return 0xFF0000 // Red
	default:
		return 0x808080 // Gray
	}
}

func getParticipantStatusEmoji(status string) string {
	switch status {
	case database.ChallengeParticipantStatusActive:
		return "🏃"
	case database.ChallengeParticipantStatusPendingValidation:
		return "⏳"
	case database.ChallengeParticipantStatusCompleted:
		return "✅"
	case database.ChallengeParticipantStatusFailed:
		return "❌"
	default:
		return "❓"
	}
}
