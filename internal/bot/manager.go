package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/Amr-9/botforge/internal/cache"
	"github.com/Amr-9/botforge/internal/database"
	"gopkg.in/telebot.v3"
)

// Manager handles the lifecycle of all child bots
type Manager struct {
	repo       *database.Repository
	cache      *cache.Redis
	bots       map[string]*telebot.Bot // token -> bot instance
	botIDs     map[string]int64        // token -> bot ID
	webhookURL string
	mu         sync.RWMutex
}

// NewManager creates a new bot manager
func NewManager(repo *database.Repository, cache *cache.Redis, webhookURL string) *Manager {
	return &Manager{
		repo:       repo,
		cache:      cache,
		bots:       make(map[string]*telebot.Bot),
		botIDs:     make(map[string]int64),
		webhookURL: webhookURL,
	}
}

// RegisterExistingBot manually adds a bot to the manager
func (m *Manager) RegisterExistingBot(token string, bot *telebot.Bot) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure webhook is set
	publicURL := fmt.Sprintf("%s/webhook/%s", m.webhookURL, token)
	webhook := &telebot.Webhook{
		Endpoint: &telebot.WebhookEndpoint{PublicURL: publicURL},
	}
	if err := bot.SetWebhook(webhook); err != nil {
		log.Printf("Failed to set webhook for existing bot: %v", err)
	}

	m.bots[token] = bot
	// For existing bots (Factory), we might not have ID or don't track it in message logs mostly
	m.botIDs[token] = 0
	log.Printf("Registered existing bot: %s...", token[:10])
}

// ServeHTTP handles incoming webhook requests
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Path format: /webhook/{token}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	token := parts[2]
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	m.mu.RLock()
	bot, exists := m.bots[token]
	m.mu.RUnlock()

	if !exists {
		http.Error(w, "Bot not found", http.StatusNotFound)
		return
	}

	// Decode update
	var update telebot.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	// Process update
	bot.ProcessUpdate(update)
}

// StartBot registers the bot with Telegram Webhook and adds it to the manager
func (m *Manager) StartBot(token string, ownerChatID int64, botID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if bot is already running
	if _, exists := m.bots[token]; exists {
		log.Printf("Bot already running: %s...", token[:10])
		return nil
	}

	// Public Webhook URL for this bot
	publicURL := fmt.Sprintf("%s/webhook/%s", m.webhookURL, token)

	// Create bot settings with Webhook poller
	settings := telebot.Settings{
		Token:  token,
		Poller: &telebot.Webhook{}, // No Listen port here
	}

	// Create bot instance
	bot, err := telebot.NewBot(settings)
	if err != nil {
		return err
	}

	// Set Webhook on Telegram side
	webhook := &telebot.Webhook{
		Endpoint: &telebot.WebhookEndpoint{PublicURL: publicURL},
	}
	if err := bot.SetWebhook(webhook); err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	// Register handlers
	m.registerChildHandlers(bot, token, ownerChatID)

	// Store bot
	m.bots[token] = bot
	m.botIDs[token] = botID
	log.Printf("Started webhook for bot: %s... (ID: %d)", token[:10], botID)

	return nil
}

// StopBot removes the bot from manager and DELETE webhook
func (m *Manager) StopBot(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if bot, exists := m.bots[token]; exists {
		go func(b *telebot.Bot) {
			b.RemoveWebhook()
		}(bot)

		delete(m.bots, token)
		delete(m.botIDs, token)
		log.Printf("Stopped bot: %s...", token[:10])
	}
}

// StopAll stops all running child bots
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for token, bot := range m.bots {
		go bot.RemoveWebhook()
		delete(m.bots, token)
		delete(m.botIDs, token)
	}
}

// GetRunningCount returns the number of running bots
func (m *Manager) GetRunningCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.bots)
}

// IsRunning checks if a bot is currently running
func (m *Manager) IsRunning(token string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.bots[token]
	return exists
}

