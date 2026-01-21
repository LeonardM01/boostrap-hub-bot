package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

// focusCommand creates the /focus command group
func focusCommand() *Command {
	return &Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "focus",
			Description: "Manage your 2-week Focus Period goals",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Start a new 2-week Focus Period",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "add",
					Description: "Add a goal to your current Focus Period",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "goal",
							Description: "The goal you want to accomplish",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
					},
				},
				{
					Name:        "complete",
					Description: "Mark a goal as completed",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "number",
							Description: "The goal number to mark as complete (e.g., 1, 2, 3)",
							Type:        discordgo.ApplicationCommandOptionInteger,
							Required:    true,
							MinValue:    floatPtr(1),
							MaxValue:    100,
						},
					},
				},
				{
					Name:        "list",
					Description: "View your current Focus Period goals",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "status",
					Description: "Get an overview of your Focus Period progress",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: handleFocusCommand,
	}
}

func floatPtr(f float64) *float64 {
	return &f
}

func handleFocusCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	case "start":
		handleFocusStart(s, i, user, guildID)
	case "add":
		goalText := options[0].Options[0].StringValue()
		handleFocusAdd(s, i, user, goalText)
	case "complete":
		goalNum := int(options[0].Options[0].IntValue())
		handleFocusComplete(s, i, user, goalNum)
	case "list":
		handleFocusList(s, i, user)
	case "status":
		handleFocusStatus(s, i, user)
	default:
		respondWithError(s, i, "Unknown subcommand")
	}
}

func handleFocusStart(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, guildID string) {
	// Check if user already has an active focus period
	existing, err := database.GetCurrentFocusPeriod(user.ID)
	if err != nil {
		log.Printf("Error checking existing focus period: %v", err)
		respondWithError(s, i, "Failed to check your current Focus Period.")
		return
	}

	if existing != nil {
		embed := &discordgo.MessageEmbed{
			Title:       "Focus Period Already Active",
			Description: fmt.Sprintf("You already have an active Focus Period with %d days remaining!\n\nUse `/focus list` to see your goals or `/focus add` to add more goals.", existing.DaysRemaining()),
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	// Create new focus period
	period, err := database.CreateFocusPeriod(user.ID, guildID)
	if err != nil {
		log.Printf("Error creating focus period: %v", err)
		respondWithError(s, i, "Failed to create your Focus Period.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "New Focus Period Started!",
		Description: "Your 2-week Focus Period has begun. Time to set your goals and crush them!",
		Color:       0x00FF00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Duration",
				Value:  fmt.Sprintf("%s - %s", period.StartDate.Format("Jan 2"), period.EndDate.Format("Jan 2, 2006")),
				Inline: true,
			},
			{
				Name:   "Days Remaining",
				Value:  fmt.Sprintf("%d days", period.DaysRemaining()),
				Inline: true,
			},
			{
				Name:   "Next Steps",
				Value:  "Use `/focus add <goal>` to add your goals.\n\n**Tip:** Set at least 3 goals to stay on track!",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Bootstrap Hub Bot - Focus on what matters",
		},
	}
	respondWithEmbed(s, i, embed)
}

func handleFocusAdd(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, goal string) {
	// Get current focus period
	period, err := database.GetCurrentFocusPeriod(user.ID)
	if err != nil {
		log.Printf("Error getting focus period: %v", err)
		respondWithError(s, i, "Failed to get your Focus Period.")
		return
	}

	if period == nil {
		embed := &discordgo.MessageEmbed{
			Title:       "No Active Focus Period",
			Description: "You don't have an active Focus Period.\n\nUse `/focus start` to begin a new 2-week Focus Period!",
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	// Add the task
	task, err := database.AddTask(period.ID, goal, "")
	if err != nil {
		log.Printf("Error adding task: %v", err)
		respondWithError(s, i, "Failed to add your goal.")
		return
	}

	// Reload tasks to get count
	tasks, _ := database.GetTasksByFocusPeriod(period.ID)
	taskCount := len(tasks)

	embed := &discordgo.MessageEmbed{
		Title:       "Goal Added!",
		Description: fmt.Sprintf("**#%d:** %s", task.Position, goal),
		Color:       0x00FF00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Total Goals",
				Value:  fmt.Sprintf("%d goals set", taskCount),
				Inline: true,
			},
			{
				Name:   "Days Remaining",
				Value:  fmt.Sprintf("%d days", period.DaysRemaining()),
				Inline: true,
			},
		},
	}

	if taskCount < database.MinimumTasksRequired {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Tip",
			Value:  fmt.Sprintf("Add %d more goal(s) to reach the recommended minimum of %d!", database.MinimumTasksRequired-taskCount, database.MinimumTasksRequired),
			Inline: false,
		})
	}

	respondWithEmbed(s, i, embed)
}

