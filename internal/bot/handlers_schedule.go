package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Amr-9/botforge/internal/models"
	"gopkg.in/telebot.v3"
)

// handleScheduleMenu shows the schedule menu
func (m *Manager) handleScheduleMenu(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		menu := &telebot.ReplyMarkup{}
		btnNew := menu.Data("📅 New Scheduled Message", "schedule_new")
		btnList := menu.Data("📋 View Scheduled", "schedule_list")
		btnBack := menu.Data("« Back to Menu", "child_main_menu")

		menu.Inline(
			menu.Row(btnNew),
			menu.Row(btnList),
			menu.Row(btnBack),
		)

		msg := `📅 <b>Schedule Messages</b>

Schedule broadcast messages to be sent automatically at specific times.

<b>Features:</b>
• One-time messages
• Daily recurring messages
• Weekly recurring messages
• Support for text, photos, videos, and documents`

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handleScheduleNewMessage starts the scheduling flow
func (m *Manager) handleScheduleNewMessage(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()

		// Set state
		if err := m.cache.SetUserState(ctx, token, c.Sender().ID, "schedule_awaiting_message"); err != nil {
			return c.Respond(&telebot.CallbackResponse{
				Text:      "Failed to start scheduling",
				ShowAlert: true,
			})
		}

		menu := &telebot.ReplyMarkup{}
		btnCancel := menu.Data("❌ Cancel", "schedule_cancel")
		menu.Inline(menu.Row(btnCancel))

		msg := `📅 <b>Schedule a Broadcast Message</b>

Please send the message you want to schedule.
You can send:
• Text
• Photo (with optional caption)
• Video (with optional caption)
• Document (with optional caption)`

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handleScheduleTypeSelection handles schedule type selection buttons
func (m *Manager) handleScheduleTypeSelection(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		scheduleType := c.Callback().Data // "once", "daily", "weekly"

		// Save schedule type
		if err := m.cache.SetTempData(ctx, token, c.Sender().ID, "schedule_type", scheduleType); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Error", ShowAlert: true})
		}

		menu := &telebot.ReplyMarkup{}
		var msg string

		switch scheduleType {
		case models.ScheduleTypeOnce:
			msg = "⏰ <b>Send Once</b>\n\nSelect when to send:"
			btn1h := menu.Data("🕐 In 1 Hour", "schedule_time_1h")
			btn3h := menu.Data("🕐 In 3 Hours", "schedule_time_3h")
			btn6h := menu.Data("🕐 In 6 Hours", "schedule_time_6h")
			btn12h := menu.Data("🕐 In 12 Hours", "schedule_time_12h")
			btnCustom := menu.Data("⏰ Custom Time", "schedule_time_custom")
			btnBack := menu.Data("« Back", "schedule_new")
			menu.Inline(
				menu.Row(btn1h, btn3h),
				menu.Row(btn6h, btn12h),
				menu.Row(btnCustom),
				menu.Row(btnBack),
			)

		case models.ScheduleTypeDaily:
			msg = "📆 <b>Send Daily</b>\n\nSelect time to send every day:"
			btn6 := menu.Data("🌅 06:00", "schedule_time_daily_06:00")
			btn9 := menu.Data("🌞 09:00", "schedule_time_daily_09:00")
			btn12 := menu.Data("🌤️ 12:00", "schedule_time_daily_12:00")
			btn15 := menu.Data("🌆 15:00", "schedule_time_daily_15:00")
			btn18 := menu.Data("🌙 18:00", "schedule_time_daily_18:00")
			btn21 := menu.Data("🌃 21:00", "schedule_time_daily_21:00")
			btnCustom := menu.Data("⏰ Custom Time", "schedule_time_custom")
			btnBack := menu.Data("« Back", "schedule_new")
			menu.Inline(
				menu.Row(btn6, btn9),
				menu.Row(btn12, btn15),
				menu.Row(btn18, btn21),
				menu.Row(btnCustom),
				menu.Row(btnBack),
			)

		case models.ScheduleTypeWeekly:
			msg = "📅 <b>Send Weekly</b>\n\nSelect the day:"
			btnSun := menu.Data("Sunday", "schedule_day_0")
			btnMon := menu.Data("Monday", "schedule_day_1")
			btnTue := menu.Data("Tuesday", "schedule_day_2")
			btnWed := menu.Data("Wednesday", "schedule_day_3")
			btnThu := menu.Data("Thursday", "schedule_day_4")
			btnFri := menu.Data("Friday", "schedule_day_5")
			btnSat := menu.Data("Saturday", "schedule_day_6")
			btnBack := menu.Data("« Back", "schedule_new")
			menu.Inline(
				menu.Row(btnSun, btnMon),
				menu.Row(btnTue, btnWed),
				menu.Row(btnThu, btnFri),
				menu.Row(btnSat),
				menu.Row(btnBack),
			)
		}

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handleScheduleDaySelection handles day selection for weekly schedules
func (m *Manager) handleScheduleDaySelection(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		day := c.Callback().Data // "0" to "6"

		// Save day
		if err := m.cache.SetTempData(ctx, token, c.Sender().ID, "schedule_day", day); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Error", ShowAlert: true})
		}

		// Show time selection
		menu := &telebot.ReplyMarkup{}
		btn6 := menu.Data("🌅 06:00", "schedule_time_weekly_06:00")
		btn9 := menu.Data("🌞 09:00", "schedule_time_weekly_09:00")
		btn12 := menu.Data("🌤️ 12:00", "schedule_time_weekly_12:00")
		btn15 := menu.Data("🌆 15:00", "schedule_time_weekly_15:00")
		btn18 := menu.Data("🌙 18:00", "schedule_time_weekly_18:00")
		btn21 := menu.Data("🌃 21:00", "schedule_time_weekly_21:00")
		btnCustom := menu.Data("⏰ Custom Time", "schedule_time_custom")
		btnBack := menu.Data("« Back", "schedule_type_weekly")
		menu.Inline(
			menu.Row(btn6, btn9),
			menu.Row(btn12, btn15),
			menu.Row(btn18, btn21),
			menu.Row(btnCustom),
			menu.Row(btnBack),
		)

		dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		dayNum, _ := strconv.Atoi(day)
		dayName := dayNames[dayNum]

		msg := fmt.Sprintf("📅 <b>Send Weekly</b>\n\nDay: <b>%s</b>\n\nSelect time:", dayName)
		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handleScheduleTimeSelection handles time selection
func (m *Manager) handleScheduleTimeSelection(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		data := c.Callback().Data // e.g., "1h", "daily_09:00", "weekly_15:00"

		var scheduledTime time.Time
		var timeOfDay string
		var nextRunAt time.Time

		now := time.Now()

		switch {
		case strings.HasSuffix(data, "h"): // For "once" type: 1h, 3h, 6h, 12h
			hours, _ := strconv.Atoi(strings.TrimSuffix(data, "h"))
			scheduledTime = now.Add(time.Duration(hours) * time.Hour)
			nextRunAt = scheduledTime

		case strings.HasPrefix(data, "daily_"): // For daily: daily_09:00
			timeStr := strings.TrimPrefix(data, "daily_")
			timeOfDay = timeStr + ":00"
			t, _ := time.Parse("15:04:05", timeOfDay)
			scheduledTime = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
			if scheduledTime.Before(now) {
				scheduledTime = scheduledTime.AddDate(0, 0, 1)
			}
			nextRunAt = scheduledTime

		case strings.HasPrefix(data, "weekly_"): // For weekly: weekly_09:00
			timeStr := strings.TrimPrefix(data, "weekly_")
			timeOfDay = timeStr + ":00"
			dayStr, _ := m.cache.GetTempData(ctx, token, c.Sender().ID, "schedule_day")
			dayNum, _ := strconv.Atoi(dayStr)

			t, _ := time.Parse("15:04:05", timeOfDay)
			scheduledTime = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())

			// Calculate days until target weekday
			targetWeekday := time.Weekday(dayNum)
			daysUntil := int(targetWeekday - now.Weekday())
			if daysUntil <= 0 || (daysUntil == 0 && scheduledTime.Before(now)) {
				daysUntil += 7
			}
			scheduledTime = scheduledTime.AddDate(0, 0, daysUntil)
			nextRunAt = scheduledTime
		}

		// Save time config
		m.cache.SetTempData(ctx, token, c.Sender().ID, "schedule_time", scheduledTime.Format("2006-01-02 15:04:05"))
		m.cache.SetTempData(ctx, token, c.Sender().ID, "time_of_day", timeOfDay)
		m.cache.SetTempData(ctx, token, c.Sender().ID, "next_run_at", nextRunAt.Format("2006-01-02 15:04:05"))

		// Show confirmation
		return m.showScheduleConfirmation(c, ctx, bot, token)
	}
}

