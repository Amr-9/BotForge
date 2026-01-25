package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Factory Bot
	FactoryBotToken string
	AdminID         int64

	// Webhook
	WebhookURL string
	ServerPort string

	// MySQL
	DBHost string
	DBUser string
	DBPass string
	DBName string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Cache TTL for message links
	MessageTTL time.Duration

	// Security
	EncryptionKey string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	cfg := &Config{
		FactoryBotToken: os.Getenv("FACTORY_BOT_TOKEN"),
		WebhookURL:      os.Getenv("WEBHOOK_URL"),
		ServerPort:      getEnvOrDefault("PORT", "4210"),
		DBHost:          getEnvOrDefault("DB_HOST", "localhost"),
		DBUser:          getEnvOrDefault("DB_USER", "root"),
		DBPass:          os.Getenv("DB_PASS"),
		DBName:          getEnvOrDefault("DB_NAME", "numgate"),
		RedisAddr:       getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
	}

	// Parse Admin ID
	adminIDStr := os.Getenv("ADMIN_ID")
	if adminIDStr != "" {
		adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ADMIN_ID: %w", err)
		}
		cfg.AdminID = adminID
	}

	// Parse Redis DB
	redisDBStr := getEnvOrDefault("REDIS_DB", "0")
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}
	cfg.RedisDB = redisDB

	// Parse Message TTL (in hours)
	ttlStr := getEnvOrDefault("MESSAGE_TTL", "48")
	ttlHours, err := strconv.Atoi(ttlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid MESSAGE_TTL: %w", err)
	}
	cfg.MessageTTL = time.Duration(ttlHours) * time.Hour

	// Validate required fields
	if cfg.FactoryBotToken == "" {
		return nil, fmt.Errorf("FACTORY_BOT_TOKEN is required")
	}
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("WEBHOOK_URL is required for webhook mode")
	}

	// Encryption Key (Must be 32 chars)
	cfg.EncryptionKey = getEnvOrDefault("BOT_ENCRYPTION_KEY", "12345678901234567890123456789012") // Default for dev only
	if len(cfg.EncryptionKey) != 32 {
		return nil, fmt.Errorf("BOT_ENCRYPTION_KEY must be exactly 32 bytes")
	}

	return cfg, nil
}

// GetDSN returns MySQL connection string
func (c *Config) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&charset=utf8mb4",
		c.DBUser, c.DBPass, c.DBHost, c.DBName)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