func handleFocusComplete(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User, goalNum int) {
	// Get current focus period
	period, err := database.GetCurrentFocusPeriod(user.ID)
	if err != nil {
		log.Printf("Error getting focus period: %v", err)
		respondWithError(s, i, "Failed to get your Focus Period.")
		return
	}

	if period == nil {
		embed := &discordgo.MessageEmbed{
			Title:       "No Active Focus Period",
			Description: "You don't have an active Focus Period.\n\nUse `/focus start` to begin a new 2-week Focus Period!",
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	// Complete the task
	task, err := database.CompleteTask(period.ID, goalNum)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	// Reload tasks to get counts
	tasks, _ := database.GetTasksByFocusPeriod(period.ID)
	completedCount := 0
	for _, t := range tasks {
		if t.Completed {
			completedCount++
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Goal Completed!",
		Description: fmt.Sprintf("**#%d:** ~~%s~~", task.Position, task.Title),
		Color:       0x00FF00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Progress",
				Value:  fmt.Sprintf("%d/%d goals completed", completedCount, len(tasks)),
				Inline: true,
			},
			{
				Name:   "Days Remaining",
				Value:  fmt.Sprintf("%d days", period.DaysRemaining()),
				Inline: true,
			},
		},
	}

	// Add celebration message if all tasks completed
	if completedCount == len(tasks) && len(tasks) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Amazing!",
			Value:  "You've completed ALL your goals for this Focus Period! You're crushing it, founder!",
			Inline: false,
		})
		embed.Color = 0xFFD700 // Gold
	}

	respondWithEmbed(s, i, embed)
}

