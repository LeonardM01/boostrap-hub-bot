package commands

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

// resourceCommand creates the /resource command with all subcommands
func resourceCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "resource",
			Description: "Manage community and private resources",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "submit",
					Description: "Submit a public resource for community voting (1-hour approval period)",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "url",
							Description: "Resource URL (must be valid http/https link)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "title",
							Description: "Resource title",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "description",
							Description: "Brief description of the resource",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
						{
							Name:        "category",
							Description: "Resource category (e.g., marketing, development, design)",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
						{
							Name:        "tags",
							Description: "Comma-separated tags for searchability",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
					},
				},
				{
					Name:        "list",
					Description: "List approved public resources",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "category",
							Description: "Filter by category",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
						{
							Name:        "search",
							Description: "Search in title, description, or tags",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    false,
						},
					},
				},
				{
					Name:        "private",
					Description: "Manage private project resources",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "add",
							Description: "Add a private resource visible to specific project roles",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "url",
									Description: "Resource URL",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
								{
									Name:        "title",
									Description: "Resource title",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
								{
									Name:        "roles",
									Description: "Project roles that can access (mention them, e.g., @project-globi)",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    true,
								},
								{
									Name:        "description",
									Description: "Brief description",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    false,
								},
								{
									Name:        "category",
									Description: "Resource category",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    false,
								},
								{
									Name:        "tags",
									Description: "Comma-separated tags",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    false,
								},
							},
						},
						{
							Name:        "list",
							Description: "List private resources you have access to",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "category",
									Description: "Filter by category",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    false,
								},
								{
									Name:        "search",
									Description: "Search in title, description, or tags",
									Type:        discordgo.ApplicationCommandOptionString,
									Required:    false,
								},
							},
						},
						{
							Name:        "remove",
							Description: "Remove a private resource you own",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "id",
									Description: "Resource ID (from /resource private list)",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
					},
				},
			},
		},
		Handler: handleResourceCommand,
	}
}

// handleResourceCommand routes to the appropriate subcommand handler
func handleResourceCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options

	if len(options) == 0 {
		respondError(s, i, "No subcommand provided")
		return
	}

	switch options[0].Name {
	case "submit":
		handleResourceSubmit(s, i, options[0].Options)
	case "list":
		handleResourceList(s, i, options[0].Options)
	case "private":
		handleResourcePrivate(s, i, options[0].Options)
	default:
		respondError(s, i, "Unknown subcommand")
	}
}

