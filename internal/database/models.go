package database

import (
	"time"

	"gorm.io/gorm"
)

// User represents a Discord user in the system
type User struct {
	gorm.Model
	DiscordID   string `gorm:"uniqueIndex;not null"` // Discord user ID
	GuildID     string `gorm:"index;not null"`       // Discord guild/server ID
	Username    string // Cached username for display
	TotalPoints int    `gorm:"default:0"` // Lifetime points earned
}

// FocusPeriod represents a 2-week goal period (like a sprint)
type FocusPeriod struct {
	gorm.Model
	UserID            uint      `gorm:"index;not null"`
	User              User      `gorm:"foreignKey:UserID"`
	GuildID           string    `gorm:"index;not null"` // Guild where this period was created
	StartDate         time.Time `gorm:"not null"`
	EndDate           time.Time `gorm:"not null"`
	Tasks             []Task    `gorm:"foreignKey:FocusPeriodID"`
	LeaderboardPosted bool      `gorm:"default:false"` // Tracks if leaderboard was posted for this period
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
	Points        int `gorm:"default:0"` // Points earned when task is completed
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusCompleted TaskStatus = "completed"
)

// PublicResource represents a community-vetted resource with emoji voting
type PublicResource struct {
	gorm.Model
	GuildID           string `gorm:"index;not null"`
	SubmitterID       string `gorm:"not null"` // Discord user ID
	SubmitterUsername string

	// Resource details
	URL         string `gorm:"not null"`
	Title       string `gorm:"not null"`
	Description string `gorm:"type:text"`
	Category    string `gorm:"index"`
	Tags        string // Comma-separated

	// Voting tracking
	VoteMessageID string    `gorm:"uniqueIndex"` // Message with reactions
	VoteChannelID string
	VoteExpiresAt time.Time

	// Vote counts
	UsefulVotes    int `gorm:"default:0"`
	NotUsefulVotes int `gorm:"default:0"`

	// Status
	Status      ResourceStatus `gorm:"index;default:'pending'"`
	ProcessedAt *time.Time
}

// PrivateResource represents role-based resource sharing
type PrivateResource struct {
	gorm.Model
	GuildID       string `gorm:"index;not null"`
	OwnerID       string `gorm:"not null"`
	OwnerUsername string

	// Resource details
	URL         string `gorm:"not null"`
	Title       string `gorm:"not null"`
	Description string `gorm:"type:text"`
	Category    string `gorm:"index"`
	Tags        string

	// Access control
	AllowedRoles []PrivateResourceRole `gorm:"foreignKey:PrivateResourceID"`
}

// PrivateResourceRole links private resources to Discord roles
type PrivateResourceRole struct {
	gorm.Model
	PrivateResourceID uint            `gorm:"index;not null"`
	PrivateResource   PrivateResource `gorm:"foreignKey:PrivateResourceID"`
	RoleID            string          `gorm:"index;not null"` // Discord role ID
	RoleName          string          // Cached role name
}

// ResourceStatus represents the approval status of a public resource
type ResourceStatus string

const (
	ResourceStatusPending  ResourceStatus = "pending"
	ResourceStatusApproved ResourceStatus = "approved"
	ResourceStatusRejected ResourceStatus = "rejected"
)

// GuildConfig stores guild-level configuration
type GuildConfig struct {
	gorm.Model
	GuildID            string `gorm:"uniqueIndex;not null"` // Discord guild/server ID
	LeaderboardChannel string // Channel ID for automated leaderboard posts
	WinsChannel        string // Channel ID for win celebrations
	MRRChannel         string // Channel ID for MRR milestone announcements
}

// SprintPoints tracks points earned in a specific focus period
type SprintPoints struct {
	gorm.Model
	FocusPeriodID uint        `gorm:"index;not null"`
	FocusPeriod   FocusPeriod `gorm:"foreignKey:FocusPeriodID"`
	UserID        uint        `gorm:"index;not null"`
	User          User        `gorm:"foreignKey:UserID"`
	GuildID       string      `gorm:"index;not null"`
	Points        int         `gorm:"default:0"`
	StartDate     time.Time   `gorm:"not null;index"` // Cached from FocusPeriod for easier querying
	EndDate       time.Time   `gorm:"not null;index"`
}

// Standup represents a daily standup/check-in
type Standup struct {
	gorm.Model
	UserID       uint      `gorm:"index;not null"`
	User         User      `gorm:"foreignKey:UserID"`
	GuildID      string    `gorm:"index;not null"`
	Date         time.Time `gorm:"index;not null"`
	Accomplished string    `gorm:"type:text"`
	WorkingOn    string    `gorm:"type:text;not null"`
	Blockers     string    `gorm:"type:text"`
}

// UserStreak tracks daily standup streaks for a user
type UserStreak struct {
	gorm.Model
	UserID          uint       `gorm:"uniqueIndex:idx_user_guild_streak;not null"`
	User            User       `gorm:"foreignKey:UserID"`
	GuildID         string     `gorm:"uniqueIndex:idx_user_guild_streak;not null"`
	CurrentStreak   int        `gorm:"default:0"`
	LongestStreak   int        `gorm:"default:0"`
	LastStandupDate *time.Time
	TotalStandups   int `gorm:"default:0"`
}

// Win represents a user-shared win/celebration
type Win struct {
	gorm.Model
	UserID     uint      `gorm:"index;not null"`
	User       User      `gorm:"foreignKey:UserID"`
	GuildID    string    `gorm:"index;not null"`
	Message    string    `gorm:"type:text;not null"`
	MessageID  string    // Discord message ID if posted to wins channel
	Category   string    // revenue, product, marketing, customer, other
	CreatedAt  time.Time `gorm:"index"`
}

