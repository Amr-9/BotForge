package factory

import (
	"log"

	"github.com/Amr-9/botforge/internal/bot"
	"github.com/Amr-9/botforge/internal/database"
	"gopkg.in/telebot.v3"
)

// Factory represents the main factory bot
type Factory struct {
	bot     *telebot.Bot
	repo    *database.Repository
	manager *bot.Manager
	adminID int64
}

// NewFactory creates a new factory bot logic instance
func NewFactory(botInstance *telebot.Bot, repo *database.Repository, manager *bot.Manager, adminID int64) (*Factory, error) {
	factory := &Factory{
		bot:     botInstance,
		repo:    repo,
		manager: manager,
		adminID: adminID,
	}

	factory.registerHandlers()

	return factory, nil
}

// GetBot returns the underlying bot instance
func (f *Factory) GetBot() *telebot.Bot {
	return f.bot
}

// Start starts the factory bot (No-op in Webhook mode as server drives it)
func (f *Factory) Start() {
	log.Println("Factory Bot Logic initialized.")
}

// Stop stops the factory bot
func (f *Factory) Stop() {
	log.Println("Stopping Factory Bot logic...")
}