func handleFocusList(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User) {
	// Get current focus period
	period, err := database.GetCurrentFocusPeriod(user.ID)
	if err != nil {
		log.Printf("Error getting focus period: %v", err)
		respondWithError(s, i, "Failed to get your Focus Period.")
		return
	}

	if period == nil {
		embed := &discordgo.MessageEmbed{
			Title:       "No Active Focus Period",
			Description: "You don't have an active Focus Period.\n\nUse `/focus start` to begin a new 2-week Focus Period!",
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	// Get tasks
	tasks, err := database.GetTasksByFocusPeriod(period.ID)
	if err != nil {
		log.Printf("Error getting tasks: %v", err)
		respondWithError(s, i, "Failed to get your goals.")
		return
	}

	var goalsList strings.Builder
	if len(tasks) == 0 {
		goalsList.WriteString("*No goals set yet!*\n\nUse `/focus add <goal>` to add your first goal.")
	} else {
		for _, task := range tasks {
			if task.Completed {
				goalsList.WriteString(fmt.Sprintf("~~**#%d:** %s~~ ✅\n", task.Position, task.Title))
			} else {
				goalsList.WriteString(fmt.Sprintf("**#%d:** %s\n", task.Position, task.Title))
			}
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Your Focus Period Goals",
		Description: goalsList.String(),
		Color:       0x5865F2, // Discord blurple
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Period",
				Value:  fmt.Sprintf("%s - %s", period.StartDate.Format("Jan 2"), period.EndDate.Format("Jan 2")),
				Inline: true,
			},
			{
				Name:   "Day",
				Value:  fmt.Sprintf("Day %d of 14", period.DayNumber()),
				Inline: true,
			},
			{
				Name:   "Progress",
				Value:  fmt.Sprintf("%d/%d completed", period.CompletedTaskCount(), len(tasks)),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /focus complete <number> to mark a goal as done",
		},
	}

	respondWithEmbed(s, i, embed)
}

func handleFocusStatus(s *discordgo.Session, i *discordgo.InteractionCreate, user *database.User) {
	// Get current focus period
	period, err := database.GetCurrentFocusPeriod(user.ID)
	if err != nil {
		log.Printf("Error getting focus period: %v", err)
		respondWithError(s, i, "Failed to get your Focus Period.")
		return
	}

	if period == nil {
		embed := &discordgo.MessageEmbed{
			Title:       "No Active Focus Period",
			Description: "You don't have an active Focus Period.\n\nUse `/focus start` to begin a new 2-week Focus Period and set goals to track your progress!",
			Color:       0xFFA500, // Orange
		}
		respondWithEmbed(s, i, embed)
		return
	}

	// Get tasks
	tasks, _ := database.GetTasksByFocusPeriod(period.ID)
	completedCount := period.CompletedTaskCount()
	totalCount := len(tasks)
	pendingCount := totalCount - completedCount

	// Calculate progress percentage
	progressPercent := 0
	if totalCount > 0 {
		progressPercent = (completedCount * 100) / totalCount
	}

	// Build progress bar
	progressBar := buildProgressBar(progressPercent)

	// Determine status color and message
	var statusColor int
	var statusMessage string

	switch {
	case totalCount == 0:
		statusColor = 0xFFA500 // Orange
		statusMessage = "You haven't set any goals yet. Add some goals to get started!"
	case completedCount == totalCount:
		statusColor = 0xFFD700 // Gold
		statusMessage = "Outstanding! You've completed all your goals!"
	case progressPercent >= 75:
		statusColor = 0x00FF00 // Green
		statusMessage = "Great progress! You're almost there!"
	case progressPercent >= 50:
		statusColor = 0x5865F2 // Blurple
		statusMessage = "Good progress! Keep pushing forward!"
	case progressPercent >= 25:
		statusColor = 0xFFA500 // Orange
		statusMessage = "You're making progress. Stay focused!"
	default:
		statusColor = 0xFF6B6B // Light red
		statusMessage = "Time to pick up the pace! You've got this!"
	}

	embed := &discordgo.MessageEmbed{
		Title: "Focus Period Status",
		Color: statusColor,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Progress",
				Value:  fmt.Sprintf("%s %d%%", progressBar, progressPercent),
				Inline: false,
			},
			{
				Name:   "Goals",
				Value:  fmt.Sprintf("✅ %d completed\n⏳ %d pending\n📋 %d total", completedCount, pendingCount, totalCount),
				Inline: true,
			},
			{
				Name:   "Time",
				Value:  fmt.Sprintf("📅 Day %d of 14\n⏰ %d days remaining", period.DayNumber(), period.DaysRemaining()),
				Inline: true,
			},
			{
				Name:   "Status",
				Value:  statusMessage,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Focus Period: %s - %s", period.StartDate.Format("Jan 2"), period.EndDate.Format("Jan 2")),
		},
	}

	if totalCount < database.MinimumTasksRequired {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Recommendation",
			Value:  fmt.Sprintf("Consider adding %d more goal(s). Having at least %d goals helps maintain momentum!", database.MinimumTasksRequired-totalCount, database.MinimumTasksRequired),
			Inline: false,
		})
	}

	respondWithEmbed(s, i, embed)
}

// buildProgressBar creates a visual progress bar
func buildProgressBar(percent int) string {
	filled := percent / 10
	empty := 10 - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return "[" + bar + "]"
}

// respondWithEmbed sends an embed response
func respondWithEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error responding with embed: %v", err)
	}
}

// respondWithError sends an error message
func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	embed := &discordgo.MessageEmbed{
		Title:       "Error",
		Description: message,
		Color:       0xFF0000, // Red
	}
	respondWithEmbed(s, i, embed)
}
