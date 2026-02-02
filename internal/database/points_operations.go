package database

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// LeaderboardEntry represents a user's position on the leaderboard
type LeaderboardEntry struct {
	Rank        int
	DiscordID   string
	Username    string
	Points      int
	TasksCount  int
	CompletedAt time.Time
}

// GetOrCreateGuildConfig gets or creates guild configuration
func GetOrCreateGuildConfig(guildID string) (*GuildConfig, error) {
	var config GuildConfig
	result := DB.Where("guild_id = ?", guildID).First(&config)

	if result.Error == gorm.ErrRecordNotFound {
		config = GuildConfig{
			GuildID: guildID,
		}
		if err := DB.Create(&config).Error; err != nil {
			return nil, fmt.Errorf("failed to create guild config: %w", err)
		}
		return &config, nil
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch guild config: %w", result.Error)
	}

	return &config, nil
}

// UpdateLeaderboardChannel updates the leaderboard channel for a guild
func UpdateLeaderboardChannel(guildID, channelID string) error {
	config, err := GetOrCreateGuildConfig(guildID)
	if err != nil {
		return err
	}

	config.LeaderboardChannel = channelID
	if err := DB.Save(config).Error; err != nil {
		return fmt.Errorf("failed to update leaderboard channel: %w", err)
	}

	return nil
}

// GetLeaderboardChannel gets the leaderboard channel for a guild
func GetLeaderboardChannel(guildID string) (string, error) {
	config, err := GetOrCreateGuildConfig(guildID)
	if err != nil {
		return "", err
	}
	return config.LeaderboardChannel, nil
}

// GetOrCreateSprintPoints gets or creates sprint points for a focus period
func GetOrCreateSprintPoints(focusPeriodID, userID uint, guildID string, startDate, endDate time.Time) (*SprintPoints, error) {
	var sp SprintPoints
	result := DB.Where("focus_period_id = ? AND user_id = ?", focusPeriodID, userID).First(&sp)

	if result.Error == gorm.ErrRecordNotFound {
		sp = SprintPoints{
			FocusPeriodID: focusPeriodID,
			UserID:        userID,
			GuildID:       guildID,
			Points:        0,
			StartDate:     startDate,
			EndDate:       endDate,
		}
		if err := DB.Create(&sp).Error; err != nil {
			return nil, fmt.Errorf("failed to create sprint points: %w", err)
		}
		return &sp, nil
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch sprint points: %w", result.Error)
	}

	return &sp, nil
}

// AddPointsToUser adds points to a user's total and sprint total
func AddPointsToUser(userID, focusPeriodID uint, points int, guildID string, startDate, endDate time.Time) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// Update user's total points
		var user User
		if err := tx.Where("id = ?", userID).First(&user).Error; err != nil {
			return fmt.Errorf("failed to fetch user: %w", err)
		}
		user.TotalPoints += points
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("failed to update user points: %w", err)
		}

		// Update sprint points
		var sp SprintPoints
		result := tx.Where("focus_period_id = ? AND user_id = ?", focusPeriodID, userID).First(&sp)

		if result.Error == gorm.ErrRecordNotFound {
			sp = SprintPoints{
				FocusPeriodID: focusPeriodID,
				UserID:        userID,
				GuildID:       guildID,
				Points:        points,
				StartDate:     startDate,
				EndDate:       endDate,
			}
			if err := tx.Create(&sp).Error; err != nil {
				return fmt.Errorf("failed to create sprint points: %w", err)
			}
		} else if result.Error != nil {
			return fmt.Errorf("failed to fetch sprint points: %w", result.Error)
		} else {
			sp.Points += points
			if err := tx.Save(&sp).Error; err != nil {
				return fmt.Errorf("failed to update sprint points: %w", err)
			}
		}

		return nil
	})
}

