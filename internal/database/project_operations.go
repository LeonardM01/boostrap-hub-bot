package database

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateProjectMapping creates or updates a role-to-category mapping
func CreateProjectMapping(guildID, roleID, roleName, categoryID, categoryName string, maxChannels int) (*ProjectMapping, error) {
	mapping := ProjectMapping{
		GuildID:      guildID,
		RoleID:       roleID,
		RoleName:     roleName,
		CategoryID:   categoryID,
		CategoryName: categoryName,
		MaxChannels:  maxChannels,
	}

	result := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "guild_id"}, {Name: "role_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"role_name", "category_id", "category_name", "max_channels",
		}),
	}).Create(&mapping)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to create project mapping: %w", result.Error)
	}

	return &mapping, nil
}

// RemoveProjectMapping deletes a role-to-category mapping
func RemoveProjectMapping(guildID, roleID string) error {
	result := DB.Where("guild_id = ? AND role_id = ?", guildID, roleID).Delete(&ProjectMapping{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove project mapping: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no mapping found for that role")
	}
	return nil
}

// GetProjectMappings returns all mappings for a guild
func GetProjectMappings(guildID string) ([]ProjectMapping, error) {
	var mappings []ProjectMapping
	result := DB.Where("guild_id = ?", guildID).Find(&mappings)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch project mappings: %w", result.Error)
	}
	return mappings, nil
}

// GetUserMappings finds mappings that match any of the user's role IDs
func GetUserMappings(guildID string, roleIDs []string) ([]ProjectMapping, error) {
	var mappings []ProjectMapping
	result := DB.Where("guild_id = ? AND role_id IN ?", guildID, roleIDs).Find(&mappings)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to fetch user mappings: %w", result.Error)
	}
	return mappings, nil
}

// CreateProjectChannel records a channel created via /project
func CreateProjectChannel(guildID, userID, channelID, categoryID, roleID, name, channelType string) (*ProjectChannel, error) {
	channel := ProjectChannel{
		GuildID:    guildID,
		UserID:     userID,
		ChannelID:  channelID,
		CategoryID: categoryID,
		RoleID:     roleID,
		Name:       name,
		Type:       channelType,
	}

	if err := DB.Create(&channel).Error; err != nil {
		return nil, fmt.Errorf("failed to record project channel: %w", err)
	}

	return &channel, nil
}

// CountUserChannelsInCategory returns how many channels a user has created in a category
func CountUserChannelsInCategory(guildID, userID, categoryID string) (int64, error) {
	var count int64
	result := DB.Model(&ProjectChannel{}).Where(
		"guild_id = ? AND user_id = ? AND category_id = ?",
		guildID, userID, categoryID,
	).Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to count user channels: %w", result.Error)
	}

	return count, nil
}

// GetUserChannelsInCategory returns all channels a user has created in a category
func GetUserChannelsInCategory(guildID, userID, categoryID string) ([]ProjectChannel, error) {
	var channels []ProjectChannel
	result := DB.Where(
		"guild_id = ? AND user_id = ? AND category_id = ?",
		guildID, userID, categoryID,
	).Find(&channels)

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to fetch user channels: %w", result.Error)
	}

	return channels, nil
}
