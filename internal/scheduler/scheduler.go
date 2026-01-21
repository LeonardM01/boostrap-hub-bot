package scheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/bootstrap-hub/bootstrap-hub-bot/internal/database"
	"github.com/bwmarrin/discordgo"
)

// Scheduler handles periodic tasks like reminders
type Scheduler struct {
	session         *discordgo.Session
	reminderChannel string // Channel ID to send reminders to
	stopChan        chan struct{}
	ticker          *time.Ticker
}

// New creates a new Scheduler instance
func New(session *discordgo.Session, reminderChannelID string) *Scheduler {
	return &Scheduler{
		session:         session,
		reminderChannel: reminderChannelID,
		stopChan:        make(chan struct{}),
	}
}

// Start begins the scheduler, checking every hour for reminders to send
func (s *Scheduler) Start() {
	log.Println("Starting reminder scheduler...")

	// Check every hour
	s.ticker = time.NewTicker(1 * time.Hour)

	go func() {
		// Run immediately on start
		s.runChecks()

		for {
			select {
			case <-s.ticker.C:
				s.runChecks()
			case <-s.stopChan:
				log.Println("Scheduler stopped")
				return
			}
		}
	}()

	log.Println("Reminder scheduler started")
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	log.Println("Stopping reminder scheduler...")
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopChan)
}

// runChecks runs all scheduled checks
func (s *Scheduler) runChecks() {
	log.Println("Running scheduled checks...")

	// Only run main checks once per day (between 9:00-10:00 AM)
	hour := time.Now().Hour()
	if hour == 9 {
		s.checkDailyReminders()
		s.checkInsufficientTasks()
	}
}

// checkDailyReminders sends reminders on specific days (3, 7, 10, 12, 13)
func (s *Scheduler) checkDailyReminders() {
	if s.reminderChannel == "" {
		log.Println("No reminder channel configured, skipping daily reminders")
		return
	}

	guildIDs, err := database.GetAllGuildsWithActivePeriods()
	if err != nil {
		log.Printf("Error fetching guilds: %v", err)
		return
	}

	for _, guildID := range guildIDs {
		s.sendDailyRemindersForGuild(guildID)
	}
}

// sendDailyRemindersForGuild sends reminders to users in a specific guild
func (s *Scheduler) sendDailyRemindersForGuild(guildID string) {
	periods, err := database.GetUsersWithActiveFocusPeriods(guildID)
	if err != nil {
		log.Printf("Error fetching focus periods for guild %s: %v", guildID, err)
		return
	}

	for _, period := range periods {
		dayNumber := period.DayNumber()

		// Check if today is a reminder day
		isReminderDay := false
		for _, rd := range database.ReminderDays {
			if dayNumber == rd {
				isReminderDay = true
				break
			}
		}

		if !isReminderDay {
			continue
		}

		// Only remind if they have pending tasks
		if period.PendingTaskCount() == 0 {
			continue
		}

		s.sendReminderMessage(&period)
	}
}

