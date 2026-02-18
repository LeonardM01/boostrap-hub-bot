package commands

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

func projectCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "project",
			Description: "Manage channels in your project category",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "admin",
					Description: "Admin commands for project mapping management",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "setup",
							Description: "Map a role to a category channel for project management",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "role",
									Description: "The project role to map",
									Type:        discordgo.ApplicationCommandOptionRole,
									Required:    true,
								},
								{
									Name:        "category",
									Description: "The category channel for this project",
									Type:        discordgo.ApplicationCommandOptionChannel,
									Required:    true,
									ChannelTypes: []discordgo.ChannelType{
										discordgo.ChannelTypeGuildCategory,
									},
								},
								{
									Name:        "max-channels",
									Description: "Maximum channels a user can create (default: 5)",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    false,
									MinValue:    func() *float64 { v := 1.0; return &v }(),
									MaxValue:    50,
								},
							},
						},
						{
							Name:        "remove-mapping",
							Description: "Remove a role-to-category mapping",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "role",
									Description: "The role to remove the mapping for",
									Type:        discordgo.ApplicationCommandOptionRole,
									Required:    true,
								},
							},
						},
						{
							Name:        "list-mappings",
							Description: "Show all project role-to-category mappings",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
					},
				},
				{
					Name:        "create-channel",
					Description: "Create a channel in your project category",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "name",
							Description: "Channel name",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "type",
							Description: "Type of channel to create",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "Text Channel", Value: "text"},
								{Name: "Voice Channel", Value: "voice"},
								{Name: "Thread", Value: "thread"},
							},
						},
						{
							Name:        "parent-channel",
							Description: "Parent text channel for threads (required for Thread type)",
							Type:        discordgo.ApplicationCommandOptionChannel,
							Required:    false,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Name:        "list-channels",
					Description: "Show channels you created in your project category",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: handleProjectCommand,
	}
}

func handleProjectCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" || i.Member == nil {
		respondWithError(s, i, "This command can only be used in a server.")
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		respondWithError(s, i, "No subcommand provided.")
		return
	}

	switch options[0].Name {
	case "admin":
		handleProjectAdmin(s, i, options[0].Options)
	case "create-channel":
		handleProjectCreateChannel(s, i, options[0].Options)
	case "list-channels":
		handleProjectListChannels(s, i)
	default:
		respondWithError(s, i, "Unknown subcommand.")
	}
}

// ==================== Admin Handlers ====================

func handleProjectAdmin(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if !hasAdminRole(s, i) {
		embed := &discordgo.MessageEmbed{
			Title:       "Permission Denied",
			Description: "You need one of these roles to use this command: Admin, Moderator, or Mod",
			Color:       0xFF0000,
		}
		respondWithEmbedEphemeral(s, i, embed, true)
		return
	}

	if len(options) == 0 {
		respondWithError(s, i, "No admin subcommand provided.")
		return
	}

	switch options[0].Name {
	case "setup":
		handleProjectAdminSetup(s, i, options[0].Options)
	case "remove-mapping":
		handleProjectAdminRemoveMapping(s, i, options[0].Options)
	case "list-mappings":
		handleProjectAdminListMappings(s, i)
	default:
		respondWithError(s, i, "Unknown admin subcommand.")
	}
}

func handleProjectAdminSetup(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	role := optionMap["role"].RoleValue(s, i.GuildID)
	channel := optionMap["category"].ChannelValue(s)

	maxChannels := database.DefaultMaxChannels
	if opt, ok := optionMap["max-channels"]; ok {
		maxChannels = int(opt.IntValue())
	}

	mapping, err := database.CreateProjectMapping(
		i.GuildID, role.ID, role.Name,
		channel.ID, channel.Name, maxChannels,
	)
	if err != nil {
		log.Printf("Error creating project mapping: %v", err)
		respondWithError(s, i, "Failed to create project mapping.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Project Mapping Created",
		Description: fmt.Sprintf("Role <@&%s> is now mapped to category **%s**.", role.ID, channel.Name),
		Color:       0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Max Channels per User", Value: fmt.Sprintf("%d", mapping.MaxChannels), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Users with this role can now create channels with /project create-channel"},
	}
	respondWithEmbedEphemeral(s, i, embed, true)
}

