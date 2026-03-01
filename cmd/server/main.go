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
	"github.com/Amr-9/botforge/internal/factory"
	"github.com/Amr-9/botforge/internal/recovery"
	"github.com/Amr-9/botforge/internal/scheduler"
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
	repo := database.NewRepository(mysql, cfg.EncryptionKey)

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

	// Create scheduler service
	schedulerService := scheduler.NewScheduler(repo, manager, 1*time.Minute)

	// Create Factory Bot with Webhook
	factorySettings := telebot.Settings{
		Token:  cfg.FactoryBotToken,
		Poller: &bot.ManualPoller{}, // Use ManualPoller to avoid port binding
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

	// Create Factory Logic
	factory, err := factory.NewFactory(factoryBot, repo, manager, cfg.AdminID)
	if err != nil {
		log.Fatalf("Failed to create factory logic: %v", err)
	}

	// Create shared panic recovery handler
	panicHandler := recovery.DefaultHandler

	// HTTP Server Routing with panic recovery middleware
	http.Handle("/webhook/", recovery.HTTPMiddleware(manager, panicHandler))

	// Start HTTP Server
	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: nil, // DefaultServeMux
	}

	// Start HTTP server with panic recovery
	// Use SafeGoWithRestartAndReset - only restart on panic, not on normal return
	recovery.SafeGoWithRestartAndReset(
		func() {
			log.Printf("Server listening on port %s...", cfg.ServerPort)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				// Panic on critical HTTP error to trigger restart
				panic(fmt.Errorf("HTTP server critical error: %v", err))
			}
		},
		map[string]string{"type": "http_server"},
		panicHandler,
		recovery.NewRestartPolicy(5, 1*time.Second, 30*time.Second),
		30*time.Second, // Reset retry counter if server runs for 30s successfully
		func() {
			log.Fatalf("[CRITICAL] HTTP server exhausted restart retries")
		},
	)

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

	// Register Factory Bot into Manager so ServeHTTP routes its webhook updates
	manager.RegisterExistingBot(cfg.FactoryBotToken, factory.GetBot())

	// Start scheduler service
	schedulerService.Start()
	log.Println("Scheduler service started")

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gracefully...")

	// Stop scheduler service
	schedulerService.Stop()

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
