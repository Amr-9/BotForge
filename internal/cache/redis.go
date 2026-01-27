package cache

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis wraps the redis client with message caching operations
type Redis struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedis creates a new Redis connection
func NewRedis(addr, password string, db int, ttl time.Duration) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Connected to Redis successfully")

	return &Redis{
		client: client,
		ttl:    ttl,
	}, nil
}

// generateKey creates a Redis key for message mapping
// Format: msg:{bot_token}:{admin_msg_id}
func (r *Redis) generateKey(botToken string, adminMsgID int) string {
	return fmt.Sprintf("msg:%s:%d", botToken, adminMsgID)
}

// SetMessageLink stores the mapping between admin message and user chat with TTL
func (r *Redis) SetMessageLink(ctx context.Context, botToken string, adminMsgID int, userChatID int64) error {
	key := r.generateKey(botToken, adminMsgID)
	value := strconv.FormatInt(userChatID, 10)

	err := r.client.Set(ctx, key, value, r.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set message link in Redis: %w", err)
	}

	return nil
}

// GetMessageLink retrieves the user chat ID for a given admin message
// Returns 0 and redis.Nil error if key not found (cache miss)
func (r *Redis) GetMessageLink(ctx context.Context, botToken string, adminMsgID int) (int64, error) {
	key := r.generateKey(botToken, adminMsgID)

	value, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, redis.Nil // Cache miss
		}
		return 0, fmt.Errorf("failed to get message link from Redis: %w", err)
	}

	userChatID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse user chat ID: %w", err)
	}

	return userChatID, nil
}

// DeleteMessageLink removes a message link from cache
func (r *Redis) DeleteMessageLink(ctx context.Context, botToken string, adminMsgID int) error {
	key := r.generateKey(botToken, adminMsgID)

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete message link from Redis: %w", err)
	}

	return nil
}

