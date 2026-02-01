package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/Amr-9/botforge/internal/cache"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// Helper function to create a test Redis instance with miniredis
func setupTestRedis(t *testing.T) (*cache.Redis, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	r, err := cache.NewRedis(mr.Addr(), "", 0, 48*time.Hour)
	if err != nil {
		mr.Close()
		t.Fatalf("Failed to create Redis client: %v", err)
	}

	return r, mr
}

// ==================== Connection Tests ====================

func TestNewRedis_Success(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}
	defer mr.Close()

	r, err := cache.NewRedis(mr.Addr(), "", 0, 48*time.Hour)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer r.Close()
}

func TestNewRedis_InvalidAddress(t *testing.T) {
	_, err := cache.NewRedis("invalid:99999", "", 0, 48*time.Hour)
	if err == nil {
		t.Error("Expected error for invalid address")
	}
}

func TestPing_Success(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	err := r.Ping(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestClose_Success(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()

	err := r.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got: %v", err)
	}
}

// ==================== Message Link Tests ====================

func TestMessageLink_SetAndGet(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot-token"
	adminMsgID := 12345
	userChatID := int64(987654321)

	// Set
	err := r.SetMessageLink(ctx, botToken, adminMsgID, userChatID)
	if err != nil {
		t.Fatalf("Failed to set message link: %v", err)
	}

	// Get
	result, err := r.GetMessageLink(ctx, botToken, adminMsgID)
	if err != nil {
		t.Fatalf("Failed to get message link: %v", err)
	}

	if result != userChatID {
		t.Errorf("Expected %d, got %d", userChatID, result)
	}
}

func TestMessageLink_NotFound(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()

	result, err := r.GetMessageLink(ctx, "non-existent", 99999)
	if !cache.IsNil(err) {
		t.Errorf("Expected redis.Nil error for cache miss, got: %v", err)
	}
	if result != 0 {
		t.Errorf("Expected 0 for cache miss, got %d", result)
	}
}

func TestMessageLink_Delete(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"
	adminMsgID := 123

	// Set then delete
	r.SetMessageLink(ctx, botToken, adminMsgID, 456)
	err := r.DeleteMessageLink(ctx, botToken, adminMsgID)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Should not find
	_, err = r.GetMessageLink(ctx, botToken, adminMsgID)
	if !cache.IsNil(err) {
		t.Error("Expected cache miss after delete")
	}
}

// ==================== Session Tests ====================

func TestSession_SetAndHas(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"
	userID := int64(123456)

	// No session initially
	has, err := r.HasSession(ctx, botToken, userID)
	if err != nil {
		t.Fatalf("Error checking session: %v", err)
	}
	if has {
		t.Error("Expected no session initially")
	}

	// Set session
	err = r.SetSession(ctx, botToken, userID, 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to set session: %v", err)
	}

	// Should have session now
	has, err = r.HasSession(ctx, botToken, userID)
	if err != nil {
		t.Fatalf("Error checking session: %v", err)
	}
	if !has {
		t.Error("Expected session to exist")
	}
}

// ==================== Broadcast Mode Tests ====================

func TestBroadcastMode_SetGetClear(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(111)

	// Not in broadcast mode initially
	inMode, err := r.GetBroadcastMode(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if inMode {
		t.Error("Should not be in broadcast mode initially")
	}

	// Set broadcast mode
	err = r.SetBroadcastMode(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Failed to set broadcast mode: %v", err)
	}

	// Should be in broadcast mode
	inMode, err = r.GetBroadcastMode(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if !inMode {
		t.Error("Expected to be in broadcast mode")
	}

	// Clear
	err = r.ClearBroadcastMode(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	// Should not be in mode anymore
	inMode, _ = r.GetBroadcastMode(ctx, botToken, adminID)
	if inMode {
		t.Error("Should not be in broadcast mode after clear")
	}
}

// ==================== User State Tests ====================

func TestUserState_SetGetClear(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"
	userID := int64(222)

	// Empty initially
	state, err := r.GetUserState(ctx, botToken, userID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if state != "" {
		t.Error("Expected empty state initially")
	}

	// Set state
	err = r.SetUserState(ctx, botToken, userID, "waiting_for_input")
	if err != nil {
		t.Fatalf("Failed to set state: %v", err)
	}

	// Get state
	state, err = r.GetUserState(ctx, botToken, userID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if state != "waiting_for_input" {
		t.Errorf("Expected 'waiting_for_input', got '%s'", state)
	}

	// Clear state
	err = r.ClearUserState(ctx, botToken, userID)
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	// Should be empty
	state, _ = r.GetUserState(ctx, botToken, userID)
	if state != "" {
		t.Error("Expected empty state after clear")
	}
}

// ==================== Ban Cache Tests ====================

func TestBanCache_SetAndCheck(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"
	userChatID := int64(333)

	// Not banned initially (cache miss)
	isBanned, cacheHit, err := r.IsUserBanned(ctx, botToken, userChatID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if cacheHit {
		t.Error("Expected cache miss initially")
	}
	if isBanned {
		t.Error("Should not be banned initially")
	}

	// Ban user
	err = r.SetUserBanned(ctx, botToken, userChatID)
	if err != nil {
		t.Fatalf("Failed to ban: %v", err)
	}

	// Should be banned now (cache hit)
	isBanned, cacheHit, err = r.IsUserBanned(ctx, botToken, userChatID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if !cacheHit {
		t.Error("Expected cache hit")
	}
	if !isBanned {
		t.Error("Expected user to be banned")
	}
}

func TestBanCache_Remove(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"
	userChatID := int64(444)

	// Ban then unban
	r.SetUserBanned(ctx, botToken, userChatID)
	err := r.RemoveUserBan(ctx, botToken, userChatID)
	if err != nil {
		t.Fatalf("Failed to remove ban: %v", err)
	}

	// Should not be in cache
	isBanned, cacheHit, _ := r.IsUserBanned(ctx, botToken, userChatID)
	if cacheHit || isBanned {
		t.Error("Ban should be removed from cache")
	}
}

func TestBanCache_NegativeCaching(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"
	userChatID := int64(555)

	// Cache that user is NOT banned
	err := r.CacheNotBanned(ctx, botToken, userChatID)
	if err != nil {
		t.Fatalf("Failed to cache not banned: %v", err)
	}

	// Should return true for IsNotBannedCached
	isCached, err := r.IsNotBannedCached(ctx, botToken, userChatID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if !isCached {
		t.Error("Expected not-banned to be cached")
	}

	// Invalidate
	err = r.InvalidateNotBannedCache(ctx, botToken, userChatID)
	if err != nil {
		t.Fatalf("Failed to invalidate: %v", err)
	}

	// Should not be cached anymore
	isCached, _ = r.IsNotBannedCached(ctx, botToken, userChatID)
	if isCached {
		t.Error("Should not be cached after invalidation")
	}
}

// ==================== Auto-Reply Cache Tests ====================

func TestAutoReply_SetAndGet(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"

	// Set auto-reply
	err := r.SetAutoReply(ctx, botToken, "hello", "Hi there!", "keyword")
	if err != nil {
		t.Fatalf("Failed to set auto-reply: %v", err)
	}

	// Get
	response, err := r.GetAutoReply(ctx, botToken, "hello", "keyword")
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if response != "Hi there!" {
		t.Errorf("Expected 'Hi there!', got '%s'", response)
	}
}

func TestAutoReply_NotFound(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()

	response, err := r.GetAutoReply(ctx, "bot", "nonexistent", "keyword")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if response != "" {
		t.Errorf("Expected empty response for non-existent, got '%s'", response)
	}
}

func TestAutoReply_Delete(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"

	// Set then delete
	r.SetAutoReply(ctx, botToken, "goodbye", "Bye!", "keyword")
	err := r.DeleteAutoReply(ctx, botToken, "goodbye", "keyword")
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Should be empty
	response, _ := r.GetAutoReply(ctx, botToken, "goodbye", "keyword")
	if response != "" {
		t.Error("Expected empty after delete")
	}
}

func TestAutoReplyWithMedia_SetAndGet(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"

	cacheData := &cache.AutoReplyCache{
		Response:    "",
		MessageType: "photo",
		FileID:      "AgACAgIAAxkBAAI...",
		Caption:     "Beautiful sunset!",
	}

	// Set
	err := r.SetAutoReplyWithMedia(ctx, botToken, "sunset", cacheData, "keyword")
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Get
	result, err := r.GetAutoReplyWithMedia(ctx, botToken, "sunset", "keyword")
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if result.MessageType != "photo" {
		t.Errorf("Expected 'photo', got '%s'", result.MessageType)
	}
	if result.FileID != "AgACAgIAAxkBAAI..." {
		t.Errorf("FileID mismatch")
	}
	if result.Caption != "Beautiful sunset!" {
		t.Errorf("Caption mismatch")
	}
}

// ==================== Temp Data Tests ====================

func TestTempData_SetGetClear(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"
	userID := int64(666)

	// Set
	err := r.SetTempData(ctx, botToken, userID, "trigger_word", "hello")
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Get
	val, err := r.GetTempData(ctx, botToken, userID, "trigger_word")
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if val != "hello" {
		t.Errorf("Expected 'hello', got '%s'", val)
	}

	// Clear
	err = r.ClearTempData(ctx, botToken, userID, "trigger_word")
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	// Should be empty
	val, _ = r.GetTempData(ctx, botToken, userID, "trigger_word")
	if val != "" {
		t.Error("Expected empty after clear")
	}
}

// ==================== Schedule State Tests ====================

func TestScheduleState_SetAndGet(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(777)

	// Set
	err := r.SetScheduleState(ctx, botToken, adminID, "awaiting_message")
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Get
	state, err := r.GetScheduleState(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if state != "awaiting_message" {
		t.Errorf("Expected 'awaiting_message', got '%s'", state)
	}
}

func TestScheduleMessageData_SetAndGet(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(888)

	// Set
	err := r.SetScheduleMessageData(ctx, botToken, adminID, "photo", "", "FileID123", "Caption here")
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Get
	msgType, _, fileID, caption, err := r.GetScheduleMessageData(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if msgType != "photo" {
		t.Errorf("Expected msgType 'photo', got '%s'", msgType)
	}
	if fileID != "FileID123" {
		t.Errorf("Expected fileID 'FileID123', got '%s'", fileID)
	}
	if caption != "Caption here" {
		t.Errorf("Expected caption 'Caption here', got '%s'", caption)
	}
}

func TestScheduleData_Clear(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"
	adminID := int64(999)

	// Set data
	r.SetScheduleState(ctx, botToken, adminID, "test-state")
	r.SetScheduleMessageData(ctx, botToken, adminID, "text", "Hello", "", "")

	// Clear all
	err := r.ClearScheduleData(ctx, botToken, adminID)
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	// Verify cleared
	state, _ := r.GetScheduleState(ctx, botToken, adminID)
	if state != "" {
		t.Error("State should be cleared")
	}
}

// ==================== Forced Subscription Tests ====================

func TestForcedSubEnabled_SetAndGet(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"

	// Set enabled
	err := r.SetForcedSubEnabled(ctx, botToken, true)
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Get
	enabled, cacheHit, err := r.GetForcedSubEnabled(ctx, botToken)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if !cacheHit {
		t.Error("Expected cache hit")
	}
	if !enabled {
		t.Error("Expected enabled=true")
	}

	// Set disabled
	r.SetForcedSubEnabled(ctx, botToken, false)
	enabled, _, _ = r.GetForcedSubEnabled(ctx, botToken)
	if enabled {
		t.Error("Expected enabled=false")
	}
}

func TestUserSubVerified_SetAndCheck(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"
	userID := int64(1111)

	// Not verified initially
	verified, err := r.IsUserSubVerified(ctx, botToken, userID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if verified {
		t.Error("Should not be verified initially")
	}

	// Set verified
	err = r.SetUserSubVerified(ctx, botToken, userID)
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Should be verified
	verified, err = r.IsUserSubVerified(ctx, botToken, userID)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if !verified {
		t.Error("Expected verified=true")
	}

	// Clear
	r.ClearUserSubVerified(ctx, botToken, userID)
	verified, _ = r.IsUserSubVerified(ctx, botToken, userID)
	if verified {
		t.Error("Should not be verified after clear")
	}
}

// ==================== Bot Settings Cache Tests ====================

func TestShowSentConfirmation_SetAndGet(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"

	// Set
	err := r.SetShowSentConfirmation(ctx, botToken, false)
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Get
	show, cacheHit, err := r.GetShowSentConfirmation(ctx, botToken)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if !cacheHit {
		t.Error("Expected cache hit")
	}
	if show {
		t.Error("Expected show=false")
	}
}

func TestStartMessage_SetAndGet(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"

	// Set
	err := r.SetStartMessage(ctx, botToken, "Welcome to my bot!")
	if err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Get
	msg, cacheHit, err := r.GetStartMessage(ctx, botToken)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if !cacheHit {
		t.Error("Expected cache hit")
	}
	if msg != "Welcome to my bot!" {
		t.Errorf("Expected 'Welcome to my bot!', got '%s'", msg)
	}
}

func TestPreloadBotSettings(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"

	// Preload all settings
	err := r.PreloadBotSettings(ctx, botToken, "Hello!", true, false, true)
	if err != nil {
		t.Fatalf("Failed to preload: %v", err)
	}

	// Verify all settings
	msg, hit, _ := r.GetStartMessage(ctx, botToken)
	if !hit || msg != "Hello!" {
		t.Error("Start message not preloaded correctly")
	}

	forward, hit, _ := r.GetForwardAutoReplies(ctx, botToken)
	if !hit || !forward {
		t.Error("Forward replies not preloaded correctly")
	}

	show, hit, _ := r.GetShowSentConfirmation(ctx, botToken)
	if !hit || show {
		t.Error("Show sent confirmation not preloaded correctly")
	}

	enabled, hit, _ := r.GetForcedSubEnabled(ctx, botToken)
	if !hit || !enabled {
		t.Error("Forced sub enabled not preloaded correctly")
	}
}

func TestInvalidateAllBotSettings(t *testing.T) {
	r, mr := setupTestRedis(t)
	defer mr.Close()
	defer r.Close()

	ctx := context.Background()
	botToken := "test-bot"

	// Preload then invalidate
	r.PreloadBotSettings(ctx, botToken, "Hello!", true, true, true)
	err := r.InvalidateAllBotSettings(ctx, botToken)
	if err != nil {
		t.Fatalf("Failed to invalidate: %v", err)
	}

	// All should be cache miss now
	_, hit, _ := r.GetStartMessage(ctx, botToken)
	if hit {
		t.Error("Start message should be invalidated")
	}

	_, hit, _ = r.GetForcedSubEnabled(ctx, botToken)
	if hit {
		t.Error("Forced sub should be invalidated")
	}
}

// ==================== IsNil Helper Test ====================

func TestIsNil(t *testing.T) {
	if !cache.IsNil(redis.Nil) {
		t.Error("IsNil should return true for redis.Nil")
	}

	if cache.IsNil(nil) {
		t.Error("IsNil should return false for nil")
	}

	if cache.IsNil(redis.TxFailedErr) {
		t.Error("IsNil should return false for other errors")
	}
}