// showScheduleConfirmation shows the final confirmation screen
func (m *Manager) showScheduleConfirmation(c telebot.Context, ctx context.Context, bot *telebot.Bot, token string) error {
	adminID := c.Sender().ID

	// Get all data
	msgType, msgText, _, caption, _ := m.cache.GetScheduleMessageData(ctx, token, adminID)
	scheduleType, _ := m.cache.GetTempData(ctx, token, adminID, "schedule_type")
	scheduleTimeStr, _ := m.cache.GetTempData(ctx, token, adminID, "schedule_time")
	dayStr, _ := m.cache.GetTempData(ctx, token, adminID, "schedule_day")

	scheduledTime, _ := time.Parse("2006-01-02 15:04:05", scheduleTimeStr)

	// Build preview
	preview := "✅ <b>Message Preview:</b>\n"
	if msgType == models.MessageTypeText {
		if len(msgText) > 100 {
			preview += msgText[:100] + "..."
		} else {
			preview += msgText
		}
	} else {
		preview += fmt.Sprintf("📎 Type: %s", msgType)
		if caption != "" {
			preview += fmt.Sprintf("\nCaption: %s", caption)
		}
	}

	// Build schedule info
	scheduleInfo := "\n\n<b>Schedule:</b> "
	switch scheduleType {
	case models.ScheduleTypeOnce:
		scheduleInfo += fmt.Sprintf("Once at %s", scheduledTime.Format("2006-01-02 15:04"))
	case models.ScheduleTypeDaily:
		scheduleInfo += fmt.Sprintf("Daily at %s", scheduledTime.Format("15:04"))
	case models.ScheduleTypeWeekly:
		dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		dayNum, _ := strconv.Atoi(dayStr)
		scheduleInfo += fmt.Sprintf("Weekly on %s at %s", dayNames[dayNum], scheduledTime.Format("15:04"))
	}

	msg := preview + scheduleInfo + "\n\n<b>Confirm schedule?</b>"

	menu := &telebot.ReplyMarkup{}
	btnConfirm := menu.Data("✅ Confirm & Schedule", "schedule_confirm")
	btnCancel := menu.Data("❌ Cancel", "schedule_cancel")
	menu.Inline(
		menu.Row(btnConfirm),
		menu.Row(btnCancel),
	)

	return c.Edit(msg, menu, telebot.ModeHTML)
}

