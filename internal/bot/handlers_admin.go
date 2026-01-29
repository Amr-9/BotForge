package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"gopkg.in/telebot.v3"
)

// handleChildStart handles the /start command for child bots
func (m *Manager) handleChildStart(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		sender := c.Sender()

		// If owner, show admin menu
		if sender.ID == ownerChat.ID {
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
			return c.Reply("🤖 <b>Bot Admin Panel</b>\n\nSelect an option:", menu, telebot.ModeHTML)
		}

		ctx := context.Background()

		// Check if user is banned - silently ignore
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		isBanned, err := m.checkUserBanned(ctx, token, botID, sender.ID)
		if err != nil {
			log.Printf("Error checking ban status: %v", err)
		}
		if isBanned {
			return nil // Silently ignore banned user
		}

		// Check forced subscription
		isSubscribed, menu, blockedMsg, err := m.checkForcedSubscription(ctx, c, bot, token, botID, sender.ID)
		if err != nil {
			log.Printf("Error checking forced subscription: %v", err)
		}
		if !isSubscribed {
			return c.Send(blockedMsg, menu, telebot.ModeHTML)
		}

		// Retrieve Start Message from DB
		botModel, err := m.repo.GetBotByToken(ctx, token)
		if err != nil {
			log.Printf("Failed to get bot for start msg: %v", err)
			return c.Send("👋 Welcome! Please send me your message.")
		}

		welcomeMsg := "👋 Welcome! Please send me your message."
		if botModel != nil && botModel.StartMessage != "" {
			welcomeMsg = botModel.StartMessage
		}

		// Send welcome message to user
		// Use ModeMarkdown to support rich text (Markdown) in start message
		return c.Send(welcomeMsg, telebot.ModeMarkdown)
	}
}

// handleChildMainMenu shows the main admin menu (Edit mode for callbacks)
func (m *Manager) handleChildMainMenu(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

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

// handleChildSettings shows settings menu
func (m *Manager) handleChildSettings(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Get banned user count for display
		bannedCount, _ := m.repo.GetBannedUserCount(ctx, botID)

		// Get auto-reply counts
		keywordCount, _ := m.repo.GetAutoReplyCount(ctx, botID, "keyword")
		commandCount, _ := m.repo.GetAutoReplyCount(ctx, botID, "command")
		autoReplyTotal := keywordCount + commandCount

		// Get forced subscription info
		forcedChannelCount, _ := m.repo.GetForcedChannelCount(ctx, botID)
		botModel, _ := m.repo.GetBotByToken(ctx, token)
		forcedSubStatus := "OFF"
		if botModel != nil && botModel.ForcedSubEnabled {
			forcedSubStatus = "ON"
		}

		// Get sent confirmation status
		sentConfirmStatus := "ON"
		if botModel != nil && !botModel.ShowSentConfirmation {
			sentConfirmStatus = "OFF"
		}

		menu := &telebot.ReplyMarkup{}
		btnSetStartMsg := menu.Data("📝 Set Start Message", "set_start_msg")
		btnAutoReplies := menu.Data(fmt.Sprintf("🤖 Auto-Replies (%d)", autoReplyTotal), "auto_replies_menu")
		btnForcedSub := menu.Data(fmt.Sprintf("🔐 Forced Sub [%s] (%d)", forcedSubStatus, forcedChannelCount), "forced_sub_menu")
		btnBannedUsers := menu.Data(fmt.Sprintf("🚫 Banned Users (%d)", bannedCount), "banned_list")
		btnSentConfirm := menu.Data(fmt.Sprintf("✅ Sent Confirmation [%s]", sentConfirmStatus), "toggle_sent_confirm")
		btnBack := menu.Data("« Back to Menu", "child_main_menu")

		menu.Inline(
			menu.Row(btnSetStartMsg),
			menu.Row(btnAutoReplies),
			menu.Row(btnForcedSub),
			menu.Row(btnBannedUsers),
			menu.Row(btnSentConfirm),
			menu.Row(btnBack),
		)

		return c.Edit("⚙️ <b>Settings</b>\n\nChoose an option:", menu, telebot.ModeHTML)
	}
}

// handleBackToSettings navigates back to settings menu
func (m *Manager) handleBackToSettings(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		// Clear any pending user state when going back
		ctx := context.Background()
		m.cache.ClearUserState(ctx, token, c.Sender().ID)
		// Just reuse handleChildSettings logic
		return m.handleChildSettings(bot, token, ownerChat)(c)
	}
}

// handleSetStartMsgBtn initiates state to set start message
func (m *Manager) handleSetStartMsgBtn(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		if err := m.cache.SetUserState(ctx, token, c.Sender().ID, "set_start_msg"); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Error setting state!", ShowAlert: true})
		}

		menu := &telebot.ReplyMarkup{}
		btnCancel := menu.Data("❌ Cancel", "back_to_settings")
		menu.Inline(menu.Row(btnCancel))

		currentBot, err := m.repo.GetBotByToken(ctx, token)
		currentMsg := "<i>(Default)</i>"
		if err == nil && currentBot != nil && currentBot.StartMessage != "" {
			// Escape HTML tags for display in the "Current Message" section to avoid rendering them
			currentMsg = strings.ReplaceAll(currentBot.StartMessage, "<", "&lt;")
			currentMsg = strings.ReplaceAll(currentMsg, ">", "&gt;")
		}

		msg := fmt.Sprintf(`📝 <b>Set Start Message</b>

<b>Current Message:</b>
<pre>%s</pre>

Please send the new welcome message for your bot.
✅ <b>Supported Formats:</b> Markdown
Example: <code>Hello *User*!</code>
_Italic_, *Bold*, [Link](http://example.com)`, currentMsg)

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handleChildStats shows bot statistics to the owner
func (m *Manager) handleChildStats(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		count, err := m.repo.GetUniqueUserCount(ctx, botID)
		if err != nil {
			return c.Edit("❌ Failed to retrieve stats.")
		}

		msg := fmt.Sprintf("📊 <b>Bot Statistics</b>\n\n👥 <b>Unique Users:</b> %d", count)

		menu := &telebot.ReplyMarkup{}
		btnBack := menu.Data("« Back to Menu", "child_main_menu")
		menu.Inline(
			menu.Row(btnBack),
		)

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

// handleToggleSentConfirmation toggles the "Message sent successfully" notification
func (m *Manager) handleToggleSentConfirmation(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		if c.Sender().ID != ownerChat.ID {
			return nil
		}

		ctx := context.Background()
		m.mu.RLock()
		botID := m.botIDs[token]
		m.mu.RUnlock()

		// Get current setting
		botModel, err := m.repo.GetBotByToken(ctx, token)
		if err != nil || botModel == nil {
			return c.Respond(&telebot.CallbackResponse{Text: "❌ Failed to get settings!", ShowAlert: true})
		}

		// Toggle the setting
		newValue := !botModel.ShowSentConfirmation
		if err := m.repo.UpdateBotShowSentConfirmation(ctx, botID, newValue); err != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "❌ Failed to update setting!", ShowAlert: true})
		}

		// Update cache immediately for better performance
		m.cache.SetShowSentConfirmation(ctx, token, newValue)

		status := "ON"
		if !newValue {
			status = "OFF"
		}

		c.Respond(&telebot.CallbackResponse{Text: fmt.Sprintf("✅ Sent confirmation is now %s", status)})

		// Refresh settings menu
		return m.handleChildSettings(bot, token, ownerChat)(c)
	}
}
