package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Amr-9/botforge/internal/models"
)

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
