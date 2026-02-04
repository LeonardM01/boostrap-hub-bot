package commands

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

const (
	// AnonRoleName is the role given to new members before onboarding
	AnonRoleName = "anon"
	// OnboardingButtonID is the custom ID for the onboarding button
	OnboardingButtonID = "onboarding_start"
	// OnboardingModalID is the custom ID for the onboarding modal
	OnboardingModalID = "onboarding_form"
)

// HandleMemberJoin is called when a new member joins the server.
// It assigns the "anon" role and sends a welcome message in the configured channel.
func HandleMemberJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	// Don't process bots
	if m.User.Bot {
		return
	}

	log.Printf("New member joined: %s in guild %s", m.User.Username, m.GuildID)

	// Assign "anon" role
	roleID, err := ensureAnonRole(s, m.GuildID)
	if err != nil {
		log.Printf("Error ensuring anon role in guild %s: %v", m.GuildID, err)
	} else {
		err = s.GuildMemberRoleAdd(m.GuildID, m.User.ID, roleID)
		if err != nil {
			log.Printf("Error assigning anon role to %s: %v", m.User.Username, err)
		} else {
			log.Printf("Assigned 'anon' role to %s", m.User.Username)
		}
	}

	// Ensure user exists in DB
	_, _ = database.GetOrCreateUser(m.User.ID, m.GuildID, m.User.Username)

	// Check if guild has a welcome channel configured
	cfg, err := database.GetGuildConfig(m.GuildID)
	if err != nil || cfg == nil || cfg.WelcomeChannelID == "" {
		log.Printf("No welcome channel configured for guild %s", m.GuildID)
		return
	}

	// Send welcome message with onboarding button
	sendWelcomeMessage(s, cfg.WelcomeChannelID, m.User)
}

// ensureAnonRole finds or creates the "anon" role in the guild
func ensureAnonRole(s *discordgo.Session, guildID string) (string, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch guild roles: %w", err)
	}

	// Look for existing "anon" role
	for _, role := range roles {
		if strings.EqualFold(role.Name, AnonRoleName) {
			return role.ID, nil
		}
	}

	// Create the role if it doesn't exist
	role, err := s.GuildRoleCreate(guildID, &discordgo.RoleParams{
		Name:  AnonRoleName,
		Color: intPtr(0x95A5A6), // Grey color
		Hoist: boolPtr(false),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create anon role: %w", err)
	}

	log.Printf("Created 'anon' role in guild %s", guildID)
	return role.ID, nil
}

func intPtr(i int) *int       { return &i }
func boolPtr(b bool) *bool    { return &b }

// sendWelcomeMessage sends the onboarding message to the welcome channel
func sendWelcomeMessage(s *discordgo.Session, channelID string, user *discordgo.User) {
	embed := &discordgo.MessageEmbed{
		Title:       "Welcome to Bootstrap Hub!",
		Description: fmt.Sprintf("Hey <@%s>, welcome to the community of solo founders!\n\nTo get started and unlock full access, complete the onboarding by clicking the button below. We'll set up your project space on the server.", user.ID),
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "What you'll need",
				Value:  "1. Your project name\n2. Your project website (if any)\n3. Your project category",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Bootstrap Hub - Built for Founders, by Founders",
		},
	}

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Start Onboarding",
						Style:    discordgo.PrimaryButton,
						CustomID: OnboardingButtonID,
						Emoji: &discordgo.ComponentEmoji{
							Name: "🚀",
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Error sending welcome message: %v", err)
	}
}

// HandleOnboardingButton handles the "Start Onboarding" button click.
// It opens a modal form for the user to fill in their project details.
func HandleOnboardingButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if user already onboarded
	user, err := database.GetOrCreateUser(i.Member.User.ID, i.GuildID, i.Member.User.Username)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if user.Onboarded {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You've already completed onboarding!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Open the onboarding modal
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: OnboardingModalID,
			Title:    "Project Onboarding",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "project_name",
							Label:       "Project Name",
							Style:       discordgo.TextInputShort,
							Placeholder: "e.g. My Awesome SaaS",
							Required:    true,
							MinLength:   2,
							MaxLength:   50,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "project_website",
							Label:       "Project Website",
							Style:       discordgo.TextInputShort,
							Placeholder: "e.g. https://myproject.com (or N/A)",
							Required:    false,
							MaxLength:   200,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "project_category",
							Label:       "Project Category",
							Style:       discordgo.TextInputShort,
							Placeholder: "e.g. SaaS, E-commerce, Agency, Mobile App, Dev Tools",
							Required:    true,
							MinLength:   2,
							MaxLength:   50,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Error opening onboarding modal: %v", err)
	}
}

