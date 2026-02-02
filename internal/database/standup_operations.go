package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CreateStandup creates a new standup entry and updates streak
func CreateStandup(userID uint, guildID, workingOn, accomplished, blockers string) (*Standup, *UserStreak, int, error) {
	today := time.Now()
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	// Check if user already posted today
	var existingStandup Standup
	result := DB.Where("user_id = ? AND guild_id = ? AND date >= ?", userID, guildID, todayStart).First(&existingStandup)
	if result.Error == nil {
		return nil, nil, 0, fmt.Errorf("you've already posted a standup today")
	}

	var standup *Standup
	var streak *UserStreak
	var bonusPoints int

	err := DB.Transaction(func(tx *gorm.DB) error {
		// Create standup
		standup = &Standup{
			UserID:       userID,
			GuildID:      guildID,
			Date:         today,
			WorkingOn:    workingOn,
			Accomplished: accomplished,
			Blockers:     blockers,
		}
		if err := tx.Create(standup).Error; err != nil {
			return fmt.Errorf("failed to create standup: %w", err)
		}

		// Get or create streak
		streak = &UserStreak{}
		result := tx.Where("user_id = ? AND guild_id = ?", userID, guildID).First(streak)
		if result.Error == gorm.ErrRecordNotFound {
			streak = &UserStreak{
				UserID:          userID,
				GuildID:         guildID,
				CurrentStreak:   0,
				LongestStreak:   0,
				TotalStandups:   0,
				LastStandupDate: nil,
			}
			if err := tx.Create(streak).Error; err != nil {
				return fmt.Errorf("failed to create streak: %w", err)
			}
		} else if result.Error != nil {
			return fmt.Errorf("failed to fetch streak: %w", result.Error)
		}

		// Update streak
		yesterday := todayStart.Add(-24 * time.Hour)
		if streak.LastStandupDate != nil {
			lastDate := time.Date(streak.LastStandupDate.Year(), streak.LastStandupDate.Month(), streak.LastStandupDate.Day(), 0, 0, 0, 0, streak.LastStandupDate.Location())
			if lastDate.Equal(yesterday) {
				// Consecutive day - increment streak
				streak.CurrentStreak++
			} else if lastDate.Before(yesterday) {
				// Streak broken - reset to 1
				streak.CurrentStreak = 1
			}
			// Same day means no change (handled by the "already posted" check above)
		} else {
			// First standup ever
			streak.CurrentStreak = 1
		}

		// Update longest streak if needed
		if streak.CurrentStreak > streak.LongestStreak {
			streak.LongestStreak = streak.CurrentStreak
		}

		streak.TotalStandups++
		streak.LastStandupDate = &today

		// Check for streak milestone bonus
		if bonus, exists := StreakMilestones[streak.CurrentStreak]; exists {
			bonusPoints = bonus
		}

		if err := tx.Save(streak).Error; err != nil {
			return fmt.Errorf("failed to update streak: %w", err)
		}

		// Award base points (1 point per standup) + bonus
		totalPoints := 1 + bonusPoints
		var user User
		if err := tx.Where("id = ?", userID).First(&user).Error; err != nil {
			return fmt.Errorf("failed to fetch user: %w", err)
		}
		user.TotalPoints += totalPoints
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("failed to update user points: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, nil, 0, err
	}

	return standup, streak, bonusPoints, nil
}

// GetUserStreak gets the streak info for a user
func GetUserStreak(userID uint, guildID string) (*UserStreak, error) {
	var streak UserStreak
	result := DB.Where("user_id = ? AND guild_id = ?", userID, guildID).First(&streak)

	if result.Error == gorm.ErrRecordNotFound {
		// Return a default streak (user hasn't posted any standups yet)
		return &UserStreak{
			UserID:        userID,
			GuildID:       guildID,
			CurrentStreak: 0,
			LongestStreak: 0,
			TotalStandups: 0,
		}, nil
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch streak: %w", result.Error)
	}

	return &streak, nil
}

// StreakLeaderboardEntry represents an entry in the streak leaderboard
type StreakLeaderboardEntry struct {
	Rank          int
	DiscordID     string
	Username      string
	CurrentStreak int
	LongestStreak int
	TotalStandups int
}

// GetStreakLeaderboard gets the streak leaderboard for a guild
func GetStreakLeaderboard(guildID string, limit int) ([]StreakLeaderboardEntry, error) {
	var entries []StreakLeaderboardEntry

	rows, err := DB.Raw(`
		SELECT
			u.discord_id,
			u.username,
			us.current_streak,
			us.longest_streak,
			us.total_standups
		FROM user_streaks us
		JOIN users u ON u.id = us.user_id
		WHERE us.guild_id = ? AND us.total_standups > 0
		ORDER BY us.current_streak DESC, us.total_standups DESC
		LIMIT ?
	`, guildID, limit).Rows()

	if err != nil {
		return nil, fmt.Errorf("failed to fetch streak leaderboard: %w", err)
	}
	defer rows.Close()

	rank := 1
	for rows.Next() {
		var entry StreakLeaderboardEntry
		if err := rows.Scan(&entry.DiscordID, &entry.Username, &entry.CurrentStreak, &entry.LongestStreak, &entry.TotalStandups); err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		entry.Rank = rank
		entries = append(entries, entry)
		rank++
	}

	return entries, nil
}

// GetUserStandups gets recent standups for a user
func GetUserStandups(userID uint, guildID string, days int) ([]Standup, error) {
	var standups []Standup
	since := time.Now().AddDate(0, 0, -days)

	result := DB.Where("user_id = ? AND guild_id = ? AND date >= ?", userID, guildID, since).
		Order("date DESC").
		Find(&standups)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch standups: %w", result.Error)
	}

	return standups, nil
}

// HasPostedStandupToday checks if a user has posted a standup today
func HasPostedStandupToday(userID uint, guildID string) (bool, error) {
	today := time.Now()
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	var count int64
	result := DB.Model(&Standup{}).Where("user_id = ? AND guild_id = ? AND date >= ?", userID, guildID, todayStart).Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("failed to check standup: %w", result.Error)
	}

	return count > 0, nil
}

// GetUsersWithoutStandupToday gets users who have active focus periods but haven't posted today
func GetUsersWithoutStandupToday(guildID string) ([]User, error) {
	today := time.Now()
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	var users []User
	result := DB.Raw(`
		SELECT DISTINCT u.*
		FROM users u
		JOIN focus_periods fp ON fp.user_id = u.id
		WHERE u.guild_id = ?
		  AND fp.start_date <= ?
		  AND fp.end_date >= ?
		  AND u.id NOT IN (
			SELECT user_id FROM standups WHERE guild_id = ? AND date >= ?
		  )
	`, guildID, today, today, guildID, todayStart).Scan(&users)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch users without standup: %w", result.Error)
	}

	return users, nil
}
