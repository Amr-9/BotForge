package bot_test

import (
	"context"
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/cache"
	"github.com/Amr-9/botforge/internal/database"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/jmoiron/sqlx"
)

// ==================== Test Setup Helper ====================

func setupAutoReplyTestEnv(t *testing.T) (*database.Repository, *cache.Redis, sqlmock.Sqlmock, *miniredis.Miniredis, func()) {
	// Setup mock MySQL
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	sqlxDB := sqlx.NewDb(db, "mysql")
	mysql := database.NewMySQLFromDB(sqlxDB)
	repo := database.NewRepository(mysql, "12345678901234567890123456789012")

	// Setup miniredis
	mr, err := miniredis.Run()
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	redisCache, err := cache.NewRedis(mr.Addr(), "", 0, 48*time.Hour)
	if err != nil {
		mr.Close()
		db.Close()
		t.Fatalf("Failed to create Redis client: %v", err)
	}

	cleanup := func() {
		redisCache.Close()
		mr.Close()
		db.Close()
	}

	return repo, redisCache, mock, mr, cleanup
}

// ==================== Auto-Reply Cache Tests ====================

func TestAutoReplyCache_SetAndGet(t *testing.T) {
	_, redisCache, _, _, cleanup := setupAutoReplyTestEnv(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot-token"

	// Set auto-reply with media
	cacheData := &cache.AutoReplyCache{
		Response:    "Hello!",
		MessageType: "text",
		FileID:      "",
		Caption:     "",
	}

	err := redisCache.SetAutoReplyWithMedia(ctx, botToken, "hello", cacheData, "keyword")
	if err != nil {
		t.Fatalf("Failed to set auto-reply: %v", err)
	}

	// Get auto-reply
	result, err := redisCache.GetAutoReplyWithMedia(ctx, botToken, "hello", "keyword")
	if err != nil {
		t.Fatalf("Failed to get auto-reply: %v", err)
	}

	if result.Response != "Hello!" {
		t.Errorf("Expected 'Hello!', got '%s'", result.Response)
	}
	if result.MessageType != "text" {
		t.Errorf("Expected 'text', got '%s'", result.MessageType)
	}
}

func TestAutoReplyCache_PhotoType(t *testing.T) {
	_, redisCache, _, _, cleanup := setupAutoReplyTestEnv(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"

	cacheData := &cache.AutoReplyCache{
		Response:    "",
		MessageType: "photo",
		FileID:      "AgACAgIAAxkBAAI123...",
		Caption:     "Beautiful sunset!",
	}

	err := redisCache.SetAutoReplyWithMedia(ctx, botToken, "sunset", cacheData, "keyword")
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	result, err := redisCache.GetAutoReplyWithMedia(ctx, botToken, "sunset", "keyword")
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}

	if result.MessageType != "photo" {
		t.Errorf("Expected 'photo', got '%s'", result.MessageType)
	}
	if result.FileID != "AgACAgIAAxkBAAI123..." {
		t.Errorf("FileID mismatch")
	}
	if result.Caption != "Beautiful sunset!" {
		t.Errorf("Caption mismatch")
	}
}

func TestAutoReplyCache_Command(t *testing.T) {
	_, redisCache, _, _, cleanup := setupAutoReplyTestEnv(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"

	cacheData := &cache.AutoReplyCache{
		Response:    "This is help text",
		MessageType: "text",
	}

	// Set as command type
	err := redisCache.SetAutoReplyWithMedia(ctx, botToken, "help", cacheData, "command")
	if err != nil {
		t.Fatalf("Failed to set command: %v", err)
	}

	// Get command
	result, err := redisCache.GetAutoReplyWithMedia(ctx, botToken, "help", "command")
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}

	if result.Response != "This is help text" {
		t.Errorf("Expected 'This is help text', got '%s'", result.Response)
	}
}

func TestAutoReplyCache_Delete(t *testing.T) {
	_, redisCache, _, _, cleanup := setupAutoReplyTestEnv(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"

	cacheData := &cache.AutoReplyCache{
		Response:    "Bye!",
		MessageType: "text",
	}

	// Set then delete
	redisCache.SetAutoReplyWithMedia(ctx, botToken, "goodbye", cacheData, "keyword")
	err := redisCache.DeleteAutoReply(ctx, botToken, "goodbye", "keyword")
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Should not find
	result, err := redisCache.GetAutoReplyWithMedia(ctx, botToken, "goodbye", "keyword")
	if err == nil && result != nil {
		t.Error("Expected nil result after delete")
	}
}

func TestAutoReplyCache_CaseSensitivity(t *testing.T) {
	_, redisCache, _, _, cleanup := setupAutoReplyTestEnv(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"

	cacheData := &cache.AutoReplyCache{
		Response:    "Found it!",
		MessageType: "text",
	}

	// Set with lowercase
	redisCache.SetAutoReplyWithMedia(ctx, botToken, "hello", cacheData, "keyword")

	// Try to get with same case
	result, err := redisCache.GetAutoReplyWithMedia(ctx, botToken, "hello", "keyword")
	if err != nil || result == nil {
		t.Error("Expected to find with same case")
	}
}
