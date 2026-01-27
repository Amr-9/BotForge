package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Amr-9/botforge/internal/bot"
	"github.com/Amr-9/botforge/internal/database"
	"github.com/Amr-9/botforge/internal/models"
	"gopkg.in/telebot.v3"
)

// Scheduler handles scheduled message processing
type Scheduler struct {
	repo     *database.Repository
	manager  *bot.Manager
	ticker   *time.Ticker
	stopCh   chan struct{}
	interval time.Duration
}

// NewScheduler creates a new scheduler instance
func NewScheduler(repo *database.Repository, manager *bot.Manager, interval time.Duration) *Scheduler {
	return &Scheduler{
		repo:     repo,
		manager:  manager,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the scheduler loop
func (s *Scheduler) Start() {
	s.ticker = time.NewTicker(s.interval)
	go s.run()
	log.Printf("[Scheduler] Started with interval: %v", s.interval)
}

// Stop halts the scheduler
func (s *Scheduler) Stop() {
	close(s.stopCh)
	if s.ticker != nil {
		s.ticker.Stop()
	}
	log.Println("[Scheduler] Stopped")
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	// Process any pending messages immediately on startup
	s.processPendingMessages()

	for {
		select {
		case <-s.ticker.C:
			s.processPendingMessages()
		case <-s.stopCh:
			return
		}
	}
}

// processPendingMessages retrieves and processes messages ready to send
func (s *Scheduler) processPendingMessages() {
	ctx := context.Background()
	now := time.Now()

	messages, err := s.repo.GetPendingScheduledMessages(ctx, now, 50)
	if err != nil {
		log.Printf("[Scheduler] Failed to get pending messages: %v", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	log.Printf("[Scheduler] Processing %d pending messages", len(messages))

	for _, msg := range messages {
		go s.processMessage(ctx, msg)
	}
}

// processMessage processes a single scheduled message
func (s *Scheduler) processMessage(ctx context.Context, msg models.ScheduledMessage) {
	log.Printf("[Scheduler] Processing message ID=%d, Bot=%d, Type=%s", msg.ID, msg.BotID, msg.ScheduleType)

	// Get bot instance
	botInstance, _, err := s.manager.GetBotByID(msg.BotID)
	if err != nil {
		log.Printf("[Scheduler] Bot not found for ID=%d: %v", msg.BotID, err)
		s.repo.UpdateScheduledMessageStatus(ctx, msg.ID, models.ScheduleStatusFailed, "Bot not running")
		return
	}

	// Get all user chat IDs
	userIDs, err := s.repo.GetAllUserChatIDs(ctx, msg.BotID)
	if err != nil {
		log.Printf("[Scheduler] Failed to get users: %v", err)
		s.repo.UpdateScheduledMessageStatus(ctx, msg.ID, models.ScheduleStatusFailed, err.Error())
		return
	}

	if len(userIDs) == 0 {
		log.Printf("[Scheduler] No users found for bot ID=%d", msg.BotID)
		s.repo.UpdateScheduledMessageStatus(ctx, msg.ID, models.ScheduleStatusSent, "No users")
		s.notifyAdmin(botInstance, msg.OwnerChatID, &msg, 0, 0)
		return
	}

	// Broadcast the message
	success, failed := s.broadcastMessage(botInstance, &msg, userIDs)
	now := time.Now()

	log.Printf("[Scheduler] Message ID=%d sent. Success=%d, Failed=%d", msg.ID, success, failed)

	// Update status based on schedule type
	if msg.ScheduleType == models.ScheduleTypeOnce {
		s.repo.UpdateScheduledMessageStatus(ctx, msg.ID, models.ScheduleStatusSent, "")
	} else {
		// Calculate next run time for recurring messages
		nextRun := s.calculateNextRun(&msg, now)
		s.repo.UpdateScheduledMessageAfterSend(ctx, msg.ID, now, nextRun)
	}

	// Notify admin
	s.notifyAdmin(botInstance, msg.OwnerChatID, &msg, success, failed)
}

// broadcastMessage sends the message to all users
func (s *Scheduler) broadcastMessage(bot *telebot.Bot, msg *models.ScheduledMessage, userIDs []int64) (int, int) {
	success := 0
	failed := 0

	for _, userID := range userIDs {
		if userID == msg.OwnerChatID {
			continue // Don't send to admin
		}

		userChat := &telebot.Chat{ID: userID}
		var err error

		switch msg.MessageType {
		case models.MessageTypeText:
			_, err = bot.Send(userChat, msg.MessageText, telebot.ModeMarkdown)

		case models.MessageTypePhoto:
			photo := &telebot.Photo{
				File:    telebot.File{FileID: msg.FileID},
				Caption: msg.Caption,
			}
			_, err = bot.Send(userChat, photo, telebot.ModeMarkdown)

		case models.MessageTypeVideo:
			video := &telebot.Video{
				File:    telebot.File{FileID: msg.FileID},
				Caption: msg.Caption,
			}
			_, err = bot.Send(userChat, video, telebot.ModeMarkdown)

		case models.MessageTypeDocument:
			doc := &telebot.Document{
				File:    telebot.File{FileID: msg.FileID},
				Caption: msg.Caption,
			}
			_, err = bot.Send(userChat, doc, telebot.ModeMarkdown)
		}

		if err != nil {
			failed++
		} else {
			success++
		}

		// Rate limiting - 40ms between messages (25 msg/sec)
		time.Sleep(40 * time.Millisecond)
	}

	return success, failed
}

// calculateNextRun calculates the next execution time for recurring messages
func (s *Scheduler) calculateNextRun(msg *models.ScheduledMessage, from time.Time) *time.Time {
	var next time.Time

	switch msg.ScheduleType {
	case models.ScheduleTypeDaily:
		// Parse time_of_day
		t, err := time.Parse("15:04:05", msg.TimeOfDay)
		if err != nil {
			log.Printf("[Scheduler] Failed to parse time_of_day: %v", err)
			return nil
		}

		next = time.Date(from.Year(), from.Month(), from.Day(),
			t.Hour(), t.Minute(), t.Second(), 0, from.Location())

		// If already passed today, schedule for tomorrow
		if next.Before(from) || next.Equal(from) {
			next = next.AddDate(0, 0, 1)
		}

	case models.ScheduleTypeWeekly:
		if msg.DayOfWeek == nil {
			log.Printf("[Scheduler] DayOfWeek is nil for weekly message ID=%d", msg.ID)
			return nil
		}

		t, err := time.Parse("15:04:05", msg.TimeOfDay)
		if err != nil {
			log.Printf("[Scheduler] Failed to parse time_of_day: %v", err)
			return nil
		}

		targetWeekday := time.Weekday(*msg.DayOfWeek)
		next = time.Date(from.Year(), from.Month(), from.Day(),
			t.Hour(), t.Minute(), t.Second(), 0, from.Location())

		// Calculate days until target weekday
		daysUntil := int(targetWeekday - from.Weekday())
		if daysUntil <= 0 || (daysUntil == 0 && next.Before(from)) {
			daysUntil += 7
		}
		next = next.AddDate(0, 0, daysUntil)
	}

	return &next
}

// notifyAdmin sends a delivery report to the admin
func (s *Scheduler) notifyAdmin(bot *telebot.Bot, adminID int64, msg *models.ScheduledMessage, success, failed int) {
	adminChat := &telebot.Chat{ID: adminID}

	scheduleInfo := ""
	switch msg.ScheduleType {
	case models.ScheduleTypeOnce:
		scheduleInfo = "One-time message"
	case models.ScheduleTypeDaily:
		scheduleInfo = "Daily recurring"
	case models.ScheduleTypeWeekly:
		scheduleInfo = "Weekly recurring"
	}

	report := fmt.Sprintf(`📢 <b>Scheduled Message Delivered</b>

📋 <b>Schedule:</b> %s
✅ <b>Success:</b> %d
❌ <b>Failed:</b> %d
👥 <b>Total:</b> %d`,
		scheduleInfo, success, failed, success+failed)

	bot.Send(adminChat, report, telebot.ModeHTML)
}
