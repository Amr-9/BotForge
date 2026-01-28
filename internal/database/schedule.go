package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Amr-9/botforge/internal/models"
)

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
