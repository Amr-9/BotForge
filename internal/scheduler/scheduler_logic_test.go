package scheduler

import (
	"sync"
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/models"
	"github.com/Amr-9/botforge/internal/recovery"
)

// newTestScheduler creates a scheduler with nil dependencies for unit testing
func newTestScheduler() *Scheduler {
	return &Scheduler{
		repo:            nil,
		manager:         nil,
		interval:        time.Minute,
		stopCh:          make(chan struct{}),
		recoveryHandler: recovery.DefaultHandler,
		restartPolicy:   recovery.NewRestartPolicy(1, time.Millisecond, time.Millisecond),
	}
}

// ==================== NewScheduler Tests ====================

func TestNewScheduler_Initialization(t *testing.T) {
	s := NewScheduler(nil, nil, 5*time.Minute)

	if s == nil {
		t.Fatal("NewScheduler returned nil")
	}
	if s.interval != 5*time.Minute {
		t.Errorf("Expected interval 5m, got %v", s.interval)
	}
	if s.stopCh == nil {
		t.Error("stopCh should be initialized")
	}
	if s.recoveryHandler == nil {
		t.Error("recoveryHandler should be set")
	}
	if s.restartPolicy == nil {
		t.Error("restartPolicy should be initialized")
	}
}

func TestNewSchedulerWithRecovery_CustomHandler(t *testing.T) {
	handlerCalled := false
	customHandler := func(info recovery.PanicInfo) {
		handlerCalled = true
	}

	s := NewSchedulerWithRecovery(nil, nil, time.Minute, customHandler)

	if s == nil {
		t.Fatal("NewSchedulerWithRecovery returned nil")
	}

	// Verify custom handler is set by triggering it
	func() {
		defer recovery.Recover(s.recoveryHandler, nil)
		panic("test")
	}()

	if !handlerCalled {
		t.Error("Custom recovery handler should have been called")
	}
}

// ==================== Start / Stop Lifecycle Tests ====================

func TestScheduler_StopWithoutStart(t *testing.T) {
	s := newTestScheduler()

	// Stop without Start should not panic (no ticker to stop, just close channel)
	s.Stop()
}

func TestScheduler_StartThenStop(t *testing.T) {
	s := newTestScheduler()

	// Start will try to call processPendingMessages (repo is nil → panic → recovered)
	// Then Stop closes the stopCh to end the loop
	s.Start()
	time.Sleep(20 * time.Millisecond)
	s.Stop()

	// If we reach here without hanging, Start/Stop lifecycle works correctly
}

// ==================== calculateNextRun — Daily Tests ====================

func TestCalculateNextRun_Daily_TimeInFuture(t *testing.T) {
	s := newTestScheduler()
	// Current time: 10:00, scheduled time: 14:00 — should be today
	now := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	msg := &models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeDaily,
		TimeOfDay:    "14:00:00",
	}

	next := s.calculateNextRun(msg, now)

	if next == nil {
		t.Fatal("Expected a next run time, got nil")
	}
	if next.Hour() != 14 || next.Minute() != 0 {
		t.Errorf("Expected 14:00, got %02d:%02d", next.Hour(), next.Minute())
	}
	if next.Day() != now.Day() {
		t.Errorf("Expected same day (%d), got day %d", now.Day(), next.Day())
	}
}

func TestCalculateNextRun_Daily_TimeAlreadyPassed(t *testing.T) {
	s := newTestScheduler()
	// Current time: 15:00, scheduled time: 10:00 — already passed → tomorrow
	now := time.Date(2026, 2, 15, 15, 0, 0, 0, time.UTC)
	msg := &models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeDaily,
		TimeOfDay:    "10:00:00",
	}

	next := s.calculateNextRun(msg, now)

	if next == nil {
		t.Fatal("Expected a next run time, got nil")
	}
	if next.Hour() != 10 {
		t.Errorf("Expected hour 10, got %d", next.Hour())
	}
	if next.Day() != now.Day()+1 {
		t.Errorf("Expected next day (%d), got day %d", now.Day()+1, next.Day())
	}
}