// handleResourceSubmit handles /resource submit
func handleResourceSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	// Extract parameters
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	urlStr := optionMap["url"].StringValue()
	title := optionMap["title"].StringValue()
	description := ""
	category := ""
	tags := ""

	if opt, ok := optionMap["description"]; ok {
		description = opt.StringValue()
	}
	if opt, ok := optionMap["category"]; ok {
		category = opt.StringValue()
	}
	if opt, ok := optionMap["tags"]; ok {
		tags = opt.StringValue()
	}

	// Validate URL
	if !isValidURL(urlStr) {
		respondError(s, i, "Invalid URL. Please provide a valid http:// or https:// link.")
		return
	}

	// Get guild and user info
	guildID := i.GuildID
	userID := i.Member.User.ID
	username := i.Member.User.Username

	// Check for duplicate URL
	duplicate, err := database.CheckDuplicateURL(guildID, urlStr)
	if err != nil {
		log.Printf("Error checking duplicate URL: %v", err)
		respondError(s, i, "Failed to check for duplicates. Please try again.")
		return
	}
	if duplicate != nil {
		respondError(s, i, fmt.Sprintf("⚠️ This URL is already approved! Check it out in `/resource list`\n\n**%s**\n%s", duplicate.Title, duplicate.URL))
		return
	}

	// Create the resource
	resource, err := database.CreatePublicResource(guildID, userID, username, urlStr, title, description, category, tags)
	if err != nil {
		log.Printf("Error creating resource: %v", err)
		respondError(s, i, "Failed to create resource. Please try again.")
		return
	}

	// Create voting embed
	voteEmbed := &discordgo.MessageEmbed{
		Title:       "📚 New Resource Submitted for Voting",
		Description: fmt.Sprintf("**%s**\n\nSubmitted by <@%s>", title, userID),
		Color:       0xFFA500, // Orange for pending
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🔗 URL",
				Value:  urlStr,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Vote with 👍 (useful) or 👎 (not useful) • Voting ends in 1 hour",
		},
		Timestamp: time.Now().Add(time.Hour).Format(time.RFC3339),
	}

	if description != "" {
		voteEmbed.Fields = append(voteEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   "📝 Description",
			Value:  description,
			Inline: false,
		})
	}
	if category != "" {
		voteEmbed.Fields = append(voteEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   "📂 Category",
			Value:  category,
			Inline: true,
		})
	}
	if tags != "" {
		voteEmbed.Fields = append(voteEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   "🏷️ Tags",
			Value:  tags,
			Inline: true,
		})
	}

	// Respond with confirmation first
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Resource submitted! Posting voting message...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
		return
	}

	// Post voting message in the same channel
	voteMsg, err := s.ChannelMessageSendEmbed(i.ChannelID, voteEmbed)
	if err != nil {
		log.Printf("Error posting vote message: %v", err)
		return
	}

	// Add reaction options
	s.MessageReactionAdd(i.ChannelID, voteMsg.ID, "👍")
	s.MessageReactionAdd(i.ChannelID, voteMsg.ID, "👎")

	// Update resource with vote message details
	expiresAt := time.Now().Add(time.Hour)
	err = database.UpdateResourceVoteMessage(resource.ID, voteMsg.ID, i.ChannelID, expiresAt)
	if err != nil {
		log.Printf("Error updating vote message: %v", err)
	}
}

// handleResourceList handles /resource list
func handleResourceList(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	// Extract parameters
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	category := ""
	search := ""

	if opt, ok := optionMap["category"]; ok {
		category = opt.StringValue()
	}
	if opt, ok := optionMap["search"]; ok {
		search = opt.StringValue()
	}

	// Get approved resources
	resources, err := database.GetApprovedPublicResources(i.GuildID, category, search)
	if err != nil {
		log.Printf("Error fetching resources: %v", err)
		respondError(s, i, "Failed to fetch resources. Please try again.")
		return
	}

	if len(resources) == 0 {
		respondError(s, i, "No approved resources found. Be the first to submit one with `/resource submit`!")
		return
	}

	// Build embed
	embed := &discordgo.MessageEmbed{
		Title:       "📚 Approved Community Resources",
		Description: fmt.Sprintf("Found %d resource(s)", len(resources)),
		Color:       0x00FF00, // Green for approved
		Fields:      []*discordgo.MessageEmbedField{},
	}

	// Limit to first 10 resources to avoid hitting embed limits
	displayCount := len(resources)
	if displayCount > 10 {
		displayCount = 10
	}

	for idx, resource := range resources[:displayCount] {
		fieldValue := fmt.Sprintf("🔗 [Link](%s)\n", resource.URL)
		if resource.Description != "" {
			fieldValue += fmt.Sprintf("%s\n", resource.Description)
		}
		if resource.Category != "" {
			fieldValue += fmt.Sprintf("📂 Category: %s\n", resource.Category)
		}
		if resource.Tags != "" {
			fieldValue += fmt.Sprintf("🏷️ Tags: %s\n", resource.Tags)
		}
		fieldValue += fmt.Sprintf("👤 By: %s • 👍 %d", resource.SubmitterUsername, resource.UsefulVotes)

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%d. %s", idx+1, resource.Title),
			Value:  fieldValue,
			Inline: false,
		})
	}

	if len(resources) > 10 {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Showing first 10 of %d resources. Use filters to narrow down results.", len(resources)),
		}
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