// WinCategory constants
const (
	WinCategoryRevenue   = "revenue"
	WinCategoryProduct   = "product"
	WinCategoryMarketing = "marketing"
	WinCategoryCustomer  = "customer"
	WinCategoryOther     = "other"
)

// BuddyRequest represents a pending accountability buddy request
type BuddyRequest struct {
	gorm.Model
	RequesterID uint      `gorm:"index;not null"`
	Requester   User      `gorm:"foreignKey:RequesterID"`
	ReceiverID  uint      `gorm:"index;not null"`
	Receiver    User      `gorm:"foreignKey:ReceiverID"`
	GuildID     string    `gorm:"index;not null"`
	Status      string    `gorm:"default:'pending'"` // pending, accepted, declined
	ExpiresAt   time.Time
}

// BuddyRequestStatus constants
const (
	BuddyRequestStatusPending  = "pending"
	BuddyRequestStatusAccepted = "accepted"
	BuddyRequestStatusDeclined = "declined"
)

// BuddyPair represents an active accountability buddy relationship
type BuddyPair struct {
	gorm.Model
	User1ID            uint   `gorm:"index;not null"`
	User1              User   `gorm:"foreignKey:User1ID"`
	User2ID            uint   `gorm:"index;not null"`
	User2              User   `gorm:"foreignKey:User2ID"`
	GuildID            string `gorm:"index;not null"`
	NotifyOnCompletion bool   `gorm:"default:true"`
}

// MaxBuddiesPerUser is the maximum number of buddies a user can have
const MaxBuddiesPerUser = 3

// Challenge represents a time-boxed challenge between buddies
type Challenge struct {
	gorm.Model
	CreatorID        uint      `gorm:"index;not null"`
	Creator          User      `gorm:"foreignKey:CreatorID"`
	GuildID          string    `gorm:"index;not null"`
	Title            string    `gorm:"not null"`
	Description      string    `gorm:"type:text"`
	StartDate        time.Time
	EndDate          time.Time `gorm:"index"`
	Status           string    `gorm:"default:'active'"` // active, completed, failed
	PointsMultiplier float64   `gorm:"default:1.5"`
}

// ChallengeStatus constants
const (
	ChallengeStatusActive    = "active"
	ChallengeStatusCompleted = "completed"
	ChallengeStatusFailed    = "failed"
)

// ChallengeParticipant represents a user's participation in a challenge
type ChallengeParticipant struct {
	gorm.Model
	ChallengeID uint      `gorm:"index;not null"`
	Challenge   Challenge `gorm:"foreignKey:ChallengeID"`
	UserID      uint      `gorm:"index;not null"`
	User        User      `gorm:"foreignKey:UserID"`
	Status      string    `gorm:"default:'active'"` // active, pending_validation, completed, failed
	ProofURL    string
	CompletedAt *time.Time
}

// ChallengeParticipantStatus constants
const (
	ChallengeParticipantStatusActive            = "active"
	ChallengeParticipantStatusPendingValidation = "pending_validation"
	ChallengeParticipantStatusCompleted         = "completed"
	ChallengeParticipantStatusFailed            = "failed"
)

// ChallengeProgress represents a progress update on a challenge
type ChallengeProgress struct {
	gorm.Model
	ChallengeID uint      `gorm:"index;not null"`
	Challenge   Challenge `gorm:"foreignKey:ChallengeID"`
	UserID      uint      `gorm:"index;not null"`
	User        User      `gorm:"foreignKey:UserID"`
	Update      string    `gorm:"type:text;not null"`
}

// ChallengeValidation represents a buddy's validation of challenge completion
type ChallengeValidation struct {
	gorm.Model
	ParticipantID uint                 `gorm:"index;not null"`
	Participant   ChallengeParticipant `gorm:"foreignKey:ParticipantID"`
	ValidatorID   uint                 `gorm:"index;not null"`
	Validator     User                 `gorm:"foreignKey:ValidatorID"`
	Approved      bool
}

// MRREntry represents a monthly recurring revenue log entry
type MRREntry struct {
	gorm.Model
	UserID   uint      `gorm:"index;not null"`
	User     User      `gorm:"foreignKey:UserID"`
	GuildID  string    `gorm:"index;not null"`
	Amount   float64   `gorm:"not null"`
	Currency string    `gorm:"default:'USD'"`
	Date     time.Time `gorm:"index"`
	Note     string
}

// MRRSettings represents user's MRR display preferences
type MRRSettings struct {
	gorm.Model
	UserID               uint   `gorm:"uniqueIndex:idx_user_guild_mrr;not null"`
	User                 User   `gorm:"foreignKey:UserID"`
	GuildID              string `gorm:"uniqueIndex:idx_user_guild_mrr;not null"`
	IsPublic             bool   `gorm:"default:false"`
	LastMilestoneReached int    `gorm:"default:0"`
	ProjectChannelID     string // User's project channel for MRR reminders
}

// MRRMilestones defines the revenue milestones to celebrate (in cents to avoid float issues)
var MRRMilestones = []int{10000, 50000, 100000, 500000, 1000000, 5000000, 10000000} // $100, $500, $1K, $5K, $10K, $50K, $100K

// StreakMilestones defines the days at which streak bonuses are awarded
var StreakMilestones = map[int]int{
	7:  10,  // 7 days = +10 points
	14: 25,  // 14 days = +25 points
	30: 50,  // 30 days = +50 points
	60: 100, // 60 days = +100 points
	90: 200, // 90 days = +200 points
}

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
