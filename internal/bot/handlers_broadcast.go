package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
)

// handleChildBroadcast initiates broadcast mode
func (m *Manager) handleChildBroadcast(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		if err := m.cache.SetBroadcastMode(ctx, token, c.Sender().ID); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Failed to start broadcast mode", ShowAlert: true})
		}

		menu := &telebot.ReplyMarkup{}
		btnCancel := menu.Data("❌ Cancel Broadcast", "cancel_broadcast")
		menu.Inline(menu.Row(btnCancel))

		return c.Edit("📢 <b>Broadcast Mode</b>\n\nSend the message you want to broadcast to all users.\nYou can send text, photos, videos, etc.", menu, telebot.ModeHTML)
	}
}

// handleCancelBroadcast cancels broadcast mode
func (m *Manager) handleCancelBroadcast(bot *telebot.Bot, token string) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		ctx := context.Background()
		m.cache.ClearBroadcastMode(ctx, token, c.Sender().ID)
		m.cache.ClearPendingBroadcast(ctx, token, c.Sender().ID)

		menu := &telebot.ReplyMarkup{}
		btnStats := menu.Data("📊 Statistics", "child_stats")
		btnBroadcast := menu.Data("📢 Broadcast", "child_broadcast")
		btnSchedule := menu.Data("📅 Schedule Message", "schedule_menu")
		btnSettings := menu.Data("⚙️ Settings", "child_settings")
		menu.Inline(
			menu.Row(btnStats),
			menu.Row(btnBroadcast),
			menu.Row(btnSchedule),
			menu.Row(btnSettings),
		)

		return c.Edit("🤖 <b>Bot Admin Panel</b>\n\nSelect an option:", menu, telebot.ModeHTML)
	}
}

// requestBroadcastConfirmation shows confirmation before broadcasting
func (m *Manager) requestBroadcastConfirmation(ctx context.Context, c telebot.Context, _ *telebot.Bot, token string) error {
	// Save the message ID for later
	if err := m.cache.SetPendingBroadcast(ctx, token, c.Sender().ID, c.Message().ID); err != nil {
		return c.Reply("❌ Failed to prepare broadcast.")
	}

	menu := &telebot.ReplyMarkup{}
	btnConfirm := menu.Data("✅ Confirm Send", "confirm_broadcast")
	btnCancel := menu.Data("❌ Cancel", "cancel_broadcast")
	menu.Inline(
		menu.Row(btnConfirm, btnCancel),
	)

	return c.Reply("⚠️ <b>Confirm Broadcast</b>\n\nAre you sure you want to send this message to all users?", menu, telebot.ModeHTML)
}

// handleConfirmBroadcast executes the broadcast after confirmation
func (m *Manager) handleConfirmBroadcast(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()

		// Get the pending broadcast message ID
		msgID, err := m.cache.GetPendingBroadcast(ctx, token, c.Sender().ID)
		if err != nil || msgID == 0 {
			return c.Edit("❌ No pending broadcast found. Please start again.")
		}

		// Clear pending state
		m.cache.ClearPendingBroadcast(ctx, token, c.Sender().ID)
		m.cache.ClearBroadcastMode(ctx, token, c.Sender().ID)

		c.Edit("⏳ Starting broadcast. This may take a while...")

		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		userIDs, err := m.repo.GetAllUserChatIDs(ctx, botID)
		if err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Failed to retrieve user list", ShowAlert: true})
		}

		// Get the original message to broadcast
		originalMsg := &telebot.Message{ID: msgID, Chat: ownerChat}

		success := 0
		blocked := 0
		failed := 0

		for _, userID := range userIDs {
			if userID == c.Sender().ID {
				continue
			}

			userChat := &telebot.Chat{ID: userID}
			_, err := bot.Copy(userChat, originalMsg)
			if err != nil {
				if strings.Contains(err.Error(), "blocked") || strings.Contains(err.Error(), "Forbidden") {
					blocked++
				} else {
					failed++
					log.Printf("Failed to broadcast to %d: %v", userID, err)
				}
			} else {
				success++
			}

			// Rate limiting: 40ms delay between messages (max ~25 msg/sec)
			time.Sleep(40 * time.Millisecond)
		}

		report := fmt.Sprintf(`📢 <b>Broadcast Report</b>

✅ <b>Success:</b> %d
🚫 <b>Blocked/Forbidden:</b> %d
❌ <b>Failed:</b> %d
👥 <b>Total Attempted:</b> %d`,
			success, blocked, failed, len(userIDs))

		menu := &telebot.ReplyMarkup{}
		btnStats := menu.Data("📊 Statistics", "child_stats")
		btnBroadcast := menu.Data("📢 Broadcast", "child_broadcast")
		btnSchedule := menu.Data("📅 Schedule Message", "schedule_menu")
		btnSettings := menu.Data("⚙️ Settings", "child_settings")
		menu.Inline(
			menu.Row(btnStats),
			menu.Row(btnBroadcast),
			menu.Row(btnSchedule),
			menu.Row(btnSettings),
		)

		return c.Send(report, menu, telebot.ModeHTML)
	}
}
