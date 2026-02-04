package commands

import (
	"log"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

// adminPermission is the permission flag for Administrator
var adminPermission int64 = discordgo.PermissionAdministrator

// configCommand creates the /config command for server admins
func configCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:                     "config",
			Description:              "Configure Bootstrap Hub Bot settings (Admin only)",
			DefaultMemberPermissions: &adminPermission,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "welcome-channel",
					Description: "Set the channel where new members get onboarded",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:         "channel",
							Description:  "The channel to use for onboarding",
							Type:         discordgo.ApplicationCommandOptionChannel,
							Required:     true,
							ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
						},
					},
				},
				{
					Name:        "view",
					Description: "View current bot configuration",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: handleConfigCommand,
	}
}

func handleConfigCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		respondWithError(s, i, "Invalid command usage")
		return
	}

	switch options[0].Name {
	case "welcome-channel":
		channel := options[0].Options[0].ChannelValue(s)
		handleSetWelcomeChannel(s, i, channel)
	case "view":
		handleConfigView(s, i)
	}
}

func handleSetWelcomeChannel(s *discordgo.Session, i *discordgo.InteractionCreate, channel *discordgo.Channel) {
	err := database.SetWelcomeChannel(i.GuildID, channel.ID)
	if err != nil {
		log.Printf("Error setting welcome channel: %v", err)
		respondWithError(s, i, "Failed to save the welcome channel setting.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Welcome Channel Set",
		Description: "The onboarding welcome channel has been configured.",
		Color:       0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Channel",
				Value:  "<#" + channel.ID + ">",
				Inline: true,
			},
			{
				Name:   "What happens next",
				Value:  "When new members join, they'll receive the **anon** role and a welcome message with an onboarding button will be posted in this channel.",
				Inline: false,
			},
		},
	}
	respondWithEmbed(s, i, embed)
}

func handleConfigView(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cfg, err := database.GetGuildConfig(i.GuildID)
	if err != nil {
		log.Printf("Error fetching guild config: %v", err)
		respondWithError(s, i, "Failed to fetch configuration.")
		return
	}

	welcomeChannel := "*Not set*"
	if cfg != nil && cfg.WelcomeChannelID != "" {
		welcomeChannel = "<#" + cfg.WelcomeChannelID + ">"
	}

	embed := &discordgo.MessageEmbed{
		Title: "Bot Configuration",
		Color: 0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Welcome Channel",
				Value:  welcomeChannel,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /config welcome-channel to update settings",
		},
	}
	respondWithEmbed(s, i, embed)
}
