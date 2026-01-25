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
		DBHost:          os.Getenv("DB_HOST"),
		DBUser:          os.Getenv("DB_USER"),
		DBPass:          os.Getenv("DB_PASS"),
		DBName:          os.Getenv("DB_NAME"),
		RedisAddr:       os.Getenv("REDIS_ADDR"),
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

	if cfg.DBHost == "" || cfg.DBUser == "" || cfg.DBName == "" {
		return nil, fmt.Errorf("database configuration (DB_HOST, DB_USER, DB_NAME) is required")
	}
	if cfg.RedisAddr == "" {
		return nil, fmt.Errorf("REDIS_ADDR is required")
	}

	// Encryption Key (Must be 32 chars)
	cfg.EncryptionKey = os.Getenv("BOT_ENCRYPTION_KEY")
	if cfg.EncryptionKey == "" {
		return nil, fmt.Errorf("BOT_ENCRYPTION_KEY is required")
	}
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