// HasSession checks if a user has an active session with a bot
func (r *Redis) HasSession(ctx context.Context, botToken string, userID int64) (bool, error) {
	key := fmt.Sprintf("session:%s:%d", botToken, userID)
	_, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// SetSession updates the session activity for a user
func (r *Redis) SetSession(ctx context.Context, botToken string, userID int64, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s:%d", botToken, userID)
	return r.client.Set(ctx, key, "active", ttl).Err()
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	return r.client.Close()
}

// Ping checks if Redis connection is alive
func (r *Redis) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// SetBroadcastMode sets the broadcast state for an admin
func (r *Redis) SetBroadcastMode(ctx context.Context, botToken string, adminID int64) error {
	key := fmt.Sprintf("broadcast_mode:%s:%d", botToken, adminID)
	return r.client.Set(ctx, key, "true", 10*time.Minute).Err()
}

// GetBroadcastMode checks if admin is in broadcast mode
func (r *Redis) GetBroadcastMode(ctx context.Context, botToken string, adminID int64) (bool, error) {
	key := fmt.Sprintf("broadcast_mode:%s:%d", botToken, adminID)
	_, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ClearBroadcastMode removes the broadcast state
func (r *Redis) ClearBroadcastMode(ctx context.Context, botToken string, adminID int64) error {
	key := fmt.Sprintf("broadcast_mode:%s:%d", botToken, adminID)
	return r.client.Del(ctx, key).Err()
}

// IsNil checks if error is redis.Nil (cache miss)
func IsNil(err error) bool {
	return err == redis.Nil
}

// SetUserState sets a temporary state for a user (e.g. waiting for input)
func (r *Redis) SetUserState(ctx context.Context, botToken string, userID int64, state string) error {
	key := fmt.Sprintf("state:%s:%d", botToken, userID)
	return r.client.Set(ctx, key, state, 5*time.Minute).Err()
}

// GetUserState retrieves the current state of a user
func (r *Redis) GetUserState(ctx context.Context, botToken string, userID int64) (string, error) {
	key := fmt.Sprintf("state:%s:%d", botToken, userID)
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// ClearUserState clears the user state
func (r *Redis) ClearUserState(ctx context.Context, botToken string, userID int64) error {
	key := fmt.Sprintf("state:%s:%d", botToken, userID)
	return r.client.Del(ctx, key).Err()
}

// SetUserBanned caches the ban status for a user
func (r *Redis) SetUserBanned(ctx context.Context, botToken string, userChatID int64) error {
	key := fmt.Sprintf("ban:%s:%d", botToken, userChatID)
	return r.client.Set(ctx, key, "1", 24*time.Hour).Err()
}

// IsUserBanned checks if user is banned (cache layer)
// Returns: (isBanned, cacheHit, error)
func (r *Redis) IsUserBanned(ctx context.Context, botToken string, userChatID int64) (bool, bool, error) {
	key := fmt.Sprintf("ban:%s:%d", botToken, userChatID)
	_, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, false, nil // Not in cache
	}
	if err != nil {
		return false, false, err
	}
	return true, true, nil // Banned and cached
}

// RemoveUserBan removes the ban status from cache
func (r *Redis) RemoveUserBan(ctx context.Context, botToken string, userChatID int64) error {
	key := fmt.Sprintf("ban:%s:%d", botToken, userChatID)
	return r.client.Del(ctx, key).Err()
}

// CacheNotBanned caches that a user is NOT banned (negative caching)
func (r *Redis) CacheNotBanned(ctx context.Context, botToken string, userChatID int64) error {
	key := fmt.Sprintf("notban:%s:%d", botToken, userChatID)
	return r.client.Set(ctx, key, "0", 5*time.Minute).Err()
}

// IsNotBannedCached checks if we have cached that user is NOT banned
func (r *Redis) IsNotBannedCached(ctx context.Context, botToken string, userChatID int64) (bool, error) {
	key := fmt.Sprintf("notban:%s:%d", botToken, userChatID)
	_, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// InvalidateNotBannedCache removes the "not banned" cache when user gets banned
func (r *Redis) InvalidateNotBannedCache(ctx context.Context, botToken string, userChatID int64) error {
	key := fmt.Sprintf("notban:%s:%d", botToken, userChatID)
	return r.client.Del(ctx, key).Err()
}

// SetPendingBroadcast stores the message ID for pending broadcast confirmation
func (r *Redis) SetPendingBroadcast(ctx context.Context, botToken string, adminID int64, msgID int) error {
	key := fmt.Sprintf("pending_broadcast:%s:%d", botToken, adminID)
	return r.client.Set(ctx, key, strconv.Itoa(msgID), 10*time.Minute).Err()
}

// GetPendingBroadcast retrieves the pending broadcast message ID
func (r *Redis) GetPendingBroadcast(ctx context.Context, botToken string, adminID int64) (int, error) {
	key := fmt.Sprintf("pending_broadcast:%s:%d", botToken, adminID)
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	msgID, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return msgID, nil
}

// ClearPendingBroadcast removes the pending broadcast message
func (r *Redis) ClearPendingBroadcast(ctx context.Context, botToken string, adminID int64) error {
	key := fmt.Sprintf("pending_broadcast:%s:%d", botToken, adminID)
	return r.client.Del(ctx, key).Err()
}

// ==================== Auto-Reply Cache Functions ====================

// SetAutoReply caches an auto-reply response
func (r *Redis) SetAutoReply(ctx context.Context, botToken, trigger, response, triggerType string) error {
	key := fmt.Sprintf("autoreply:%s:%s:%s", botToken, triggerType, trigger)
	return r.client.Set(ctx, key, response, 24*time.Hour).Err()
}

// GetAutoReply retrieves a cached auto-reply response
func (r *Redis) GetAutoReply(ctx context.Context, botToken, trigger, triggerType string) (string, error) {
	key := fmt.Sprintf("autoreply:%s:%s:%s", botToken, triggerType, trigger)
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// DeleteAutoReply removes a cached auto-reply
func (r *Redis) DeleteAutoReply(ctx context.Context, botToken, trigger, triggerType string) error {
	key := fmt.Sprintf("autoreply:%s:%s:%s", botToken, triggerType, trigger)
	return r.client.Del(ctx, key).Err()
}

// GetAllAutoReplies loads all auto-replies of a specific type for a bot from cache
// Returns a map of trigger -> response
func (r *Redis) GetAllAutoReplies(ctx context.Context, botToken, triggerType string) (map[string]string, error) {
	pattern := fmt.Sprintf("autoreply:%s:%s:*", botToken, triggerType)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, nil
	}

	result := make(map[string]string)
	prefix := fmt.Sprintf("autoreply:%s:%s:", botToken, triggerType)

	for _, key := range keys {
		val, err := r.client.Get(ctx, key).Result()
		if err == nil {
			// Extract trigger from key
			trigger := key[len(prefix):]
			result[trigger] = val
		}
	}

	return result, nil
}

// ==================== Temp Data Cache Functions ====================

// SetTempData stores temporary data during multi-step flows
func (r *Redis) SetTempData(ctx context.Context, botToken string, userID int64, key, value string) error {
	redisKey := fmt.Sprintf("temp:%s:%d:%s", botToken, userID, key)
	return r.client.Set(ctx, redisKey, value, 10*time.Minute).Err()
}

// GetTempData retrieves temporary data
func (r *Redis) GetTempData(ctx context.Context, botToken string, userID int64, key string) (string, error) {
	redisKey := fmt.Sprintf("temp:%s:%d:%s", botToken, userID, key)
	val, err := r.client.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// ClearTempData removes temporary data
func (r *Redis) ClearTempData(ctx context.Context, botToken string, userID int64, key string) error {
	redisKey := fmt.Sprintf("temp:%s:%d:%s", botToken, userID, key)
	return r.client.Del(ctx, redisKey).Err()
}
