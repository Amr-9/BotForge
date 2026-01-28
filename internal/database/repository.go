package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Amr-9/botforge/internal/models"
	"github.com/Amr-9/botforge/internal/utils/crypto"
)

// Repository handles all database operations
type Repository struct {
	mysql         *MySQL
	encryptionKey string
}

// NewRepository creates a new repository instance
func NewRepository(mysql *MySQL, encryptionKey string) *Repository {
	return &Repository{
		mysql:         mysql,
		encryptionKey: encryptionKey,
	}
}

// CreateBot inserts a new bot into the database
func (r *Repository) CreateBot(ctx context.Context, token string, ownerChatID int64) (*models.Bot, error) {
	encryptedToken, err := crypto.EncryptDeterministic(token, r.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	query := `INSERT INTO bots (token, owner_chat_id, is_active, start_message) VALUES (?, ?, TRUE, '')`

	result, err := r.mysql.db.ExecContext(ctx, query, encryptedToken, ownerChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &models.Bot{
		ID:           id,
		Token:        token, // Return original token to caller
		OwnerChatID:  ownerChatID,
		IsActive:     true,
		StartMessage: "",
		CreatedAt:    time.Now(),
	}, nil
}

// GetBotByToken retrieves a bot by its token (excludes soft-deleted bots)
func (r *Repository) GetBotByToken(ctx context.Context, token string) (*models.Bot, error) {
	encryptedToken, err := crypto.EncryptDeterministic(token, r.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token for lookup: %w", err)
	}

	var bot models.Bot
	query := `SELECT id, token, owner_chat_id, is_active, COALESCE(start_message, '') as start_message,
			  COALESCE(forward_auto_replies, TRUE) as forward_auto_replies,
			  COALESCE(forced_sub_enabled, FALSE) as forced_sub_enabled,
			  COALESCE(forced_sub_message, '') as forced_sub_message, created_at
			  FROM bots WHERE token = ? AND deleted_at IS NULL`

	err = r.mysql.db.GetContext(ctx, &bot, query, encryptedToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get bot: %w", err)
	}

	// Decrypt token before returning (though we already know it matches input)
	decryptedToken, err := crypto.DecryptDeterministic(bot.Token, r.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("database data corruption: failed to decrypt token: %w", err)
	}
	bot.Token = decryptedToken

	return &bot, nil
}

// GetDeletedBotByToken retrieves a soft-deleted bot by its token (for restore)
func (r *Repository) GetDeletedBotByToken(ctx context.Context, token string) (*models.Bot, error) {
	encryptedToken, err := crypto.EncryptDeterministic(token, r.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token for lookup: %w", err)
	}

	var bot models.Bot
	query := `SELECT id, token, owner_chat_id, is_active, COALESCE(start_message, '') as start_message, created_at
			  FROM bots WHERE token = ? AND deleted_at IS NOT NULL`

	err = r.mysql.db.GetContext(ctx, &bot, query, encryptedToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get deleted bot: %w", err)
	}

	decryptedToken, err := crypto.DecryptDeterministic(bot.Token, r.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("database data corruption: failed to decrypt token: %w", err)
	}
	bot.Token = decryptedToken

	return &bot, nil
}

// RestoreBot restores a soft-deleted bot
func (r *Repository) RestoreBot(ctx context.Context, token string, ownerChatID int64) error {
	encryptedToken, err := crypto.EncryptDeterministic(token, r.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	query := `UPDATE bots SET deleted_at = NULL, is_active = TRUE, owner_chat_id = ? WHERE token = ?`

	_, err = r.mysql.db.ExecContext(ctx, query, ownerChatID, encryptedToken)
	if err != nil {
		return fmt.Errorf("failed to restore bot: %w", err)
	}

	return nil
}

// GetAllBots retrieves all non-deleted bots (both active and inactive)
func (r *Repository) GetAllBots(ctx context.Context) ([]models.Bot, error) {
	var bots []models.Bot
	query := `SELECT id, token, owner_chat_id, is_active, COALESCE(start_message, '') as start_message, created_at
			  FROM bots WHERE deleted_at IS NULL`

	err := r.mysql.db.SelectContext(ctx, &bots, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all bots: %w", err)
	}

	// Decrypt all tokens
	for i := range bots {
		decrypted, err := crypto.DecryptDeterministic(bots[i].Token, r.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt bot token (ID: %d): %w", bots[i].ID, err)
		}
		bots[i].Token = decrypted
	}

	return bots, nil
}

// GetDeletedBotsCount returns the count of soft-deleted bots
func (r *Repository) GetDeletedBotsCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM bots WHERE deleted_at IS NOT NULL`

	err := r.mysql.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted bots count: %w", err)
	}

	return count, nil
}

// GetActiveBots retrieves all active bots (excludes soft-deleted)
func (r *Repository) GetActiveBots(ctx context.Context) ([]models.Bot, error) {
	var bots []models.Bot
	query := `SELECT id, token, owner_chat_id, is_active, COALESCE(start_message, '') as start_message, created_at
			  FROM bots WHERE is_active = TRUE AND deleted_at IS NULL`

	err := r.mysql.db.SelectContext(ctx, &bots, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active bots: %w", err)
	}

	// Decrypt all tokens
	for i := range bots {
		decrypted, err := crypto.DecryptDeterministic(bots[i].Token, r.encryptionKey)
		if err != nil {
			// creating a placeholder or skipping? failing here is critical.
			// Let's log error but maybe valid for now? No, better error out.
			return nil, fmt.Errorf("failed to decrypt bot token (ID: %d): %w", bots[i].ID, err)
		}
		bots[i].Token = decrypted
	}

	return bots, nil
}

// DeactivateBot sets is_active to false for a bot
func (r *Repository) DeactivateBot(ctx context.Context, token string) error {
	encryptedToken, err := crypto.EncryptDeterministic(token, r.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	query := `UPDATE bots SET is_active = FALSE WHERE token = ?`

	_, err = r.mysql.db.ExecContext(ctx, query, encryptedToken)
	if err != nil {
		return fmt.Errorf("failed to deactivate bot: %w", err)
	}

	return nil
}

// ActivateBot sets is_active to true for a bot
func (r *Repository) ActivateBot(ctx context.Context, token string) error {
	encryptedToken, err := crypto.EncryptDeterministic(token, r.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	query := `UPDATE bots SET is_active = TRUE WHERE token = ?`

	_, err = r.mysql.db.ExecContext(ctx, query, encryptedToken)
	if err != nil {
		return fmt.Errorf("failed to activate bot: %w", err)
	}

	return nil
}

// UpdateBotStartMessage updates the welcome message for a bot
func (r *Repository) UpdateBotStartMessage(ctx context.Context, botID int64, message string) error {
	query := `UPDATE bots SET start_message = ? WHERE id = ?`

	_, err := r.mysql.db.ExecContext(ctx, query, message, botID)
	if err != nil {
		return fmt.Errorf("failed to update start message: %w", err)
	}

	return nil
}

// UpdateBotForwardAutoReplies updates the forward_auto_replies setting for a bot
func (r *Repository) UpdateBotForwardAutoReplies(ctx context.Context, botID int64, forward bool) error {
	query := `UPDATE bots SET forward_auto_replies = ? WHERE id = ?`

	_, err := r.mysql.db.ExecContext(ctx, query, forward, botID)
	if err != nil {
		return fmt.Errorf("failed to update forward_auto_replies: %w", err)
	}

	return nil
}

// DeleteBot performs a soft delete by setting deleted_at timestamp
func (r *Repository) DeleteBot(ctx context.Context, token string) error {
	encryptedToken, err := crypto.EncryptDeterministic(token, r.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	query := `UPDATE bots SET deleted_at = NOW(), is_active = FALSE WHERE token = ? AND deleted_at IS NULL`

	_, err = r.mysql.db.ExecContext(ctx, query, encryptedToken)
	if err != nil {
		return fmt.Errorf("failed to soft delete bot: %w", err)
	}

	return nil
}

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
			return time.Time{}, nil // Should ideally not happen if calling logic is correct
		}
		return time.Time{}, fmt.Errorf("failed to get first message date: %w", err)
	}

	return createdAt, nil
}

// GetBotsByOwner retrieves all bots owned by a specific user (excludes soft-deleted)
func (r *Repository) GetBotsByOwner(ctx context.Context, ownerChatID int64) ([]models.Bot, error) {
	var bots []models.Bot
	query := `SELECT id, token, owner_chat_id, is_active, COALESCE(start_message, '') as start_message, created_at
			  FROM bots WHERE owner_chat_id = ? AND deleted_at IS NULL`

	err := r.mysql.db.SelectContext(ctx, &bots, query, ownerChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bots by owner: %w", err)
	}

	// Decrypt all tokens
	for i := range bots {
		decrypted, err := crypto.DecryptDeterministic(bots[i].Token, r.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt bot token: %w", err)
		}
		bots[i].Token = decrypted
	}

	return bots, nil
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

// ==================== Auto-Reply Functions ====================

// CreateAutoReply creates a new auto-reply or custom command
func (r *Repository) CreateAutoReply(ctx context.Context, botID int64, trigger, response, triggerType, matchType string) error {
	query := `INSERT INTO auto_replies (bot_id, trigger_word, response, trigger_type, match_type, is_active)
			  VALUES (?, ?, ?, ?, ?, TRUE)
			  ON DUPLICATE KEY UPDATE response = ?, match_type = ?, is_active = TRUE`

	_, err := r.mysql.db.ExecContext(ctx, query, botID, trigger, response, triggerType, matchType, response, matchType)
	if err != nil {
		return fmt.Errorf("failed to create auto-reply: %w", err)
	}
	return nil
}

// GetAutoReplies retrieves all auto-replies or commands for a bot
func (r *Repository) GetAutoReplies(ctx context.Context, botID int64, triggerType string) ([]models.AutoReply, error) {
	var replies []models.AutoReply
	query := `SELECT id, bot_id, trigger_word, response, trigger_type, match_type, is_active, created_at
			  FROM auto_replies WHERE bot_id = ? AND trigger_type = ? AND is_active = TRUE
			  ORDER BY created_at DESC`

	err := r.mysql.db.SelectContext(ctx, &replies, query, botID, triggerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get auto-replies: %w", err)
	}
	return replies, nil
}

// GetAutoReplyByTrigger finds an auto-reply by its trigger word
func (r *Repository) GetAutoReplyByTrigger(ctx context.Context, botID int64, trigger, triggerType string) (*models.AutoReply, error) {
	var reply models.AutoReply
	query := `SELECT id, bot_id, trigger_word, response, trigger_type, match_type, is_active, created_at
			  FROM auto_replies WHERE bot_id = ? AND trigger_word = ? AND trigger_type = ?`

	err := r.mysql.db.GetContext(ctx, &reply, query, botID, trigger, triggerType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get auto-reply: %w", err)
	}
	return &reply, nil
}

// GetAutoReplyByID retrieves an auto-reply by its ID
func (r *Repository) GetAutoReplyByID(ctx context.Context, replyID int64) (*models.AutoReply, error) {
	var reply models.AutoReply
	query := `SELECT id, bot_id, trigger_word, response, trigger_type, match_type, is_active, created_at
			  FROM auto_replies WHERE id = ?`

	err := r.mysql.db.GetContext(ctx, &reply, query, replyID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get auto-reply by ID: %w", err)
	}
	return &reply, nil
}

// DeleteAutoReply removes an auto-reply by ID
func (r *Repository) DeleteAutoReply(ctx context.Context, botID, replyID int64) error {
	query := `DELETE FROM auto_replies WHERE id = ? AND bot_id = ?`
	_, err := r.mysql.db.ExecContext(ctx, query, replyID, botID)
	if err != nil {
		return fmt.Errorf("failed to delete auto-reply: %w", err)
	}
	return nil
}

// GetAutoReplyCount returns the count of auto-replies for a bot by type
func (r *Repository) GetAutoReplyCount(ctx context.Context, botID int64, triggerType string) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM auto_replies WHERE bot_id = ? AND trigger_type = ? AND is_active = TRUE`
	err := r.mysql.db.GetContext(ctx, &count, query, botID, triggerType)
	if err != nil {
		return 0, fmt.Errorf("failed to get auto-reply count: %w", err)
	}
	return count, nil
}

// ==================== Scheduled Messages Methods ====================

// CreateScheduledMessage inserts a new scheduled message
func (r *Repository) CreateScheduledMessage(ctx context.Context, msg *models.ScheduledMessage) (int64, error) {
	query := `INSERT INTO scheduled_messages
		(bot_id, owner_chat_id, message_type, message_text, file_id, caption,
		schedule_type, scheduled_time, time_of_day, day_of_week, status, next_run_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.mysql.db.ExecContext(ctx, query,
		msg.BotID, msg.OwnerChatID, msg.MessageType, msg.MessageText, msg.FileID, msg.Caption,
		msg.ScheduleType, msg.ScheduledTime, msg.TimeOfDay, msg.DayOfWeek, msg.Status, msg.NextRunAt)

	if err != nil {
		return 0, fmt.Errorf("failed to create scheduled message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// GetScheduledMessagesByBot retrieves all scheduled messages for a bot
func (r *Repository) GetScheduledMessagesByBot(ctx context.Context, botID int64, limit, offset int) ([]models.ScheduledMessage, error) {
	var messages []models.ScheduledMessage
	query := `SELECT * FROM scheduled_messages
		WHERE bot_id = ? AND status IN ('pending', 'paused')
		ORDER BY created_at DESC LIMIT ? OFFSET ?`

	err := r.mysql.db.SelectContext(ctx, &messages, query, botID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled messages: %w", err)
	}
	return messages, nil
}

// GetScheduledMessagesCount returns count of scheduled messages for a bot
func (r *Repository) GetScheduledMessagesCount(ctx context.Context, botID int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM scheduled_messages WHERE bot_id = ? AND status IN ('pending', 'paused')`
	err := r.mysql.db.GetContext(ctx, &count, query, botID)
	if err != nil {
		return 0, fmt.Errorf("failed to get scheduled messages count: %w", err)
	}
	return count, nil
}

// GetPendingScheduledMessages retrieves messages ready to be sent
func (r *Repository) GetPendingScheduledMessages(ctx context.Context, beforeTime time.Time, limit int) ([]models.ScheduledMessage, error) {
	var messages []models.ScheduledMessage
	query := `SELECT * FROM scheduled_messages
		WHERE status = 'pending'
		AND next_run_at IS NOT NULL
		AND next_run_at <= ?
		ORDER BY next_run_at ASC LIMIT ?`

	err := r.mysql.db.SelectContext(ctx, &messages, query, beforeTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending scheduled messages: %w", err)
	}
	return messages, nil
}

// UpdateScheduledMessageAfterSend updates message after sending
func (r *Repository) UpdateScheduledMessageAfterSend(ctx context.Context, msgID int64, lastSent time.Time, nextRun *time.Time) error {
	query := `UPDATE scheduled_messages
		SET last_sent_at = ?, next_run_at = ?, updated_at = NOW()
		WHERE id = ?`

	_, err := r.mysql.db.ExecContext(ctx, query, lastSent, nextRun, msgID)
	if err != nil {
		return fmt.Errorf("failed to update scheduled message: %w", err)
	}
	return nil
}

// UpdateScheduledMessageStatus updates the status of a message
func (r *Repository) UpdateScheduledMessageStatus(ctx context.Context, msgID int64, status, failureReason string) error {
	query := `UPDATE scheduled_messages
		SET status = ?, failure_reason = ?, updated_at = NOW()
		WHERE id = ?`

	_, err := r.mysql.db.ExecContext(ctx, query, status, failureReason, msgID)
	if err != nil {
		return fmt.Errorf("failed to update message status: %w", err)
	}
	return nil
}

// PauseScheduledMessage pauses a scheduled message
func (r *Repository) PauseScheduledMessage(ctx context.Context, msgID, botID int64) error {
	query := `UPDATE scheduled_messages
		SET status = 'paused', updated_at = NOW()
		WHERE id = ? AND bot_id = ? AND status = 'pending'`

	_, err := r.mysql.db.ExecContext(ctx, query, msgID, botID)
	if err != nil {
		return fmt.Errorf("failed to pause message: %w", err)
	}
	return nil
}

// ResumeScheduledMessage resumes a paused message
func (r *Repository) ResumeScheduledMessage(ctx context.Context, msgID, botID int64) error {
	query := `UPDATE scheduled_messages
		SET status = 'pending', updated_at = NOW()
		WHERE id = ? AND bot_id = ? AND status = 'paused'`

	_, err := r.mysql.db.ExecContext(ctx, query, msgID, botID)
	if err != nil {
		return fmt.Errorf("failed to resume message: %w", err)
	}
	return nil
}

// DeleteScheduledMessage cancels a scheduled message
func (r *Repository) DeleteScheduledMessage(ctx context.Context, msgID, botID int64) error {
	query := `UPDATE scheduled_messages
		SET status = 'cancelled', updated_at = NOW()
		WHERE id = ? AND bot_id = ?`

	_, err := r.mysql.db.ExecContext(ctx, query, msgID, botID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	return nil
}

// GetScheduledMessage retrieves a single scheduled message by ID
func (r *Repository) GetScheduledMessage(ctx context.Context, msgID int64) (*models.ScheduledMessage, error) {
	var msg models.ScheduledMessage
	query := `SELECT * FROM scheduled_messages WHERE id = ?`

	err := r.mysql.db.GetContext(ctx, &msg, query, msgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get scheduled message: %w", err)
	}
	return &msg, nil
}

// ==================== Forced Channel Subscription Methods ====================

// CreateForcedChannel adds a new forced subscription channel
func (r *Repository) CreateForcedChannel(ctx context.Context, botID, channelID int64, username, title, inviteLink string) error {
	query := `INSERT INTO forced_channels (bot_id, channel_id, channel_username, channel_title, invite_link, is_active)
			  VALUES (?, ?, ?, ?, ?, TRUE)
			  ON DUPLICATE KEY UPDATE channel_username = ?, channel_title = ?, invite_link = ?, is_active = TRUE`

	_, err := r.mysql.db.ExecContext(ctx, query, botID, channelID, username, title, inviteLink, username, title, inviteLink)
	if err != nil {
		return fmt.Errorf("failed to create forced channel: %w", err)
	}
	return nil
}

// GetForcedChannels retrieves all active forced channels for a bot
func (r *Repository) GetForcedChannels(ctx context.Context, botID int64) ([]models.ForcedChannel, error) {
	var channels []models.ForcedChannel
	query := `SELECT id, bot_id, channel_id, COALESCE(channel_username, '') as channel_username,
			  COALESCE(channel_title, '') as channel_title, COALESCE(invite_link, '') as invite_link,
			  is_active, created_at
			  FROM forced_channels WHERE bot_id = ? AND is_active = TRUE
			  ORDER BY created_at ASC`

	err := r.mysql.db.SelectContext(ctx, &channels, query, botID)
	if err != nil {
		return nil, fmt.Errorf("failed to get forced channels: %w", err)
	}
	return channels, nil
}

// GetForcedChannel retrieves a single forced channel by bot and channel ID
func (r *Repository) GetForcedChannel(ctx context.Context, botID, channelID int64) (*models.ForcedChannel, error) {
	var channel models.ForcedChannel
	query := `SELECT id, bot_id, channel_id, COALESCE(channel_username, '') as channel_username,
			  COALESCE(channel_title, '') as channel_title, COALESCE(invite_link, '') as invite_link,
			  is_active, created_at
			  FROM forced_channels WHERE bot_id = ? AND channel_id = ?`

	err := r.mysql.db.GetContext(ctx, &channel, query, botID, channelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get forced channel: %w", err)
	}
	return &channel, nil
}

// DeleteForcedChannel removes a channel from forced subscription list
func (r *Repository) DeleteForcedChannel(ctx context.Context, botID, channelID int64) error {
	query := `DELETE FROM forced_channels WHERE bot_id = ? AND channel_id = ?`
	_, err := r.mysql.db.ExecContext(ctx, query, botID, channelID)
	if err != nil {
		return fmt.Errorf("failed to delete forced channel: %w", err)
	}
	return nil
}

// GetForcedChannelCount returns count of active forced channels for a bot
func (r *Repository) GetForcedChannelCount(ctx context.Context, botID int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM forced_channels WHERE bot_id = ? AND is_active = TRUE`
	err := r.mysql.db.GetContext(ctx, &count, query, botID)
	if err != nil {
		return 0, fmt.Errorf("failed to get forced channel count: %w", err)
	}
	return count, nil
}

// UpdateForcedSubEnabled toggles the forced subscription feature for a bot
func (r *Repository) UpdateForcedSubEnabled(ctx context.Context, botID int64, enabled bool) error {
	query := `UPDATE bots SET forced_sub_enabled = ? WHERE id = ?`
	_, err := r.mysql.db.ExecContext(ctx, query, enabled, botID)
	if err != nil {
		return fmt.Errorf("failed to update forced_sub_enabled: %w", err)
	}
	return nil
}

// UpdateForcedSubMessage updates the custom message for non-subscribers
func (r *Repository) UpdateForcedSubMessage(ctx context.Context, botID int64, message string) error {
	query := `UPDATE bots SET forced_sub_message = ? WHERE id = ?`
	_, err := r.mysql.db.ExecContext(ctx, query, message, botID)
	if err != nil {
		return fmt.Errorf("failed to update forced_sub_message: %w", err)
	}
	return nil
}
