package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CreateBuddyRequest creates a new buddy request
func CreateBuddyRequest(requesterID, receiverID uint, guildID string) (*BuddyRequest, error) {
	// Check if requester has reached max buddies
	count, err := GetBuddyCount(requesterID, guildID)
	if err != nil {
		return nil, err
	}
	if count >= MaxBuddiesPerUser {
		return nil, fmt.Errorf("you've reached the maximum of %d buddies", MaxBuddiesPerUser)
	}

	// Check if receiver has reached max buddies
	count, err = GetBuddyCount(receiverID, guildID)
	if err != nil {
		return nil, err
	}
	if count >= MaxBuddiesPerUser {
		return nil, fmt.Errorf("this user has reached their maximum of %d buddies", MaxBuddiesPerUser)
	}

	// Check if they're already buddies
	isBuddy, err := AreBuddies(requesterID, receiverID, guildID)
	if err != nil {
		return nil, err
	}
	if isBuddy {
		return nil, fmt.Errorf("you're already buddies with this user")
	}

	// Check if there's a pending request
	var existingRequest BuddyRequest
	result := DB.Where("requester_id = ? AND receiver_id = ? AND guild_id = ? AND status = ?",
		requesterID, receiverID, guildID, BuddyRequestStatusPending).First(&existingRequest)
	if result.Error == nil {
		return nil, fmt.Errorf("you already have a pending request to this user")
	}

	// Check if there's a pending request from them to you
	result = DB.Where("requester_id = ? AND receiver_id = ? AND guild_id = ? AND status = ?",
		receiverID, requesterID, guildID, BuddyRequestStatusPending).First(&existingRequest)
	if result.Error == nil {
		return nil, fmt.Errorf("this user already sent you a request - use `/buddy accept` to accept it")
	}

	request := &BuddyRequest{
		RequesterID: requesterID,
		ReceiverID:  receiverID,
		GuildID:     guildID,
		Status:      BuddyRequestStatusPending,
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour), // 7 days to accept
	}

	if err := DB.Create(request).Error; err != nil {
		return nil, fmt.Errorf("failed to create buddy request: %w", err)
	}

	return request, nil
}

// AcceptBuddyRequest accepts a buddy request and creates a buddy pair
func AcceptBuddyRequest(requesterID, receiverID uint, guildID string) (*BuddyPair, error) {
	var request BuddyRequest
	result := DB.Where("requester_id = ? AND receiver_id = ? AND guild_id = ? AND status = ?",
		requesterID, receiverID, guildID, BuddyRequestStatusPending).First(&request)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("no pending buddy request from this user")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch buddy request: %w", result.Error)
	}

	// Check if request has expired
	if time.Now().After(request.ExpiresAt) {
		request.Status = BuddyRequestStatusDeclined
		DB.Save(&request)
		return nil, fmt.Errorf("this buddy request has expired")
	}

	var pair *BuddyPair
	err := DB.Transaction(func(tx *gorm.DB) error {
		// Update request status
		request.Status = BuddyRequestStatusAccepted
		if err := tx.Save(&request).Error; err != nil {
			return fmt.Errorf("failed to update request: %w", err)
		}

		// Create buddy pair
		pair = &BuddyPair{
			User1ID:            requesterID,
			User2ID:            receiverID,
			GuildID:            guildID,
			NotifyOnCompletion: true,
		}
		if err := tx.Create(pair).Error; err != nil {
			return fmt.Errorf("failed to create buddy pair: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return pair, nil
}

// DeclineBuddyRequest declines a buddy request
func DeclineBuddyRequest(requesterID, receiverID uint, guildID string) error {
	result := DB.Model(&BuddyRequest{}).
		Where("requester_id = ? AND receiver_id = ? AND guild_id = ? AND status = ?",
			requesterID, receiverID, guildID, BuddyRequestStatusPending).
		Update("status", BuddyRequestStatusDeclined)

	if result.RowsAffected == 0 {
		return fmt.Errorf("no pending buddy request from this user")
	}
	if result.Error != nil {
		return fmt.Errorf("failed to decline request: %w", result.Error)
	}

	return nil
}

// RemoveBuddy removes a buddy relationship
func RemoveBuddy(userID1, userID2 uint, guildID string) error {
	result := DB.Where(
		"((user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)) AND guild_id = ?",
		userID1, userID2, userID2, userID1, guildID,
	).Delete(&BuddyPair{})

	if result.RowsAffected == 0 {
		return fmt.Errorf("you're not buddies with this user")
	}
	if result.Error != nil {
		return fmt.Errorf("failed to remove buddy: %w", result.Error)
	}

	return nil
}

// GetBuddyCount returns the number of buddies a user has
func GetBuddyCount(userID uint, guildID string) (int, error) {
	var count int64
	result := DB.Model(&BuddyPair{}).
		Where("(user1_id = ? OR user2_id = ?) AND guild_id = ?", userID, userID, guildID).
		Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to count buddies: %w", result.Error)
	}

	return int(count), nil
}

// AreBuddies checks if two users are buddies
func AreBuddies(userID1, userID2 uint, guildID string) (bool, error) {
	var count int64
	result := DB.Model(&BuddyPair{}).
		Where("((user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)) AND guild_id = ?",
			userID1, userID2, userID2, userID1, guildID).
		Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("failed to check buddy status: %w", result.Error)
	}

	return count > 0, nil
}

// GetUserBuddies returns all buddies for a user
func GetUserBuddies(userID uint, guildID string) ([]User, error) {
	var buddies []User

	// Get buddy pairs where user is either user1 or user2
	var pairs []BuddyPair
	result := DB.Where("(user1_id = ? OR user2_id = ?) AND guild_id = ?", userID, userID, guildID).Find(&pairs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch buddy pairs: %w", result.Error)
	}

	// Collect buddy IDs
	buddyIDs := make([]uint, 0, len(pairs))
	for _, pair := range pairs {
		if pair.User1ID == userID {
			buddyIDs = append(buddyIDs, pair.User2ID)
		} else {
			buddyIDs = append(buddyIDs, pair.User1ID)
		}
	}

	if len(buddyIDs) == 0 {
		return buddies, nil
	}

	// Fetch buddy users
	result = DB.Where("id IN ?", buddyIDs).Find(&buddies)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch buddies: %w", result.Error)
	}

	return buddies, nil
}

