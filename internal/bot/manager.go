package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Amr-9/botforge/internal/cache"
	"github.com/Amr-9/botforge/internal/database"
	"github.com/Amr-9/botforge/internal/recovery"
	"gopkg.in/telebot.v3"
)

// Manager handles the lifecycle of all child bots
type Manager struct {
	repo               *database.Repository
	cache              *cache.Redis
	bots               map[string]*telebot.Bot // token -> bot instance
	botIDs             map[string]int64        // token -> bot ID
	webhookURL         string
	mu                 sync.RWMutex
	recoveryHandler    recovery.Handler
	restartPolicies    map[string]*recovery.RestartPolicy     // token -> restart policy
	restartControllers map[string]*recovery.RestartController // token -> restart controller
}

// NewManager creates a new bot manager with default recovery handler
func NewManager(repo *database.Repository, cache *cache.Redis, webhookURL string) *Manager {
	return NewManagerWithRecovery(repo, cache, webhookURL, recovery.DefaultHandler)
}

// NewManagerWithRecovery creates a new bot manager with custom recovery handler
func NewManagerWithRecovery(repo *database.Repository, cache *cache.Redis, webhookURL string, handler recovery.Handler) *Manager {
	return &Manager{
		repo:               repo,
		cache:              cache,
		bots:               make(map[string]*telebot.Bot),
		botIDs:             make(map[string]int64),
		webhookURL:         webhookURL,
		recoveryHandler:    handler,
		restartPolicies:    make(map[string]*recovery.RestartPolicy),
		restartControllers: make(map[string]*recovery.RestartController),
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

	// Create restart policy and controller for factory bot
	policy := recovery.NewRestartPolicy(3, 5*time.Second, 1*time.Minute)
	m.restartPolicies[token] = policy
	controller := recovery.NewRestartController()
	m.restartControllers[token] = controller

	// Start the bot dispatcher in the background with panic recovery and cancellation support
	tokenPrefix := token[:10]
	recovery.SafeGoWithRestartAndController(
		func() { bot.Start() },
		map[string]string{
			"type":  "factory_bot",
			"token": tokenPrefix + "...",
		},
		m.recoveryHandler,
		policy,
		controller,
		func() {
			log.Printf("[CRITICAL] Factory bot %s... exhausted restart retries", tokenPrefix)
		},
	)

	log.Printf("Registered existing bot: %s...", tokenPrefix)
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

	// Process update with panic recovery
	tokenPrefix := token
	if len(token) > 10 {
		tokenPrefix = token[:10] + "..."
	}
	func() {
		defer recovery.Recover(m.recoveryHandler, map[string]string{
			"type":  "process_update",
			"token": tokenPrefix,
		})
		bot.ProcessUpdate(update)
	}()
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
		Poller: &ManualPoller{}, // Use ManualPoller to avoid port binding
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

	// Create restart policy and controller for child bot
	policy := recovery.NewRestartPolicy(3, 5*time.Second, 1*time.Minute)
	m.restartPolicies[token] = policy
	controller := recovery.NewRestartController()
	m.restartControllers[token] = controller

	// Start the bot dispatcher in the background with panic recovery and cancellation support
	tokenPrefix := token[:10]
	recovery.SafeGoWithRestartAndController(
		func() { bot.Start() },
		map[string]string{
			"type":  "child_bot",
			"token": tokenPrefix + "...",
			"botID": fmt.Sprintf("%d", botID),
		},
		m.recoveryHandler,
		policy,
		controller,
		func() {
			log.Printf("[CRITICAL] Child bot %s... (ID: %d) exhausted restart retries", tokenPrefix, botID)
		},
	)

	log.Printf("Started webhook for bot: %s... (ID: %d)", tokenPrefix, botID)

	return nil
}

// StopBot removes the bot from manager and DELETE webhook
func (m *Manager) StopBot(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if bot, exists := m.bots[token]; exists {
		tokenPrefix := token[:10]

		// Stop the restart controller first to cancel the goroutine
		if controller, ctrlExists := m.restartControllers[token]; ctrlExists {
			controller.Stop()
			delete(m.restartControllers, token)
		}

		botCopy := bot
		recovery.SafeGo(
			func() { botCopy.RemoveWebhook() },
			map[string]string{
				"type":  "webhook_cleanup",
				"token": tokenPrefix + "...",
			},
			m.recoveryHandler,
		)

		delete(m.bots, token)
		delete(m.botIDs, token)
		delete(m.restartPolicies, token)
		log.Printf("Stopped bot: %s...", tokenPrefix)
	}
}

// StopAll stops all running child bots
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for token, bot := range m.bots {
		tokenPrefix := token[:10]

		// Stop the restart controller first
		if controller, ctrlExists := m.restartControllers[token]; ctrlExists {
			controller.Stop()
			delete(m.restartControllers, token)
		}

		botCopy := bot
		recovery.SafeGo(
			func() { botCopy.RemoveWebhook() },
			map[string]string{
				"type":  "webhook_cleanup_all",
				"token": tokenPrefix + "...",
			},
			m.recoveryHandler,
		)
		delete(m.bots, token)
		delete(m.botIDs, token)
		delete(m.restartPolicies, token)
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

// GetBotByID retrieves a bot instance by bot ID
func (m *Manager) GetBotByID(botID int64) (*telebot.Bot, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find the token by botID
	for token, id := range m.botIDs {
		if id == botID {
			bot, exists := m.bots[token]
			if !exists {
				return nil, "", fmt.Errorf("bot with ID %d is not running", botID)
			}
			return bot, token, nil
		}
	}

	return nil, "", fmt.Errorf("bot with ID %d not found", botID)
}

// ManualPoller is a custom poller that does nothing but block.
// It is used when we drive the bot updates manually via ProcessUpdate.
// This allows us to call bot.Start() to run the dispatcher without
// starting a built-in HTTP server or LongPolling loop.
type ManualPoller struct{}

func (p *ManualPoller) Poll(b *telebot.Bot, dest chan telebot.Update, stop chan struct{}) {
	// Block until stop is closed
	<-stop
}
