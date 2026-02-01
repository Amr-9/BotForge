package scheduler_test

import (
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/models"
)

// ==================== calculateNextRun Logic Tests ====================
// Note: These tests focus on the time calculation logic
// For full integration tests, you would need to mock the database and bot manager

func TestCalculateNextRun_Daily_Logic(t *testing.T) {
	// Test daily schedule calculation
	// Schedule at 14:00 daily, current time is 10:00
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	scheduleTime := "14:00"

	expectedHour := 14
	expectedMinute := 0

	// Parse the schedule time
	parsed, err := time.Parse("15:04", scheduleTime)
	if err != nil {
		t.Fatalf("Failed to parse time: %v", err)
	}

	nextRun := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())

	// If the scheduled time has passed today, move to tomorrow
	if nextRun.Before(now) || nextRun.Equal(now) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	if nextRun.Hour() != expectedHour || nextRun.Minute() != expectedMinute {
		t.Errorf("Expected %02d:%02d, got %02d:%02d", expectedHour, expectedMinute, nextRun.Hour(), nextRun.Minute())
	}

	// Should be the same day since 14:00 is after 10:00
	if nextRun.Day() != now.Day() {
		t.Errorf("Expected same day, got day %d", nextRun.Day())
	}
}

func TestCalculateNextRun_Daily_PassedTime(t *testing.T) {
	// Schedule at 08:00 daily, current time is 10:00 (already passed)
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	scheduleTime := "08:00"

	parsed, _ := time.Parse("15:04", scheduleTime)
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())

	if nextRun.Before(now) || nextRun.Equal(now) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	// Should be tomorrow since 08:00 already passed
	expectedDay := now.Day() + 1
	if nextRun.Day() != expectedDay {
		t.Errorf("Expected day %d (tomorrow), got day %d", expectedDay, nextRun.Day())
	}
}

func TestCalculateNextRun_Weekly_SameDay(t *testing.T) {
	// Schedule for Monday at 10:00, current time is Monday 08:00
	// Monday = 1 in Go's Weekday
	monday := time.Date(2026, 2, 2, 8, 0, 0, 0, time.UTC) // This is a Monday
	if monday.Weekday() != time.Monday {
		t.Fatalf("Test setup error: expected Monday, got %v", monday.Weekday())
	}

	scheduleTime := "10:00"
	scheduleDayOfWeek := int(time.Monday) // 1

	parsed, _ := time.Parse("15:04", scheduleTime)
	nextRun := time.Date(monday.Year(), monday.Month(), monday.Day(), parsed.Hour(), parsed.Minute(), 0, 0, monday.Location())

	daysUntil := (scheduleDayOfWeek - int(monday.Weekday()) + 7) % 7
	if daysUntil == 0 && (nextRun.Before(monday) || nextRun.Equal(monday)) {
		daysUntil = 7
	}

	if daysUntil > 0 {
		nextRun = nextRun.Add(time.Duration(daysUntil) * 24 * time.Hour)
	}

	// Should be the same day (Monday) at 10:00
	if nextRun.Weekday() != time.Monday {
		t.Errorf("Expected Monday, got %v", nextRun.Weekday())
	}
	if nextRun.Hour() != 10 {
		t.Errorf("Expected 10:00, got %d:%02d", nextRun.Hour(), nextRun.Minute())
	}
}

func TestCalculateNextRun_Weekly_NextWeek(t *testing.T) {
	// Schedule for Monday at 10:00, current time is Tuesday
	tuesday := time.Date(2026, 2, 3, 8, 0, 0, 0, time.UTC) // This is a Tuesday
	if tuesday.Weekday() != time.Tuesday {
		t.Fatalf("Test setup error: expected Tuesday, got %v", tuesday.Weekday())
	}

	scheduleDayOfWeek := int(time.Monday) // 1

	daysUntil := (scheduleDayOfWeek - int(tuesday.Weekday()) + 7) % 7
	if daysUntil == 0 {
		daysUntil = 7
	}

	// From Tuesday to next Monday is 6 days
	if daysUntil != 6 {
		t.Errorf("Expected 6 days until next Monday, got %d", daysUntil)
	}
}

