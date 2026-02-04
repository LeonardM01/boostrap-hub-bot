package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database connection
var DB *gorm.DB

// Initialize sets up the database connection and runs migrations
func Initialize(dbPath string) error {
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	err = DB.AutoMigrate(&GuildConfig{}, &User{}, &Project{}, &FocusPeriod{}, &Task{})
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

// GetOrCreateUser gets an existing user or creates a new one
func GetOrCreateUser(discordID, guildID, username string) (*User, error) {
	var user User
	result := DB.Where("discord_id = ? AND guild_id = ?", discordID, guildID).First(&user)

	if result.Error == gorm.ErrRecordNotFound {
		user = User{
			DiscordID: discordID,
			GuildID:   guildID,
			Username:  username,
		}
		if err := DB.Create(&user).Error; err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		return &user, nil
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", result.Error)
	}

	// Update username if changed
	if user.Username != username {
		user.Username = username
		DB.Save(&user)
	}

	return &user, nil
}

// GetCurrentFocusPeriod returns the active focus period for a user, if any
func GetCurrentFocusPeriod(userID uint) (*FocusPeriod, error) {
	var period FocusPeriod
	now := time.Now()

	result := DB.Preload("Tasks", func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC")
	}).Where("user_id = ? AND start_date <= ? AND end_date >= ?", userID, now, now).First(&period)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch focus period: %w", result.Error)
	}

	return &period, nil
}

// CreateFocusPeriod creates a new focus period for a user
func CreateFocusPeriod(userID uint, guildID string) (*FocusPeriod, error) {
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endDate := startDate.Add(FocusPeriodDuration)

	period := FocusPeriod{
		UserID:    userID,
		GuildID:   guildID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	if err := DB.Create(&period).Error; err != nil {
		return nil, fmt.Errorf("failed to create focus period: %w", err)
	}

	return &period, nil
}

// AddTask adds a task to a focus period
func AddTask(focusPeriodID uint, title, description string) (*Task, error) {
	// Get the next position
	var maxPosition int
	DB.Model(&Task{}).Where("focus_period_id = ?", focusPeriodID).Select("COALESCE(MAX(position), 0)").Scan(&maxPosition)

	task := Task{
		FocusPeriodID: focusPeriodID,
		Title:         title,
		Description:   description,
		Completed:     false,
		Position:      maxPosition + 1,
	}

	if err := DB.Create(&task).Error; err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return &task, nil
}

// CompleteTask marks a task as completed
func CompleteTask(focusPeriodID uint, position int) (*Task, error) {
	var task Task
	result := DB.Where("focus_period_id = ? AND position = ?", focusPeriodID, position).First(&task)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("task #%d not found", position)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch task: %w", result.Error)
	}

	if task.Completed {
		return nil, fmt.Errorf("task #%d is already completed", position)
	}

	now := time.Now()
	task.Completed = true
	task.CompletedAt = &now

	if err := DB.Save(&task).Error; err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return &task, nil
}

// GetTasksByFocusPeriod returns all tasks for a focus period
func GetTasksByFocusPeriod(focusPeriodID uint) ([]Task, error) {
	var tasks []Task
	result := DB.Where("focus_period_id = ?", focusPeriodID).Order("position ASC").Find(&tasks)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", result.Error)
	}

	return tasks, nil
}

// GetUsersWithActiveFocusPeriods returns all users who have active focus periods in a guild
func GetUsersWithActiveFocusPeriods(guildID string) ([]FocusPeriod, error) {
	var periods []FocusPeriod
	now := time.Now()

	result := DB.Preload("User").Preload("Tasks").
		Where("guild_id = ? AND start_date <= ? AND end_date >= ?", guildID, now, now).
		Find(&periods)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch focus periods: %w", result.Error)
	}

	return periods, nil
}

// GetUsersWithInsufficientTasks returns users who have less than minimum required tasks
func GetUsersWithInsufficientTasks(guildID string) ([]FocusPeriod, error) {
	periods, err := GetUsersWithActiveFocusPeriods(guildID)
	if err != nil {
		return nil, err
	}

	var insufficient []FocusPeriod
	for _, period := range periods {
		if len(period.Tasks) < MinimumTasksRequired {
			insufficient = append(insufficient, period)
		}
	}

	return insufficient, nil
}

// GetAllGuildsWithActivePeriods returns all guild IDs that have active focus periods
func GetAllGuildsWithActivePeriods() ([]string, error) {
	var guildIDs []string
	now := time.Now()

	result := DB.Model(&FocusPeriod{}).
		Distinct("guild_id").
		Where("start_date <= ? AND end_date >= ?", now, now).
		Pluck("guild_id", &guildIDs)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch guild IDs: %w", result.Error)
	}

	return guildIDs, nil
}

// GetFocusPeriodsForReminder returns focus periods that need a reminder on the current day
func GetFocusPeriodsForReminder(guildID string, dayNumber int) ([]FocusPeriod, error) {
	periods, err := GetUsersWithActiveFocusPeriods(guildID)
	if err != nil {
		return nil, err
	}

	var needsReminder []FocusPeriod
	for _, period := range periods {
		if period.DayNumber() == dayNumber && period.PendingTaskCount() > 0 {
			needsReminder = append(needsReminder, period)
		}
	}

	return needsReminder, nil
}

// --- Guild Config Operations ---

// GetGuildConfig returns the configuration for a guild
func GetGuildConfig(guildID string) (*GuildConfig, error) {
	var config GuildConfig
	result := DB.Where("guild_id = ?", guildID).First(&config)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch guild config: %w", result.Error)
	}
	return &config, nil
}

// SetWelcomeChannel sets the welcome channel for a guild
func SetWelcomeChannel(guildID, channelID string) error {
	var config GuildConfig
	result := DB.Where("guild_id = ?", guildID).First(&config)

	if result.Error == gorm.ErrRecordNotFound {
		config = GuildConfig{
			GuildID:          guildID,
			WelcomeChannelID: channelID,
		}
		return DB.Create(&config).Error
	}
	if result.Error != nil {
		return result.Error
	}

	config.WelcomeChannelID = channelID
	return DB.Save(&config).Error
}

// --- Project Operations ---

// CreateProject stores a new founder project
func CreateProject(guildID string, userID uint, name, website, category, roleID, categoryID, channelID string) (*Project, error) {
	project := Project{
		GuildID:    guildID,
		UserID:     userID,
		Name:       name,
		Website:    website,
		Category:   category,
		RoleID:     roleID,
		CategoryID: categoryID,
		ChannelID:  channelID,
	}
	if err := DB.Create(&project).Error; err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}
	return &project, nil
}

// GetProjectByUser returns the project for a user in a guild
func GetProjectByUser(userID uint, guildID string) (*Project, error) {
	var project Project
	result := DB.Where("user_id = ? AND guild_id = ?", userID, guildID).First(&project)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch project: %w", result.Error)
	}
	return &project, nil
}

// MarkUserOnboarded marks a user as having completed onboarding
func MarkUserOnboarded(userID uint) error {
	return DB.Model(&User{}).Where("id = ?", userID).Update("onboarded", true).Error
}
