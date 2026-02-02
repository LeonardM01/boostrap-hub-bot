package database

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(
		&User{},
		&FocusPeriod{},
		&Task{},
		&GuildConfig{},
		&SprintPoints{},
	)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Set global DB for tests
	DB = db
	return db
}

func TestGetOrCreateGuildConfig(t *testing.T) {
	setupTestDB(t)

	guildID := "test-guild-123"

	// Create config
	config1, err := GetOrCreateGuildConfig(guildID)
	if err != nil {
		t.Fatalf("Failed to create guild config: %v", err)
	}

	if config1.GuildID != guildID {
		t.Errorf("Expected GuildID %s, got %s", guildID, config1.GuildID)
	}

	// Get existing config
	config2, err := GetOrCreateGuildConfig(guildID)
	if err != nil {
		t.Fatalf("Failed to get guild config: %v", err)
	}

	if config1.ID != config2.ID {
		t.Errorf("Expected same config ID, got %d and %d", config1.ID, config2.ID)
	}
}

func TestUpdateLeaderboardChannel(t *testing.T) {
	setupTestDB(t)

	guildID := "test-guild-123"
	channelID := "channel-456"

	err := UpdateLeaderboardChannel(guildID, channelID)
	if err != nil {
		t.Fatalf("Failed to update leaderboard channel: %v", err)
	}

	// Verify update
	config, err := GetOrCreateGuildConfig(guildID)
	if err != nil {
		t.Fatalf("Failed to get guild config: %v", err)
	}

	if config.LeaderboardChannel != channelID {
		t.Errorf("Expected LeaderboardChannel %s, got %s", channelID, config.LeaderboardChannel)
	}
}

func TestAddPointsToUser(t *testing.T) {
	setupTestDB(t)

	guildID := "test-guild-123"
	userDiscordID := "user-789"

	// Create user
	user := User{
		DiscordID:   userDiscordID,
		GuildID:     guildID,
		Username:    "TestUser",
		TotalPoints: 0,
	}
	DB.Create(&user)

	// Create focus period
	startDate := time.Now()
	endDate := startDate.Add(14 * 24 * time.Hour)
	period := FocusPeriod{
		UserID:    user.ID,
		GuildID:   guildID,
		StartDate: startDate,
		EndDate:   endDate,
	}
	DB.Create(&period)

	// Add points
	points := 7
	err := AddPointsToUser(user.ID, period.ID, points, guildID, startDate, endDate)
	if err != nil {
		t.Fatalf("Failed to add points: %v", err)
	}

	// Verify user points
	var updatedUser User
	DB.First(&updatedUser, user.ID)
	if updatedUser.TotalPoints != points {
		t.Errorf("Expected TotalPoints %d, got %d", points, updatedUser.TotalPoints)
	}

	// Verify sprint points
	var sp SprintPoints
	err = DB.Where("focus_period_id = ? AND user_id = ?", period.ID, user.ID).First(&sp).Error
	if err != nil {
		t.Fatalf("Failed to get sprint points: %v", err)
	}

	if sp.Points != points {
		t.Errorf("Expected sprint points %d, got %d", points, sp.Points)
	}

	// Add more points
	additionalPoints := 3
	err = AddPointsToUser(user.ID, period.ID, additionalPoints, guildID, startDate, endDate)
	if err != nil {
		t.Fatalf("Failed to add additional points: %v", err)
	}

	// Verify cumulative points
	DB.First(&updatedUser, user.ID)
	expectedTotal := points + additionalPoints
	if updatedUser.TotalPoints != expectedTotal {
		t.Errorf("Expected TotalPoints %d, got %d", expectedTotal, updatedUser.TotalPoints)
	}
}

