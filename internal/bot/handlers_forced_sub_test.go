package bot

import (
	"context"
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/cache"
	"github.com/alicebob/miniredis/v2"
)

// ==================== Setup ====================

func setupForcedSubCache(t *testing.T) (*cache.Redis, func()) {
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

	return redisCache, cleanup
}

// ==================== Forced Sub Enabled Tests ====================

func TestForcedSubEnabled_SetAndGet(t *testing.T) {
	redisCache, cleanup := setupForcedSubCache(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"

	// Initially cache miss
	_, hit, _ := redisCache.GetForcedSubEnabled(ctx, botToken)
	if hit {
		t.Error("Expected cache miss initially")
	}

	// Set enabled = true
	err := redisCache.SetForcedSubEnabled(ctx, botToken, true)
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Should be cache hit with enabled = true
	enabled, hit, err := redisCache.GetForcedSubEnabled(ctx, botToken)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if !hit {
		t.Error("Expected cache hit")
	}
	if !enabled {
		t.Error("Expected enabled=true")
	}

	// Set enabled = false
	redisCache.SetForcedSubEnabled(ctx, botToken, false)
	enabled, _, _ = redisCache.GetForcedSubEnabled(ctx, botToken)
	if enabled {
		t.Error("Expected enabled=false")
	}
}

func TestForcedSubEnabled_Toggle(t *testing.T) {
	redisCache, cleanup := setupForcedSubCache(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "bot1"

	// Enable
	redisCache.SetForcedSubEnabled(ctx, botToken, true)
	enabled, _, _ := redisCache.GetForcedSubEnabled(ctx, botToken)
	if !enabled {
		t.Error("Should be enabled")
	}

	// Disable
	redisCache.SetForcedSubEnabled(ctx, botToken, false)
	enabled, _, _ = redisCache.GetForcedSubEnabled(ctx, botToken)
	if enabled {
		t.Error("Should be disabled")
	}
}

func TestForcedSubEnabled_DifferentBots(t *testing.T) {
	redisCache, cleanup := setupForcedSubCache(t)
	defer cleanup()

	ctx := context.Background()

	// Different settings for different bots
	redisCache.SetForcedSubEnabled(ctx, "bot1", true)
	redisCache.SetForcedSubEnabled(ctx, "bot2", false)

	e1, _, _ := redisCache.GetForcedSubEnabled(ctx, "bot1")
	e2, _, _ := redisCache.GetForcedSubEnabled(ctx, "bot2")

	if !e1 {
		t.Error("bot1 should have forced sub enabled")
	}
	if e2 {
		t.Error("bot2 should have forced sub disabled")
	}
}

func TestForcedSubEnabled_Invalidation(t *testing.T) {
	redisCache, cleanup := setupForcedSubCache(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "bot1"

	// Set and verify
	redisCache.SetForcedSubEnabled(ctx, botToken, true)
	_, hit, _ := redisCache.GetForcedSubEnabled(ctx, botToken)
	if !hit {
		t.Error("Expected cache hit")
	}

	// Invalidate
	err := redisCache.InvalidateForcedSubEnabled(ctx, botToken)
	if err != nil {
		t.Fatalf("Failed to invalidate: %v", err)
	}

	// Should be cache miss now
	_, hit, _ = redisCache.GetForcedSubEnabled(ctx, botToken)
	if hit {
		t.Error("Expected cache miss after invalidation")
	}
}

// ==================== User Subscription Verification Tests ====================

func TestUserSubVerified_SetAndCheck(t *testing.T) {
	redisCache, cleanup := setupForcedSubCache(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "bot1"
	userID := int64(12345)

	// Initially not verified
	verified, err := redisCache.IsUserSubVerified(ctx, botToken, userID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if verified {
		t.Error("Should not be verified initially")
	}

	// Mark as verified
	err = redisCache.SetUserSubVerified(ctx, botToken, userID)
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Now should be verified
	verified, err = redisCache.IsUserSubVerified(ctx, botToken, userID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if !verified {
		t.Error("Should be verified after SetUserSubVerified")
	}
}

func TestUserSubVerified_Clear(t *testing.T) {
	redisCache, cleanup := setupForcedSubCache(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "bot1"
	userID := int64(12345)

	// Verify then clear
	redisCache.SetUserSubVerified(ctx, botToken, userID)
	redisCache.ClearUserSubVerified(ctx, botToken, userID)

	verified, _ := redisCache.IsUserSubVerified(ctx, botToken, userID)
	if verified {
		t.Error("Should not be verified after clear")
	}
}

func TestUserSubVerified_DifferentUsers(t *testing.T) {
	redisCache, cleanup := setupForcedSubCache(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "bot1"

	// Verify different users
	redisCache.SetUserSubVerified(ctx, botToken, 111)
	redisCache.SetUserSubVerified(ctx, botToken, 222)

	v1, _ := redisCache.IsUserSubVerified(ctx, botToken, 111)
	v2, _ := redisCache.IsUserSubVerified(ctx, botToken, 222)
	v3, _ := redisCache.IsUserSubVerified(ctx, botToken, 333)

	if !v1 || !v2 {
		t.Error("Users 111 and 222 should be verified")
	}
	if v3 {
		t.Error("User 333 should not be verified")
	}
}

// ==================== User State Tests ====================

func TestUserState_ForcedSubStates(t *testing.T) {
	redisCache, cleanup := setupForcedSubCache(t)
	defer cleanup()

	ctx := context.Background()
	botToken := "test-bot"
	userID := int64(12345)

	// Test add_forced_channel state
	err := redisCache.SetUserState(ctx, botToken, userID, "add_forced_channel")
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	state, err := redisCache.GetUserState(ctx, botToken, userID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if state != "add_forced_channel" {
		t.Errorf("Expected 'add_forced_channel', got '%s'", state)
	}

	// Test set_forced_sub_message state
	redisCache.SetUserState(ctx, botToken, userID, "set_forced_sub_message")
	state, _ = redisCache.GetUserState(ctx, botToken, userID)
	if state != "set_forced_sub_message" {
		t.Errorf("Expected 'set_forced_sub_message', got '%s'", state)
	}

	// Clear state
	redisCache.ClearUserState(ctx, botToken, userID)
	state, _ = redisCache.GetUserState(ctx, botToken, userID)
	if state != "" {
		t.Error("Expected empty state after clear")
	}
}