// HandleOnboardingModalSubmit processes the submitted onboarding form.
// It creates a project role, server category, and text channel.
func HandleOnboardingModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	// Extract form values
	var projectName, projectWebsite, projectCategory string
	for _, row := range data.Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			input := comp.(*discordgo.TextInput)
			switch input.CustomID {
			case "project_name":
				projectName = input.Value
			case "project_website":
				projectWebsite = input.Value
			case "project_category":
				projectCategory = input.Value
			}
		}
	}

	// Acknowledge immediately - project setup takes a moment
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error deferring modal response: %v", err)
		return
	}

	guildID := i.GuildID
	userDiscordID := i.Member.User.ID
	username := i.Member.User.Username

	// Get or create user
	user, err := database.GetOrCreateUser(userDiscordID, guildID, username)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		editDeferredError(s, i, "Something went wrong. Please try again.")
		return
	}

	// Check if already onboarded
	if user.Onboarded {
		editDeferredError(s, i, "You've already completed onboarding!")
		return
	}

	// 1. Create a role for the project
	safeName := sanitizeName(projectName)
	role, err := s.GuildRoleCreate(guildID, &discordgo.RoleParams{
		Name:  safeName,
		Color: intPtr(0x3498DB), // Blue
		Hoist: boolPtr(false),
	})
	if err != nil {
		log.Printf("Error creating project role: %v", err)
		editDeferredError(s, i, "Failed to create project role. Please contact an admin.")
		return
	}

	// 2. Create a server category for the project
	category, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name: safeName,
		Type: discordgo.ChannelTypeGuildCategory,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{
				// Deny @everyone from viewing
				ID:   guildID,
				Type: discordgo.PermissionOverwriteTypeRole,
				Deny: discordgo.PermissionViewChannel,
			},
			{
				// Allow the project role to view
				ID:    role.ID,
				Type:  discordgo.PermissionOverwriteTypeRole,
				Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory,
			},
		},
	})
	if err != nil {
		log.Printf("Error creating project category: %v", err)
		// Clean up: remove the role we just created
		_ = s.GuildRoleDelete(guildID, role.ID)
		editDeferredError(s, i, "Failed to create project category. Please contact an admin.")
		return
	}

	// 3. Create a text channel under the category
	channel, err := s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:     "general",
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: category.ID,
		Topic:    fmt.Sprintf("%s - %s | %s", projectName, projectCategory, projectWebsite),
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{
				ID:   guildID,
				Type: discordgo.PermissionOverwriteTypeRole,
				Deny: discordgo.PermissionViewChannel,
			},
			{
				ID:    role.ID,
				Type:  discordgo.PermissionOverwriteTypeRole,
				Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionReadMessageHistory,
			},
		},
	})
	if err != nil {
		log.Printf("Error creating project channel: %v", err)
		// Clean up
		_, _ = s.ChannelDelete(category.ID)
		_ = s.GuildRoleDelete(guildID, role.ID)
		editDeferredError(s, i, "Failed to create project channel. Please contact an admin.")
		return
	}

	// 4. Assign project role to user
	err = s.GuildMemberRoleAdd(guildID, userDiscordID, role.ID)
	if err != nil {
		log.Printf("Error assigning project role: %v", err)
	}

	// 5. Remove "anon" role from user
	removeAnonRole(s, guildID, userDiscordID)

	// 6. Save to database
	_, err = database.CreateProject(guildID, user.ID, projectName, projectWebsite, projectCategory, role.ID, category.ID, channel.ID)
	if err != nil {
		log.Printf("Error saving project to database: %v", err)
	}

	err = database.MarkUserOnboarded(user.ID)
	if err != nil {
		log.Printf("Error marking user onboarded: %v", err)
	}

	// 7. Send a welcome message in the new project channel
	projectEmbed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Welcome to %s!", projectName),
		Description: fmt.Sprintf("<@%s>, your project space is all set up. This is your dedicated channel to track progress, share updates, and get feedback from the community.", userDiscordID),
		Color:       0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Project",
				Value:  projectName,
				Inline: true,
			},
			{
				Name:   "Category",
				Value:  projectCategory,
				Inline: true,
			},
			{
				Name:   "Website",
				Value:  formatWebsite(projectWebsite),
				Inline: true,
			},
			{
				Name:   "Next Steps",
				Value:  "1. Use `/focus start` to begin your first 2-week Focus Period\n2. Share your goals and progress here\n3. Connect with other founders in the community!",
				Inline: false,
			},
		},
	}
	_, _ = s.ChannelMessageSendEmbed(channel.ID, projectEmbed)

	// 8. Edit the deferred response to confirm
	embed := &discordgo.MessageEmbed{
		Title:       "Onboarding Complete!",
		Description: fmt.Sprintf("Your project **%s** has been set up.", projectName),
		Color:       0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Your Project Channel",
				Value:  fmt.Sprintf("<#%s>", channel.ID),
				Inline: true,
			},
			{
				Name:   "Your Project Role",
				Value:  fmt.Sprintf("<@&%s>", role.ID),
				Inline: true,
			},
		},
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		log.Printf("Error editing deferred response: %v", err)
	}
}

// removeAnonRole removes the "anon" role from a user
func removeAnonRole(s *discordgo.Session, guildID, userID string) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return
	}

	for _, role := range roles {
		if strings.EqualFold(role.Name, AnonRoleName) {
			err = s.GuildMemberRoleRemove(guildID, userID, role.ID)
			if err != nil {
				log.Printf("Error removing anon role from user %s: %v", userID, err)
			}
			return
		}
	}
}

// sanitizeName creates a safe Discord channel/role name from a project name
func sanitizeName(name string) string {
	// Lowercase and trim
	safe := strings.ToLower(strings.TrimSpace(name))
	// Replace spaces and special chars with hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	safe = reg.ReplaceAllString(safe, "-")
	// Remove leading/trailing hyphens
	safe = strings.Trim(safe, "-")
	// Limit length
	if len(safe) > 50 {
		safe = safe[:50]
	}
	if safe == "" {
		safe = "project"
	}
	return safe
}

// formatWebsite formats the website for display
func formatWebsite(website string) string {
	if website == "" || strings.EqualFold(website, "n/a") {
		return "*Not provided*"
	}
	return website
}

// editDeferredError edits a deferred response with an error message
func editDeferredError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	embed := &discordgo.MessageEmbed{
		Title:       "Error",
		Description: message,
		Color:       0xFF0000,
	}
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}
