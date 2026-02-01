package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Amr-9/botforge/internal/models"
)

// ==================== Message Log & User Analytics Functions ====================

// SaveMessageLog stores the message link in database
func (r *Repository) SaveMessageLog(ctx context.Context, adminMsgID int, userChatID int64, botID int64) error {
	query := `INSERT INTO message_logs (admin_msg_id, user_chat_id, bot_id) VALUES (?, ?, ?)`

	_, err := r.mysql.db.ExecContext(ctx, query, adminMsgID, userChatID, botID)
	if err != nil {
		return fmt.Errorf("failed to save message log: %w", err)
	}

	return nil
}

// GetUserChatID retrieves the user chat ID for a given admin message
func (r *Repository) GetUserChatID(ctx context.Context, adminMsgID int, botID int64) (int64, error) {
	var userChatID int64
	query := `SELECT user_chat_id FROM message_logs WHERE admin_msg_id = ? AND bot_id = ? LIMIT 1`

	err := r.mysql.db.GetContext(ctx, &userChatID, query, adminMsgID, botID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get user chat id: %w", err)
	}

	return userChatID, nil
}

// HasUserInteracted checks if a user has ever messaged a bot
func (r *Repository) HasUserInteracted(ctx context.Context, botID int64, userChatID int64) (bool, error) {
	var exists int
	query := `SELECT 1 FROM message_logs WHERE bot_id = ? AND user_chat_id = ? LIMIT 1`

	err := r.mysql.db.GetContext(ctx, &exists, query, botID, userChatID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check interaction: %w", err)
	}

	return true, nil
}

// GetFirstMessageDate retrieves the timestamp of the first message from a user
func (r *Repository) GetFirstMessageDate(ctx context.Context, botID int64, userChatID int64) (time.Time, error) {
	var createdAt time.Time
	query := `SELECT created_at FROM message_logs WHERE bot_id = ? AND user_chat_id = ? ORDER BY id ASC LIMIT 1`

	err := r.mysql.db.GetContext(ctx, &createdAt, query, botID, userChatID)
	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to get first message date: %w", err)
	}

	return createdAt, nil
}

// GetUniqueUserCount returns the number of unique users tracked for a bot
func (r *Repository) GetUniqueUserCount(ctx context.Context, botID int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(DISTINCT user_chat_id) FROM message_logs WHERE bot_id = ?`

	err := r.mysql.db.GetContext(ctx, &count, query, botID)
	if err != nil {
		return 0, fmt.Errorf("failed to get unique user count: %w", err)
	}

	return count, nil
}

// GetAllUserChatIDs returns all unique user chat IDs for a bot
func (r *Repository) GetAllUserChatIDs(ctx context.Context, botID int64) ([]int64, error) {
	var userChatIDs []int64
	query := `SELECT DISTINCT user_chat_id FROM message_logs WHERE bot_id = ?`

	err := r.mysql.db.SelectContext(ctx, &userChatIDs, query, botID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all user chat ids: %w", err)
	}

	return userChatIDs, nil
}

// ==================== Ban Functions ====================

// BanUser adds a user to the banned list for a bot
func (r *Repository) BanUser(ctx context.Context, botID, userChatID, bannedBy int64) error {
	query := `INSERT INTO banned_users (bot_id, user_chat_id, banned_by)
			  VALUES (?, ?, ?)
			  ON DUPLICATE KEY UPDATE banned_by = ?, created_at = CURRENT_TIMESTAMP`
	_, err := r.mysql.db.ExecContext(ctx, query, botID, userChatID, bannedBy, bannedBy)
	if err != nil {
		return fmt.Errorf("failed to ban user: %w", err)
	}
	return nil
}

// UnbanUser removes a user from the banned list
func (r *Repository) UnbanUser(ctx context.Context, botID, userChatID int64) error {
	query := `DELETE FROM banned_users WHERE bot_id = ? AND user_chat_id = ?`
	_, err := r.mysql.db.ExecContext(ctx, query, botID, userChatID)
	if err != nil {
		return fmt.Errorf("failed to unban user: %w", err)
	}
	return nil
}

// IsUserBanned checks if a user is banned for a specific bot
func (r *Repository) IsUserBanned(ctx context.Context, botID, userChatID int64) (bool, error) {
	var exists int
	query := `SELECT 1 FROM banned_users WHERE bot_id = ? AND user_chat_id = ? LIMIT 1`
	err := r.mysql.db.GetContext(ctx, &exists, query, botID, userChatID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check ban status: %w", err)
	}
	return true, nil
}

// GetBannedUsers retrieves all banned users for a bot with pagination
func (r *Repository) GetBannedUsers(ctx context.Context, botID int64, limit, offset int) ([]models.BannedUser, error) {
	var users []models.BannedUser
	query := `SELECT id, bot_id, user_chat_id, banned_by, created_at
			  FROM banned_users WHERE bot_id = ?
			  ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.mysql.db.SelectContext(ctx, &users, query, botID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get banned users: %w", err)
	}
	return users, nil
}

// GetBannedUserCount returns the count of banned users for a bot
func (r *Repository) GetBannedUserCount(ctx context.Context, botID int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM banned_users WHERE bot_id = ?`
	err := r.mysql.db.GetContext(ctx, &count, query, botID)
	if err != nil {
		return 0, fmt.Errorf("failed to get banned user count: %w", err)
	}
	return count, nil
}

// ==================== Statistics Functions ====================

// GetTotalMessageCount returns the total number of messages for a bot
func (r *Repository) GetTotalMessageCount(ctx context.Context, botID int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM message_logs WHERE bot_id = ?`
	err := r.mysql.db.GetContext(ctx, &count, query, botID)
	if err != nil {
		return 0, fmt.Errorf("failed to get total message count: %w", err)
	}
	return count, nil
}