// handleConfirmSchedule confirms and saves the scheduled message
func (m *Manager) handleConfirmSchedule(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		adminID := c.Sender().ID

		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Get all data
		msgType, msgText, fileID, caption, _ := m.cache.GetScheduleMessageData(ctx, token, adminID)
		scheduleType, _ := m.cache.GetTempData(ctx, token, adminID, "schedule_type")
		scheduleTimeStr, _ := m.cache.GetTempData(ctx, token, adminID, "schedule_time")
		timeOfDay, _ := m.cache.GetTempData(ctx, token, adminID, "time_of_day")
		dayStr, _ := m.cache.GetTempData(ctx, token, adminID, "schedule_day")
		nextRunStr, _ := m.cache.GetTempData(ctx, token, adminID, "next_run_at")

		scheduledTime, _ := time.Parse("2006-01-02 15:04:05", scheduleTimeStr)
		nextRunAt, _ := time.Parse("2006-01-02 15:04:05", nextRunStr)

		var dayOfWeek *int
		if dayStr != "" {
			day, _ := strconv.Atoi(dayStr)
			dayOfWeek = &day
		}

		// Create scheduled message
		msg := &models.ScheduledMessage{
			BotID:         botID,
			OwnerChatID:   adminID,
			MessageType:   msgType,
			MessageText:   msgText,
			FileID:        fileID,
			Caption:       caption,
			ScheduleType:  scheduleType,
			ScheduledTime: scheduledTime,
			TimeOfDay:     timeOfDay,
			DayOfWeek:     dayOfWeek,
			Status:        models.ScheduleStatusPending,
			NextRunAt:     &nextRunAt,
		}

		msgID, err := m.repo.CreateScheduledMessage(ctx, msg)
		if err != nil {
			log.Printf("Failed to create scheduled message: %v", err)
			return c.Respond(&telebot.CallbackResponse{
				Text:      "Failed to schedule message",
				ShowAlert: true,
			})
		}

		// Clear cache
		m.cache.ClearScheduleData(ctx, token, adminID)
		m.cache.ClearUserState(ctx, token, adminID)

		c.Respond(&telebot.CallbackResponse{Text: "✅ Message scheduled!"})

		menu := &telebot.ReplyMarkup{}
		btnView := menu.Data("📋 View Scheduled", "schedule_list")
		btnBack := menu.Data("« Back to Menu", "child_main_menu")
		menu.Inline(
			menu.Row(btnView),
			menu.Row(btnBack),
		)

		successMsg := fmt.Sprintf(`✅ <b>Message Scheduled Successfully!</b>

<b>Message ID:</b> #%d
<b>Type:</b> %s
<b>Schedule:</b> %s

Your message will be broadcast to all users at the scheduled time.`, msgID, scheduleType, nextRunAt.Format("2006-01-02 15:04"))

		return c.Edit(successMsg, menu, telebot.ModeHTML)
	}
}