// handleResourcePrivate routes private resource subcommands
func handleResourcePrivate(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(options) == 0 {
		respondError(s, i, "No subcommand provided")
		return
	}

	switch options[0].Name {
	case "add":
		handleResourcePrivateAdd(s, i, options[0].Options)
	case "list":
		handleResourcePrivateList(s, i, options[0].Options)
	case "remove":
		handleResourcePrivateRemove(s, i, options[0].Options)
	default:
		respondError(s, i, "Unknown private subcommand")
	}
}

// handleResourcePrivateAdd handles /resource private add
func handleResourcePrivateAdd(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	// Extract parameters
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	urlStr := optionMap["url"].StringValue()
	title := optionMap["title"].StringValue()
	rolesStr := optionMap["roles"].StringValue()
	description := ""
	category := ""
	tags := ""

	if opt, ok := optionMap["description"]; ok {
		description = opt.StringValue()
	}
	if opt, ok := optionMap["category"]; ok {
		category = opt.StringValue()
	}
	if opt, ok := optionMap["tags"]; ok {
		tags = opt.StringValue()
	}

	// Validate URL
	if !isValidURL(urlStr) {
		respondError(s, i, "Invalid URL. Please provide a valid http:// or https:// link.")
		return
	}

	// Extract role IDs from mentions
	roleIDs := extractRoleIDs(rolesStr)
	if len(roleIDs) == 0 {
		respondError(s, i, "No valid roles provided. Please mention at least one role (e.g., @project-globi).")
		return
	}

	// Validate roles
	guildRoles, err := s.GuildRoles(i.GuildID)
	if err != nil {
		log.Printf("Error fetching guild roles: %v", err)
		respondError(s, i, "Failed to fetch server roles. Please try again.")
		return
	}

	// Create role map
	roleMap := make(map[string]*discordgo.Role)
	for _, role := range guildRoles {
		roleMap[role.ID] = role
	}

	// Validate user has roles and they're project roles
	memberRoleMap := make(map[string]bool)
	for _, roleID := range i.Member.Roles {
		memberRoleMap[roleID] = true
	}

	var validatedRoles []struct {
		RoleID   string
		RoleName string
	}

	for _, roleID := range roleIDs {
		// Check if role exists
		role, exists := roleMap[roleID]
		if !exists {
			respondError(s, i, fmt.Sprintf("Role <@&%s> doesn't exist in this server.", roleID))
			return
		}

		// Check if user has the role
		if !memberRoleMap[roleID] {
			respondError(s, i, fmt.Sprintf("You don't have the role @%s. You can only share resources with roles you belong to.", role.Name))
			return
		}

		// Check if it's a project role
		if !strings.HasPrefix(role.Name, "project-") {
			respondError(s, i, fmt.Sprintf("Role @%s is not a project role. Only roles starting with 'project-' can be used (e.g., project-globi).", role.Name))
			return
		}

		validatedRoles = append(validatedRoles, struct {
			RoleID   string
			RoleName string
		}{
			RoleID:   roleID,
			RoleName: role.Name,
		})
	}

	// Create the private resource
	userID := i.Member.User.ID
	username := i.Member.User.Username

	resource, err := database.CreatePrivateResourceWithRoles(i.GuildID, userID, username, urlStr, title, description, category, tags, validatedRoles)
	if err != nil {
		log.Printf("Error creating private resource: %v", err)
		respondError(s, i, "Failed to create private resource. Please try again.")
		return
	}

	// Build role mentions for confirmation
	var roleMentions []string
	for _, role := range validatedRoles {
		roleMentions = append(roleMentions, fmt.Sprintf("<@&%s>", role.RoleID))
	}

	// Send confirmation
	embed := &discordgo.MessageEmbed{
		Title:       "✅ Private Resource Added",
		Description: fmt.Sprintf("**%s**\n\nResource ID: `%d`", title, resource.ID),
		Color:       0x5865F2, // Discord blurple
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🔗 URL",
				Value:  urlStr,
				Inline: false,
			},
			{
				Name:   "👥 Accessible to",
				Value:  strings.Join(roleMentions, ", "),
				Inline: false,
			},
		},
	}

	if description != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📝 Description",
			Value:  description,
			Inline: false,
		})
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

