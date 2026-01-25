package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Amr-9/botforge/internal/models"
)

// Repository handles all database operations
type Repository struct {
	mysql *MySQL
}

// NewRepository creates a new repository instance
func NewRepository(mysql *MySQL) *Repository {
	return &Repository{mysql: mysql}
}

// CreateBot inserts a new bot into the database
func (r *Repository) CreateBot(ctx context.Context, token string, ownerChatID int64) (*models.Bot, error) {
	query := `INSERT INTO bots (token, owner_chat_id, is_active, start_message) VALUES (?, ?, TRUE, '')`

	result, err := r.mysql.db.ExecContext(ctx, query, token, ownerChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &models.Bot{
		ID:           id,
		Token:        token,
		OwnerChatID:  ownerChatID,
		IsActive:     true,
		StartMessage: "",
		CreatedAt:    time.Now(),
	}, nil
}

// GetBotByToken retrieves a bot by its token
func (r *Repository) GetBotByToken(ctx context.Context, token string) (*models.Bot, error) {
	var bot models.Bot
	query := `SELECT id, token, owner_chat_id, is_active, start_message, created_at FROM bots WHERE token = ?`

	err := r.mysql.db.GetContext(ctx, &bot, query, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get bot: %w", err)
	}

	return &bot, nil
}

// GetActiveBots retrieves all active bots
func (r *Repository) GetActiveBots(ctx context.Context) ([]models.Bot, error) {
	var bots []models.Bot
	query := `SELECT id, token, owner_chat_id, is_active, start_message, created_at FROM bots WHERE is_active = TRUE`

	err := r.mysql.db.SelectContext(ctx, &bots, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active bots: %w", err)
	}

	return bots, nil
}

// DeactivateBot sets is_active to false for a bot
func (r *Repository) DeactivateBot(ctx context.Context, token string) error {
	query := `UPDATE bots SET is_active = FALSE WHERE token = ?`

	_, err := r.mysql.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to deactivate bot: %w", err)
	}

	return nil
}

// ActivateBot sets is_active to true for a bot
func (r *Repository) ActivateBot(ctx context.Context, token string) error {
	query := `UPDATE bots SET is_active = TRUE WHERE token = ?`

	_, err := r.mysql.db.ExecContext(ctx, query, token)
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

// DeleteBot removes a bot from the database
func (r *Repository) DeleteBot(ctx context.Context, token string) error {
	query := `DELETE FROM bots WHERE token = ?`

	_, err := r.mysql.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete bot: %w", err)
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

// GetBotsByOwner retrieves all bots owned by a specific user
func (r *Repository) GetBotsByOwner(ctx context.Context, ownerChatID int64) ([]models.Bot, error) {
	var bots []models.Bot
	query := `SELECT id, token, owner_chat_id, is_active, start_message, created_at FROM bots WHERE owner_chat_id = ?`

	err := r.mysql.db.SelectContext(ctx, &bots, query, ownerChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bots by owner: %w", err)
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