// handleListScheduledMessages shows list of scheduled messages
func (m *Manager) handleListScheduledMessages(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		messages, err := m.repo.GetScheduledMessagesByBot(ctx, botID, 20, 0)
		if err != nil {
			log.Printf("Failed to get scheduled messages: %v", err)
			return c.Edit("❌ Failed to load scheduled messages", &telebot.ReplyMarkup{})
		}

		if len(messages) == 0 {
			menu := &telebot.ReplyMarkup{}
			btnNew := menu.Data("📅 Schedule New", "schedule_new")
			btnBack := menu.Data("« Back", "child_main_menu")
			menu.Inline(
				menu.Row(btnNew),
				menu.Row(btnBack),
			)
			return c.Edit("📭 <b>No Scheduled Messages</b>\n\nYou don't have any scheduled messages yet.", menu, telebot.ModeHTML)
		}

		msg := fmt.Sprintf("📋 <b>Scheduled Messages (%d active)</b>\n\n", len(messages))

		menu := &telebot.ReplyMarkup{}
		var rows []telebot.Row

		for i, schedMsg := range messages {
			// Build status icon
			statusIcon := "⏳"
			if schedMsg.Status == models.ScheduleStatusPaused {
				statusIcon = "⏸️"
			}

			// Build schedule info
			var scheduleInfo string
			switch schedMsg.ScheduleType {
			case models.ScheduleTypeOnce:
				scheduleInfo = fmt.Sprintf("Once at %s", schedMsg.ScheduledTime.Format("01-02 15:04"))
			case models.ScheduleTypeDaily:
				scheduleInfo = fmt.Sprintf("Daily at %s", schedMsg.ScheduledTime.Format("15:04"))
			case models.ScheduleTypeWeekly:
				dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
				scheduleInfo = fmt.Sprintf("Weekly on %s", dayNames[*schedMsg.DayOfWeek])
			}

			msg += fmt.Sprintf("%d️⃣ %s %s\n   Type: %s | Status: %s\n   Next: %s\n",
				i+1, statusIcon, scheduleInfo, schedMsg.MessageType, schedMsg.Status,
				schedMsg.NextRunAt.Format("2006-01-02 15:04"))

			// Add action buttons
			if schedMsg.Status == models.ScheduleStatusPending {
				btnPause := menu.Data("⏸️", fmt.Sprintf("schedule_pause_%d", schedMsg.ID))
				btnDelete := menu.Data("❌", fmt.Sprintf("schedule_delete_%d", schedMsg.ID))
				rows = append(rows, menu.Row(btnPause, btnDelete))
			} else if schedMsg.Status == models.ScheduleStatusPaused {
				btnResume := menu.Data("▶️", fmt.Sprintf("schedule_resume_%d", schedMsg.ID))
				btnDelete := menu.Data("❌", fmt.Sprintf("schedule_delete_%d", schedMsg.ID))
				rows = append(rows, menu.Row(btnResume, btnDelete))
			}
		}

		btnBack := menu.Data("« Back", "child_main_menu")
		rows = append(rows, menu.Row(btnBack))
		menu.Inline(rows...)

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handlePauseScheduledMessage pauses a scheduled message
func (m *Manager) handlePauseScheduledMessage(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		data := c.Callback().Data
		msgID, _ := strconv.ParseInt(data, 10, 64)

		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		if err := m.repo.PauseScheduledMessage(ctx, msgID, botID); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Failed to pause", ShowAlert: true})
		}

		c.Respond(&telebot.CallbackResponse{Text: "⏸️ Paused"})
		return m.handleListScheduledMessages(bot, token, ownerChat)(c)
	}
}

