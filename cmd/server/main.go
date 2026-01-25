package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Amr-9/botforge/internal/bot"
	"github.com/Amr-9/botforge/internal/cache"
	"github.com/Amr-9/botforge/internal/config"
	"github.com/Amr-9/botforge/internal/database"
	"gopkg.in/telebot.v3"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Bot Factory (Webhook Mode)...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to MySQL
	mysql, err := database.NewMySQL(cfg.GetDSN())
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer mysql.Close()

	// Create repository
	repo := database.NewRepository(mysql)

	// Connect to Redis
	redisCache, err := cache.NewRedis(
		cfg.RedisAddr,
		cfg.RedisPassword,
		cfg.RedisDB,
		cfg.MessageTTL,
	)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()

	// Create bot manager with Webhook support
	manager := bot.NewManager(repo, redisCache, cfg.WebhookURL)

	// Create Factory Bot with Webhook
	factorySettings := telebot.Settings{
		Token:  cfg.FactoryBotToken,
		Poller: &telebot.Webhook{}, // No Listen port here
	}

	factoryBot, err := telebot.NewBot(factorySettings)
	if err != nil {
		log.Fatalf("Failed to create factory bot: %v", err)
	}

	// Set Factory Webhook
	factoryPublicURL := fmt.Sprintf("%s/webhook/%s", cfg.WebhookURL, cfg.FactoryBotToken)
	webhook := &telebot.Webhook{
		Endpoint: &telebot.WebhookEndpoint{PublicURL: factoryPublicURL},
	}
	if err := factoryBot.SetWebhook(webhook); err != nil {
		log.Fatalf("Failed to set factory webhook: %v", err)
	}

	// Use existing Factory logic (just attach the bot instance)
	// We need to slightly adapt NewFactory to accept an existing bot or update it
	// For now, let's create it and then swap the poller/webhook manually if needed
	// Actually, better to just modify Factory struct or inject the bot
	// Let's rely on manager's routing for everyone including factory

	// Create Factory Logic
	factory, err := bot.NewFactory(factoryBot, repo, manager, cfg.AdminID)
	if err != nil {
		log.Fatalf("Failed to create factory logic: %v", err)
	}
	// Note: NewFactory internally creates a LongPoller bot.
	// To fix this without changing Factory signature too much, we will rely on valid architecture.
	// But `NewFactory` currently creates a NEW bot.
	// We should update `Internal/bot/factory.go` to accept WebhookURL too or just use the manager's router
	// For simplicity in this step, I will let NewFactory run, but we must update `factory.go` next.
	// Because `NewFactory` currently forces LongPoller.

	// Register Factory in Manager so it handles its updates via ServeHTTP
	// We can manually add it to manager's map if we expose a method, or just use `StartBot` logic adapted.
	// But Factory has special handlers.

	// Let's Pause here. I need to update Factory.go first to support Webhook or pass the bot in.
	// I will act proactively and update Factory.go in the next tool call.

	// For Main.go, we setup the HTTP server.

	// HTTP Server Routing
	http.Handle("/webhook/", manager)

	// Start HTTP Server
	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: nil, // DefaultServeMux
	}

	go func() {
		log.Printf("Server listening on port %s...", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Load and start all active bots (Set Webhook for them)
	ctx := context.Background()
	activeBots, err := repo.GetActiveBots(ctx)
	if err != nil {
		log.Printf("Warning: Failed to load active bots: %v", err)
	} else {
		log.Printf("Loading %d active bots...", len(activeBots))
		for _, b := range activeBots {
			if err := manager.StartBot(b.Token, b.OwnerChatID, b.ID); err != nil {
				log.Printf("Failed to start bot %s: %v", maskToken(b.Token), err)
			}
		}
		log.Printf("Started %d child bots successfully", manager.GetRunningCount())
	}

	// Manually register Factory Bot into Manager's map so ServeHTTP finds it
	// Accessing private map is not allowed. We need a method `RegisterBot` or similar in Manager.
	// Or simply, we treat Factory as just another bot in the manager but with special handlers attached?

	// Better approach:
	// 1. Update Factory to use the passed `manager` for Webhook registration?
	// 2. Or just let Manager handle routing for Factory too.

	// Let's Assume I will add `RegisterExistingBot` to manager.
	manager.RegisterExistingBot(cfg.FactoryBotToken, factory.GetBot()) // Method to be added

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gracefully...")

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	// Remove Webhooks
	manager.StopAll()
	factory.Stop() // This currently stops the bot instance

	log.Println("Shutdown complete")
}

// maskToken masks a token for logging
func maskToken(token string) string {
	if len(token) > 15 {
		return token[:10] + "..."
	}
	return "***"
}
