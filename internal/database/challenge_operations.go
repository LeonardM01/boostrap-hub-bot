package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CreateChallenge creates a new challenge with participants
func CreateChallenge(creatorID uint, guildID, title, description string, days int, participantIDs []uint, multiplier float64) (*Challenge, error) {
	if multiplier <= 0 {
		multiplier = 1.5
	}
	if multiplier > 3.0 {
		multiplier = 3.0
	}

	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endDate := startDate.Add(time.Duration(days) * 24 * time.Hour)

	var challenge *Challenge
	err := DB.Transaction(func(tx *gorm.DB) error {
		challenge = &Challenge{
			CreatorID:        creatorID,
			GuildID:          guildID,
			Title:            title,
			Description:      description,
			StartDate:        startDate,
			EndDate:          endDate,
			Status:           ChallengeStatusActive,
			PointsMultiplier: multiplier,
		}

		if err := tx.Create(challenge).Error; err != nil {
			return fmt.Errorf("failed to create challenge: %w", err)
		}

		// Add creator as a participant
		allParticipants := append([]uint{creatorID}, participantIDs...)

		for _, userID := range allParticipants {
			participant := ChallengeParticipant{
				ChallengeID: challenge.ID,
				UserID:      userID,
				Status:      ChallengeParticipantStatusActive,
			}
			if err := tx.Create(&participant).Error; err != nil {
				return fmt.Errorf("failed to add participant: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return challenge, nil
}

// GetChallenge gets a challenge by ID
func GetChallenge(challengeID uint) (*Challenge, error) {
	var challenge Challenge
	result := DB.Preload("Creator").First(&challenge, challengeID)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("challenge not found")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch challenge: %w", result.Error)
	}
	return &challenge, nil
}

// GetChallengeWithParticipants gets a challenge with its participants
func GetChallengeWithParticipants(challengeID uint) (*Challenge, []ChallengeParticipant, error) {
	var challenge Challenge
	result := DB.Preload("Creator").First(&challenge, challengeID)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil, fmt.Errorf("challenge not found")
	}
	if result.Error != nil {
		return nil, nil, fmt.Errorf("failed to fetch challenge: %w", result.Error)
	}

	var participants []ChallengeParticipant
	result = DB.Preload("User").Where("challenge_id = ?", challengeID).Find(&participants)
	if result.Error != nil {
		return nil, nil, fmt.Errorf("failed to fetch participants: %w", result.Error)
	}

	return &challenge, participants, nil
}

// GetUserChallenges gets challenges for a user
func GetUserChallenges(userID uint, guildID string, status string) ([]Challenge, error) {
	var challenges []Challenge

	query := DB.Preload("Creator").
		Joins("JOIN challenge_participants cp ON cp.challenge_id = challenges.id").
		Where("cp.user_id = ? AND challenges.guild_id = ?", userID, guildID)

	if status != "" {
		query = query.Where("challenges.status = ?", status)
	}

	result := query.Order("challenges.created_at DESC").Find(&challenges)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch challenges: %w", result.Error)
	}

	return challenges, nil
}

// GetActiveChallenges gets all active challenges in a guild
func GetActiveChallenges(guildID string) ([]Challenge, error) {
	var challenges []Challenge
	now := time.Now()

	result := DB.Preload("Creator").
		Where("guild_id = ? AND status = ? AND end_date > ?", guildID, ChallengeStatusActive, now).
		Order("end_date ASC").
		Find(&challenges)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch active challenges: %w", result.Error)
	}

	return challenges, nil
}

// AddChallengeProgress adds a progress update to a challenge
func AddChallengeProgress(challengeID, userID uint, update string) (*ChallengeProgress, error) {
	// Verify user is a participant
	var participant ChallengeParticipant
	result := DB.Where("challenge_id = ? AND user_id = ?", challengeID, userID).First(&participant)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("you're not a participant in this challenge")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to verify participation: %w", result.Error)
	}

	if participant.Status != ChallengeParticipantStatusActive {
		return nil, fmt.Errorf("this challenge is no longer active for you")
	}

	progress := &ChallengeProgress{
		ChallengeID: challengeID,
		UserID:      userID,
		Update:      update,
	}

	if err := DB.Create(progress).Error; err != nil {
		return nil, fmt.Errorf("failed to add progress: %w", err)
	}

	return progress, nil
}

// GetChallengeProgress gets progress updates for a challenge
func GetChallengeProgress(challengeID uint) ([]ChallengeProgress, error) {
	var progress []ChallengeProgress
	result := DB.Preload("User").
		Where("challenge_id = ?", challengeID).
		Order("created_at DESC").
		Find(&progress)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch progress: %w", result.Error)
	}

	return progress, nil
}

// SubmitChallengeCompletion marks a participant as pending validation
func SubmitChallengeCompletion(challengeID, userID uint, proofURL string) (*ChallengeParticipant, error) {
	var participant ChallengeParticipant
	result := DB.Where("challenge_id = ? AND user_id = ?", challengeID, userID).First(&participant)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("you're not a participant in this challenge")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch participant: %w", result.Error)
	}

	if participant.Status != ChallengeParticipantStatusActive {
		return nil, fmt.Errorf("you've already submitted or this challenge is no longer active")
	}

	participant.Status = ChallengeParticipantStatusPendingValidation
	participant.ProofURL = proofURL

	if err := DB.Save(&participant).Error; err != nil {
		return nil, fmt.Errorf("failed to submit completion: %w", err)
	}

	return &participant, nil
}