func TestGetAllTimeLeaderboard(t *testing.T) {
	db := setupTestDB(t)

	guildID := "test-guild-123"

	// Create test users with points
	users := []User{
		{DiscordID: "user1", GuildID: guildID, Username: "User1", TotalPoints: 50},
		{DiscordID: "user2", GuildID: guildID, Username: "User2", TotalPoints: 30},
		{DiscordID: "user3", GuildID: guildID, Username: "User3", TotalPoints: 0},
		{DiscordID: "user4", GuildID: guildID, Username: "User4", TotalPoints: 40},
	}

	for _, u := range users {
		db.Create(&u)
	}

	// Create focus periods and tasks for users with points
	for i, u := range users {
		if u.TotalPoints > 0 {
			period := FocusPeriod{
				UserID:    u.ID,
				GuildID:   guildID,
				StartDate: time.Now().Add(-14 * 24 * time.Hour),
				EndDate:   time.Now(),
			}
			db.Create(&period)

			// Create completed task
			now := time.Now()
			task := Task{
				FocusPeriodID: period.ID,
				Title:         "Test task",
				Position:      i + 1,
				Completed:     true,
				CompletedAt:   &now,
			}
			db.Create(&task)
		}
	}

	// Get leaderboard
	entries, err := GetAllTimeLeaderboard(guildID, 10)
	if err != nil {
		t.Fatalf("Failed to get leaderboard: %v", err)
	}

	// Should only include users with points > 0
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}

	// Verify ordering (highest points first)
	if len(entries) > 0 && entries[0].Points != 50 {
		t.Errorf("Expected first entry to have 50 points, got %d", entries[0].Points)
	}

	if len(entries) > 0 && entries[0].Rank != 1 {
		t.Errorf("Expected rank 1, got %d", entries[0].Rank)
	}

	if len(entries) > 1 && entries[1].Points != 40 {
		t.Errorf("Expected second entry to have 40 points, got %d", entries[1].Points)
	}

	if len(entries) > 2 && entries[2].Points != 30 {
		t.Errorf("Expected third entry to have 30 points, got %d", entries[2].Points)
	}
}

func TestMarkLeaderboardPosted(t *testing.T) {
	setupTestDB(t)

	guildID := "test-guild-123"

	// Create user and focus period
	user := User{
		DiscordID: "user1",
		GuildID:   guildID,
		Username:  "User1",
	}
	DB.Create(&user)

	startDate := time.Now().Add(-15 * 24 * time.Hour)
	endDate := startDate.Add(14 * 24 * time.Hour)
	period := FocusPeriod{
		UserID:            user.ID,
		GuildID:           guildID,
		StartDate:         startDate,
		EndDate:           endDate,
		LeaderboardPosted: false,
	}
	DB.Create(&period)

	// Mark as posted
	err := MarkLeaderboardPosted(period.ID)
	if err != nil {
		t.Fatalf("Failed to mark leaderboard posted: %v", err)
	}

	// Verify
	var updatedPeriod FocusPeriod
	DB.First(&updatedPeriod, period.ID)
	if !updatedPeriod.LeaderboardPosted {
		t.Error("Expected LeaderboardPosted to be true")
	}
}

func TestGetEndedPeriodsNeedingLeaderboard(t *testing.T) {
	setupTestDB(t)

	guildID := "test-guild-123"

	// Create user
	user := User{
		DiscordID: "user1",
		GuildID:   guildID,
		Username:  "User1",
	}
	DB.Create(&user)

	now := time.Now()

	// Create ended period needing leaderboard
	period1 := FocusPeriod{
		UserID:            user.ID,
		GuildID:           guildID,
		StartDate:         now.Add(-20 * 24 * time.Hour),
		EndDate:           now.Add(-6 * 24 * time.Hour),
		LeaderboardPosted: false,
	}
	DB.Create(&period1)

	// Create ended period already posted
	period2 := FocusPeriod{
		UserID:            user.ID,
		GuildID:           guildID,
		StartDate:         now.Add(-40 * 24 * time.Hour),
		EndDate:           now.Add(-26 * 24 * time.Hour),
		LeaderboardPosted: true,
	}
	DB.Create(&period2)

	// Create active period
	period3 := FocusPeriod{
		UserID:            user.ID,
		GuildID:           guildID,
		StartDate:         now.Add(-5 * 24 * time.Hour),
		EndDate:           now.Add(9 * 24 * time.Hour),
		LeaderboardPosted: false,
	}
	DB.Create(&period3)

	// Get periods needing leaderboard
	periods, err := GetEndedPeriodsNeedingLeaderboard(guildID)
	if err != nil {
		t.Fatalf("Failed to get ended periods: %v", err)
	}

	// Should only return period1
	if len(periods) != 1 {
		t.Errorf("Expected 1 period, got %d", len(periods))
	}

	if len(periods) > 0 && periods[0].ID != period1.ID {
		t.Errorf("Expected period %d, got %d", period1.ID, periods[0].ID)
	}
}