func handleProjectAdminRemoveMapping(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	role := options[0].RoleValue(s, i.GuildID)

	err := database.RemoveProjectMapping(i.GuildID, role.ID)
	if err != nil {
		log.Printf("Error removing project mapping: %v", err)
		respondWithError(s, i, fmt.Sprintf("Failed to remove mapping: %s", err.Error()))
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Mapping Removed",
		Description: fmt.Sprintf("Project mapping for role <@&%s> has been removed.", role.ID),
		Color:       0xFFA500,
	}
	respondWithEmbedEphemeral(s, i, embed, true)
}

func handleProjectAdminListMappings(s *discordgo.Session, i *discordgo.InteractionCreate) {
	mappings, err := database.GetProjectMappings(i.GuildID)
	if err != nil {
		log.Printf("Error fetching project mappings: %v", err)
		respondWithError(s, i, "Failed to fetch mappings.")
		return
	}

	if len(mappings) == 0 {
		respondWithError(s, i, "No project mappings configured. Use `/project admin setup` to create one.")
		return
	}

	var fields []*discordgo.MessageEmbedField
	for _, m := range mappings {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   m.RoleName,
			Value:  fmt.Sprintf("Role: <@&%s>\nCategory: **%s**\nMax Channels: %d", m.RoleID, m.CategoryName, m.MaxChannels),
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Project Mappings",
		Color:  0x5865F2,
		Fields: fields,
	}
	respondWithEmbedEphemeral(s, i, embed, true)
}

// ==================== User Handlers ====================

func handleProjectCreateChannel(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	name := optionMap["name"].StringValue()
	channelType := optionMap["type"].StringValue()

	// Find user's project mappings
	if len(i.Member.Roles) == 0 {
		respondWithError(s, i, "You don't have any project roles. Ask an admin to set up a mapping.")
		return
	}

	mappings, err := database.GetUserMappings(i.GuildID, i.Member.Roles)
	if err != nil {
		log.Printf("Error fetching user mappings: %v", err)
		respondWithError(s, i, "Failed to look up your project roles.")
		return
	}

	if len(mappings) == 0 {
		respondWithError(s, i, "You don't have any project roles. Ask an admin to set up a mapping.")
		return
	}

	// Disambiguate if multiple mappings
	var mapping database.ProjectMapping
	if len(mappings) == 1 {
		mapping = mappings[0]
	} else {
		// Try to resolve by the category the invoking channel belongs to
		ch, err := s.Channel(i.ChannelID)
		if err != nil {
			log.Printf("Error fetching channel: %v", err)
			respondWithError(s, i, "Failed to determine your project category.")
			return
		}

		found := false
		for _, m := range mappings {
			if ch.ParentID == m.CategoryID {
				mapping = m
				found = true
				break
			}
		}

		if !found {
			var categoryList []string
			for _, m := range mappings {
				categoryList = append(categoryList, fmt.Sprintf("- **%s** (<@&%s>)", m.CategoryName, m.RoleID))
			}
			respondWithError(s, i, fmt.Sprintf(
				"You have multiple project roles. Please run this command from within one of your project categories:\n%s",
				strings.Join(categoryList, "\n"),
			))
			return
		}
	}

	// Check channel limit
	count, err := database.CountUserChannelsInCategory(i.GuildID, i.Member.User.ID, mapping.CategoryID)
	if err != nil {
		log.Printf("Error counting user channels: %v", err)
		respondWithError(s, i, "Failed to check your channel quota.")
		return
	}

	if int(count) >= mapping.MaxChannels {
		respondWithError(s, i, fmt.Sprintf(
			"You've reached the maximum of %d channels in **%s**. Remove one or ask an admin to increase the limit.",
			mapping.MaxChannels, mapping.CategoryName,
		))
		return
	}

	// Sanitize channel name
	sanitized := sanitizeChannelName(name)
	if sanitized == "" {
		respondWithError(s, i, "Invalid channel name. Use letters, numbers, hyphens, or underscores.")
		return
	}

	var createdID string

	switch channelType {
	case "text":
		ch, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
			Name:     sanitized,
			Type:     discordgo.ChannelTypeGuildText,
			ParentID: mapping.CategoryID,
		})
		if err != nil {
			log.Printf("Error creating text channel: %v", err)
			respondWithError(s, i, "Failed to create channel. The bot may lack Manage Channels permission.")
			return
		}
		createdID = ch.ID

	case "voice":
		ch, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{
			Name:     sanitized,
			Type:     discordgo.ChannelTypeGuildVoice,
			ParentID: mapping.CategoryID,
		})
		if err != nil {
			log.Printf("Error creating voice channel: %v", err)
			respondWithError(s, i, "Failed to create channel. The bot may lack Manage Channels permission.")
			return
		}
		createdID = ch.ID

	case "thread":
		parentOpt, ok := optionMap["parent-channel"]
		if !ok {
			respondWithError(s, i, "When creating a thread, you must specify the `parent-channel` option.")
			return
		}
		parentChannel := parentOpt.ChannelValue(s)

		// Verify the parent channel is in the same category
		if parentChannel.ParentID != mapping.CategoryID {
			respondWithError(s, i, "The selected parent channel is not in your project category.")
			return
		}

		thread, err := s.ThreadStartComplex(parentChannel.ID, &discordgo.ThreadStart{
			Name:                sanitized,
			AutoArchiveDuration: 10080, // 7 days
			Type:                discordgo.ChannelTypeGuildPublicThread,
		})
		if err != nil {
			log.Printf("Error creating thread: %v", err)
			respondWithError(s, i, "Failed to create thread. The bot may lack thread permissions.")
			return
		}
		createdID = thread.ID

	default:
		respondWithError(s, i, "Invalid channel type.")
		return
	}

	// Record in DB
	_, err = database.CreateProjectChannel(
		i.GuildID, i.Member.User.ID, createdID,
		mapping.CategoryID, mapping.RoleID, sanitized, channelType,
	)
	if err != nil {
		log.Printf("Error recording project channel: %v", err)
		// Channel was created but DB record failed — not a blocker for the user
	}

	remaining := mapping.MaxChannels - int(count) - 1
	embed := &discordgo.MessageEmbed{
		Title:       "Channel Created",
		Description: fmt.Sprintf("Created %s channel <#%s> in **%s**.", channelType, createdID, mapping.CategoryName),
		Color:       0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Quota Remaining", Value: fmt.Sprintf("%d / %d", remaining, mapping.MaxChannels), Inline: true},
		},
	}
	respondWithEmbedEphemeral(s, i, embed, true)
}

