package database

import (
	"fmt"
	"time"
)

// CreateWin creates a new win entry
func CreateWin(userID uint, guildID, message, category string) (*Win, error) {
	// Validate category
	validCategories := map[string]bool{
		WinCategoryRevenue:   true,
		WinCategoryProduct:   true,
		WinCategoryMarketing: true,
		WinCategoryCustomer:  true,
		WinCategoryOther:     true,
	}

	if category == "" {
		category = WinCategoryOther
	} else if !validCategories[category] {
		category = WinCategoryOther
	}

	win := &Win{
		UserID:   userID,
		GuildID:  guildID,
		Message:  message,
		Category: category,
	}

	if err := DB.Create(win).Error; err != nil {
		return nil, fmt.Errorf("failed to create win: %w", err)
	}

	// Award 2 points for sharing a win
	var user User
	if err := DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return win, nil // Win created but points not awarded - not critical
	}
	user.TotalPoints += 2
	DB.Save(&user)

	return win, nil
}

// UpdateWinMessageID updates the Discord message ID for a win
func UpdateWinMessageID(winID uint, messageID string) error {
	result := DB.Model(&Win{}).Where("id = ?", winID).Update("message_id", messageID)
	if result.Error != nil {
		return fmt.Errorf("failed to update win message ID: %w", result.Error)
	}
	return nil
}

// GetRecentWins gets recent wins for a guild
func GetRecentWins(guildID string, days int) ([]Win, error) {
	var wins []Win
	since := time.Now().AddDate(0, 0, -days)

	result := DB.Preload("User").
		Where("guild_id = ? AND created_at >= ?", guildID, since).
		Order("created_at DESC").
		Find(&wins)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch wins: %w", result.Error)
	}

	return wins, nil
}

// WinStats represents statistics about wins
type WinStats struct {
	TotalWins       int
	WinsByCategory  map[string]int
	RecentWinsCount int
	TopSharers      []WinSharerStats
}

// WinSharerStats represents a user's win sharing stats
type WinSharerStats struct {
	DiscordID string
	Username  string
	WinCount  int
}

// GetWinStats gets win statistics for a guild
func GetWinStats(guildID string) (*WinStats, error) {
	stats := &WinStats{
		WinsByCategory: make(map[string]int),
	}

	// Total wins
	var totalWins int64
	DB.Model(&Win{}).Where("guild_id = ?", guildID).Count(&totalWins)
	stats.TotalWins = int(totalWins)

	// Wins by category
	rows, err := DB.Model(&Win{}).
		Select("category, COUNT(*) as count").
		Where("guild_id = ?", guildID).
		Group("category").
		Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch category stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			continue
		}
		stats.WinsByCategory[category] = count
	}

	// Recent wins (last 7 days)
	since := time.Now().AddDate(0, 0, -7)
	var recentWins int64
	DB.Model(&Win{}).Where("guild_id = ? AND created_at >= ?", guildID, since).Count(&recentWins)
	stats.RecentWinsCount = int(recentWins)

	// Top sharers
	topRows, err := DB.Raw(`
		SELECT u.discord_id, u.username, COUNT(w.id) as win_count
		FROM wins w
		JOIN users u ON u.id = w.user_id
		WHERE w.guild_id = ?
		GROUP BY w.user_id
		ORDER BY win_count DESC
		LIMIT 5
	`, guildID).Rows()
	if err != nil {
		return stats, nil // Return partial stats
	}
	defer topRows.Close()

	for topRows.Next() {
		var sharer WinSharerStats
		if err := topRows.Scan(&sharer.DiscordID, &sharer.Username, &sharer.WinCount); err != nil {
			continue
		}
		stats.TopSharers = append(stats.TopSharers, sharer)
	}

	return stats, nil
}

// GetWinsChannel gets the wins channel for a guild
func GetWinsChannel(guildID string) (string, error) {
	config, err := GetOrCreateGuildConfig(guildID)
	if err != nil {
		return "", err
	}
	return config.WinsChannel, nil
}

// UpdateWinsChannel updates the wins channel for a guild
func UpdateWinsChannel(guildID, channelID string) error {
	config, err := GetOrCreateGuildConfig(guildID)
	if err != nil {
		return err
	}

	config.WinsChannel = channelID
	if err := DB.Save(config).Error; err != nil {
		return fmt.Errorf("failed to update wins channel: %w", err)
	}

	return nil
}

// GetMonthlyTopWins gets top wins for the previous month
func GetMonthlyTopWins(guildID string, limit int) ([]Win, error) {
	now := time.Now()
	// Get first day of previous month
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	firstOfLastMonth := firstOfThisMonth.AddDate(0, -1, 0)

	var wins []Win
	result := DB.Preload("User").
		Where("guild_id = ? AND created_at >= ? AND created_at < ?", guildID, firstOfLastMonth, firstOfThisMonth).
		Order("created_at DESC").
		Limit(limit).
		Find(&wins)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch monthly wins: %w", result.Error)
	}

	return wins, nil
}

// GetUserWinCount gets the total number of wins for a user
func GetUserWinCount(userID uint, guildID string) (int64, error) {
	var count int64
	result := DB.Model(&Win{}).Where("user_id = ? AND guild_id = ?", userID, guildID).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count wins: %w", result.Error)
	}
	return count, nil
}