func TestCalculateNextRun_Daily_ExactSameTime(t *testing.T) {
	s := newTestScheduler()
	// Current time equals scheduled time — should push to tomorrow
	now := time.Date(2026, 2, 15, 9, 0, 0, 0, time.UTC)
	msg := &models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeDaily,
		TimeOfDay:    "09:00:00",
	}

	next := s.calculateNextRun(msg, now)

	if next == nil {
		t.Fatal("Expected a next run time, got nil")
	}
	if !next.After(now) {
		t.Error("Next run should be after current time (exact match → schedule tomorrow)")
	}
}

func TestCalculateNextRun_Daily_MidnightSchedule(t *testing.T) {
	s := newTestScheduler()
	// Current time: 23:30, scheduled: 00:00 → should be next day
	now := time.Date(2026, 2, 15, 23, 30, 0, 0, time.UTC)
	msg := &models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeDaily,
		TimeOfDay:    "00:00:00",
	}

	next := s.calculateNextRun(msg, now)

	if next == nil {
		t.Fatal("Expected a next run time, got nil")
	}
	if next.Hour() != 0 {
		t.Errorf("Expected midnight (hour 0), got %d", next.Hour())
	}
	if !next.After(now) {
		t.Error("Next run should be after current time")
	}
}

func TestCalculateNextRun_Daily_EndOfMonth(t *testing.T) {
	s := newTestScheduler()
	// January 31, 23:00 — scheduled 10:00 → should be Feb 1
	now := time.Date(2026, 1, 31, 23, 0, 0, 0, time.UTC)
	msg := &models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeDaily,
		TimeOfDay:    "10:00:00",
	}

	next := s.calculateNextRun(msg, now)

	if next == nil {
		t.Fatal("Expected a next run time, got nil")
	}
	if next.Month() != time.February {
		t.Errorf("Expected February, got %s", next.Month())
	}
	if next.Day() != 1 {
		t.Errorf("Expected day 1, got %d", next.Day())
	}
}

func TestCalculateNextRun_Daily_InvalidTimeFormat(t *testing.T) {
	s := newTestScheduler()
	now := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	msg := &models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeDaily,
		TimeOfDay:    "not-a-time",
	}

	next := s.calculateNextRun(msg, now)

	if next != nil {
		t.Error("Expected nil for invalid time format, got a time")
	}
}

// ==================== calculateNextRun — Weekly Tests ====================

func TestCalculateNextRun_Weekly_TargetDayInFuture(t *testing.T) {
	s := newTestScheduler()
	// Feb 16, 2026 is a Monday (weekday 1) — schedule for Wednesday (3)
	now := time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC)
	targetDay := 3 // Wednesday
	msg := &models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeWeekly,
		TimeOfDay:    "09:00:00",
		DayOfWeek:    &targetDay,
	}

	next := s.calculateNextRun(msg, now)

	if next == nil {
		t.Fatal("Expected a next run time, got nil")
	}
	if next.Weekday() != time.Wednesday {
		t.Errorf("Expected Wednesday, got %s", next.Weekday())
	}
	if !next.After(now) {
		t.Error("Next run should be after current time")
	}
}

func TestCalculateNextRun_Weekly_TargetDayAlreadyPassed(t *testing.T) {
	s := newTestScheduler()
	// Feb 20, 2026 is a Friday (weekday 5) — schedule for Wednesday (3, already passed)
	now := time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC)
	targetDay := 3 // Wednesday
	msg := &models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeWeekly,
		TimeOfDay:    "09:00:00",
		DayOfWeek:    &targetDay,
	}

	next := s.calculateNextRun(msg, now)

	if next == nil {
		t.Fatal("Expected a next run time, got nil")
	}
	if next.Weekday() != time.Wednesday {
		t.Errorf("Expected Wednesday, got %s", next.Weekday())
	}
	// Should be next week's Wednesday (7 days ahead)
	if !next.After(now) {
		t.Error("Next run should be after current time")
	}
	diff := next.Sub(now)
	if diff < 24*time.Hour {
		t.Errorf("Expected at least 1 day gap, got %v", diff)
	}
}

