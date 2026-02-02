package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CreateMRREntry creates a new MRR entry
func CreateMRREntry(userID uint, guildID string, amount float64, currency, note string) (*MRREntry, int, error) {
	if currency == "" {
		currency = "USD"
	}

	entry := &MRREntry{
		UserID:   userID,
		GuildID:  guildID,
		Amount:   amount,
		Currency: currency,
		Date:     time.Now(),
		Note:     note,
	}

	if err := DB.Create(entry).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to create MRR entry: %w", err)
	}

	// Check for new milestone
	milestone := checkMRRMilestone(userID, guildID, amount)

	return entry, milestone, nil
}

// checkMRRMilestone checks if user has reached a new milestone
func checkMRRMilestone(userID uint, guildID string, amount float64) int {
	settings, err := GetMRRSettings(userID, guildID)
	if err != nil {
		return 0
	}

	// Convert amount to cents for comparison
	amountCents := int(amount * 100)

	// Find the highest milestone reached
	highestReached := 0
	for _, milestone := range MRRMilestones {
		if amountCents >= milestone && milestone > settings.LastMilestoneReached {
			highestReached = milestone
		}
	}

	if highestReached > 0 {
		// Update last milestone reached
		settings.LastMilestoneReached = highestReached
		DB.Save(settings)
		return highestReached
	}

	return 0
}

// GetMRRSettings gets or creates MRR settings for a user
func GetMRRSettings(userID uint, guildID string) (*MRRSettings, error) {
	var settings MRRSettings
	result := DB.Where("user_id = ? AND guild_id = ?", userID, guildID).First(&settings)

	if result.Error == gorm.ErrRecordNotFound {
		settings = MRRSettings{
			UserID:               userID,
			GuildID:              guildID,
			IsPublic:             false,
			LastMilestoneReached: 0,
		}
		if err := DB.Create(&settings).Error; err != nil {
			return nil, fmt.Errorf("failed to create MRR settings: %w", err)
		}
		return &settings, nil
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch MRR settings: %w", result.Error)
	}

	return &settings, nil
}

// UpdateMRRVisibility updates the public visibility of a user's MRR
func UpdateMRRVisibility(userID uint, guildID string, isPublic bool) error {
	settings, err := GetMRRSettings(userID, guildID)
	if err != nil {
		return err
	}

	settings.IsPublic = isPublic
	if err := DB.Save(settings).Error; err != nil {
		return fmt.Errorf("failed to update MRR visibility: %w", err)
	}

	return nil
}

// GetLatestMRR gets the most recent MRR entry for a user
func GetLatestMRR(userID uint, guildID string) (*MRREntry, error) {
	var entry MRREntry
	result := DB.Where("user_id = ? AND guild_id = ?", userID, guildID).
		Order("date DESC").
		First(&entry)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch MRR: %w", result.Error)
	}

	return &entry, nil
}

// GetMRRHistory gets MRR history for a user
func GetMRRHistory(userID uint, guildID string, months int) ([]MRREntry, error) {
	var entries []MRREntry
	since := time.Now().AddDate(0, -months, 0)

	result := DB.Where("user_id = ? AND guild_id = ? AND date >= ?", userID, guildID, since).
		Order("date DESC").
		Find(&entries)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch MRR history: %w", result.Error)
	}

	return entries, nil
}

// MRRLeaderboardEntry represents an entry in the MRR leaderboard
type MRRLeaderboardEntry struct {
	Rank      int
	DiscordID string
	Username  string
	Amount    float64
	Currency  string
	Growth    float64 // Percentage growth from previous entry
}

