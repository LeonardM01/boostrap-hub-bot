package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

// leaderboardCommand creates the /leaderboard command
func leaderboardCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "leaderboard",
			Description: "View the leaderboard rankings",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "alltime",
					Description: "View all-time leaderboard rankings",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "sprint",
					Description: "View current sprint leaderboard rankings",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: handleLeaderboardCommand,
	}
}

func handleLeaderboardCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		respondWithError(s, i, "Invalid command usage")
		return
	}

	subCommand := options[0].Name
	guildID := i.GuildID
	if guildID == "" {
		guildID = "DM"
	}

	switch subCommand {
	case "alltime":
		handleLeaderboardAllTime(s, i, guildID)
	case "sprint":
		handleLeaderboardSprint(s, i, guildID)
	default:
		respondWithError(s, i, "Unknown subcommand")
	}
}

func handleLeaderboardAllTime(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) {
	entries, err := database.GetAllTimeLeaderboard(guildID, 10)
	if err != nil {
		log.Printf("Error fetching all-time leaderboard: %v", err)
		respondWithError(s, i, "Failed to fetch leaderboard.")
		return
	}

	if len(entries) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "All-Time Leaderboard",
			Description: "No one has earned points yet!\n\nStart a Focus Period and complete tasks to earn points and climb the leaderboard.",
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	var leaderboardText strings.Builder
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

		leaderboardText.WriteString(fmt.Sprintf("%s **%s** - %d points (%d tasks)\n",
			medal, entry.Username, entry.Points, entry.TasksCount))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🏆 All-Time Leaderboard",
		Description: leaderboardText.String(),
		Color:       0xFFD700, // Gold
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Keep crushing those goals to climb the ranks!",
		},
	}

	respondWithEmbed(s, i, embed)
}

func handleLeaderboardSprint(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) {
	entries, err := database.GetSprintLeaderboard(guildID, 10)
	if err != nil {
		log.Printf("Error fetching sprint leaderboard: %v", err)
		respondWithError(s, i, "Failed to fetch leaderboard.")
		return
	}

	if len(entries) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "Current Sprint Leaderboard",
			Description: "No one has earned points in this sprint yet!\n\nComplete tasks during your active Focus Period to earn points.",
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	var leaderboardText strings.Builder
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

		leaderboardText.WriteString(fmt.Sprintf("%s **%s** - %d points (%d tasks)\n",
			medal, entry.Username, entry.Points, entry.TasksCount))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🏃 Current Sprint Leaderboard",
		Description: leaderboardText.String(),
		Color:       0x5865F2, // Blurple
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Showing rankings for active Focus Periods",
		},
	}

	respondWithEmbed(s, i, embed)
}