func handleProjectListChannels(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if len(i.Member.Roles) == 0 {
		respondWithError(s, i, "You don't have any project roles.")
		return
	}

	mappings, err := database.GetUserMappings(i.GuildID, i.Member.Roles)
	if err != nil {
		log.Printf("Error fetching user mappings: %v", err)
		respondWithError(s, i, "Failed to look up your project roles.")
		return
	}

	if len(mappings) == 0 {
		respondWithError(s, i, "You don't have any project roles. Ask an admin to set up a mapping.")
		return
	}

	var fields []*discordgo.MessageEmbedField

	for _, m := range mappings {
		channels, err := database.GetUserChannelsInCategory(i.GuildID, i.Member.User.ID, m.CategoryID)
		if err != nil {
			log.Printf("Error fetching channels for category %s: %v", m.CategoryID, err)
			continue
		}

		count := len(channels)
		var lines []string
		for _, ch := range channels {
			lines = append(lines, fmt.Sprintf("<#%s> (%s)", ch.ChannelID, ch.Type))
		}

		value := fmt.Sprintf("Quota: **%d / %d** used\n", count, m.MaxChannels)
		if len(lines) > 0 {
			value += strings.Join(lines, "\n")
		} else {
			value += "_No channels created yet_"
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s (<@&%s>)", m.CategoryName, m.RoleID),
			Value:  value,
			Inline: false,
		})
	}

	if len(fields) == 0 {
		respondWithError(s, i, "No project data found.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Your Project Channels",
		Color:  0x5865F2,
		Fields: fields,
	}
	respondWithEmbedEphemeral(s, i, embed, true)
}

// sanitizeChannelName normalizes a name for Discord channel naming rules
func sanitizeChannelName(name string) string {
	// Lowercase
	name = strings.ToLower(name)
	// Replace spaces with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	// Strip anything that isn't alphanumeric, hyphen, or underscore
	re := regexp.MustCompile(`[^a-z0-9\-_]`)
	name = re.ReplaceAllString(name, "")
	// Collapse multiple hyphens
	re2 := regexp.MustCompile(`-{2,}`)
	name = re2.ReplaceAllString(name, "-")
	// Trim leading/trailing hyphens
	name = strings.Trim(name, "-_")
	// Cap at 100 chars
	if len(name) > 100 {
		name = name[:100]
	}
	return name
}