// GetPendingBuddyRequests returns pending requests sent to a user
func GetPendingBuddyRequests(userID uint, guildID string) ([]BuddyRequest, error) {
	var requests []BuddyRequest

	result := DB.Preload("Requester").
		Where("receiver_id = ? AND guild_id = ? AND status = ? AND expires_at > ?",
			userID, guildID, BuddyRequestStatusPending, time.Now()).
		Find(&requests)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch pending requests: %w", result.Error)
	}

	return requests, nil
}

// GetSentBuddyRequests returns pending requests sent by a user
func GetSentBuddyRequests(userID uint, guildID string) ([]BuddyRequest, error) {
	var requests []BuddyRequest

	result := DB.Preload("Receiver").
		Where("requester_id = ? AND guild_id = ? AND status = ? AND expires_at > ?",
			userID, guildID, BuddyRequestStatusPending, time.Now()).
		Find(&requests)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch sent requests: %w", result.Error)
	}

	return requests, nil
}

// GetBuddyPair gets the buddy pair between two users
func GetBuddyPair(userID1, userID2 uint, guildID string) (*BuddyPair, error) {
	var pair BuddyPair
	result := DB.Where(
		"((user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)) AND guild_id = ?",
		userID1, userID2, userID2, userID1, guildID,
	).First(&pair)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch buddy pair: %w", result.Error)
	}

	return &pair, nil
}

// UpdateBuddyNotification updates the notification setting for a buddy pair
func UpdateBuddyNotification(userID1, userID2 uint, guildID string, notify bool) error {
	result := DB.Model(&BuddyPair{}).
		Where("((user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)) AND guild_id = ?",
			userID1, userID2, userID2, userID1, guildID).
		Update("notify_on_completion", notify)

	if result.Error != nil {
		return fmt.Errorf("failed to update notification setting: %w", result.Error)
	}

	return nil
}

// GetBuddiesWithNotifications returns buddies who should be notified when a user completes a task
func GetBuddiesWithNotifications(userID uint, guildID string) ([]User, error) {
	var buddies []User

	rows, err := DB.Raw(`
		SELECT u.*
		FROM users u
		JOIN buddy_pairs bp ON (
			(bp.user1_id = u.id AND bp.user2_id = ?) OR
			(bp.user2_id = u.id AND bp.user1_id = ?)
		)
		WHERE bp.guild_id = ? AND bp.notify_on_completion = true
	`, userID, userID, guildID).Rows()

	if err != nil {
		return nil, fmt.Errorf("failed to fetch buddies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var buddy User
		if err := DB.ScanRows(rows, &buddy); err != nil {
			continue
		}
		buddies = append(buddies, buddy)
	}

	return buddies, nil
}

// CleanupExpiredRequests removes expired buddy requests
func CleanupExpiredRequests() error {
	result := DB.Where("status = ? AND expires_at < ?", BuddyRequestStatusPending, time.Now()).
		Delete(&BuddyRequest{})

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired requests: %w", result.Error)
	}

	return nil
}
