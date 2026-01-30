package database

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ==================== Public Resource Operations ====================

// CreatePublicResource creates a new public resource pending approval
func CreatePublicResource(guildID, submitterID, submitterUsername, url, title, description, category, tags string) (*PublicResource, error) {
	resource := PublicResource{
		GuildID:           guildID,
		SubmitterID:       submitterID,
		SubmitterUsername: submitterUsername,
		URL:               url,
		Title:             title,
		Description:       description,
		Category:          category,
		Tags:              tags,
		Status:            ResourceStatusPending,
		UsefulVotes:       0,
		NotUsefulVotes:    0,
	}

	if err := DB.Create(&resource).Error; err != nil {
		return nil, fmt.Errorf("failed to create public resource: %w", err)
	}

	return &resource, nil
}

// UpdateResourceVoteMessage updates the vote message details for a resource
func UpdateResourceVoteMessage(resourceID uint, messageID, channelID string, expiresAt time.Time) error {
	result := DB.Model(&PublicResource{}).
		Where("id = ?", resourceID).
		Updates(map[string]interface{}{
			"vote_message_id": messageID,
			"vote_channel_id": channelID,
			"vote_expires_at": expiresAt,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update vote message: %w", result.Error)
	}

	return nil
}

// GetPublicResourceByVoteMessageID retrieves a resource by its vote message ID
func GetPublicResourceByVoteMessageID(messageID string) (*PublicResource, error) {
	var resource PublicResource
	result := DB.Where("vote_message_id = ?", messageID).First(&resource)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch resource: %w", result.Error)
	}

	return &resource, nil
}

// GetPendingResourcesWithExpiredVotes returns pending resources with expired voting periods
func GetPendingResourcesWithExpiredVotes() ([]PublicResource, error) {
	var resources []PublicResource
	now := time.Now()

	result := DB.Where("status = ? AND vote_expires_at <= ? AND vote_expires_at IS NOT NULL", ResourceStatusPending, now).Find(&resources)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch expired resources: %w", result.Error)
	}

	return resources, nil
}

// UpdateResourceStatus updates the status of a resource
func UpdateResourceStatus(resourceID uint, status ResourceStatus, processedAt time.Time) error {
	result := DB.Model(&PublicResource{}).
		Where("id = ?", resourceID).
		Updates(map[string]interface{}{
			"status":       status,
			"processed_at": processedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update resource status: %w", result.Error)
	}

	return nil
}

// IncrementUsefulVotes increments the useful vote count for a resource
func IncrementUsefulVotes(resourceID uint) error {
	result := DB.Model(&PublicResource{}).
		Where("id = ?", resourceID).
		UpdateColumn("useful_votes", gorm.Expr("useful_votes + ?", 1))

	if result.Error != nil {
		return fmt.Errorf("failed to increment useful votes: %w", result.Error)
	}

	return nil
}

// DecrementUsefulVotes decrements the useful vote count for a resource
func DecrementUsefulVotes(resourceID uint) error {
	result := DB.Model(&PublicResource{}).
		Where("id = ? AND useful_votes > 0", resourceID).
		UpdateColumn("useful_votes", gorm.Expr("useful_votes - ?", 1))

	if result.Error != nil {
		return fmt.Errorf("failed to decrement useful votes: %w", result.Error)
	}

	return nil
}

// IncrementNotUsefulVotes increments the not useful vote count for a resource
func IncrementNotUsefulVotes(resourceID uint) error {
	result := DB.Model(&PublicResource{}).
		Where("id = ?", resourceID).
		UpdateColumn("not_useful_votes", gorm.Expr("not_useful_votes + ?", 1))

	if result.Error != nil {
		return fmt.Errorf("failed to increment not useful votes: %w", result.Error)
	}

	return nil
}

// DecrementNotUsefulVotes decrements the not useful vote count for a resource
func DecrementNotUsefulVotes(resourceID uint) error {
	result := DB.Model(&PublicResource{}).
		Where("id = ? AND not_useful_votes > 0", resourceID).
		UpdateColumn("not_useful_votes", gorm.Expr("not_useful_votes - ?", 1))

	if result.Error != nil {
		return fmt.Errorf("failed to decrement not useful votes: %w", result.Error)
	}

	return nil
}

// GetApprovedPublicResources retrieves approved public resources with optional filters
func GetApprovedPublicResources(guildID string, category string, search string) ([]PublicResource, error) {
	var resources []PublicResource
	query := DB.Where("guild_id = ? AND status = ?", guildID, ResourceStatusApproved)

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ? OR LOWER(tags) LIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	result := query.Order("created_at DESC").Find(&resources)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch approved resources: %w", result.Error)
	}

	return resources, nil
}

