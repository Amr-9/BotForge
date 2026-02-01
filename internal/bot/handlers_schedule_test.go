package bot_test

import (
	"context"
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/cache"
	"github.com/alicebob/miniredis/v2"
)

// ==================== Schedule State Cache Tests ====================

func setupScheduleTestRedis(t *testing.T) (*cache.Redis, *miniredis.Miniredis, func()) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	redisCache, err := cache.NewRedis(mr.Addr(), "", 0, 48*time.Hour)
	if err != nil {
		mr.Close()
		t.Fatalf("Failed to create Redis client: %v", err)
	}

	cleanup := func() {
		redisCache.Close()
		mr.Close()
	}

	return redisCache, mr, cleanup
}

func TestScheduleState_SetAndGet(t *testing.T) {
	redisCache, _, cleanup := setupScheduleTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(12345)

	// Set state
	err := redisCache.SetScheduleState(ctx, botToken, adminID, "awaiting_message")
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Get state
	state, err := redisCache.GetScheduleState(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if state != "awaiting_message" {
		t.Errorf("Expected 'awaiting_message', got '%s'", state)
	}
}

func TestScheduleState_Empty(t *testing.T) {
	redisCache, _, cleanup := setupScheduleTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	// Get non-existent state
	state, err := redisCache.GetScheduleState(ctx, "nonexistent", 99999)
	if err != nil {
		t.Logf("Error (may be expected): %v", err)
	}
	if state != "" {
		t.Errorf("Expected empty state, got '%s'", state)
	}
}

// ==================== Schedule Message Data Tests ====================

func TestScheduleMessageData_TextMessage(t *testing.T) {
	redisCache, _, cleanup := setupScheduleTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(12345)

	// Set text message data
	err := redisCache.SetScheduleMessageData(ctx, botToken, adminID, "text", "Hello World", "", "")
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Get data
	msgType, msgText, fileID, caption, err := redisCache.GetScheduleMessageData(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if msgType != "text" {
		t.Errorf("Expected 'text', got '%s'", msgType)
	}
	if msgText != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", msgText)
	}
	if fileID != "" {
		t.Errorf("Expected empty fileID, got '%s'", fileID)
	}
	if caption != "" {
		t.Errorf("Expected empty caption, got '%s'", caption)
	}
}

func TestScheduleMessageData_PhotoMessage(t *testing.T) {
	redisCache, _, cleanup := setupScheduleTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(12345)

	// Set photo message data
	err := redisCache.SetScheduleMessageData(ctx, botToken, adminID, "photo", "", "FileID123", "Nice photo!")
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Get data
	msgType, _, fileID, caption, err := redisCache.GetScheduleMessageData(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if msgType != "photo" {
		t.Errorf("Expected 'photo', got '%s'", msgType)
	}
	if fileID != "FileID123" {
		t.Errorf("Expected 'FileID123', got '%s'", fileID)
	}
	if caption != "Nice photo!" {
		t.Errorf("Expected 'Nice photo!', got '%s'", caption)
	}
}

func TestScheduleData_Clear(t *testing.T) {
	redisCache, _, cleanup := setupScheduleTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(12345)

	// Set data
	redisCache.SetScheduleState(ctx, botToken, adminID, "test-state")
	redisCache.SetScheduleMessageData(ctx, botToken, adminID, "text", "Test", "", "")

	// Clear all
	err := redisCache.ClearScheduleData(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	// State should be empty
	state, _ := redisCache.GetScheduleState(ctx, botToken, adminID)
	if state != "" {
		t.Error("State should be cleared")
	}
}

// ==================== Schedule Config Tests ====================

func TestScheduleConfig_SetAndGet(t *testing.T) {
	redisCache, _, cleanup := setupScheduleTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(12345)

	// Set schedule config
	err := redisCache.SetScheduleConfig(ctx, botToken, adminID, "daily", "14:00", "")
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Get schedule config
	schedType, schedTime, _, err := redisCache.GetScheduleConfig(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if schedType != "daily" {
		t.Errorf("Expected 'daily', got '%s'", schedType)
	}
	if schedTime != "14:00" {
		t.Errorf("Expected '14:00', got '%s'", schedTime)
	}
}

func TestScheduleConfig_Weekly(t *testing.T) {
	redisCache, _, cleanup := setupScheduleTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(12345)

	// Set weekly config with day
	redisCache.SetScheduleConfig(ctx, botToken, adminID, "weekly", "10:00", "1")

	schedType, schedTime, day, _ := redisCache.GetScheduleConfig(ctx, botToken, adminID)
	if schedType != "weekly" {
		t.Errorf("Expected 'weekly', got '%s'", schedType)
	}
	if schedTime != "10:00" {
		t.Errorf("Expected '10:00', got '%s'", schedTime)
	}
	if day != "1" {
		t.Errorf("Expected day '1', got '%s'", day)
	}
}

// ==================== Time Calculation Tests ====================

func TestScheduleTimeCalculation_Daily(t *testing.T) {
	// Test daily schedule at 14:00, current time is 10:00
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	scheduleTime := "14:00"

	parsed, err := time.Parse("15:04", scheduleTime)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	nextRun := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
	if nextRun.Before(now) || nextRun.Equal(now) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	// Should be today at 14:00
	if nextRun.Hour() != 14 {
		t.Errorf("Expected 14:00, got %d:%02d", nextRun.Hour(), nextRun.Minute())
	}
	if nextRun.Day() != 1 {
		t.Errorf("Expected day 1, got %d", nextRun.Day())
	}
}

func TestScheduleTimeCalculation_DailyPassed(t *testing.T) {
	// Test daily schedule at 08:00, current time is 10:00 (passed)
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	scheduleTime := "08:00"

	parsed, _ := time.Parse("15:04", scheduleTime)
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
	if nextRun.Before(now) || nextRun.Equal(now) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	// Should be tomorrow
	if nextRun.Day() != 2 {
		t.Errorf("Expected day 2 (tomorrow), got %d", nextRun.Day())
	}
}