// ValidateChallengeCompletion validates or rejects a completion
func ValidateChallengeCompletion(challengeID, validatorID, targetUserID uint, approved bool) error {
	// Get the participant
	var participant ChallengeParticipant
	result := DB.Where("challenge_id = ? AND user_id = ?", challengeID, targetUserID).First(&participant)
	if result.Error == gorm.ErrRecordNotFound {
		return fmt.Errorf("participant not found")
	}
	if result.Error != nil {
		return fmt.Errorf("failed to fetch participant: %w", result.Error)
	}

	if participant.Status != ChallengeParticipantStatusPendingValidation {
		return fmt.Errorf("this participant hasn't submitted completion yet")
	}

	// Verify validator is a participant
	var validatorParticipant ChallengeParticipant
	result = DB.Where("challenge_id = ? AND user_id = ?", challengeID, validatorID).First(&validatorParticipant)
	if result.Error != nil {
		return fmt.Errorf("you're not a participant in this challenge")
	}

	// Can't validate own submission
	if validatorID == targetUserID {
		return fmt.Errorf("you can't validate your own submission")
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		// Create validation record
		validation := ChallengeValidation{
			ParticipantID: participant.ID,
			ValidatorID:   validatorID,
			Approved:      approved,
		}
		if err := tx.Create(&validation).Error; err != nil {
			return fmt.Errorf("failed to create validation: %w", err)
		}

		if approved {
			// Check if we have enough validations (at least 1 for small challenges)
			var validationCount int64
			tx.Model(&ChallengeValidation{}).
				Where("participant_id = ? AND approved = ?", participant.ID, true).
				Count(&validationCount)

			// For now, 1 validation is enough
			if validationCount >= 1 {
				now := time.Now()
				participant.Status = ChallengeParticipantStatusCompleted
				participant.CompletedAt = &now
				if err := tx.Save(&participant).Error; err != nil {
					return fmt.Errorf("failed to update participant: %w", err)
				}

				// Award points with multiplier
				var challenge Challenge
				tx.First(&challenge, challengeID)
				points := int(10 * challenge.PointsMultiplier) // Base 10 points * multiplier

				var user User
				if err := tx.First(&user, targetUserID).Error; err == nil {
					user.TotalPoints += points
					tx.Save(&user)
				}
			}
		} else {
			// Rejection - move back to active
			participant.Status = ChallengeParticipantStatusActive
			participant.ProofURL = ""
			if err := tx.Save(&participant).Error; err != nil {
				return fmt.Errorf("failed to update participant: %w", err)
			}
		}

		return nil
	})

	return err
}

// GetChallengeParticipant gets a participant's status
func GetChallengeParticipant(challengeID, userID uint) (*ChallengeParticipant, error) {
	var participant ChallengeParticipant
	result := DB.Preload("User").Where("challenge_id = ? AND user_id = ?", challengeID, userID).First(&participant)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch participant: %w", result.Error)
	}
	return &participant, nil
}

// CheckAndFailExpiredChallenges marks expired challenges as failed
func CheckAndFailExpiredChallenges() error {
	now := time.Now()

	// Find active challenges that have ended
	var challenges []Challenge
	result := DB.Where("status = ? AND end_date < ?", ChallengeStatusActive, now).Find(&challenges)
	if result.Error != nil {
		return fmt.Errorf("failed to fetch expired challenges: %w", result.Error)
	}

	for _, challenge := range challenges {
		err := DB.Transaction(func(tx *gorm.DB) error {
			// Mark incomplete participants as failed
			tx.Model(&ChallengeParticipant{}).
				Where("challenge_id = ? AND status IN ?", challenge.ID,
					[]string{ChallengeParticipantStatusActive, ChallengeParticipantStatusPendingValidation}).
				Update("status", ChallengeParticipantStatusFailed)

			// Check if all participants completed
			var completedCount int64
			var totalCount int64
			tx.Model(&ChallengeParticipant{}).Where("challenge_id = ?", challenge.ID).Count(&totalCount)
			tx.Model(&ChallengeParticipant{}).Where("challenge_id = ? AND status = ?",
				challenge.ID, ChallengeParticipantStatusCompleted).Count(&completedCount)

			if completedCount == totalCount {
				challenge.Status = ChallengeStatusCompleted
			} else {
				challenge.Status = ChallengeStatusFailed
			}
			return tx.Save(&challenge).Error
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// GetChallengesNeedingReminder gets active challenges for reminder
func GetChallengesNeedingReminder(guildID string) ([]Challenge, error) {
	now := time.Now()
	var challenges []Challenge

	result := DB.Preload("Creator").
		Where("guild_id = ? AND status = ? AND end_date > ?", guildID, ChallengeStatusActive, now).
		Find(&challenges)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch challenges: %w", result.Error)
	}

	return challenges, nil
}

// IsUserInChallenge checks if a user is a participant in a challenge
func IsUserInChallenge(challengeID, userID uint) (bool, error) {
	var count int64
	result := DB.Model(&ChallengeParticipant{}).
		Where("challenge_id = ? AND user_id = ?", challengeID, userID).
		Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("failed to check participation: %w", result.Error)
	}

	return count > 0, nil
}