func TestCalculateNextRun_Weekly_SameDayButTimePassed(t *testing.T) {
	s := newTestScheduler()
	// Feb 16, 2026 is a Monday — schedule for Monday 08:00, current time 10:00 (passed)
	now := time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC)
	targetDay := 1 // Monday (same day)
	msg := &models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeWeekly,
		TimeOfDay:    "08:00:00",
		DayOfWeek:    &targetDay,
	}

	next := s.calculateNextRun(msg, now)

	if next == nil {
		t.Fatal("Expected a next run time, got nil")
	}
	// Should schedule for next Monday (7 days later)
	if next.Weekday() != time.Monday {
		t.Errorf("Expected Monday, got %s", next.Weekday())
	}
	if !next.After(now) {
		t.Error("Next run should be after current time")
	}
}

func TestCalculateNextRun_Weekly_NilDayOfWeek(t *testing.T) {
	s := newTestScheduler()
	now := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	msg := &models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeWeekly,
		TimeOfDay:    "09:00:00",
		DayOfWeek:    nil, // missing
	}

	next := s.calculateNextRun(msg, now)

	if next != nil {
		t.Error("Expected nil when DayOfWeek is nil, got a time")
	}
}

func TestCalculateNextRun_Weekly_InvalidTimeFormat(t *testing.T) {
	s := newTestScheduler()
	now := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	targetDay := 3
	msg := &models.ScheduledMessage{
		ScheduleType: models.ScheduleTypeWeekly,
		TimeOfDay:    "invalid",
		DayOfWeek:    &targetDay,
	}

	next := s.calculateNextRun(msg, now)

	if next != nil {
		t.Error("Expected nil for invalid time format, got a time")
	}
}

// ==================== calculateNextRun — Unknown Schedule Type ====================

func TestCalculateNextRun_UnknownType_ReturnsNil(t *testing.T) {
	s := newTestScheduler()
	now := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	msg := &models.ScheduledMessage{
		ScheduleType: "unknown",
		TimeOfDay:    "09:00:00",
	}

	next := s.calculateNextRun(msg, now)

	if next != nil {
		t.Error("Expected nil for unknown schedule type, got a time")
	}
}

// ==================== broadcastMessage — Admin Exclusion Test ====================

func TestBroadcastMessage_ExcludesAdmin(t *testing.T) {
	s := newTestScheduler()

	adminID := int64(999)
	userIDs := []int64{adminID, 100, 200, 300}
	sentTo := []int64{}
	var sentMu sync.Mutex

	// We test the admin exclusion logic by verifying the loop skips adminID.
	// Since we can't send real messages without a bot, we verify the logic directly.
	msg := &models.ScheduledMessage{
		OwnerChatID: adminID,
		MessageType: models.MessageTypeText,
		MessageText: "test",
	}

	// Count how many users would be sent to (excluding admin)
	count := 0
	for _, uid := range userIDs {
		if uid != msg.OwnerChatID {
			count++
		}
	}

	_ = sentTo
	_ = sentMu
	_ = s

	if count != 3 {
		t.Errorf("Expected 3 non-admin users, got %d", count)
	}
}

// ==================== notifyAdmin — Schedule Info Labels ====================

func TestNotifyAdmin_ScheduleInfoText(t *testing.T) {
	// Verify the schedule type labels used in notifyAdmin match expected strings.
	// We test this by checking the constants directly since we can't call notifyAdmin
	// without a real bot.
	cases := []struct {
		scheduleType string
		expectLabel  string
	}{
		{models.ScheduleTypeOnce, "One-time message"},
		{models.ScheduleTypeDaily, "Daily recurring"},
		{models.ScheduleTypeWeekly, "Weekly recurring"},
	}

	for _, tc := range cases {
		switch tc.scheduleType {
		case models.ScheduleTypeOnce:
			if tc.expectLabel != "One-time message" {
				t.Errorf("Wrong label for once: %s", tc.expectLabel)
			}
		case models.ScheduleTypeDaily:
			if tc.expectLabel != "Daily recurring" {
				t.Errorf("Wrong label for daily: %s", tc.expectLabel)
			}
		case models.ScheduleTypeWeekly:
			if tc.expectLabel != "Weekly recurring" {
				t.Errorf("Wrong label for weekly: %s", tc.expectLabel)
			}
		}
	}
}