// sendReminderMessage sends a reminder to the configured channel
func (s *Scheduler) sendReminderMessage(period *database.FocusPeriod) {
	if s.reminderChannel == "" {
		return
	}

	dayNumber := period.DayNumber()
	daysRemaining := period.DaysRemaining()
	pendingCount := period.PendingTaskCount()
	completedCount := period.CompletedTaskCount()

	// Build urgency message based on day
	var urgencyMessage string
	var color int

	switch {
	case dayNumber == 13:
		urgencyMessage = "**Final day tomorrow!** Time for a last push!"
		color = 0xFF0000 // Red
	case dayNumber == 12:
		urgencyMessage = "**Only 2 days left!** Let's finish strong!"
		color = 0xFF6B6B // Light red
	case dayNumber == 10:
		urgencyMessage = "**4 days remaining.** Great time to make progress!"
		color = 0xFFA500 // Orange
	case dayNumber == 7:
		urgencyMessage = "**Halfway through!** How are those goals coming along?"
		color = 0x5865F2 // Blurple
	case dayNumber == 3:
		urgencyMessage = "**Day 3 check-in.** Building momentum!"
		color = 0x00FF00 // Green
	default:
		urgencyMessage = "Keep pushing towards your goals!"
		color = 0x5865F2
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Focus Period Reminder - Day %d", dayNumber),
		Description: fmt.Sprintf("<@%s>, %s", period.User.DiscordID, urgencyMessage),
		Color:       color,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Your Progress",
				Value:  fmt.Sprintf("✅ %d completed | ⏳ %d pending", completedCount, pendingCount),
				Inline: true,
			},
			{
				Name:   "Time Left",
				Value:  fmt.Sprintf("%d days remaining", daysRemaining),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /focus list to see your goals | /focus complete <#> to mark done",
		},
	}

	_, err := s.session.ChannelMessageSendEmbed(s.reminderChannel, embed)
	if err != nil {
		log.Printf("Error sending reminder to channel %s: %v", s.reminderChannel, err)
	} else {
		log.Printf("Sent Day %d reminder to user %s", dayNumber, period.User.DiscordID)
	}
}

// checkInsufficientTasks reminds users who have less than 3 tasks
func (s *Scheduler) checkInsufficientTasks() {
	if s.reminderChannel == "" {
		log.Println("No reminder channel configured, skipping insufficient tasks check")
		return
	}

	guildIDs, err := database.GetAllGuildsWithActivePeriods()
	if err != nil {
		log.Printf("Error fetching guilds: %v", err)
		return
	}

	for _, guildID := range guildIDs {
		periods, err := database.GetUsersWithInsufficientTasks(guildID)
		if err != nil {
			log.Printf("Error fetching insufficient tasks for guild %s: %v", guildID, err)
			continue
		}

		for _, period := range periods {
			// Only remind once per period (on day 2 or 3)
			dayNumber := period.DayNumber()
			if dayNumber != 2 && dayNumber != 3 {
				continue
			}

			s.sendInsufficientTasksReminder(&period)
		}
	}
}

// sendInsufficientTasksReminder reminds a user to add more tasks
func (s *Scheduler) sendInsufficientTasksReminder(period *database.FocusPeriod) {
	if s.reminderChannel == "" {
		return
	}

	taskCount := len(period.Tasks)
	needed := database.MinimumTasksRequired - taskCount

	embed := &discordgo.MessageEmbed{
		Title:       "Set Your Focus Period Goals",
		Description: fmt.Sprintf("<@%s>, you currently have **%d goal(s)** set for this Focus Period.", period.User.DiscordID, taskCount),
		Color:       0xFFA500, // Orange
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Recommendation",
				Value:  fmt.Sprintf("Consider adding **%d more goal(s)** to reach the recommended minimum of %d.\n\nHaving clear goals helps maintain focus and momentum!", needed, database.MinimumTasksRequired),
				Inline: false,
			},
			{
				Name:   "How to Add Goals",
				Value:  "Use `/focus add <your goal>` to add a new goal.",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Bootstrap Hub Bot - Helping founders stay focused",
		},
	}

	_, err := s.session.ChannelMessageSendEmbed(s.reminderChannel, embed)
	if err != nil {
		log.Printf("Error sending insufficient tasks reminder: %v", err)
	} else {
		log.Printf("Sent insufficient tasks reminder to user %s", period.User.DiscordID)
	}
}

// SendManualReminder allows sending a manual reminder (for testing)
func (s *Scheduler) SendManualReminder(userDiscordID string, message string) error {
	if s.reminderChannel == "" {
		return fmt.Errorf("no reminder channel configured")
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Focus Period Reminder",
		Description: fmt.Sprintf("<@%s>, %s", userDiscordID, message),
		Color:       0x5865F2,
	}

	_, err := s.session.ChannelMessageSendEmbed(s.reminderChannel, embed)
	return err
}
