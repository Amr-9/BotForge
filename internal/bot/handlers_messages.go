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
	bot.Handle(&telebot.Btn{Unique: "confirm_broadcast"}, m.handleConfirmBroadcast(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "back_to_settings"}, m.handleBackToSettings(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "child_main_menu"}, m.handleChildMainMenu(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "banned_list"}, m.handleBannedUsersList(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "unban_user"}, m.handleUnbanUser(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "toggle_sent_confirm"}, m.handleToggleSentConfirmation(bot, token, ownerChat))

	// Auto-Replies handlers
	bot.Handle(&telebot.Btn{Unique: "auto_replies_menu"}, m.handleAutoRepliesMenu(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "add_auto_reply"}, m.handleAddAutoReply(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "add_custom_cmd"}, m.handleAddCustomCommand(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "list_auto_replies"}, m.handleListAutoReplies(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "list_custom_cmds"}, m.handleListCustomCommands(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "del_reply"}, m.handleDeleteAutoReply(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "toggle_forward_replies"}, m.handleToggleForwardReplies(bot, token, ownerChat))

	// Forced Subscription handlers
	bot.Handle(&telebot.Btn{Unique: "forced_sub_menu"}, m.handleForcedSubMenu(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "toggle_forced_sub"}, m.handleToggleForcedSub(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "add_forced_channel"}, m.handleAddForcedChannel(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "list_forced_channels"}, m.handleListForcedChannels(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "del_forced_channel"}, m.handleRemoveForcedChannel(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "set_forced_sub_msg"}, m.handleSetForcedSubMsg(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "clear_forced_sub_msg"}, m.handleClearForcedSubMsg(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "check_subscription"}, m.handleCheckSubscription(bot, token, ownerChat))

	// Schedule handlers
	bot.Handle(&telebot.Btn{Unique: "schedule_menu"}, m.handleScheduleMenu(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_new"}, m.handleScheduleNewMessage(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_list"}, m.handleListScheduledMessages(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_type_once"}, m.handleScheduleTypeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_type_daily"}, m.handleScheduleTypeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_type_weekly"}, m.handleScheduleTypeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_1h"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_3h"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_6h"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_12h"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_daily_06:00"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_daily_09:00"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_daily_12:00"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_daily_15:00"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_daily_18:00"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_daily_21:00"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_weekly_06:00"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_weekly_09:00"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_weekly_12:00"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_weekly_15:00"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_weekly_18:00"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_time_weekly_21:00"}, m.handleScheduleTimeSelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_day_0"}, m.handleScheduleDaySelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_day_1"}, m.handleScheduleDaySelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_day_2"}, m.handleScheduleDaySelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_day_3"}, m.handleScheduleDaySelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_day_4"}, m.handleScheduleDaySelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_day_5"}, m.handleScheduleDaySelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_day_6"}, m.handleScheduleDaySelection(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_confirm"}, m.handleConfirmSchedule(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_cancel"}, m.handleCancelSchedule(bot, token))
	bot.Handle(&telebot.Btn{Unique: "schedule_pause"}, m.handlePauseScheduledMessage(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_resume"}, m.handleResumeScheduledMessage(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "schedule_delete"}, m.handleDeleteScheduledMessage(bot, token, ownerChat))

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

			// Handle auto-reply states
			if strings.HasPrefix(state, "add_auto_reply") || strings.HasPrefix(state, "add_custom_cmd") {
				handled, err := m.processAutoReplyState(ctx, c, token, state)
				if handled {
					return err
				}
			}

			// Handle schedule states
			if strings.HasPrefix(state, "schedule_") {
				handled, err := m.processScheduleState(ctx, c, token, state)
				if handled {
					return err
				}
			}

			// Handle forced subscription states
			if state == "add_forced_channel" || state == "set_forced_sub_message" {
				handled, err := m.processForcedSubState(ctx, c, bot, token, state)
				if handled {
					return err
				}
			}

			return m.handleAdminReply(ctx, c, bot, token)
		}

		return m.handleUserMessage(ctx, c, bot, token, ownerChat)
	}
}

// handleUserMessage forwards user message to admin with dual write
func (m *Manager) handleUserMessage(ctx context.Context, c telebot.Context, bot *telebot.Bot, token string, ownerChat *telebot.Chat) error {
	sender := c.Sender()
	text := c.Text()

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

	// Check forced subscription
	isSubscribed, menu, blockedMsg, err := m.checkForcedSubscription(ctx, c, bot, token, botID, sender.ID)
	if err != nil {
		log.Printf("Error checking forced subscription: %v", err)
	}
	if !isSubscribed {
		return c.Send(blockedMsg, menu, telebot.ModeHTML)
	}

	// Check custom commands and auto-replies
	autoReplied := false
	if text != "" {
		if response := m.checkCustomCommand(ctx, token, botID, text); response != "" {
			c.Send(response, telebot.ModeMarkdown)
			autoReplied = true
		}

		// Check auto-reply keywords (exact match only)
		if response := m.checkAutoReply(ctx, token, botID, text); response != "" {
			c.Send(response, telebot.ModeMarkdown)
			autoReplied = true
		}
	}

	// Check forward setting - if auto-replied and forwarding is disabled, stop here
	if autoReplied {
		botModel, _ := m.repo.GetBotByToken(ctx, token)
		if botModel != nil && !botModel.ForwardAutoReplies {
			return nil // Don't forward to admin
		}
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
		return m.requestBroadcastConfirmation(ctx, c, bot, token)
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

	// Check if we should show confirmation (use cache for performance)
	showConfirmation := true // default
	cachedValue, cacheHit, cacheErr := m.cache.GetShowSentConfirmation(ctx, token)
	if cacheErr != nil {
		log.Printf("Cache error: %v", cacheErr)
	}

	if cacheHit {
		showConfirmation = cachedValue
	} else {
		// Cache miss - load from DB and cache it
		botModel, _ := m.repo.GetBotByToken(ctx, token)
		if botModel != nil {
			showConfirmation = botModel.ShowSentConfirmation
			// Cache the value for future requests
			m.cache.SetShowSentConfirmation(ctx, token, showConfirmation)
		}
	}

	if showConfirmation {
		err = bot.React(msg.Chat, msg, telebot.ReactionOptions{
			Reactions: []telebot.Reaction{{Type: "emoji", Emoji: "👍"}},
		})
		if err != nil {
			log.Printf("⚠️ Reaction Failed: %v", err)
		}
	}

	return nil
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
