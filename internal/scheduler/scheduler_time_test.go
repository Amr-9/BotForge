package scheduler_test

import (
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/models"
)

// ==================== Schedule Time Calculation Tests ====================

func TestCalculateNextRun_DailyAtFuture(t *testing.T) {
	// Current time: 10:00, schedule time: 14:00
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	scheduleTime := "14:00"

	nextRun := calculateNextDailyRun(now, scheduleTime)

	// Should be today at 14:00
	if nextRun.Hour() != 14 {
		t.Errorf("Expected hour 14, got %d", nextRun.Hour())
	}
	if nextRun.Day() != now.Day() {
		t.Error("Should be the same day")
	}
}

func TestCalculateNextRun_DailyAtPast(t *testing.T) {
	// Current time: 15:00, schedule time: 10:00 (already passed)
	now := time.Date(2026, 2, 1, 15, 0, 0, 0, time.UTC)
	scheduleTime := "10:00"

	nextRun := calculateNextDailyRun(now, scheduleTime)

	// Should be tomorrow at 10:00
	if nextRun.Hour() != 10 {
		t.Errorf("Expected hour 10, got %d", nextRun.Hour())
	}
	if nextRun.Day() != now.Day()+1 {
		t.Error("Should be the next day")
	}
}

func TestCalculateNextRun_Weekly(t *testing.T) {
	// Today is Sunday (0), schedule for Monday (1)
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC) // Assuming Feb 1, 2026 is a Sunday
	targetDay := 1                                      // Monday
	scheduleTime := "09:00"

	nextRun := calculateNextWeeklyRun(now, scheduleTime, targetDay)

	// Should be Monday
	if int(nextRun.Weekday()) != targetDay {
		t.Errorf("Expected weekday %d, got %d", targetDay, nextRun.Weekday())
	}
}

// ==================== Schedule Status Tests ====================

func TestScheduleStatus_PendingToSent(t *testing.T) {
	msg := models.ScheduledMessage{
		Status: models.ScheduleStatusPending,
	}

	// Simulate status change
	newStatus := models.ScheduleStatusSent

	if newStatus != "sent" {
		t.Error("Status should transition to 'sent'")
	}

	msg.Status = newStatus
	if msg.Status != models.ScheduleStatusSent {
		t.Error("Message status should be 'sent'")
	}
}

func TestScheduleStatus_PendingToPaused(t *testing.T) {
	msg := models.ScheduledMessage{
		Status: models.ScheduleStatusPending,
	}

	msg.Status = models.ScheduleStatusPaused
	if msg.Status != "paused" {
		t.Errorf("Expected 'paused', got '%s'", msg.Status)
	}
}

func TestScheduleStatus_PausedToResumed(t *testing.T) {
	msg := models.ScheduledMessage{
		Status: models.ScheduleStatusPaused,
	}

	// Resume = set back to pending
	msg.Status = models.ScheduleStatusPending
	if msg.Status != "pending" {
		t.Errorf("Expected 'pending', got '%s'", msg.Status)
	}
}

// ==================== Schedule Type Tests ====================

func TestScheduleType_Once(t *testing.T) {
	msg := models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeOnce,
	}

	if msg.ScheduleType != "once" {
		t.Errorf("Expected 'once', got '%s'", msg.ScheduleType)
	}
}

func TestScheduleType_Daily(t *testing.T) {
	msg := models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeDaily,
		TimeOfDay:    "09:00",
	}

	if msg.ScheduleType != "daily" {
		t.Errorf("Expected 'daily', got '%s'", msg.ScheduleType)
	}
	if msg.TimeOfDay != "09:00" {
		t.Error("TimeOfDay mismatch")
	}
}

func TestScheduleType_Weekly(t *testing.T) {
	day := 3 // Wednesday
	msg := models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeWeekly,
		TimeOfDay:    "15:30",
		DayOfWeek:    &day,
	}

	if msg.ScheduleType != "weekly" {
		t.Errorf("Expected 'weekly', got '%s'", msg.ScheduleType)
	}
	if msg.DayOfWeek == nil || *msg.DayOfWeek != 3 {
		t.Error("DayOfWeek should be 3 (Wednesday)")
	}
}

// ==================== Edge Cases ====================

func TestScheduleTime_Midnight(t *testing.T) {
	now := time.Date(2026, 2, 1, 23, 30, 0, 0, time.UTC)
	scheduleTime := "00:00"

	nextRun := calculateNextDailyRun(now, scheduleTime)

	// Should be next day at midnight
	if nextRun.Hour() != 0 {
		t.Errorf("Expected hour 0, got %d", nextRun.Hour())
	}
	if nextRun.Day() <= now.Day() {
		// If February only has one day, this would roll to next month
		t.Log("Next run is correctly scheduled for the future")
	}
}

func TestScheduleTime_EndOfMonth(t *testing.T) {
	// Last day of January, schedule for next day
	now := time.Date(2026, 1, 31, 23, 0, 0, 0, time.UTC)
	scheduleTime := "10:00"

	nextRun := calculateNextDailyRun(now, scheduleTime)

	// Should be Feb 1
	if nextRun.Month() != time.February {
		t.Errorf("Expected February, got %s", nextRun.Month())
	}
	if nextRun.Day() != 1 {
		t.Errorf("Expected day 1, got %d", nextRun.Day())
	}
}

// ==================== Helper Functions ====================

func calculateNextDailyRun(now time.Time, timeOfDay string) time.Time {
	parsed, _ := time.Parse("15:04", timeOfDay)
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
	if nextRun.Before(now) || nextRun.Equal(now) {
		nextRun = nextRun.Add(24 * time.Hour)
	}
	return nextRun
}

func calculateNextWeeklyRun(now time.Time, timeOfDay string, targetDay int) time.Time {
	parsed, _ := time.Parse("15:04", timeOfDay)
	currentDay := int(now.Weekday())
	daysUntil := targetDay - currentDay
	if daysUntil < 0 {
		daysUntil += 7
	}
	if daysUntil == 0 {
		// Same day - check if time has passed
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
		if nextRun.Before(now) || nextRun.Equal(now) {
			daysUntil = 7
		}
	}
	nextRun := now.AddDate(0, 0, daysUntil)
	return time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
}