// handleResumeScheduledMessage resumes a paused message
func (m *Manager) handleResumeScheduledMessage(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		data := c.Callback().Data
		msgID, _ := strconv.ParseInt(data, 10, 64)

		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		if err := m.repo.ResumeScheduledMessage(ctx, msgID, botID); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Failed to resume", ShowAlert: true})
		}

		c.Respond(&telebot.CallbackResponse{Text: "▶️ Resumed"})
		return m.handleListScheduledMessages(bot, token, ownerChat)(c)
	}
}

// handleDeleteScheduledMessage deletes a scheduled message
func (m *Manager) handleDeleteScheduledMessage(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		data := c.Callback().Data
		msgID, _ := strconv.ParseInt(data, 10, 64)

		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		if err := m.repo.DeleteScheduledMessage(ctx, msgID, botID); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Failed to delete", ShowAlert: true})
		}

		c.Respond(&telebot.CallbackResponse{Text: "❌ Deleted"})
		return m.handleListScheduledMessages(bot, token, ownerChat)(c)
	}
}

// handleCancelSchedule cancels the scheduling process
func (m *Manager) handleCancelSchedule(bot *telebot.Bot, token string) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		ctx := context.Background()
		m.cache.ClearScheduleData(ctx, token, c.Sender().ID)
		m.cache.ClearUserState(ctx, token, c.Sender().ID)

		c.Respond(&telebot.CallbackResponse{Text: "Cancelled"})

		menu := &telebot.ReplyMarkup{}
		btnBack := menu.Data("« Back to Menu", "child_main_menu")
		menu.Inline(menu.Row(btnBack))

		return c.Edit("❌ Schedule cancelled", menu)
	}
}

// processScheduleState processes schedule-related states
func (m *Manager) processScheduleState(ctx context.Context, c telebot.Context, token string, state string) (bool, error) {
	if state == "schedule_awaiting_message" {
		msgType := ""
		text := c.Text()
		fileID := ""
		caption := ""

		// Determine message type
		if c.Message().Photo != nil {
			msgType = models.MessageTypePhoto
			fileID = c.Message().Photo.FileID
			caption = c.Message().Caption
		} else if c.Message().Video != nil {
			msgType = models.MessageTypeVideo
			fileID = c.Message().Video.FileID
			caption = c.Message().Caption
		} else if c.Message().Document != nil {
			msgType = models.MessageTypeDocument
			fileID = c.Message().Document.FileID
			caption = c.Message().Caption
		} else if c.Text() != "" {
			msgType = models.MessageTypeText
		} else {
			return true, c.Reply("⚠️ Unsupported message type. Please send text, photo, video, or document.")
		}

		// Validation
		if msgType == models.MessageTypeText && len(text) > 4096 {
			return true, c.Reply("⚠️ Text too long (max 4096 characters)")
		}

		// Save to Redis
		m.cache.SetScheduleMessageData(ctx, token, c.Sender().ID, msgType, text, fileID, caption)
		m.cache.SetUserState(ctx, token, c.Sender().ID, "schedule_select_type")

		// Show type selection
		menu := &telebot.ReplyMarkup{}
		btnOnce := menu.Data("⏰ Once", "schedule_type_once")
		btnDaily := menu.Data("📆 Daily", "schedule_type_daily")
		btnWeekly := menu.Data("📅 Weekly", "schedule_type_weekly")
		btnCancel := menu.Data("❌ Cancel", "schedule_cancel")
		menu.Inline(
			menu.Row(btnOnce),
			menu.Row(btnDaily, btnWeekly),
			menu.Row(btnCancel),
		)

		preview := "✅ Message received!\n\n"
		if msgType == models.MessageTypeText {
			if len(text) > 50 {
				preview += fmt.Sprintf("📝 Text: %s...", text[:50])
			} else {
				preview += fmt.Sprintf("📝 Text: %s", text)
			}
		} else {
			preview += fmt.Sprintf("📎 Type: %s", msgType)
		}

		preview += "\n\n<b>Select schedule type:</b>"

		return true, c.Reply(preview, menu, telebot.ModeHTML)
	}

	return false, nil
}
