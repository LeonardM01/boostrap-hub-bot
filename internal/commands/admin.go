package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

// configCommand creates the /config command for admins
func configCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "config",
			Description: "Configure bot settings (admin only)",
			DefaultMemberPermissions: func() *int64 {
				perms := int64(discordgo.PermissionAdministrator)
				return &perms
			}(),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "leaderboard-channel",
					Description: "Set the channel for automated leaderboard posts",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "channel",
							Description: "The channel to post leaderboards in",
							Type:        discordgo.ApplicationCommandOptionChannel,
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Name:        "wins-channel",
					Description: "Set the channel for win celebrations",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "channel",
							Description: "The channel to post wins in",
							Type:        discordgo.ApplicationCommandOptionChannel,
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
				{
					Name:        "mrr-channel",
					Description: "Set the channel for MRR milestone announcements",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "channel",
							Description: "The channel to post MRR milestones in",
							Type:        discordgo.ApplicationCommandOptionChannel,
							Required:    true,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
						},
					},
				},
			},
		},
		Handler: handleConfigCommand,
	}
}

// hasAdminRole checks if the user has one of the required admin roles
func hasAdminRole(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	// Ensure we're in a guild context
	if i.GuildID == "" || i.Member == nil {
		return false
	}

	// Get all roles in the guild
	guild, err := s.State.Guild(i.GuildID)
	if err != nil {
		// Try fetching from API if not in state
		guild, err = s.Guild(i.GuildID)
		if err != nil {
			log.Printf("Error fetching guild: %v", err)
			return false
		}
	}

	// Build a map of role IDs to role names for quick lookup
	roleMap := make(map[string]string)
	for _, role := range guild.Roles {
		roleMap[role.ID] = strings.ToLower(role.Name)
	}

	// Check if user has any of the required roles
	requiredRoles := []string{"admin", "moderator", "mod"}
	for _, roleID := range i.Member.Roles {
		if roleName, exists := roleMap[roleID]; exists {
			for _, required := range requiredRoles {
				if roleName == required {
					return true
				}
			}
		}
	}

	return false
}

func handleConfigCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check for required role
	if !hasAdminRole(s, i) {
		embed := &discordgo.MessageEmbed{
			Title:       "Permission Denied",
			Description: "You need one of these roles to use this command: Admin, Moderator, or Mod",
			Color:       0xFF0000, // Red
		}
		respondWithEmbedEphemeral(s, i, embed, true)
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		respondWithError(s, i, "Invalid command usage")
		return
	}

	subCommand := options[0].Name
	guildID := i.GuildID
	if guildID == "" {
		respondWithError(s, i, "This command can only be used in a server.")
		return
	}

	switch subCommand {
	case "leaderboard-channel":
		channelID := options[0].Options[0].ChannelValue(s).ID
		handleConfigLeaderboardChannel(s, i, guildID, channelID)
	case "wins-channel":
		channelID := options[0].Options[0].ChannelValue(s).ID
		handleConfigWinsChannel(s, i, guildID, channelID)
	case "mrr-channel":
		channelID := options[0].Options[0].ChannelValue(s).ID
		handleConfigMRRChannel(s, i, guildID, channelID)
	default:
		respondWithError(s, i, "Unknown subcommand")
	}
}

func handleConfigLeaderboardChannel(s *discordgo.Session, i *discordgo.InteractionCreate, guildID, channelID string) {
	err := database.UpdateLeaderboardChannel(guildID, channelID)
	if err != nil {
		log.Printf("Error updating leaderboard channel: %v", err)
		respondWithError(s, i, "Failed to update leaderboard channel.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Configuration Updated",
		Description: fmt.Sprintf("Leaderboard channel set to <#%s>\n\nBi-weekly leaderboards will now be automatically posted to this channel when Focus Periods end.", channelID),
		Color:       0x00FF00, // Green
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Bootstrap Hub Bot Configuration",
		},
	}

	respondWithEmbedEphemeral(s, i, embed, true)
}

func handleConfigWinsChannel(s *discordgo.Session, i *discordgo.InteractionCreate, guildID, channelID string) {
	err := database.UpdateWinsChannel(guildID, channelID)
	if err != nil {
		log.Printf("Error updating wins channel: %v", err)
		respondWithError(s, i, "Failed to update wins channel.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Configuration Updated",
		Description: fmt.Sprintf("Wins channel set to <#%s>\n\nWin celebrations will now be automatically posted to this channel when users share wins.", channelID),
		Color:       0x00FF00, // Green
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Bootstrap Hub Bot Configuration",
		},
	}

	respondWithEmbedEphemeral(s, i, embed, true)
}

func handleConfigMRRChannel(s *discordgo.Session, i *discordgo.InteractionCreate, guildID, channelID string) {
	err := database.UpdateMRRChannel(guildID, channelID)
	if err != nil {
		log.Printf("Error updating MRR channel: %v", err)
		respondWithError(s, i, "Failed to update MRR channel.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Configuration Updated",
		Description: fmt.Sprintf("MRR channel set to <#%s>\n\nMRR milestone celebrations will now be automatically posted to this channel.", channelID),
		Color:       0x00FF00, // Green
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Bootstrap Hub Bot Configuration",
		},
	}

	respondWithEmbedEphemeral(s, i, embed, true)
}
