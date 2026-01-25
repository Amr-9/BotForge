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
	webhookURL string
	mu         sync.RWMutex
}

// NewManager creates a new bot manager
func NewManager(repo *database.Repository, cache *cache.Redis, webhookURL string) *Manager {
	return &Manager{
		repo:       repo,
		cache:      cache,
		bots:       make(map[string]*telebot.Bot),
		webhookURL: webhookURL,
	}
}

// RegisterExistingBot manually adds a bot to the manager (e.g. Factory Bot)
func (m *Manager) RegisterExistingBot(token string, bot *telebot.Bot) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure webhook is set
	publicURL := fmt.Sprintf("%s/webhook/%s", m.webhookURL, token)
	webhook := &telebot.Webhook{
		Endpoint: &telebot.WebhookEndpoint{PublicURL: publicURL},
	}
	// We assume caller sets the webhook or we set it here
	if err := bot.SetWebhook(webhook); err != nil {
		log.Printf("Failed to set webhook for existing bot: %v", err)
	}

	m.bots[token] = bot
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
func (m *Manager) StartBot(token string, ownerChatID int64) error {
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
	// We handle errors gently here to avoid crashing entire loop on startup
	if err := bot.SetWebhook(webhook); err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	// Register handlers
	m.registerChildHandlers(bot, token, ownerChatID)

	// Store bot
	m.bots[token] = bot
	log.Printf("Started webhook for bot: %s...", token[:10])

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
			return m.handleAdminReply(ctx, c, bot, token)
		}

		return m.handleUserMessage(ctx, c, bot, token, ownerChat)
	}
}

// handleUserMessage forwards user message to admin with dual write
func (m *Manager) handleUserMessage(ctx context.Context, c telebot.Context, bot *telebot.Bot, token string, ownerChat *telebot.Chat) error {
	sender := c.Sender()

	userInfo := formatUserInfo(sender)
	_, err := bot.Send(ownerChat, userInfo, telebot.ModeHTML)
	if err != nil {
		log.Printf("Failed to send user info: %v", err)
	}

	sent, err := bot.Copy(ownerChat, c.Message())
	if err != nil {
		log.Printf("Failed to copy message to admin: %v", err)
		return c.Reply("Sorry, failed to deliver your message. Please try again later.")
	}

	adminMsgID := sent.ID
	if err := m.repo.SaveMessageLog(ctx, adminMsgID, sender.ID, token); err != nil {
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

	if msg.ReplyTo == nil {
		return c.Reply("Please reply to a user's message to send a response.")
	}

	replyToID := msg.ReplyTo.ID
	var userChatID int64
	var err error

	userChatID, err = m.cache.GetMessageLink(ctx, token, replyToID)
	if err != nil {
		if cache.IsNil(err) {
			log.Printf("Cache miss for msg %d, falling back to MySQL", replyToID)
			userChatID, err = m.repo.GetUserChatID(ctx, replyToID, token)
			if err != nil {
				log.Printf("Failed to get user chat ID from MySQL: %v", err)
				return c.Reply("Failed to find the original message sender.")
			}
		} else {
			log.Printf("Redis error: %v, falling back to MySQL", err)
			userChatID, err = m.repo.GetUserChatID(ctx, replyToID, token)
			if err != nil {
				log.Printf("Failed to get user chat ID from MySQL: %v", err)
				return c.Reply("Failed to find the original message sender.")
			}
		}
	}

	if userChatID == 0 {
		return c.Reply("Could not find the original message sender. The message may be too old.")
	}

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
