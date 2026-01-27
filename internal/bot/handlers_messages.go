package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Amr-9/botforge/internal/cache"
	"gopkg.in/telebot.v3"
)

// registerChildHandlers sets up message handlers for a child bot
func (m *Manager) registerChildHandlers(bot *telebot.Bot, token string, ownerChatID int64) {
	ownerChat := &telebot.Chat{ID: ownerChatID}

	// Admin commands (Owner only)
	bot.Handle("/start", m.handleChildStart(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "child_stats"}, m.handleChildStats(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "child_broadcast"}, m.handleChildBroadcast(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "child_settings"}, m.handleChildSettings(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "set_start_msg"}, m.handleSetStartMsgBtn(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "cancel_broadcast"}, m.handleCancelBroadcast(bot, token))
	bot.Handle(&telebot.Btn{Unique: "back_to_settings"}, m.handleBackToSettings(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "child_main_menu"}, m.handleChildMainMenu(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "banned_list"}, m.handleBannedUsersList(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "unban_user"}, m.handleUnbanUser(bot, token, ownerChat))

	bot.Handle(telebot.OnText, m.createMessageHandler(bot, token, ownerChat))
	bot.Handle(telebot.OnPhoto, m.createMessageHandler(bot, token, ownerChat))
	bot.Handle(telebot.OnVideo, m.createMessageHandler(bot, token, ownerChat))
	bot.Handle(telebot.OnDocument, m.createMessageHandler(bot, token, ownerChat))
	bot.Handle(telebot.OnAudio, m.createMessageHandler(bot, token, ownerChat))
	bot.Handle(telebot.OnVoice, m.createMessageHandler(bot, token, ownerChat))
	bot.Handle(telebot.OnSticker, m.createMessageHandler(bot, token, ownerChat))
	bot.Handle(telebot.OnAnimation, m.createMessageHandler(bot, token, ownerChat))
	bot.Handle(telebot.OnVideoNote, m.createMessageHandler(bot, token, ownerChat))
	bot.Handle(telebot.OnContact, m.createMessageHandler(bot, token, ownerChat))
	bot.Handle(telebot.OnLocation, m.createMessageHandler(bot, token, ownerChat))
}

// createMessageHandler returns a handler function for processing messages
func (m *Manager) createMessageHandler(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		ctx := context.Background()
		sender := c.Sender()

		if sender.ID == ownerChat.ID {
			// Check user state
			state, err := m.cache.GetUserState(ctx, token, sender.ID)
			if err != nil {
				log.Printf("Error getting user state: %v", err)
			}

			if state == "set_start_msg" {
				// Update Start Message
				m.mu.RLock()
				botID := m.botIDs[token]
				m.mu.RUnlock()

				newMsg := c.Text()
				if newMsg == "" {
					return c.Reply("⚠️ Please send a text message.")
				}

				if err := m.repo.UpdateBotStartMessage(ctx, botID, newMsg); err != nil {
					return c.Reply("❌ Failed to update start message.")
				}

				// Clear state
				m.cache.ClearUserState(ctx, token, sender.ID)

				c.Reply("✅ <b>Start Message Updated!</b>\n\nHere is how it will look:", telebot.ModeHTML)
				return c.Send(newMsg, telebot.ModeMarkdown)
			}

			return m.handleAdminReply(ctx, c, bot, token)
		}

		return m.handleUserMessage(ctx, c, bot, token, ownerChat)
	}
}

// handleUserMessage forwards user message to admin with dual write
func (m *Manager) handleUserMessage(ctx context.Context, c telebot.Context, bot *telebot.Bot, token string, ownerChat *telebot.Chat) error {
	sender := c.Sender()

	m.mu.RLock()
	botID := m.botIDs[token]
	m.mu.RUnlock()

	// Check if user is banned - silently ignore their messages
	isBanned, err := m.checkUserBanned(ctx, token, botID, sender.ID)
	if err != nil {
		log.Printf("Error checking ban status: %v", err)
	}
	if isBanned {
		return nil // Silently ignore banned user messages
	}

	// Check if session exists
	hasSession, err := m.cache.HasSession(ctx, token, sender.ID)
	if err != nil {
		log.Printf("Error checking session: %v", err)
	}

	// If NOT in Redis, check DB
	if !hasSession {
		hasInteracted, err := m.repo.HasUserInteracted(ctx, botID, sender.ID)
		if err != nil {
			log.Printf("Error checking DB interaction: %v", err)
		} else if hasInteracted {
			hasSession = true
			m.cache.SetSession(ctx, token, sender.ID, 0)
		}
	}

	// If still NO session (truly first time), send Header
	if !hasSession {
		userInfo := formatUserInfo(sender)
		_, err := bot.Send(ownerChat, userInfo, telebot.ModeHTML)
		if err != nil {
			log.Printf("Failed to send user info: %v", err)
		}

		if err := m.cache.SetSession(ctx, token, sender.ID, 0); err != nil {
			log.Printf("Failed to update session: %v", err)
		}
	}

	sent, err := bot.Forward(ownerChat, c.Message())
	if err != nil {
		log.Printf("Failed to forward message to admin: %v", err)
		return c.Reply("Sorry, failed to deliver your message. Please try again later.")
	}

	adminMsgID := sent.ID
	if err := m.repo.SaveMessageLog(ctx, adminMsgID, sender.ID, botID); err != nil {
		log.Printf("Failed to save message log to MySQL: %v", err)
	}

	if err := m.cache.SetMessageLink(ctx, token, adminMsgID, sender.ID); err != nil {
		log.Printf("Failed to save message link to Redis: %v", err)
	}

	return nil
}

