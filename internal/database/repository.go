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
	query := `INSERT INTO bots (token, owner_chat_id, is_active) VALUES (?, ?, TRUE)`

	result, err := r.mysql.db.ExecContext(ctx, query, token, ownerChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &models.Bot{
		ID:          id,
		Token:       token,
		OwnerChatID: ownerChatID,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}, nil
}

// GetBotByToken retrieves a bot by its token
func (r *Repository) GetBotByToken(ctx context.Context, token string) (*models.Bot, error) {
	var bot models.Bot
	query := `SELECT id, token, owner_chat_id, is_active, created_at FROM bots WHERE token = ?`

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
	query := `SELECT id, token, owner_chat_id, is_active, created_at FROM bots WHERE is_active = TRUE`

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
func (r *Repository) SaveMessageLog(ctx context.Context, adminMsgID int, userChatID int64, botToken string) error {
	query := `INSERT INTO message_logs (admin_msg_id, user_chat_id, bot_token) VALUES (?, ?, ?)`

	_, err := r.mysql.db.ExecContext(ctx, query, adminMsgID, userChatID, botToken)
	if err != nil {
		return fmt.Errorf("failed to save message log: %w", err)
	}

	return nil
}

// GetUserChatID retrieves the user chat ID for a given admin message
func (r *Repository) GetUserChatID(ctx context.Context, adminMsgID int, botToken string) (int64, error) {
	var userChatID int64
	query := `SELECT user_chat_id FROM message_logs WHERE admin_msg_id = ? AND bot_token = ? LIMIT 1`

	err := r.mysql.db.GetContext(ctx, &userChatID, query, adminMsgID, botToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get user chat id: %w", err)
	}

	return userChatID, nil
}

// GetBotsByOwner retrieves all bots owned by a specific user
func (r *Repository) GetBotsByOwner(ctx context.Context, ownerChatID int64) ([]models.Bot, error) {
	var bots []models.Bot
	query := `SELECT id, token, owner_chat_id, is_active, created_at FROM bots WHERE owner_chat_id = ?`

	err := r.mysql.db.SelectContext(ctx, &bots, query, ownerChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bots by owner: %w", err)
	}

	return bots, nil
}
