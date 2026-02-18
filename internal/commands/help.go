package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// HelpCategory represents a command category for the help system
type HelpCategory struct {
	ID          string
	Name        string
	Emoji       string
	Description string
	Commands    string // Pre-formatted command list
}

var helpCategories = []HelpCategory{
	{
		ID:          "focus",
		Name:        "Goal Tracking",
		Emoji:       "\U0001F3AF", // Target emoji
		Description: "Manage your 2-week Focus Periods",
		Commands:    "`/focus start` - Start a new 2-week Focus Period\n`/focus add <goal>` - Add a goal (AI calculates points)\n`/focus complete <#>` - Mark a goal as completed\n`/focus list` - View your current goals\n`/focus status` - See your progress overview",
	},
	{
		ID:          "standup",
		Name:        "Daily Check-ins",
		Emoji:       "\U0001F4DD", // Memo emoji
		Description: "Post daily standups and track streaks",
		Commands:    "`/standup post` - Post your daily standup\n`/standup streak` - View your streak stats\n`/standup leaderboard` - View streak rankings\n`/standup history` - View recent standups",
	},
	{
		ID:          "win",
		Name:        "Wins & Celebration",
		Emoji:       "\U0001F389", // Party popper emoji
		Description: "Share and celebrate wins",
		Commands:    "`/win share <message>` - Share a win with the community\n`/win recent` - View recent community wins\n`/win stats` - View win statistics",
	},
	{
		ID:          "accountability",
		Name:        "Accountability",
		Emoji:       "\U0001F91D", // Handshake emoji
		Description: "Buddy system and challenges",
		Commands:    "**Buddy Commands**\n`/buddy request @user` - Send a buddy request\n`/buddy accept @user` - Accept a buddy request\n`/buddy decline @user` - Decline a request\n`/buddy status` - View buddy progress\n`/buddy list` - List your buddies\n`/buddy remove @user` - Remove a buddy\n\n**Challenge Commands**\n`/challenge create` - Create a challenge\n`/challenge progress` - Log your progress\n`/challenge complete` - Submit with proof\n`/challenge validate` - Validate buddy completion\n`/challenge list` - View your challenges\n`/challenge view` - View challenge details",
	},
	{
		ID:          "mrr",
		Name:        "Revenue Tracking",
		Emoji:       "\U0001F4B0", // Money bag emoji
		Description: "Track and share your MRR",
		Commands:    "`/mrr update <amount>` - Log your current MRR\n`/mrr history` - View your MRR trend\n`/mrr stats` - View your MRR statistics\n`/mrr leaderboard` - View public MRR rankings\n`/mrr public` - Make your MRR visible\n`/mrr private` - Hide your MRR\n`/mrr milestone` - View milestone progress",
	},
	{
		ID:          "leaderboard",
		Name:        "Leaderboards",
		Emoji:       "\U0001F3C6", // Trophy emoji
		Description: "View community rankings",
		Commands:    "`/leaderboard alltime` - All-time point rankings\n`/leaderboard sprint` - Current sprint rankings",
	},
	{
		ID:          "resource",
		Name:        "Resources",
		Emoji:       "\U0001F4DA", // Books emoji
		Description: "Community and private resources",
		Commands:    "**Public Resources**\n`/resource submit` - Submit a resource for voting\n`/resource list` - Browse approved resources\n\n**Private Resources**\n`/resource private add` - Add a private resource\n`/resource private list` - List resources you can access\n`/resource private remove` - Remove your private resource",
	},
	{
		ID:          "project",
		Name:        "Project Channels",
		Emoji:       "\U0001F4C1", // File folder emoji
		Description: "Manage channels in your project category",
		Commands:    "**User Commands**\n`/project create-channel` - Create a text, voice, or thread channel\n`/project list-channels` - View your channels and quota\n\n**Admin Commands**\n`/project admin setup` - Map a role to a category\n`/project admin remove-mapping` - Remove a mapping\n`/project admin list-mappings` - Show all mappings",
	},
	{
		ID:          "admin",
		Name:        "Admin",
		Emoji:       "\u2699\uFE0F", // Gear emoji
		Description: "Server configuration",
		Commands:    "`/config leaderboard-channel` - Set the leaderboard channel\n`/config wins-channel` - Set the wins announcement channel\n`/config mrr-channel` - Set the MRR milestone channel",
	},
}

// helpCommand creates the interactive help command
func helpCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "help",
			Description: "Get information about Bootstrap Hub Bot and available commands",
		},
		Handler: handleHelp,
	}
}

func handleHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := buildMainHelpEmbed()
	selectMenu := buildCategorySelectMenu()

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{selectMenu},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Error responding to help command: %v", err)
	}
}

func buildMainHelpEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "\U0001F680 Bootstrap Hub Bot",
		Description: "Your AI-powered assistant for solo founders and entrepreneurs.\n\n**Select a category below** to explore commands, or use the quick start guide to get going!",
		Color:       0x5865F2, // Discord blurple
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Quick Start",
				Value:  "`/focus start` - Begin your 2-week journey\n`/standup post` - Daily check-in\n`/win share` - Celebrate your wins",
				Inline: false,
			},
			{
				Name:   "Categories",
				Value:  "\U0001F3AF Goal Tracking \u2022 \U0001F4DD Daily Check-ins \u2022 \U0001F389 Wins\n\U0001F91D Accountability \u2022 \U0001F4B0 Revenue \u2022 \U0001F3C6 Leaderboards\n\U0001F4DA Resources \u2022 \U0001F4C1 Project Channels \u2022 \u2699\uFE0F Admin",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Tip: Type / to see all commands with Discord's autocomplete!",
		},
	}
}

func buildCategorySelectMenu() discordgo.SelectMenu {
	var options []discordgo.SelectMenuOption
	for _, cat := range helpCategories {
		options = append(options, discordgo.SelectMenuOption{
			Label:       cat.Name,
			Value:       cat.ID,
			Description: cat.Description,
			Emoji: &discordgo.ComponentEmoji{
				Name: cat.Emoji,
			},
		})
	}

	return discordgo.SelectMenu{
		CustomID:    "help_category_select",
		Placeholder: "Choose a category to explore...",
		Options:     options,
	}
}

// HandleHelpComponent handles select menu interactions for the help system
func HandleHelpComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	if data.CustomID != "help_category_select" {
		return
	}

	if len(data.Values) == 0 {
		return
	}

	categoryID := data.Values[0]
	var category *HelpCategory
	for idx := range helpCategories {
		if helpCategories[idx].ID == categoryID {
			category = &helpCategories[idx]
			break
		}
	}

	if category == nil {
		log.Printf("Unknown help category selected: %s", categoryID)
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s %s", category.Emoji, category.Name),
		Description: category.Commands,
		Color:       0x5865F2, // Discord blurple
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use the dropdown above to explore other categories",
		},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{buildCategorySelectMenu()},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Error responding to help component: %v", err)
	}
}
