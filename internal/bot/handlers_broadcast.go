package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

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

		menu := &telebot.ReplyMarkup{}
		btnStats := menu.Data("📊 Statistics", "child_stats")
		btnBroadcast := menu.Data("📢 Broadcast", "child_broadcast")
		menu.Inline(
			menu.Row(btnStats),
			menu.Row(btnBroadcast),
		)

		return c.Edit("📢 Broadcast cancelled.", menu, telebot.ModeHTML)
	}
}

// executeBroadcast runs the broadcast process
func (m *Manager) executeBroadcast(ctx context.Context, c telebot.Context, bot *telebot.Bot, token string) error {
	m.mu.RLock()
	botID := m.botIDs[token]
	m.mu.RUnlock()

	// Exit broadcast mode immediately to prevent accidental double sends
	m.cache.ClearBroadcastMode(ctx, token, c.Sender().ID)

	c.Reply("⏳ Starting broadcast. This may take a while...")

	userIDs, err := m.repo.GetAllUserChatIDs(ctx, botID)
	if err != nil {
		return c.Reply("❌ Failed to retrieve user list.")
	}

	success := 0
	blocked := 0
	failed := 0

	for _, userID := range userIDs {
		// Skip sending to the admin themselves if they are in the list
		if userID == c.Sender().ID {
			continue
		}

		userChat := &telebot.Chat{ID: userID}
		_, err := bot.Copy(userChat, c.Message())
		if err != nil {
			// Check for blocked user error (usually "Forbidden: bot was blocked by the user")
			if strings.Contains(err.Error(), "blocked") || strings.Contains(err.Error(), "Forbidden") {
				blocked++
			} else {
				failed++
				log.Printf("Failed to broadcast to %d: %v", userID, err)
			}
		} else {
			success++
		}
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
	menu.Inline(
		menu.Row(btnStats),
		menu.Row(btnBroadcast),
	)

	return c.Reply(report, menu, telebot.ModeHTML)
}