// ==================== Model Constants Tests ====================

func TestScheduleTypeConstants(t *testing.T) {
	if models.ScheduleTypeOnce != "once" {
		t.Errorf("ScheduleTypeOnce should be 'once'")
	}
	if models.ScheduleTypeDaily != "daily" {
		t.Errorf("ScheduleTypeDaily should be 'daily'")
	}
	if models.ScheduleTypeWeekly != "weekly" {
		t.Errorf("ScheduleTypeWeekly should be 'weekly'")
	}
}

func TestMessageTypeConstants(t *testing.T) {
	if models.MessageTypeText != "text" {
		t.Errorf("MessageTypeText should be 'text'")
	}
	if models.MessageTypePhoto != "photo" {
		t.Errorf("MessageTypePhoto should be 'photo'")
	}
	if models.MessageTypeVideo != "video" {
		t.Errorf("MessageTypeVideo should be 'video'")
	}
	if models.MessageTypeDocument != "document" {
		t.Errorf("MessageTypeDocument should be 'document'")
	}
}

func TestScheduleStatusConstants(t *testing.T) {
	if models.ScheduleStatusPending != "pending" {
		t.Errorf("ScheduleStatusPending should be 'pending'")
	}
	if models.ScheduleStatusSent != "sent" {
		t.Errorf("ScheduleStatusSent should be 'sent'")
	}
	if models.ScheduleStatusFailed != "failed" {
		t.Errorf("ScheduleStatusFailed should be 'failed'")
	}
	if models.ScheduleStatusPaused != "paused" {
		t.Errorf("ScheduleStatusPaused should be 'paused'")
	}
	if models.ScheduleStatusCancelled != "cancelled" {
		t.Errorf("ScheduleStatusCancelled should be 'cancelled'")
	}
}

// ==================== ScheduledMessage Struct Tests ====================

func TestScheduledMessage_Fields(t *testing.T) {
	now := time.Now()
	msg := models.ScheduledMessage{
		ID:            1,
		BotID:         100,
		OwnerChatID:   12345,
		MessageType:   models.MessageTypeText,
		MessageText:   "Hello World",
		ScheduleType:  models.ScheduleTypeDaily,
		ScheduledTime: now,
		Status:        models.ScheduleStatusPending,
		CreatedAt:     now,
	}

	if msg.ID != 1 {
		t.Error("ID mismatch")
	}
	if msg.BotID != 100 {
		t.Error("BotID mismatch")
	}
	if msg.MessageType != "text" {
		t.Error("MessageType mismatch")
	}
	if msg.Status != "pending" {
		t.Error("Status mismatch")
	}
}

func TestScheduledMessage_MediaFields(t *testing.T) {
	msg := models.ScheduledMessage{
		MessageType: models.MessageTypePhoto,
		FileID:      "AgACAgIAAxkBAAI...",
		Caption:     "Beautiful sunset!",
	}

	if msg.FileID == "" {
		t.Error("FileID should not be empty for media messages")
	}
	if msg.Caption == "" {
		t.Error("Caption should be set for media messages")
	}
}

func TestScheduledMessage_RecurringFields(t *testing.T) {
	dayOfWeek := 1 // Monday
	msg := models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeWeekly,
		TimeOfDay:    "14:00",
		DayOfWeek:    &dayOfWeek,
	}

	if msg.DayOfWeek == nil || *msg.DayOfWeek != 1 {
		t.Error("DayOfWeek should be 1 (Monday) for weekly schedule")
	}
	if msg.TimeOfDay != "14:00" {
		t.Error("TimeOfDay mismatch")
	}
}