// GetAllTimeLeaderboard gets the all-time leaderboard for a guild
func GetAllTimeLeaderboard(guildID string, limit int) ([]LeaderboardEntry, error) {
	var entries []LeaderboardEntry

	rows, err := DB.Raw(`
		SELECT
			u.discord_id,
			u.username,
			u.total_points as points,
			COUNT(DISTINCT t.id) as tasks_count,
			MAX(t.completed_at) as completed_at
		FROM users u
		LEFT JOIN focus_periods fp ON fp.user_id = u.id
		LEFT JOIN tasks t ON t.focus_period_id = fp.id AND t.completed = 1
		WHERE u.guild_id = ?
		GROUP BY u.id
		HAVING u.total_points > 0
		ORDER BY u.total_points DESC, completed_at ASC
		LIMIT ?
	`, guildID, limit).Rows()

	if err != nil {
		return nil, fmt.Errorf("failed to fetch all-time leaderboard: %w", err)
	}
	defer rows.Close()

	rank := 1
	for rows.Next() {
		var entry LeaderboardEntry
		var completedAt sql.NullTime
		if err := rows.Scan(&entry.DiscordID, &entry.Username, &entry.Points, &entry.TasksCount, &completedAt); err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		if completedAt.Valid {
			entry.CompletedAt = completedAt.Time
		}
		entry.Rank = rank
		entries = append(entries, entry)
		rank++
	}

	return entries, nil
}

// GetSprintLeaderboard gets the current sprint leaderboard for a guild
func GetSprintLeaderboard(guildID string, limit int) ([]LeaderboardEntry, error) {
	now := time.Now()
	var entries []LeaderboardEntry

	rows, err := DB.Raw(`
		SELECT
			u.discord_id,
			u.username,
			sp.points,
			COUNT(DISTINCT t.id) as tasks_count,
			MAX(t.completed_at) as completed_at
		FROM sprint_points sp
		JOIN users u ON u.id = sp.user_id
		LEFT JOIN tasks t ON t.focus_period_id = sp.focus_period_id AND t.completed = 1
		WHERE sp.guild_id = ?
		  AND sp.start_date <= ?
		  AND sp.end_date >= ?
		GROUP BY sp.id
		HAVING sp.points > 0
		ORDER BY sp.points DESC, completed_at ASC
		LIMIT ?
	`, guildID, now, now, limit).Rows()

	if err != nil {
		return nil, fmt.Errorf("failed to fetch sprint leaderboard: %w", err)
	}
	defer rows.Close()

	rank := 1
	for rows.Next() {
		var entry LeaderboardEntry
		var completedAt sql.NullTime
		if err := rows.Scan(&entry.DiscordID, &entry.Username, &entry.Points, &entry.TasksCount, &completedAt); err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		if completedAt.Valid {
			entry.CompletedAt = completedAt.Time
		}
		entry.Rank = rank
		entries = append(entries, entry)
		rank++
	}

	return entries, nil
}

// GetCompletedFocusPeriodsByDate gets focus periods that ended on a specific date
func GetCompletedFocusPeriodsByDate(guildID string, endDate time.Time) ([]FocusPeriod, error) {
	var periods []FocusPeriod

	// Find periods that ended on the given date (within the same day)
	startOfDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	result := DB.Preload("User").
		Where("guild_id = ? AND end_date >= ? AND end_date < ?", guildID, startOfDay, endOfDay).
		Find(&periods)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch completed focus periods: %w", result.Error)
	}

	return periods, nil
}

// MarkLeaderboardPosted marks that the leaderboard has been posted for a focus period
func MarkLeaderboardPosted(focusPeriodID uint) error {
	result := DB.Model(&FocusPeriod{}).Where("id = ?", focusPeriodID).Update("leaderboard_posted", true)
	if result.Error != nil {
		return fmt.Errorf("failed to mark leaderboard as posted: %w", result.Error)
	}
	return nil
}

// GetEndedPeriodsNeedingLeaderboard gets focus periods that ended but haven't had leaderboard posted
func GetEndedPeriodsNeedingLeaderboard(guildID string) ([]FocusPeriod, error) {
	var periods []FocusPeriod
	now := time.Now()

	result := DB.Preload("User").
		Where("guild_id = ? AND end_date < ? AND leaderboard_posted = ?", guildID, now, false).
		Find(&periods)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch ended periods: %w", result.Error)
	}

	return periods, nil
}
