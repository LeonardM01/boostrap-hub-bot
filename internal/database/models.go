package database

import (
	"time"

	"gorm.io/gorm"
)

// GuildConfig stores per-guild bot configuration
type GuildConfig struct {
	gorm.Model
	GuildID          string `gorm:"uniqueIndex;not null"` // Discord guild/server ID
	WelcomeChannelID string // Channel where onboarding messages are sent
}

// Project represents an onboarded founder project
type Project struct {
	gorm.Model
	GuildID    string `gorm:"index;not null"`
	UserID     uint   `gorm:"index;not null"`
	User       User   `gorm:"foreignKey:UserID"`
	Name       string `gorm:"not null"`
	Website    string
	Category   string // Project category (SaaS, E-commerce, etc.)
	RoleID     string // Discord role ID created for this project
	CategoryID string // Discord category (channel group) ID
	ChannelID  string // Discord text channel ID under the category
}

// User represents a Discord user in the system
type User struct {
	gorm.Model
	DiscordID  string `gorm:"uniqueIndex;not null"` // Discord user ID
	GuildID    string `gorm:"index;not null"`        // Discord guild/server ID
	Username   string // Cached username for display
	Onboarded  bool   `gorm:"default:false"` // Whether user completed onboarding
}

// FocusPeriod represents a 2-week goal period (like a sprint)
type FocusPeriod struct {
	gorm.Model
	UserID    uint      `gorm:"index;not null"`
	User      User      `gorm:"foreignKey:UserID"`
	GuildID   string    `gorm:"index;not null"` // Guild where this period was created
	StartDate time.Time `gorm:"not null"`
	EndDate   time.Time `gorm:"not null"`
	Tasks     []Task    `gorm:"foreignKey:FocusPeriodID"`
}

// Task represents a goal/task within a focus period
type Task struct {
	gorm.Model
	FocusPeriodID uint        `gorm:"index;not null"`
	FocusPeriod   FocusPeriod `gorm:"foreignKey:FocusPeriodID"`
	Title         string      `gorm:"not null"`
	Description   string
	Completed     bool
	CompletedAt   *time.Time
	Position      int // Order within the focus period (1, 2, 3, etc.)
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusCompleted TaskStatus = "completed"
)

// FocusPeriodDuration is the length of a focus period
const FocusPeriodDuration = 14 * 24 * time.Hour // 2 weeks

// MinimumTasksRequired is the minimum number of tasks a user should set
const MinimumTasksRequired = 3

// ReminderDays are the days on which to send reminders (1-indexed from period start)
var ReminderDays = []int{3, 7, 10, 12, 13}

// IsActive returns true if the focus period is currently active
func (fp *FocusPeriod) IsActive() bool {
	now := time.Now()
	return now.After(fp.StartDate) && now.Before(fp.EndDate)
}

// DaysRemaining returns the number of days left in the focus period
func (fp *FocusPeriod) DaysRemaining() int {
	remaining := time.Until(fp.EndDate)
	if remaining < 0 {
		return 0
	}
	return int(remaining.Hours() / 24)
}

// DayNumber returns the current day number within the focus period (1-14)
func (fp *FocusPeriod) DayNumber() int {
	elapsed := time.Since(fp.StartDate)
	if elapsed < 0 {
		return 0
	}
	day := int(elapsed.Hours()/24) + 1
	if day > 14 {
		return 14
	}
	return day
}

// CompletedTaskCount returns the number of completed tasks
func (fp *FocusPeriod) CompletedTaskCount() int {
	count := 0
	for _, task := range fp.Tasks {
		if task.Completed {
			count++
		}
	}
	return count
}

// PendingTaskCount returns the number of pending tasks
func (fp *FocusPeriod) PendingTaskCount() int {
	count := 0
	for _, task := range fp.Tasks {
		if !task.Completed {
			count++
		}
	}
	return count
}