// registerChildHandlers sets up message handlers for a child bot
func (m *Manager) registerChildHandlers(bot *telebot.Bot, token string, ownerChatID int64) {
	ownerChat := &telebot.Chat{ID: ownerChatID}

	// Admin commands (Owner only)
	bot.Handle("/start", m.handleChildStart(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "child_stats"}, m.handleChildStats(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "child_broadcast"}, m.handleChildBroadcast(bot, token, ownerChat))
	bot.Handle(&telebot.Btn{Unique: "cancel_broadcast"}, m.handleCancelBroadcast(bot, token))

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

// handleChildStart handles the /start command for child bots
func (m *Manager) handleChildStart(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		sender := c.Sender()

		// If owner, show admin menu
		if sender.ID == ownerChat.ID {
			menu := &telebot.ReplyMarkup{}
			btnStats := menu.Data("📊 Statistics", "child_stats")
			btnBroadcast := menu.Data("📢 Broadcast", "child_broadcast")
			menu.Inline(
				menu.Row(btnStats),
				menu.Row(btnBroadcast),
			)
			return c.Reply("🤖 <b>Bot Admin Panel</b>\n\nSelect an option:", menu, telebot.ModeHTML)
		}

		// Check session for regular users
		return m.handleUserMessage(context.Background(), c, bot, token, ownerChat)
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

		// Back button
		menu := &telebot.ReplyMarkup{}
		// We can recreate the main menu or offer a back button?
		// For simplicity, let's keep the user on this message or allow them to go back if we had a distinct menu state.
		// Re-adding the main menu buttons to allow navigation
		btnStats := menu.Data("📊 Statistics", "child_stats")
		btnBroadcast := menu.Data("📢 Broadcast", "child_broadcast")
		menu.Inline(
			menu.Row(btnStats),
			menu.Row(btnBroadcast),
		)

		return c.Edit(msg, menu, telebot.ModeHTML)
	}
}

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

// createMessageHandler returns a handler function for processing messages
func (m *Manager) createMessageHandler(bot *telebot.Bot, token string, ownerChat *telebot.Chat) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		ctx := context.Background()
		sender := c.Sender()

		if sender.ID == ownerChat.ID {
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

	// Check if we should send user info (Session Logic)
	hasSession, err := m.cache.HasSession(ctx, token, sender.ID)
	if err != nil {
		log.Printf("Error checking session: %v", err)
	}

	// If NOT in Redis, check DB (Persistent Check)
	if !hasSession {
		hasInteracted, err := m.repo.HasUserInteracted(ctx, botID, sender.ID)
		if err != nil {
			log.Printf("Error checking DB interaction: %v", err)
		} else if hasInteracted {
			hasSession = true
			// Populate Redis so we don't check DB next time
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

		// Set infinite session in Redis
		// 0 means no expiration
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

	// INFO Command: Check if admin sent "info" (case-insensitive)
	if strings.ToLower(strings.TrimSpace(msg.Text)) == "info" {
		// Get user info from Telegram (Wait, we only have ID)
		// We can try to get Chat info if the bot has seen them recently or just use what we have.
		// `bot.ChatByID` might work if we have the ID.
		chat, err := bot.ChatByID(userChatID)
		if err != nil {
			log.Printf("Failed to get chat info: %v", err)
			// Fallback if we can't get chat info, just show ID
			chat = &telebot.Chat{ID: userChatID}
		}

		// Get first message date
		firstMsgDate, err := m.repo.GetFirstMessageDate(ctx, botID, userChatID)
		dateStr := "Unknown"
		if err == nil && !firstMsgDate.IsZero() {
			dateStr = firstMsgDate.Format("2006-01-02 15:04:05")
		}

		infoText := fmt.Sprintf(`👤 <b>From:</b> %s %s
🔗 <b>Username:</b> @%s
🆔 <b>ID:</b> <code>%d</code>

📅 <b>First Message:</b> %s`,
			chat.FirstName, chat.LastName, chat.Username, chat.ID, dateStr)

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
		// Small delay to treat API gently
		// time.Sleep(30 * time.Millisecond)
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
