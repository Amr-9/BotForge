package bot_test

import (
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/bot"
	"github.com/Amr-9/botforge/internal/cache"
	"github.com/Amr-9/botforge/internal/database"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/jmoiron/sqlx"
)

// ==================== Test Setup Helpers ====================

func setupTestManager(t *testing.T) (*bot.Manager, sqlmock.Sqlmock, *miniredis.Miniredis, func()) {
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

	// Create manager
	manager := bot.NewManager(repo, redisCache, "https://example.com/webhook")

	cleanup := func() {
		redisCache.Close()
		mr.Close()
		db.Close()
	}

	return manager, mock, mr, cleanup
}

// ==================== NewManager Tests ====================

func TestNewManager_Creation(t *testing.T) {
	manager, _, _, cleanup := setupTestManager(t)
	defer cleanup()

	if manager == nil {
		t.Error("Expected manager to be created")
	}
}

func TestNewManager_InitialRunningCount(t *testing.T) {
	manager, _, _, cleanup := setupTestManager(t)
	defer cleanup()

	count := manager.GetRunningCount()
	if count != 0 {
		t.Errorf("Expected 0 running bots initially, got %d", count)
	}
}

// ==================== GetRunningCount Tests ====================

func TestGetRunningCount_Empty(t *testing.T) {
	manager, _, _, cleanup := setupTestManager(t)
	defer cleanup()

	count := manager.GetRunningCount()
	if count != 0 {
		t.Errorf("Expected 0, got %d", count)
	}
}

// ==================== IsRunning Tests ====================

func TestIsRunning_NotStarted(t *testing.T) {
	manager, _, _, cleanup := setupTestManager(t)
	defer cleanup()

	isRunning := manager.IsRunning("nonexistent:token")
	if isRunning {
		t.Error("Expected bot to not be running")
	}
}

func TestIsRunning_EmptyToken(t *testing.T) {
	manager, _, _, cleanup := setupTestManager(t)
	defer cleanup()

	isRunning := manager.IsRunning("")
	if isRunning {
		t.Error("Expected empty token to not be running")
	}
}

// ==================== ManualPoller Tests ====================

func TestManualPoller_Creation(t *testing.T) {
	poller := &bot.ManualPoller{}
	if poller == nil {
		t.Error("Expected ManualPoller to be created")
	}
}

// ==================== Integration-style Tests ====================

func TestManager_TokenNotFound(t *testing.T) {
	manager, _, _, cleanup := setupTestManager(t)
	defer cleanup()

	// Try to get a bot that doesn't exist
	_, _, err := manager.GetBotByID(99999)
	if err == nil {
		t.Error("Expected error for non-existent bot ID")
	}
}

func TestManager_GetBotByID_NotRegistered(t *testing.T) {
	manager, _, _, cleanup := setupTestManager(t)
	defer cleanup()

	// No bots registered, should return error
	teleBot, token, err := manager.GetBotByID(12345)
	if err == nil {
		t.Error("Expected error when no bots are registered")
	}
	if teleBot != nil {
		t.Error("Expected nil bot")
	}
	if token != "" {
		t.Error("Expected empty token")
	}
}