// GetMessageCountSince returns the number of messages since a given time
func (r *Repository) GetMessageCountSince(ctx context.Context, botID int64, since time.Time) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM message_logs WHERE bot_id = ? AND created_at >= ?`
	err := r.mysql.db.GetContext(ctx, &count, query, botID, since)
	if err != nil {
		return 0, fmt.Errorf("failed to get message count since: %w", err)
	}
	return count, nil
}

// GetActiveUserCount returns the number of unique users active since a given time
func (r *Repository) GetActiveUserCount(ctx context.Context, botID int64, since time.Time) (int64, error) {
	var count int64
	query := `SELECT COUNT(DISTINCT user_chat_id) FROM message_logs WHERE bot_id = ? AND created_at >= ?`
	err := r.mysql.db.GetContext(ctx, &count, query, botID, since)
	if err != nil {
		return 0, fmt.Errorf("failed to get active user count: %w", err)
	}
	return count, nil
}

// GetNewUserCount returns the number of new users (first message) since a given time
// Uses LEFT JOIN for better performance compared to correlated subquery
func (r *Repository) GetNewUserCount(ctx context.Context, botID int64, since time.Time) (int64, error) {
	var count int64
	query := `SELECT COUNT(DISTINCT ml1.user_chat_id)
			  FROM message_logs ml1
			  LEFT JOIN message_logs ml2
				  ON ml1.bot_id = ml2.bot_id
				  AND ml1.user_chat_id = ml2.user_chat_id
				  AND ml2.created_at < ?
			  WHERE ml1.bot_id = ?
				  AND ml1.created_at >= ?
				  AND ml2.id IS NULL`
	err := r.mysql.db.GetContext(ctx, &count, query, since, botID, since)
	if err != nil {
		return 0, fmt.Errorf("failed to get new user count: %w", err)
	}
	return count, nil
}

// GetBotCreatedAt returns the creation date of a bot (first message received)
func (r *Repository) GetBotFirstActivity(ctx context.Context, botID int64) (time.Time, error) {
	var createdAt time.Time
	query := `SELECT MIN(created_at) FROM message_logs WHERE bot_id = ?`
	err := r.mysql.db.GetContext(ctx, &createdAt, query, botID)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get bot first activity: %w", err)
	}
	return createdAt, nil
}

// ==================== Global Statistics Functions (All Bots) ====================

// GetGlobalUniqueUserCount returns the total unique users across all bots
func (r *Repository) GetGlobalUniqueUserCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(DISTINCT user_chat_id) FROM message_logs`
	err := r.mysql.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get global unique user count: %w", err)
	}
	return count, nil
}

// GetGlobalActiveUserCount returns the total active users across all bots since a given time
func (r *Repository) GetGlobalActiveUserCount(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	query := `SELECT COUNT(DISTINCT user_chat_id) FROM message_logs WHERE created_at >= ?`
	err := r.mysql.db.GetContext(ctx, &count, query, since)
	if err != nil {
		return 0, fmt.Errorf("failed to get global active user count: %w", err)
	}
	return count, nil
}

// GetGlobalNewUserCount returns new users across all bots since a given time
func (r *Repository) GetGlobalNewUserCount(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	query := `SELECT COUNT(DISTINCT user_chat_id) FROM message_logs
			  WHERE user_chat_id NOT IN (
				  SELECT DISTINCT user_chat_id FROM message_logs
				  WHERE created_at < ?
			  )
			  AND created_at >= ?`
	err := r.mysql.db.GetContext(ctx, &count, query, since, since)
	if err != nil {
		return 0, fmt.Errorf("failed to get global new user count: %w", err)
	}
	return count, nil
}

// GetGlobalTotalMessageCount returns the total messages across all bots
func (r *Repository) GetGlobalTotalMessageCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM message_logs`
	err := r.mysql.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get global total message count: %w", err)
	}
	return count, nil
}

// GetGlobalMessageCountSince returns total messages across all bots since a given time
func (r *Repository) GetGlobalMessageCountSince(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM message_logs WHERE created_at >= ?`
	err := r.mysql.db.GetContext(ctx, &count, query, since)
	if err != nil {
		return 0, fmt.Errorf("failed to get global message count since: %w", err)
	}
	return count, nil
}

// GetGlobalBannedUserCount returns total banned users across all bots
func (r *Repository) GetGlobalBannedUserCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM banned_users`
	err := r.mysql.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get global banned user count: %w", err)
	}
	return count, nil
}

// GetGlobalAutoReplyCount returns total auto-replies across all bots
func (r *Repository) GetGlobalAutoReplyCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM auto_replies WHERE is_active = TRUE`
	err := r.mysql.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get global auto-reply count: %w", err)
	}
	return count, nil
}

// GetGlobalForcedChannelCount returns total forced channels across all bots
func (r *Repository) GetGlobalForcedChannelCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM forced_channels WHERE is_active = TRUE`
	err := r.mysql.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get global forced channel count: %w", err)
	}
	return count, nil
}

// GetUniqueOwnerCount returns the number of unique bot owners
func (r *Repository) GetUniqueOwnerCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(DISTINCT owner_chat_id) FROM bots WHERE deleted_at IS NULL`
	err := r.mysql.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get unique owner count: %w", err)
	}
	return count, nil
}
