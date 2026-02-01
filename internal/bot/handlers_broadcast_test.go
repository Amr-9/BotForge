package bot_test

import (
	"context"
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/cache"
	"github.com/alicebob/miniredis/v2"
)

// ==================== Broadcast Mode Cache Tests ====================

func setupBroadcastTestRedis(t *testing.T) (*cache.Redis, *miniredis.Miniredis, func()) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	redisCache, err := cache.NewRedis(mr.Addr(), "", 0, 48*time.Hour)
	if err != nil {
		mr.Close()
		t.Fatalf("Failed to create Redis client: %v", err)
	}

	cleanup := func() {
		redisCache.Close()
		mr.Close()
	}

	return redisCache, mr, cleanup
}

func TestBroadcastMode_SetAndGet(t *testing.T) {
	redisCache, _, cleanup := setupBroadcastTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(12345)

	// Not in broadcast mode initially
	inMode, err := redisCache.GetBroadcastMode(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Error getting broadcast mode: %v", err)
	}
	if inMode {
		t.Error("Should not be in broadcast mode initially")
	}

	// Set broadcast mode
	err = redisCache.SetBroadcastMode(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Failed to set broadcast mode: %v", err)
	}

	// Should be in broadcast mode now
	inMode, err = redisCache.GetBroadcastMode(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if !inMode {
		t.Error("Expected to be in broadcast mode")
	}
}

func TestBroadcastMode_Clear(t *testing.T) {
	redisCache, _, cleanup := setupBroadcastTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(12345)

	// Set then clear
	redisCache.SetBroadcastMode(ctx, botToken, adminID)
	err := redisCache.ClearBroadcastMode(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	// Should not be in mode anymore
	inMode, _ := redisCache.GetBroadcastMode(ctx, botToken, adminID)
	if inMode {
		t.Error("Should not be in broadcast mode after clear")
	}
}

func TestBroadcastMode_DifferentAdmins(t *testing.T) {
	redisCache, _, cleanup := setupBroadcastTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"
	admin1 := int64(111)
	admin2 := int64(222)

	// Set for admin1 only
	redisCache.SetBroadcastMode(ctx, botToken, admin1)

	// admin1 should be in mode
	inMode1, _ := redisCache.GetBroadcastMode(ctx, botToken, admin1)
	if !inMode1 {
		t.Error("admin1 should be in broadcast mode")
	}

	// admin2 should not be in mode
	inMode2, _ := redisCache.GetBroadcastMode(ctx, botToken, admin2)
	if inMode2 {
		t.Error("admin2 should not be in broadcast mode")
	}
}

// ==================== Pending Broadcast Tests ====================

func TestPendingBroadcast_SetAndGet(t *testing.T) {
	redisCache, _, cleanup := setupBroadcastTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(12345)
	messageID := 54321

	// Set pending broadcast
	err := redisCache.SetPendingBroadcast(ctx, botToken, adminID, messageID)
	if err != nil {
		t.Fatalf("Failed to set pending broadcast: %v", err)
	}

	// Get pending broadcast
	result, err := redisCache.GetPendingBroadcast(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Failed to get pending broadcast: %v", err)
	}

	if result != messageID {
		t.Errorf("Expected message ID %d, got %d", messageID, result)
	}
}

func TestPendingBroadcast_Clear(t *testing.T) {
	redisCache, _, cleanup := setupBroadcastTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(12345)

	// Set then clear
	redisCache.SetPendingBroadcast(ctx, botToken, adminID, 999)
	err := redisCache.ClearPendingBroadcast(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	// Should return 0
	result, _ := redisCache.GetPendingBroadcast(ctx, botToken, adminID)
	if result != 0 {
		t.Errorf("Expected 0 after clear, got %d", result)
	}
}

func TestPendingBroadcast_NotFound(t *testing.T) {
	redisCache, _, cleanup := setupBroadcastTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	// Try to get non-existent
	result, err := redisCache.GetPendingBroadcast(ctx, "nonexistent", 99999)
	if err != nil {
		t.Logf("Error (expected): %v", err)
	}
	if result != 0 {
		t.Errorf("Expected 0 for non-existent, got %d", result)
	}
}
