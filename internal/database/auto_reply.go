package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Amr-9/botforge/internal/models"
)

// ==================== Auto-Reply Functions ====================

// CreateAutoReply creates a new auto-reply or custom command with optional media support
func (r *Repository) CreateAutoReply(ctx context.Context, botID int64, trigger, response, messageType, fileID, caption, triggerType, matchType string) error {
	query := `INSERT INTO auto_replies (bot_id, trigger_word, response, message_type, file_id, caption, trigger_type, match_type, is_active)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE)
			  ON DUPLICATE KEY UPDATE response = ?, message_type = ?, file_id = ?, caption = ?, match_type = ?, is_active = TRUE`

	_, err := r.mysql.db.ExecContext(ctx, query,
		botID, trigger, response, messageType, fileID, caption, triggerType, matchType,
		response, messageType, fileID, caption, matchType)
	if err != nil {
		return fmt.Errorf("failed to create auto-reply: %w", err)
	}
	return nil
}

// GetAutoReplies retrieves all auto-replies or commands for a bot
func (r *Repository) GetAutoReplies(ctx context.Context, botID int64, triggerType string) ([]models.AutoReply, error) {
	var replies []models.AutoReply
	query := `SELECT id, bot_id, trigger_word, response, message_type, file_id, caption, trigger_type, match_type, is_active, created_at
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
	query := `SELECT id, bot_id, trigger_word, response, message_type, file_id, caption, trigger_type, match_type, is_active, created_at
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
	query := `SELECT id, bot_id, trigger_word, response, message_type, file_id, caption, trigger_type, match_type, is_active, created_at
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
