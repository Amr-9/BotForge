package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Amr-9/botforge/internal/models"
	"github.com/Amr-9/botforge/internal/utils/crypto"
)

// ==================== Bot Functions ====================

// CreateBot inserts a new bot into the database
func (r *Repository) CreateBot(ctx context.Context, token string, username string, ownerChatID int64) (*models.Bot, error) {
	encryptedToken, err := crypto.EncryptDeterministic(token, r.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	query := `INSERT INTO bots (token, username, owner_chat_id, is_active, start_message) VALUES (?, ?, ?, TRUE, '')`

	result, err := r.mysql.db.ExecContext(ctx, query, encryptedToken, username, ownerChatID)
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
		Username:     username,
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
	query := `SELECT id, token, COALESCE(username, '') as username, owner_chat_id, is_active, COALESCE(start_message, '') as start_message,
			  COALESCE(forward_auto_replies, TRUE) as forward_auto_replies,
			  COALESCE(forced_sub_enabled, FALSE) as forced_sub_enabled,
			  COALESCE(forced_sub_message, '') as forced_sub_message,
			  COALESCE(show_sent_confirmation, TRUE) as show_sent_confirmation, created_at
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
	query := `SELECT id, token, COALESCE(username, '') as username, owner_chat_id, is_active, COALESCE(start_message, '') as start_message, created_at
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
func (r *Repository) RestoreBot(ctx context.Context, token string, username string, ownerChatID int64) error {
	encryptedToken, err := crypto.EncryptDeterministic(token, r.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	query := `UPDATE bots SET deleted_at = NULL, is_active = TRUE, owner_chat_id = ?, username = ? WHERE token = ?`

	_, err = r.mysql.db.ExecContext(ctx, query, ownerChatID, username, encryptedToken)
	if err != nil {
		return fmt.Errorf("failed to restore bot: %w", err)
	}

	return nil
}

// GetAllBots retrieves all non-deleted bots (both active and inactive)
func (r *Repository) GetAllBots(ctx context.Context) ([]models.Bot, error) {
	var bots []models.Bot
	query := `SELECT id, token, COALESCE(username, '') as username, owner_chat_id, is_active, COALESCE(start_message, '') as start_message, created_at
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
	query := `SELECT id, token, COALESCE(username, '') as username, owner_chat_id, is_active, COALESCE(start_message, '') as start_message, created_at
			  FROM bots WHERE is_active = TRUE AND deleted_at IS NULL`

	err := r.mysql.db.SelectContext(ctx, &bots, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active bots: %w", err)
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

// UpdateBotShowSentConfirmation updates the show_sent_confirmation setting for a bot
func (r *Repository) UpdateBotShowSentConfirmation(ctx context.Context, botID int64, show bool) error {
	query := `UPDATE bots SET show_sent_confirmation = ? WHERE id = ?`

	_, err := r.mysql.db.ExecContext(ctx, query, show, botID)
	if err != nil {
		return fmt.Errorf("failed to update show_sent_confirmation: %w", err)
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

// GetBotsByOwner retrieves all bots owned by a specific user (excludes soft-deleted)
func (r *Repository) GetBotsByOwner(ctx context.Context, ownerChatID int64) ([]models.Bot, error) {
	var bots []models.Bot
	query := `SELECT id, token, COALESCE(username, '') as username, owner_chat_id, is_active, COALESCE(start_message, '') as start_message, created_at
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