// handleResourcePrivateList handles /resource private list
func handleResourcePrivateList(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	// Extract parameters
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	category := ""
	search := ""

	if opt, ok := optionMap["category"]; ok {
		category = opt.StringValue()
	}
	if opt, ok := optionMap["search"]; ok {
		search = opt.StringValue()
	}

	// Get user's role IDs
	userRoleIDs := i.Member.Roles

	// Get accessible resources
	resources, err := database.GetPrivateResourcesForUser(i.GuildID, userRoleIDs, category, search)
	if err != nil {
		log.Printf("Error fetching private resources: %v", err)
		respondError(s, i, "Failed to fetch resources. Please try again.")
		return
	}

	if len(resources) == 0 {
		respondError(s, i, "No private resources found. Create one with `/resource private add`!")
		return
	}

	// Build embed
	embed := &discordgo.MessageEmbed{
		Title:       "🔒 Your Private Resources",
		Description: fmt.Sprintf("Found %d resource(s) you have access to", len(resources)),
		Color:       0x5865F2, // Discord blurple
		Fields:      []*discordgo.MessageEmbedField{},
	}

	// Limit to first 10 resources
	displayCount := len(resources)
	if displayCount > 10 {
		displayCount = 10
	}

	for _, resource := range resources[:displayCount] {
		fieldValue := fmt.Sprintf("🔗 [Link](%s)\n", resource.URL)
		if resource.Description != "" {
			fieldValue += fmt.Sprintf("%s\n", resource.Description)
		}
		if resource.Category != "" {
			fieldValue += fmt.Sprintf("📂 Category: %s\n", resource.Category)
		}
		if resource.Tags != "" {
			fieldValue += fmt.Sprintf("🏷️ Tags: %s\n", resource.Tags)
		}

		// List allowed roles
		var roleNames []string
		for _, role := range resource.AllowedRoles {
			roleNames = append(roleNames, role.RoleName)
		}
		fieldValue += fmt.Sprintf("👥 Roles: %s\n", strings.Join(roleNames, ", "))
		fieldValue += fmt.Sprintf("👤 By: %s • ID: `%d`", resource.OwnerUsername, resource.ID)

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   resource.Title,
			Value:  fieldValue,
			Inline: false,
		})
	}

	if len(resources) > 10 {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Showing first 10 of %d resources. Use filters to narrow down results.", len(resources)),
		}
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:   discordgo.MessageFlagsEphemeral, // Only visible to the user
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

// handleResourcePrivateRemove handles /resource private remove
func handleResourcePrivateRemove(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	// Extract parameters
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	resourceID := uint(optionMap["id"].IntValue())
	userID := i.Member.User.ID

	// Delete the resource
	err := database.DeletePrivateResource(resourceID, userID)
	if err != nil {
		log.Printf("Error deleting resource: %v", err)
		respondError(s, i, fmt.Sprintf("Failed to delete resource: %s", err.Error()))
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Private resource `%d` has been deleted.", resourceID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

// ==================== Helper Functions ====================

// isValidURL validates if a string is a valid HTTP/HTTPS URL
func isValidURL(str string) bool {
	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// extractRoleIDs extracts Discord role IDs from a string with mentions
func extractRoleIDs(str string) []string {
	// Discord role mention format: <@&ROLE_ID>
	re := regexp.MustCompile(`<@&(\d+)>`)
	matches := re.FindAllStringSubmatch(str, -1)

	var roleIDs []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			roleID := match[1]
			if !seen[roleID] {
				roleIDs = append(roleIDs, roleID)
				seen[roleID] = true
			}
		}
	}

	return roleIDs
}

// respondError sends an error message to the user
func respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error sending error response: %v", err)
	}
}