// CheckDuplicateURL checks if a URL already exists as an approved resource in the guild
func CheckDuplicateURL(guildID, url string) (*PublicResource, error) {
	var resource PublicResource
	result := DB.Where("guild_id = ? AND url = ? AND status = ?", guildID, url, ResourceStatusApproved).First(&resource)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to check duplicate: %w", result.Error)
	}

	return &resource, nil
}

// ==================== Private Resource Operations ====================

// CreatePrivateResourceWithRoles creates a private resource with associated roles
func CreatePrivateResourceWithRoles(guildID, ownerID, ownerUsername, url, title, description, category, tags string, roles []struct {
	RoleID   string
	RoleName string
}) (*PrivateResource, error) {
	resource := PrivateResource{
		GuildID:       guildID,
		OwnerID:       ownerID,
		OwnerUsername: ownerUsername,
		URL:           url,
		Title:         title,
		Description:   description,
		Category:      category,
		Tags:          tags,
	}

	// Create resource and roles in a transaction
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&resource).Error; err != nil {
			return fmt.Errorf("failed to create resource: %w", err)
		}

		// Create role associations
		for _, role := range roles {
			resourceRole := PrivateResourceRole{
				PrivateResourceID: resource.ID,
				RoleID:            role.RoleID,
				RoleName:          role.RoleName,
			}
			if err := tx.Create(&resourceRole).Error; err != nil {
				return fmt.Errorf("failed to create role association: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Reload with associations
	DB.Preload("AllowedRoles").First(&resource, resource.ID)

	return &resource, nil
}

// GetPrivateResourcesForUser retrieves private resources accessible to a user based on their roles
func GetPrivateResourcesForUser(guildID string, userRoleIDs []string, category string, search string) ([]PrivateResource, error) {
	var resources []PrivateResource

	if len(userRoleIDs) == 0 {
		return resources, nil
	}

	query := DB.Preload("AllowedRoles").
		Joins("JOIN private_resource_roles ON private_resource_roles.private_resource_id = private_resources.id").
		Where("private_resources.guild_id = ?", guildID).
		Where("private_resource_roles.role_id IN ?", userRoleIDs).
		Group("private_resources.id")

	if category != "" {
		query = query.Where("private_resources.category = ?", category)
	}

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(private_resources.title) LIKE ? OR LOWER(private_resources.description) LIKE ? OR LOWER(private_resources.tags) LIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	result := query.Order("private_resources.created_at DESC").Find(&resources)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch private resources: %w", result.Error)
	}

	return resources, nil
}

// DeletePrivateResource deletes a private resource (only owner can delete)
func DeletePrivateResource(resourceID uint, ownerID string) error {
	result := DB.Where("id = ? AND owner_id = ?", resourceID, ownerID).Delete(&PrivateResource{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete resource: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("resource not found or you don't have permission to delete it")
	}

	return nil
}

// GetPrivateResourceByID retrieves a private resource by ID
func GetPrivateResourceByID(resourceID uint) (*PrivateResource, error) {
	var resource PrivateResource
	result := DB.Preload("AllowedRoles").First(&resource, resourceID)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("resource not found")
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch resource: %w", result.Error)
	}

	return &resource, nil
}