// GetMRRLeaderboard gets the public MRR leaderboard
func GetMRRLeaderboard(guildID string, limit int) ([]MRRLeaderboardEntry, error) {
	var entries []MRRLeaderboardEntry

	// Get latest MRR for each user who has public MRR
	rows, err := DB.Raw(`
		SELECT
			u.discord_id,
			u.username,
			mrr.amount,
			mrr.currency
		FROM mrr_entries mrr
		JOIN users u ON u.id = mrr.user_id
		JOIN mrr_settings ms ON ms.user_id = mrr.user_id AND ms.guild_id = mrr.guild_id
		WHERE mrr.guild_id = ?
		  AND ms.is_public = true
		  AND mrr.date = (
			SELECT MAX(m2.date) FROM mrr_entries m2
			WHERE m2.user_id = mrr.user_id AND m2.guild_id = mrr.guild_id
		  )
		ORDER BY mrr.amount DESC
		LIMIT ?
	`, guildID, limit).Rows()

	if err != nil {
		return nil, fmt.Errorf("failed to fetch MRR leaderboard: %w", err)
	}
	defer rows.Close()

	rank := 1
	for rows.Next() {
		var entry MRRLeaderboardEntry
		if err := rows.Scan(&entry.DiscordID, &entry.Username, &entry.Amount, &entry.Currency); err != nil {
			continue
		}
		entry.Rank = rank
		entries = append(entries, entry)
		rank++
	}

	return entries, nil
}

// GetMRRChannel gets the MRR milestone channel for a guild
func GetMRRChannel(guildID string) (string, error) {
	config, err := GetOrCreateGuildConfig(guildID)
	if err != nil {
		return "", err
	}
	return config.MRRChannel, nil
}

// UpdateMRRChannel updates the MRR channel for a guild
func UpdateMRRChannel(guildID, channelID string) error {
	config, err := GetOrCreateGuildConfig(guildID)
	if err != nil {
		return err
	}

	config.MRRChannel = channelID
	if err := DB.Save(config).Error; err != nil {
		return fmt.Errorf("failed to update MRR channel: %w", err)
	}

	return nil
}

// FormatMRRMilestone formats a milestone amount to a readable string
func FormatMRRMilestone(cents int) string {
	dollars := float64(cents) / 100
	if dollars >= 1000 {
		return fmt.Sprintf("$%.0fK", dollars/1000)
	}
	return fmt.Sprintf("$%.0f", dollars)
}

// GetMRRGrowth calculates the growth percentage between two entries
func GetMRRGrowth(current, previous float64) float64 {
	if previous == 0 {
		return 0
	}
	return ((current - previous) / previous) * 100
}

// GetMRRStats gets MRR statistics for a user
type MRRStats struct {
	CurrentMRR     float64
	Currency       string
	AllTimeHigh    float64
	MonthlyGrowth  float64
	TotalEntries   int
	FirstEntry     *time.Time
	IsPublic       bool
	NextMilestone  int
	MilestonesHit  int
}

func GetMRRStats(userID uint, guildID string) (*MRRStats, error) {
	stats := &MRRStats{}

	// Get latest entry
	latest, err := GetLatestMRR(userID, guildID)
	if err != nil {
		return nil, err
	}
	if latest != nil {
		stats.CurrentMRR = latest.Amount
		stats.Currency = latest.Currency
	}

	// Get all-time high
	var maxEntry MRREntry
	DB.Where("user_id = ? AND guild_id = ?", userID, guildID).
		Order("amount DESC").
		First(&maxEntry)
	stats.AllTimeHigh = maxEntry.Amount

	// Get first entry
	var firstEntry MRREntry
	result := DB.Where("user_id = ? AND guild_id = ?", userID, guildID).
		Order("date ASC").
		First(&firstEntry)
	if result.Error == nil {
		stats.FirstEntry = &firstEntry.CreatedAt
	}

	// Count entries
	var count int64
	DB.Model(&MRREntry{}).Where("user_id = ? AND guild_id = ?", userID, guildID).Count(&count)
	stats.TotalEntries = int(count)

	// Get monthly growth
	oneMonthAgo := time.Now().AddDate(0, -1, 0)
	var previousEntry MRREntry
	result = DB.Where("user_id = ? AND guild_id = ? AND date <= ?", userID, guildID, oneMonthAgo).
		Order("date DESC").
		First(&previousEntry)
	if result.Error == nil && latest != nil {
		stats.MonthlyGrowth = GetMRRGrowth(latest.Amount, previousEntry.Amount)
	}

	// Get settings
	settings, _ := GetMRRSettings(userID, guildID)
	if settings != nil {
		stats.IsPublic = settings.IsPublic

		// Count milestones hit
		currentCents := int(stats.CurrentMRR * 100)
		for _, milestone := range MRRMilestones {
			if currentCents >= milestone {
				stats.MilestonesHit++
			} else if stats.NextMilestone == 0 {
				stats.NextMilestone = milestone
			}
		}
	}

	return stats, nil
}
