package models

import "time"

// Bot represents a child bot registered by a user
type Bot struct {
	ID                   int64     `db:"id"`
	Token                string    `db:"token"`
	Username             string    `db:"username"`
	OwnerChatID          int64     `db:"owner_chat_id"`
	IsActive             bool      `db:"is_active"`
	StartMessage         string    `db:"start_message"`
	ForwardAutoReplies   bool      `db:"forward_auto_replies"`   // Forward auto-replied messages to admin
	ForcedSubEnabled     bool      `db:"forced_sub_enabled"`     // Enable forced channel subscription
	ForcedSubMessage     string    `db:"forced_sub_message"`     // Custom message for non-subscribers
	ShowSentConfirmation bool      `db:"show_sent_confirmation"` // Show "Message sent successfully" to admin
	CreatedAt            time.Time `db:"created_at"`
}

// MessageLog stores the mapping between admin message and user chat
type MessageLog struct {
	ID         int64     `db:"id"`
	AdminMsgID int       `db:"admin_msg_id"`
	UserChatID int64     `db:"user_chat_id"`
	BotID      int64     `db:"bot_id"`
	CreatedAt  time.Time `db:"created_at"`
}

// BannedUser represents a banned user for a specific bot
type BannedUser struct {
	ID         int64     `db:"id"`
	BotID      int64     `db:"bot_id"`
	UserChatID int64     `db:"user_chat_id"`
	BannedBy   int64     `db:"banned_by"`
	CreatedAt  time.Time `db:"created_at"`
}

// ForcedChannel represents a channel that users must subscribe to
type ForcedChannel struct {
	ID              int64     `db:"id"`
	BotID           int64     `db:"bot_id"`
	ChannelID       int64     `db:"channel_id"`
	ChannelUsername string    `db:"channel_username"` // For public channels (@username)
	ChannelTitle    string    `db:"channel_title"`
	InviteLink      string    `db:"invite_link"` // For private channels
	IsActive        bool      `db:"is_active"`
	CreatedAt       time.Time `db:"created_at"`
}

// AutoReply represents an auto-reply rule or custom command for a bot
type AutoReply struct {
	ID          int64     `db:"id"`
	BotID       int64     `db:"bot_id"`
	TriggerWord string    `db:"trigger_word"` // Keyword or command name (without /)
	Response    string    `db:"response"`     // Response text (supports Markdown) - used for text type
	MessageType string    `db:"message_type"` // "text", "photo", "video", "audio", "voice", "document", "animation", "video_note", "sticker"
	FileID      string    `db:"file_id"`      // Telegram FileID for media
	Caption     string    `db:"caption"`      // Caption for media (supports Markdown)
	TriggerType string    `db:"trigger_type"` // "keyword" or "command"
	MatchType   string    `db:"match_type"`   // "exact" or "contains" (for keywords)
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
}

// ScheduledMessage represents a scheduled broadcast message
type ScheduledMessage struct {
	ID            int64      `db:"id"`
	BotID         int64      `db:"bot_id"`
	OwnerChatID   int64      `db:"owner_chat_id"`
	MessageType   string     `db:"message_type"`
	MessageText   string     `db:"message_text"`
	FileID        string     `db:"file_id"`
	Caption       string     `db:"caption"`
	ScheduleType  string     `db:"schedule_type"`
	ScheduledTime time.Time  `db:"scheduled_time"`
	TimeOfDay     string     `db:"time_of_day"`
	DayOfWeek     *int       `db:"day_of_week"`
	Status        string     `db:"status"`
	LastSentAt    *time.Time `db:"last_sent_at"`
	NextRunAt     *time.Time `db:"next_run_at"`
	FailureReason *string    `db:"failure_reason"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}

// Schedule type constants
const (
	ScheduleTypeOnce   = "once"
	ScheduleTypeDaily  = "daily"
	ScheduleTypeWeekly = "weekly"
)

// Message type constants
const (
	MessageTypeText      = "text"
	MessageTypePhoto     = "photo"
	MessageTypeVideo     = "video"
	MessageTypeDocument  = "document"
	MessageTypeAudio     = "audio"
	MessageTypeVoice     = "voice"
	MessageTypeAnimation = "animation"
	MessageTypeVideoNote = "video_note"
	MessageTypeSticker   = "sticker"
)

// Schedule status constants
const (
	ScheduleStatusPending   = "pending"
	ScheduleStatusSent      = "sent"
	ScheduleStatusFailed    = "failed"
	ScheduleStatusPaused    = "paused"
	ScheduleStatusCancelled = "cancelled"
)
