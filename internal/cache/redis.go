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

// IsNil checks if error is redis.Nil (cache miss)
func IsNil(err error) bool {
	return err == redis.Nil
}