// handleAdminReply handles admin's reply to a user
func (m *Manager) handleAdminReply(ctx context.Context, c telebot.Context, bot *telebot.Bot, token string) error {
	msg := c.Message()

	// Check Broadcast Mode
	isBroadcast, err := m.cache.GetBroadcastMode(ctx, token, c.Sender().ID)
	if err == nil && isBroadcast {
		return m.executeBroadcast(ctx, c, bot, token)
	}

	m.mu.RLock()
	botID := m.botIDs[token]
	m.mu.RUnlock()

	if msg.ReplyTo == nil {
		return c.Reply("Please reply to a user's message to send a response.")
	}

	replyToID := msg.ReplyTo.ID
	var userChatID int64

	userChatID, err = m.cache.GetMessageLink(ctx, token, replyToID)
	if err != nil {
		if cache.IsNil(err) {
			log.Printf("Cache miss for msg %d, falling back to MySQL", replyToID)
			userChatID, err = m.repo.GetUserChatID(ctx, replyToID, botID)
			if err != nil {
				log.Printf("Failed to get user chat ID from MySQL: %v", err)
				return c.Reply("Failed to find the original message sender.")
			}
		} else {
			log.Printf("Redis error: %v, falling back to MySQL", err)
			userChatID, err = m.repo.GetUserChatID(ctx, replyToID, botID)
			if err != nil {
				log.Printf("Failed to get user chat ID from MySQL: %v", err)
				return c.Reply("Failed to find the original message sender.")
			}
		}
	}

	if userChatID == 0 {
		return c.Reply("Could not find the original message sender. The message may be too old.")
	}

	// Get command text (lowercase, trimmed)
	cmdText := strings.ToLower(strings.TrimSpace(msg.Text))

	// BAN Command: Check if admin sent "ban" or "/ban"
	if cmdText == "ban" || cmdText == "/ban" {
		return m.handleBanCommand(ctx, c, bot, token, userChatID)
	}

	// INFO Command: Check if admin sent "info" (case-insensitive)
	if cmdText == "info" {
		chat, err := bot.ChatByID(userChatID)
		if err != nil {
			log.Printf("Failed to get chat info: %v", err)
			chat = &telebot.Chat{ID: userChatID}
		}

		firstMsgDate, err := m.repo.GetFirstMessageDate(ctx, botID, userChatID)
		dateStr := "Unknown"
		if err == nil && !firstMsgDate.IsZero() {
			dateStr = firstMsgDate.Format("2006-01-02 15:04:05")
		}

		// Check ban status
		isBanned, _ := m.repo.IsUserBanned(ctx, botID, userChatID)
		banStatus := "No"
		if isBanned {
			banStatus = "Yes"
		}

		infoText := fmt.Sprintf(`👤 <b>From:</b> %s %s
🔗 <b>Username:</b> @%s
🆔 <b>ID:</b> <code>%d</code>

📅 <b>First Message:</b> %s
🚫 <b>Banned:</b> %s`,
			chat.FirstName, chat.LastName, chat.Username, chat.ID, dateStr, banStatus)

		return c.Reply(infoText, telebot.ModeHTML)
	}

	// Normal Reply -> Forward to user
	userChat := &telebot.Chat{ID: userChatID}
	_, err = bot.Copy(userChat, msg)
	if err != nil {
		log.Printf("Failed to send reply to user %d: %v", userChatID, err)
		return c.Reply("Failed to send message to user. They may have blocked the bot.")
	}

	return c.Reply("✅ Message sent successfully!")
}

// formatUserInfo creates a formatted user info header
func formatUserInfo(user *telebot.User) string {
	info := "📩 <b>New Message</b>\n"
	info += "━━━━━━━━━━━━━━━\n"
	info += "👤 <b>From:</b> "

	if user.FirstName != "" {
		info += user.FirstName
	}
	if user.LastName != "" {
		info += " " + user.LastName
	}
	info += "\n"

	if user.Username != "" {
		info += "🔗 <b>Username:</b> @" + user.Username + "\n"
	}

	info += "🆔 <b>ID:</b> <code>" + formatInt64(user.ID) + "</code>\n"
	info += "━━━━━━━━━━━━━━━"

	return info
}

// formatInt64 converts int64 to string
func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}
